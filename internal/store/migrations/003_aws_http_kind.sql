-- Allow SigV4-signed AWS service calls as governed tools.
ALTER TABLE tools DROP CONSTRAINT tools_kind;
ALTER TABLE tools ADD CONSTRAINT tools_kind CHECK (
    kind IN ('mcp_http', 'postgres', 'mcp_stdio', 'http', 'aws_http')
);
