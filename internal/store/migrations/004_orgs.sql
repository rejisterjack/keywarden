-- Multi-tenant organizations. Existing rows backfill into the default org.
CREATE TABLE organizations (
    id uuid PRIMARY KEY,
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_slug_slug CHECK (slug ~ '^[a-z][a-z0-9_-]{0,62}$')
);

INSERT INTO organizations (id, slug, name)
VALUES ('00000000-0000-4000-8000-000000000001', 'default', 'Default');

CREATE TABLE org_admin_tokens (
    id uuid PRIMARY KEY,
    org_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    label text NOT NULL DEFAULT '',
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX org_admin_tokens_org_idx ON org_admin_tokens (org_id);

ALTER TABLE agents ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE agents SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE agents ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE tools ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE tools SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE tools ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE policies ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE policies SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE policies ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE tasks ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE tasks SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE tasks ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE tool_calls ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE tool_calls SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE tool_calls ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE approvals ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE approvals SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE approvals ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE audit_events ADD COLUMN org_id uuid REFERENCES organizations (id);
UPDATE audit_events SET org_id = '00000000-0000-4000-8000-000000000001';
ALTER TABLE audit_events ALTER COLUMN org_id SET NOT NULL;

ALTER TABLE tasks DROP CONSTRAINT tasks_agent_id_fkey;

ALTER TABLE agents DROP CONSTRAINT agents_pkey;
ALTER TABLE agents DROP CONSTRAINT agents_name_key;
ALTER TABLE agents ADD PRIMARY KEY (org_id, id);
ALTER TABLE agents ADD UNIQUE (org_id, name);

ALTER TABLE tools DROP CONSTRAINT tools_pkey;
ALTER TABLE tools ADD PRIMARY KEY (org_id, id);

ALTER TABLE policies DROP CONSTRAINT policies_version_key;
DROP INDEX IF EXISTS policies_one_active;
ALTER TABLE policies ADD UNIQUE (org_id, version);
CREATE UNIQUE INDEX policies_one_active_per_org ON policies (org_id) WHERE active;

ALTER TABLE tasks
    ADD CONSTRAINT tasks_agent_fkey
    FOREIGN KEY (org_id, agent_id) REFERENCES agents (org_id, id);

CREATE INDEX tasks_org_idx ON tasks (org_id);
CREATE INDEX tool_calls_org_idx ON tool_calls (org_id);
CREATE INDEX approvals_org_status_idx ON approvals (org_id, status);
CREATE INDEX audit_events_org_seq_idx ON audit_events (org_id, seq);
