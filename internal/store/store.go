package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rejisterjack/keywarden/internal/fail"
	"github.com/rejisterjack/keywarden/internal/policy"
	"github.com/rejisterjack/keywarden/internal/secret"
	"github.com/rejisterjack/keywarden/internal/slug"
	"github.com/rejisterjack/keywarden/internal/token"
	"github.com/rejisterjack/keywarden/internal/vault"
)

const (
	defaultTaskTTL = 15 * time.Minute
	maxTaskTTL     = 24 * time.Hour
)

// Options opens a store against one deployment.
type Options struct {
	DatabaseURL  string
	Vault        *vault.Vault
	Demo         bool
	AllowStdio   bool
	FixtureURL   string
	OrdersDSN    string
	SingleTenant bool
}

// Store is the Postgres control plane.
type Store struct {
	pool         *pgxpool.Pool
	vault        *vault.Vault
	demo         bool
	allowStdio   bool
	fixtureURL   string
	ordersDSN    string
	singleTenant bool
	defaultOrgID uuid.UUID
}

// Open connects and checks the database.
func Open(ctx context.Context, opt Options) (*Store, error) {
	if opt.Vault == nil {
		return nil, fmt.Errorf("vault is required")
	}
	cfg, err := pgxpool.ParseConfig(opt.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database url: %w", err)
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	ping, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ping); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{
		pool:         pool,
		vault:        opt.Vault,
		demo:         opt.Demo,
		allowStdio:   opt.AllowStdio,
		fixtureURL:   opt.FixtureURL,
		ordersDSN:    opt.OrdersDSN,
		singleTenant: opt.SingleTenant,
		defaultOrgID: DefaultOrgID,
	}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping reports whether the database accepts queries.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// CreateAgent inserts an agent. The id is the policy match key.
func (s *Store) CreateAgent(ctx context.Context, id, name string) (Agent, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return Agent{}, err
	}
	if !slug.OK(id) {
		return Agent{}, fail.New(400, "agent id must be a lowercase slug")
	}
	if name == "" || len(name) > 200 {
		return Agent{}, fail.New(400, "agent name is required")
	}
	var a Agent
	err = s.pool.QueryRow(ctx, `
		INSERT INTO agents (org_id, id, name) VALUES ($1, $2, $3)
		RETURNING id, name, created_at`, orgID, id, name).Scan(&a.ID, &a.Name, &a.CreatedAt)
	if err != nil {
		return Agent{}, mapWrite(err, "agent already exists")
	}
	if err := s.AppendAudit(ctx, "agent.created", map[string]any{"agent_id": a.ID, "name": a.Name}); err != nil {
		return Agent{}, err
	}
	return a, nil
}

// ListAgents returns every agent in the active organization.
func (s *Store) ListAgents(ctx context.Context) ([]Agent, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, created_at FROM agents WHERE org_id = $1 ORDER BY id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Agent{}
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAgent loads one agent.
func (s *Store) GetAgent(ctx context.Context, id string) (Agent, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return Agent{}, err
	}
	var a Agent
	err = s.pool.QueryRow(ctx, `SELECT id, name, created_at FROM agents WHERE org_id = $1 AND id = $2`, orgID, id).Scan(&a.ID, &a.Name, &a.CreatedAt)
	if err != nil {
		return Agent{}, mapRead(err, "agent not found")
	}
	return a, nil
}

// DeleteAgent removes an agent that has no tasks.
func (s *Store) DeleteAgent(ctx context.Context, id string) error {
	orgID, err := s.org(ctx)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM agents WHERE org_id = $1 AND id = $2`, orgID, id)
	if err != nil {
		return mapWrite(err, "agent is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fail.New(404, "agent not found")
	}
	return s.AppendAudit(ctx, "agent.deleted", map[string]any{"agent_id": id})
}

// CreateToolInput is a new upstream.
type CreateToolInput struct {
	ID     string
	Name   string
	Kind   string
	Config json.RawMessage
	Secret json.RawMessage
	Seeded bool
}

// CreateTool seals the secret and stores the tool. The secret is not returned.
func (s *Store) CreateTool(ctx context.Context, in CreateToolInput) (ToolPublic, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return ToolPublic{}, err
	}
	if !slug.OK(in.ID) {
		return ToolPublic{}, fail.New(400, "tool id must be a lowercase slug")
	}
	if in.Name == "" || len(in.Name) > 200 {
		return ToolPublic{}, fail.New(400, "tool name is required")
	}
	if in.Kind == "mcp_stdio" && !s.allowStdio {
		return ToolPublic{}, fail.New(400, "stdio tools are disabled")
	}
	if len(bytesTrim(in.Config)) == 0 {
		in.Config = []byte(`{}`)
	}
	if err := secret.Validate(in.Kind, in.Config, in.Secret, s.demo, s.fixtureURL, s.ordersDSN); err != nil {
		return ToolPublic{}, err
	}
	ct, nonce, err := s.vault.Seal(in.ID, in.Secret)
	if err != nil {
		return ToolPublic{}, err
	}
	var pub ToolPublic
	var cfg []byte
	err = s.pool.QueryRow(ctx, `
		INSERT INTO tools (org_id, id, name, kind, config, secret_ciphertext, secret_nonce, seeded)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8)
		RETURNING id, name, kind, config, seeded, created_at`,
		orgID, in.ID, in.Name, in.Kind, string(in.Config), ct, nonce, in.Seeded,
	).Scan(&pub.ID, &pub.Name, &pub.Kind, &cfg, &pub.Seeded, &pub.CreatedAt)
	if err != nil {
		return ToolPublic{}, mapWrite(err, "tool already exists")
	}
	pub.Config = cfg
	if err := s.AppendAudit(ctx, "tool.created", map[string]any{
		"tool_id": pub.ID,
		"name":    pub.Name,
		"kind":    pub.Kind,
		"seeded":  pub.Seeded,
	}); err != nil {
		return ToolPublic{}, err
	}
	return pub, nil
}

// ListTools returns tools without secrets.
func (s *Store) ListTools(ctx context.Context) ([]ToolPublic, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, kind, config, seeded, created_at FROM tools WHERE org_id = $1 ORDER BY id`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ToolPublic{}
	for rows.Next() {
		pub, err := scanToolPublic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pub)
	}
	return out, rows.Err()
}

// GetTool loads a tool, including the sealed secret for proxy use.
func (s *Store) GetTool(ctx context.Context, id string) (Tool, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return Tool{}, err
	}
	var tool Tool
	var cfg []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, name, kind, config, secret_ciphertext, secret_nonce, seeded, created_at
		FROM tools WHERE org_id = $1 AND id = $2`, orgID, id).Scan(
		&tool.ID, &tool.Name, &tool.Kind, &cfg, &tool.Ciphertext, &tool.Nonce, &tool.Seeded, &tool.CreatedAt)
	if err != nil {
		return Tool{}, mapRead(err, "tool not found")
	}
	tool.Config = cfg
	return tool, nil
}

// DeleteTool removes a tool. Seeded demo tools stay in place.
func (s *Store) DeleteTool(ctx context.Context, id string) error {
	orgID, err := s.org(ctx)
	if err != nil {
		return err
	}
	if s.demo {
		var seeded bool
		err := s.pool.QueryRow(ctx, `SELECT seeded FROM tools WHERE org_id = $1 AND id = $2`, orgID, id).Scan(&seeded)
		if err != nil {
			return mapRead(err, "tool not found")
		}
		if seeded {
			return fail.New(403, "seeded demo tools cannot be deleted")
		}
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM tools WHERE org_id = $1 AND id = $2`, orgID, id)
	if err != nil {
		return mapWrite(err, "tool is still referenced")
	}
	if tag.RowsAffected() == 0 {
		return fail.New(404, "tool not found")
	}
	return s.AppendAudit(ctx, "tool.deleted", map[string]any{"tool_id": id})
}

// OpenSecret decrypts a tool secret. Demo mode refuses anything outside the fake set.
func (s *Store) OpenSecret(tool Tool) ([]byte, error) {
	plain, err := s.vault.Open(tool.ID, tool.Ciphertext, tool.Nonce)
	if err != nil {
		return nil, fail.New(500, "stored secret could not be opened")
	}
	if err := secret.Validate(tool.Kind, tool.Config, plain, s.demo, s.fixtureURL, s.ordersDSN); err != nil {
		return nil, err
	}
	return plain, nil
}

// AssertDemoSecrets fails startup when any stored secret is outside the fake set.
func (s *Store) AssertDemoSecrets(ctx context.Context) error {
	if !s.demo {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, kind, config, secret_ciphertext, secret_nonce, seeded, created_at
		FROM tools`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tool Tool
		var cfg []byte
		if err := rows.Scan(&tool.ID, &tool.Name, &tool.Kind, &cfg, &tool.Ciphertext, &tool.Nonce, &tool.Seeded, &tool.CreatedAt); err != nil {
			return err
		}
		tool.Config = cfg
		if _, err := s.OpenSecret(tool); err != nil {
			return fmt.Errorf("tool %s is not in the demo set", tool.ID)
		}
	}
	return rows.Err()
}

// ApplyPolicy stores a new active version and retires the previous one.
func (s *Store) ApplyPolicy(ctx context.Context, raw []byte) (Policy, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return Policy{}, err
	}
	doc, canon, err := policy.Parse(raw)
	if err != nil {
		return Policy{}, fail.New(400, err.Error())
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Policy{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE policies SET active = false WHERE org_id = $1 AND active`, orgID); err != nil {
		return Policy{}, err
	}
	var version int
	if err := tx.QueryRow(ctx, `SELECT coalesce(max(version), 0) + 1 FROM policies WHERE org_id = $1`, orgID).Scan(&version); err != nil {
		return Policy{}, err
	}
	var p Policy
	var document string
	err = tx.QueryRow(ctx, `
		INSERT INTO policies (id, org_id, version, document, active)
		VALUES ($1, $2, $3, $4::jsonb, true)
		RETURNING id, version, document, active, created_at`,
		uuid.New(), orgID, version, string(canon),
	).Scan(&p.ID, &p.Version, &document, &p.Active, &p.CreatedAt)
	if err != nil {
		return Policy{}, err
	}
	p.Document = json.RawMessage(document)
	var decoded any
	if err := json.Unmarshal(canon, &decoded); err != nil {
		return Policy{}, err
	}
	if err := s.appendAuditTx(ctx, tx, "policy.applied", map[string]any{
		"policy_id": p.ID.String(),
		"version":   version,
		"document":  decoded,
	}); err != nil {
		return Policy{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Policy{}, err
	}
	_ = doc
	return p, nil
}

// ActivePolicy returns the current document.
func (s *Store) ActivePolicy(ctx context.Context) (Policy, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	var document string
	err = s.pool.QueryRow(ctx, `
		SELECT id, version, document, active, created_at
		FROM policies WHERE org_id = $1 AND active`, orgID).Scan(&p.ID, &p.Version, &document, &p.Active, &p.CreatedAt)
	if err != nil {
		return Policy{}, mapRead(err, "no active policy")
	}
	p.Document = json.RawMessage(document)
	return p, nil
}

// ActiveDocument parses the active policy, or an empty deny-all document.
func (s *Store) ActiveDocument(ctx context.Context) (policy.Document, error) {
	p, err := s.ActivePolicy(ctx)
	if fail.IsStatus(err, 404) {
		return policy.Document{Version: 1}, nil
	}
	if err != nil {
		return policy.Document{}, err
	}
	doc, _, err := policy.Parse(p.Document)
	if err != nil {
		return policy.Document{}, fmt.Errorf("stored policy is invalid: %w", err)
	}
	return doc, nil
}

// ListPolicies returns versions, newest first.
func (s *Store) ListPolicies(ctx context.Context) ([]Policy, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, version, document, active, created_at
		FROM policies WHERE org_id = $1 ORDER BY version DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Policy{}
	for rows.Next() {
		var p Policy
		var document string
		if err := rows.Scan(&p.ID, &p.Version, &document, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Document = json.RawMessage(document)
		out = append(out, p)
	}
	return out, rows.Err()
}

// IssueTask creates a task and returns the plaintext token once.
func (s *Store) IssueTask(ctx context.Context, agentID string, toolIDs []string, ttl time.Duration) (Task, string, error) {
	if _, err := s.GetAgent(ctx, agentID); err != nil {
		return Task{}, "", err
	}
	if ttl == 0 {
		ttl = defaultTaskTTL
	}
	if ttl < time.Second || ttl > maxTaskTTL {
		return Task{}, "", fail.New(400, "ttl must be between 1 second and 24 hours")
	}
	unrestricted := toolIDs == nil
	if !unrestricted {
		if len(toolIDs) == 0 {
			return Task{}, "", fail.New(400, "tool_ids is empty; omit it to allow every tool")
		}
		for _, id := range toolIDs {
			if _, err := s.GetTool(ctx, id); err != nil {
				return Task{}, "", err
			}
		}
	}
	raw, hash, err := token.NewToken()
	if err != nil {
		return Task{}, "", err
	}
	var encoded *string
	if !unrestricted {
		b, err := json.Marshal(toolIDs)
		if err != nil {
			return Task{}, "", err
		}
		text := string(b)
		encoded = &text
	}
	orgID, err := s.org(ctx)
	if err != nil {
		return Task{}, "", err
	}
	var task Task
	var tools []byte
	err = s.pool.QueryRow(ctx, `
		INSERT INTO tasks (id, org_id, agent_id, token_hash, tool_ids, expires_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING id, agent_id, tool_ids, expires_at, revoked_at, created_at, org_id`,
		uuid.New(), orgID, agentID, hash, encoded, time.Now().UTC().Add(ttl),
	).Scan(&task.ID, &task.AgentID, &tools, &task.ExpiresAt, &task.RevokedAt, &task.CreatedAt, &task.OrgID)
	if err != nil {
		return Task{}, "", err
	}
	if err := decodeTools(tools, &task); err != nil {
		return Task{}, "", err
	}
	ids := []any{}
	for _, id := range task.ToolIDs {
		ids = append(ids, id)
	}
	if err := s.AppendAudit(ctx, "task.issued", map[string]any{
		"task_id":      task.ID.String(),
		"agent_id":     task.AgentID,
		"tool_ids":     ids,
		"unrestricted": task.Unrestricted,
		"expires_at":   task.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return Task{}, "", err
	}
	return task, raw, nil
}

// ListTasks returns tasks without token hashes.
func (s *Store) ListTasks(ctx context.Context) ([]Task, error) {
	orgID, err := s.org(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, agent_id, tool_ids, expires_at, revoked_at, created_at, org_id
		FROM tasks WHERE org_id = $1 ORDER BY created_at DESC LIMIT 200`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

// RevokeTask invalidates a task immediately.
func (s *Store) RevokeTask(ctx context.Context, id uuid.UUID) error {
	orgID, err := s.org(ctx)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE tasks SET revoked_at = now()
		WHERE org_id = $1 AND id = $2 AND revoked_at IS NULL`, orgID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks WHERE org_id = $1 AND id = $2)`, orgID, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fail.New(404, "task not found")
		}
		return nil
	}
	return s.AppendAudit(ctx, "task.revoked", map[string]any{"task_id": id.String()})
}

// Authenticate resolves a task token. The token itself is not stored.
func (s *Store) Authenticate(ctx context.Context, raw string) (Task, error) {
	hash, err := token.Hash(raw)
	if err != nil {
		return Task{}, fail.New(401, "unauthorized")
	}
	var task Task
	var tools []byte
	err = s.pool.QueryRow(ctx, `
		SELECT id, agent_id, tool_ids, expires_at, revoked_at, created_at, org_id
		FROM tasks WHERE token_hash = $1`, hash).Scan(
		&task.ID, &task.AgentID, &tools, &task.ExpiresAt, &task.RevokedAt, &task.CreatedAt, &task.OrgID)
	if err != nil {
		return Task{}, fail.New(401, "unauthorized")
	}
	if err := decodeTools(tools, &task); err != nil {
		return Task{}, err
	}
	if task.RevokedAt != nil || !time.Now().Before(task.ExpiresAt) {
		return Task{}, fail.New(401, "unauthorized")
	}
	return task, nil
}

// Allows reports whether the task may address toolID.
func (t Task) Allows(toolID string) bool {
	if t.Unrestricted {
		return true
	}
	for _, id := range t.ToolIDs {
		if id == toolID {
			return true
		}
	}
	return false
}

func scanTask(row pgx.Row) (Task, error) {
	var task Task
	var tools []byte
	if err := row.Scan(&task.ID, &task.AgentID, &tools, &task.ExpiresAt, &task.RevokedAt, &task.CreatedAt, &task.OrgID); err != nil {
		return Task{}, err
	}
	if err := decodeTools(tools, &task); err != nil {
		return Task{}, err
	}
	return task, nil
}

func decodeTools(raw []byte, task *Task) error {
	if len(raw) == 0 {
		task.Unrestricted = true
		return nil
	}
	if err := json.Unmarshal(raw, &task.ToolIDs); err != nil {
		return err
	}
	return nil
}

func scanToolPublic(row pgx.Row) (ToolPublic, error) {
	var pub ToolPublic
	var cfg []byte
	if err := row.Scan(&pub.ID, &pub.Name, &pub.Kind, &cfg, &pub.Seeded, &pub.CreatedAt); err != nil {
		return ToolPublic{}, err
	}
	pub.Config = cfg
	return pub, nil
}

func bytesTrim(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\t' || b[i] == '\r') {
		i++
	}
	return b[i:]
}

func mapRead(err error, missing string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fail.New(404, missing)
	}
	return err
}

func mapWrite(err error, conflict string) error {
	if err == nil {
		return nil
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505":
			return fail.New(409, conflict)
		case "23503":
			return fail.New(409, conflict)
		case "23514":
			return fail.New(400, "invalid value")
		}
	}
	return err
}
