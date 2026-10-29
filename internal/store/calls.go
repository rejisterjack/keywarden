package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rejisterjack/keywarden/internal/fail"
)

// CallInput is a governed invocation to persist before any upstream side effect.
type CallInput struct {
	TaskID         uuid.UUID
	AgentID        string
	ToolID         string
	MCPToolName    string
	ArgsRedacted   json.RawMessage
	ArgsHash       []byte
	Decision       string
	RuleID         string
	Reason         string
	RequestID      string
	UpstreamStatus string
	AuditType      string
	Audit          map[string]any
}

// RecordCall inserts the call and its audit event in one transaction.
func (s *Store) RecordCall(ctx context.Context, in CallInput) (Call, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Call{}, err
	}
	defer tx.Rollback(ctx)
	call, err := insertCall(ctx, tx, in, nil)
	if err != nil {
		return Call{}, err
	}
	if err := appendAuditTx(ctx, tx, in.AuditType, in.Audit); err != nil {
		return Call{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Call{}, err
	}
	return call, nil
}

// RecordPending inserts a call, its approval, and the audit event together.
func (s *Store) RecordPending(ctx context.Context, in CallInput, timeout time.Duration) (Call, Approval, error) {
	if timeout < time.Second {
		timeout = time.Second
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Call{}, Approval{}, err
	}
	defer tx.Rollback(ctx)
	in.Decision = "pending"
	in.UpstreamStatus = ""
	approvalID := uuid.New()
	call, err := insertCall(ctx, tx, in, &approvalID)
	if err != nil {
		return Call{}, Approval{}, err
	}
	var approval Approval
	expires := time.Now().UTC().Add(timeout)
	err = tx.QueryRow(ctx, `
		INSERT INTO approvals (id, tool_call_id, args_hash, status, expires_at)
		VALUES ($1, $2, $3, 'pending', $4)
		RETURNING id, tool_call_id, args_hash, status, decided_by, comment, decided_at, expires_at, created_at`,
		approvalID, call.ID, in.ArgsHash, expires,
	).Scan(&approval.ID, &approval.ToolCallID, &approval.ArgsHash, &approval.Status, &approval.DecidedBy, &approval.Comment, &approval.DecidedAt, &approval.ExpiresAt, &approval.CreatedAt)
	if err != nil {
		return Call{}, Approval{}, err
	}
	if in.Audit == nil {
		in.Audit = map[string]any{}
	}
	in.Audit["approval_id"] = approval.ID.String()
	in.Audit["expires_at"] = approval.ExpiresAt.UTC().Format(time.RFC3339Nano)
	if err := appendAuditTx(ctx, tx, "tool_call.pending", in.Audit); err != nil {
		return Call{}, Approval{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Call{}, Approval{}, err
	}
	call.ApprovalID = &approval.ID
	return call, approval, nil
}

func insertCall(ctx context.Context, tx pgx.Tx, in CallInput, approvalID *uuid.UUID) (Call, error) {
	var call Call
	err := tx.QueryRow(ctx, `
		INSERT INTO tool_calls (
			id, task_id, agent_id, tool_id, mcp_tool_name, args_redacted, args_hash,
			decision, policy_rule_id, reason, approval_id, request_id, upstream_status
		) VALUES (
			$1, $2, $3, $4, $5, $6::jsonb, $7,
			$8, $9, $10, $11, $12, $13
		) RETURNING id, task_id, agent_id, tool_id, mcp_tool_name, args_redacted, args_hash,
			decision, policy_rule_id, reason, approval_id, request_id, upstream_status, created_at`,
		uuid.New(), in.TaskID, in.AgentID, in.ToolID, in.MCPToolName, string(in.ArgsRedacted), in.ArgsHash,
		in.Decision, in.RuleID, in.Reason, approvalID, in.RequestID, in.UpstreamStatus,
	).Scan(
		&call.ID, &call.TaskID, &call.AgentID, &call.ToolID, &call.MCPToolName, &call.ArgsRedacted, &call.ArgsHash,
		&call.Decision, &call.PolicyRuleID, &call.Reason, &call.ApprovalID, &call.RequestID, &call.UpstreamStatus, &call.CreatedAt,
	)
	return call, err
}

// MarkRunning claims the call for an upstream attempt. A second claim fails.
func (s *Store) MarkRunning(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE tool_calls
		SET upstream_status = 'running', updated_at = now()
		WHERE id = $1 AND upstream_status = '' AND decision IN ('allow', 'approved')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fail.New(409, "call is no longer runnable")
	}
	return nil
}

// FinishCall records the upstream outcome and an audit event.
func (s *Store) FinishCall(ctx context.Context, id uuid.UUID, upstream, auditType string, payload map[string]any) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE tool_calls SET upstream_status = $2, updated_at = now() WHERE id = $1`, id, upstream); err != nil {
		return err
	}
	if err := appendAuditTx(ctx, tx, auditType, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetApproval loads one approval.
func (s *Store) GetApproval(ctx context.Context, id uuid.UUID) (Approval, error) {
	var a Approval
	err := s.pool.QueryRow(ctx, `
		SELECT id, tool_call_id, args_hash, status, decided_by, comment, decided_at, expires_at, created_at
		FROM approvals WHERE id = $1`, id).Scan(
		&a.ID, &a.ToolCallID, &a.ArgsHash, &a.Status, &a.DecidedBy, &a.Comment, &a.DecidedAt, &a.ExpiresAt, &a.CreatedAt)
	if err != nil {
		return Approval{}, mapRead(err, "approval not found")
	}
	return a, nil
}

// ListApprovals returns reviewer rows. An empty status returns every state.
func (s *Store) ListApprovals(ctx context.Context, status string) ([]ApprovalView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.tool_call_id, a.status, a.comment, a.decided_by, a.expires_at, a.decided_at, a.created_at,
			c.agent_id, c.tool_id, coalesce(t.name, c.tool_id), c.args_redacted, encode(c.args_hash, 'hex'),
			c.policy_rule_id, c.reason, c.request_id, c.decision
		FROM approvals a
		JOIN tool_calls c ON c.id = a.tool_call_id
		LEFT JOIN tools t ON t.id = c.tool_id
		WHERE ($1 = '' OR a.status = $1)
		ORDER BY a.created_at DESC
		LIMIT 200`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ApprovalView{}
	for rows.Next() {
		var v ApprovalView
		var args []byte
		if err := rows.Scan(
			&v.ID, &v.ToolCallID, &v.Status, &v.Comment, &v.DecidedBy, &v.ExpiresAt, &v.DecidedAt, &v.CreatedAt,
			&v.AgentID, &v.ToolID, &v.ToolName, &args, &v.ArgsHash,
			&v.RuleID, &v.Reason, &v.RequestID, &v.Decision,
		); err != nil {
			return nil, err
		}
		v.ArgsRedacted = args
		out = append(out, v)
	}
	return out, rows.Err()
}

// Decide resolves a pending approval. The same decision is idempotent.
// A different decision, or a decision after expiry, is rejected.
func (s *Store) Decide(ctx context.Context, id uuid.UUID, approve bool, comment, by string) (Approval, error) {
	if len(comment) > 2000 {
		return Approval{}, fail.New(400, "comment is too long")
	}
	want := "rejected"
	callDecision := "rejected"
	if approve {
		want = "approved"
		callDecision = "approved"
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Approval{}, err
	}
	defer tx.Rollback(ctx)
	var current Approval
	err = tx.QueryRow(ctx, `
		SELECT id, tool_call_id, args_hash, status, decided_by, comment, decided_at, expires_at, created_at
		FROM approvals WHERE id = $1 FOR UPDATE`, id).Scan(
		&current.ID, &current.ToolCallID, &current.ArgsHash, &current.Status, &current.DecidedBy, &current.Comment, &current.DecidedAt, &current.ExpiresAt, &current.CreatedAt)
	if err != nil {
		return Approval{}, mapRead(err, "approval not found")
	}
	if current.Status == want {
		return current, nil
	}
	if current.Status != "pending" {
		return Approval{}, fail.New(409, "approval is already "+current.Status)
	}
	if !time.Now().Before(current.ExpiresAt) {
		if err := expireTx(ctx, tx, current, "deadline passed"); err != nil {
			return Approval{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Approval{}, err
		}
		return Approval{}, fail.New(409, "approval expired")
	}
	var callHash []byte
	err = tx.QueryRow(ctx, `SELECT args_hash FROM tool_calls WHERE id = $1 FOR UPDATE`, current.ToolCallID).Scan(&callHash)
	if err != nil {
		return Approval{}, err
	}
	if !bytesEqual(callHash, current.ArgsHash) {
		return Approval{}, fail.New(409, "approval does not match the call")
	}
	err = tx.QueryRow(ctx, `
		UPDATE approvals
		SET status = $2, comment = $3, decided_by = $4, decided_at = now()
		WHERE id = $1
		RETURNING id, tool_call_id, args_hash, status, decided_by, comment, decided_at, expires_at, created_at`,
		id, want, comment, by,
	).Scan(&current.ID, &current.ToolCallID, &current.ArgsHash, &current.Status, &current.DecidedBy, &current.Comment, &current.DecidedAt, &current.ExpiresAt, &current.CreatedAt)
	if err != nil {
		return Approval{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tool_calls SET decision = $2, updated_at = now() WHERE id = $1 AND decision = 'pending'`,
		current.ToolCallID, callDecision); err != nil {
		return Approval{}, err
	}
	if err := appendAuditTx(ctx, tx, "approval.decided", map[string]any{
		"approval_id":  current.ID.String(),
		"tool_call_id": current.ToolCallID.String(),
		"status":       current.Status,
		"comment":      comment,
		"decided_by":   by,
		"args_hash":    hex.EncodeToString(current.ArgsHash),
	}); err != nil {
		return Approval{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Approval{}, err
	}
	return current, nil
}

// ExpireApproval marks a pending approval expired and returns the resulting status.
// If the approval was already decided, that status is returned and nothing is forwarded.
func (s *Store) ExpireApproval(ctx context.Context, id uuid.UUID, reason string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var current Approval
	err = tx.QueryRow(ctx, `
		SELECT id, tool_call_id, args_hash, status, decided_by, comment, decided_at, expires_at, created_at
		FROM approvals WHERE id = $1 FOR UPDATE`, id).Scan(
		&current.ID, &current.ToolCallID, &current.ArgsHash, &current.Status, &current.DecidedBy, &current.Comment, &current.DecidedAt, &current.ExpiresAt, &current.CreatedAt)
	if err != nil {
		return "", mapRead(err, "approval not found")
	}
	if current.Status != "pending" {
		return current.Status, nil
	}
	if err := expireTx(ctx, tx, current, reason); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return "expired", nil
}

func expireTx(ctx context.Context, tx pgx.Tx, current Approval, reason string) error {
	if _, err := tx.Exec(ctx, `
		UPDATE approvals SET status = 'expired', decided_at = now() WHERE id = $1 AND status = 'pending'`, current.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tool_calls SET decision = 'expired', upstream_status = 'not_called', updated_at = now()
		WHERE id = $1 AND decision = 'pending'`, current.ToolCallID); err != nil {
		return err
	}
	return appendAuditTx(ctx, tx, "approval.expired", map[string]any{
		"approval_id":  current.ID.String(),
		"tool_call_id": current.ToolCallID.String(),
		"reason":       reason,
		"args_hash":    hex.EncodeToString(current.ArgsHash),
	})
}

// InterruptInFlight expires work a previous process can no longer finish.
// It never forwards those calls.
func (s *Store) InterruptInFlight(ctx context.Context) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT c.id, c.decision, a.id
		FROM tool_calls c
		LEFT JOIN approvals a ON a.tool_call_id = c.id
		WHERE c.decision = 'pending'
		   OR (c.decision IN ('allow', 'approved') AND c.upstream_status IN ('', 'running'))
		FOR UPDATE OF c`)
	if err != nil {
		return 0, err
	}
	type item struct {
		callID     uuid.UUID
		decision   string
		approvalID *uuid.UUID
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.callID, &it.decision, &it.approvalID); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, it := range items {
		if it.approvalID != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE approvals SET status = 'expired', decided_at = now()
				WHERE id = $1 AND status = 'pending'`, *it.approvalID); err != nil {
				return 0, err
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tool_calls
			SET decision = 'expired', upstream_status = 'interrupted', updated_at = now()
			WHERE id = $1`, it.callID); err != nil {
			return 0, err
		}
		if err := appendAuditTx(ctx, tx, "tool_call.interrupted", map[string]any{
			"tool_call_id": it.callID.String(),
			"was":          it.decision,
			"reason":       "process restart",
		}); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
