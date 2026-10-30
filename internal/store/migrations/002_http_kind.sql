-- Allow plain HTTP APIs as governed tools, not just MCP servers.
ALTER TABLE tools DROP CONSTRAINT tools_kind;
ALTER TABLE tools ADD CONSTRAINT tools_kind CHECK (
    kind IN ('mcp_http', 'postgres', 'mcp_stdio', 'http')
);
