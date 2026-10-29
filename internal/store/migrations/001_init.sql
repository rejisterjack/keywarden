CREATE TABLE agents (
    id text PRIMARY KEY,
    name text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT agents_id_slug CHECK (id ~ '^[a-z][a-z0-9_-]{0,62}$')
);

CREATE TABLE tools (
    id text PRIMARY KEY,
    name text NOT NULL,
    kind text NOT NULL,
    config jsonb NOT NULL,
    secret_ciphertext bytea NOT NULL,
    secret_nonce bytea NOT NULL,
    seeded boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tools_id_slug CHECK (id ~ '^[a-z][a-z0-9_-]{0,62}$'),
    CONSTRAINT tools_id_sep CHECK (position('__' IN id) = 0),
    CONSTRAINT tools_kind CHECK (kind IN ('mcp_http', 'postgres', 'mcp_stdio'))
);

CREATE TABLE policies (
    id uuid PRIMARY KEY,
    version integer NOT NULL UNIQUE,
    document jsonb NOT NULL,
    active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX policies_one_active ON policies (active) WHERE active;

CREATE TABLE tasks (
    id uuid PRIMARY KEY,
    agent_id text NOT NULL REFERENCES agents (id),
    token_hash bytea NOT NULL UNIQUE,
    tool_ids jsonb,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tool_calls (
    id uuid PRIMARY KEY,
    task_id uuid NOT NULL REFERENCES tasks (id),
    agent_id text NOT NULL,
    tool_id text NOT NULL,
    mcp_tool_name text NOT NULL,
    args_redacted jsonb NOT NULL,
    args_hash bytea NOT NULL,
    decision text NOT NULL,
    policy_rule_id text NOT NULL DEFAULT '',
    reason text NOT NULL DEFAULT '',
    approval_id uuid,
    request_id text NOT NULL DEFAULT '',
    upstream_status text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tool_calls_decision CHECK (
        decision IN ('allow', 'deny', 'pending', 'approved', 'rejected', 'expired', 'error')
    )
);

CREATE TABLE approvals (
    id uuid PRIMARY KEY,
    tool_call_id uuid NOT NULL UNIQUE REFERENCES tool_calls (id),
    args_hash bytea NOT NULL,
    status text NOT NULL,
    decided_by text NOT NULL DEFAULT '',
    comment text NOT NULL DEFAULT '',
    decided_at timestamptz,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT approvals_status CHECK (status IN ('pending', 'approved', 'rejected', 'expired'))
);

CREATE INDEX approvals_status_idx ON approvals (status);

CREATE TABLE audit_events (
    seq bigserial PRIMARY KEY,
    prev_hash bytea NOT NULL,
    hash bytea NOT NULL UNIQUE,
    event_type text NOT NULL,
    payload text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
