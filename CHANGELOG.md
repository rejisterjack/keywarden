# Changelog

## 0.1.0

First release of the least-privilege proxy for agent tool calls.

- Short-lived task tokens. The real secret stays in the proxy.
- Default-deny policy, with ordered rules or an in-process Rego module.
- Human approval for one call, bound to that call's argument hash.
- Hash-chained audit log, verified with `keywarden audit verify`.
- Tool kinds: MCP over HTTP, MCP over stdio, Postgres, plain HTTPS, AWS SigV4, and GCP or Azure Bearer HTTPS.
- Single-tenant by default, with org admin tokens when one database serves more than one team.
- Dashboard for approvals, tasks, agents, tools, policy, and the audit chain.
