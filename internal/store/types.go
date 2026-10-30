package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Agent is a caller that can be issued task tokens.
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Tool is a registered upstream, including the sealed secret.
// Ciphertext and nonce are never serialized.
type Tool struct {
	ID         string
	Name       string
	Kind       string
	Config     json.RawMessage
	Ciphertext []byte `json:"-"`
	Nonce      []byte `json:"-"`
	Seeded     bool
	CreatedAt  time.Time
}

// ToolPublic is safe to return from the control plane.
type ToolPublic struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Config    json.RawMessage `json:"config"`
	Seeded    bool            `json:"seeded"`
	CreatedAt time.Time       `json:"created_at"`
}

// Policy is one stored document. Only one row is active.
type Policy struct {
	ID        uuid.UUID       `json:"id"`
	Version   int             `json:"version"`
	Document  json.RawMessage `json:"document"`
	Active    bool            `json:"active"`
	CreatedAt time.Time       `json:"created_at"`
}

// Task is a short-lived credential for one agent.
type Task struct {
	ID           uuid.UUID  `json:"id"`
	OrgID        uuid.UUID  `json:"org_id"`
	AgentID      string     `json:"agent_id"`
	ToolIDs      []string   `json:"tool_ids"`
	Unrestricted bool       `json:"unrestricted"`
	ExpiresAt    time.Time  `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Call is one governed tool invocation.
type Call struct {
	ID             uuid.UUID
	TaskID         uuid.UUID
	AgentID        string
	ToolID         string
	MCPToolName    string
	PublicName     string
	ArgsRedacted   json.RawMessage
	ArgsHash       []byte
	Decision       string
	PolicyRuleID   string
	Reason         string
	SQLClass       string
	ApprovalID     *uuid.UUID
	RequestID      string
	UpstreamStatus string
	CreatedAt      time.Time
}

// Approval is a human decision bound to an argument hash.
type Approval struct {
	ID         uuid.UUID  `json:"id"`
	ToolCallID uuid.UUID  `json:"tool_call_id"`
	ArgsHash   []byte     `json:"-"`
	Status     string     `json:"status"`
	DecidedBy  string     `json:"decided_by"`
	Comment    string     `json:"comment"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ApprovalView is an approval plus the redacted call a reviewer needs.
type ApprovalView struct {
	ID           uuid.UUID       `json:"id"`
	ToolCallID   uuid.UUID       `json:"tool_call_id"`
	Status       string          `json:"status"`
	Comment      string          `json:"comment"`
	DecidedBy    string          `json:"decided_by"`
	ExpiresAt    time.Time       `json:"expires_at"`
	DecidedAt    *time.Time      `json:"decided_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	AgentID      string          `json:"agent_id"`
	ToolID       string          `json:"tool_id"`
	ToolName     string          `json:"tool_name"`
	ArgsRedacted json.RawMessage `json:"args_redacted"`
	ArgsHash     string          `json:"args_hash"`
	RuleID       string          `json:"rule_id"`
	Reason       string          `json:"reason"`
	RequestID    string          `json:"request_id"`
	Decision     string          `json:"decision"`
}

// AuditEvent is one hash-chained record. Payload is canonical JSON.
type AuditEvent struct {
	Seq       int64           `json:"seq"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// VerifyResult is the outcome of walking the hash chain.
type VerifyResult struct {
	OK        bool   `json:"ok"`
	Events    int    `json:"events"`
	Head      string `json:"head,omitempty"`
	BrokenSeq int64  `json:"broken_seq,omitempty"`
	Reason    string `json:"reason,omitempty"`
}
