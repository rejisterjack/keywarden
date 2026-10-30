# Architecture

Keywarden is one Go process and a Postgres database. The dashboard is a separate Next.js app that calls the admin API with a token the browser never sees.

```
Agent -- task token --> POST /mcp
                           |
                     policy engine
                           |
              allow | deny | wait for approval
                           |
                     encrypted vault
                           |
         upstream MCP / Postgres / HTTP / AWS
                           |
                     hash-chained audit
```

## Control plane

`/v1` is the admin API. Every admin request is scoped to one organization. In the default single-tenant mode (`KEYWARDEN_SINGLE_TENANT=1`), `KEYWARDEN_ADMIN_TOKEN` authenticates as the migration-seeded default org. In multi-tenant mode, each org uses its own `kwa_…` admin token; the process env token is not accepted.

Agents, tools, policies, tasks, approvals, and audit events all carry `org_id`. A task token (`kwt_…`) is bound to one agent in one org, an optional tool subset, and an expiry. The default lifetime is 15 minutes. The maximum is 24 hours. The plaintext is returned once.

## Data plane

`POST /mcp` speaks Streamable HTTP MCP. `tools/list` returns only tools the task may use and that some allow or approval rule could match. Names are `toolId__toolName`, split on the first `__`.

`tools/call` canonicalizes the arguments, redacts secret-shaped values, evaluates policy, and writes an audit event before any upstream call. The policy engine is the ordered rules evaluator by default; a policy with `"engine": "rego"` is compiled at apply time and evaluated with the same input (`agent_id`, `tool_id`, `name`, `sql_class`). An undefined or malformed Rego decision denies the call. `tools/list` visibility for a rego policy uses the module's `visible` rule when it defines one, and otherwise falls back to the decision with the argument predicates unknown.

Effects:

- `allow` forwards immediately.
- `deny`, and no matching rule, return a tool error and do not open an upstream connection.
- `require_approval` inserts a pending row bound to the argument hash and waits until a human approves, denies, or the rule timeout passes. A timeout or a process restart expires the wait. The call is not forwarded on the way back up.

The Postgres tool accepts one SQL statement. The classifier labels it `read`, `mutating`, or `destructive`. That label is a predicate the policy uses. Multi-statement batches are refused by the tool itself.

`mcp_http` keeps an upstream MCP session and injects the sealed headers. `mcp_stdio` is off unless `KEYWARDEN_ALLOW_STDIO=1`, and it is refused entirely in demo mode.

`http` tools expose a single MCP tool named `request` that forwards to a sealed base URL with an allow-listed method and path prefix. `aws_http` is the same shape, but the proxy SigV4-signs the request with sealed static keys or an STS AssumeRole session. Demo mode refuses both kinds.

## Audit

Each organization has its own hash chain. A transaction-scoped advisory lock is keyed per org so writers do not serialize across tenants. The genesis previous hash is 32 zero bytes. `GET /v1/audit/verify` and `keywarden audit verify` recompute the active org's chain.

## Dashboard

The Next.js app checks a bootstrap password and sets an HMAC session cookie. Server actions call Keywarden with the admin token for the configured organization. Approvals, the audit chain, agents, tools, and policy are the five screens.

## Tenancy

Compose and local demos run single-tenant against the default org. Set `KEYWARDEN_SINGLE_TENANT=0` and issue per-org admin tokens when one database serves more than one team. MCP requests inherit the org from the task token; admin routes inherit it from the admin token.
