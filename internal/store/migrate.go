package store

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version integer PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		var already bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&already); err != nil {
			return err
		}
		if already {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return s.EnsureDefaultOrg(ctx)
}

func migrationVersion(name string) (int, error) {
	prefix, _, _ := strings.Cut(name, "_")
	n, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration name %s", name)
	}
	return n, nil
}

// ResetForTests clears product data. It is used by tests against a disposable database.
func (s *Store) ResetForTests(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		TRUNCATE TABLE audit_events, approvals, tool_calls, tasks, policies, tools, agents, org_admin_tokens
		RESTART IDENTITY CASCADE`)
	if err != nil {
		return err
	}
	return s.EnsureDefaultOrg(ctx)
}

// Pool exposes the connection pool for tests that tamper with rows directly.
func (s *Store) Pool() *pgxpool.Pool { return s.pool }
