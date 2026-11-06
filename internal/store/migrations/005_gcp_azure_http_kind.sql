-- Allow basic GCP and Azure HTTPS helpers as governed tools.
ALTER TABLE tools DROP CONSTRAINT tools_kind;
ALTER TABLE tools ADD CONSTRAINT tools_kind CHECK (
    kind IN ('mcp_http', 'postgres', 'mcp_stdio', 'http', 'aws_http', 'gcp_http', 'azure_http')
);
