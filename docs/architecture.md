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
              upstream MCP  or  Postgres
                           |
                     hash-chained audit
```

## Control plane

`/v1` is the admin API. It is authenticated with `KEYWARDEN_ADMIN_TOKEN`. It registers agents and tools, applies one active policy version at a time, issues task tokens, and records approval decisions.

A task token (`kwt_…`) is bound to one agent, an optional tool subset, and an expiry. The default lifetime is 15 minutes. The maximum is 24 hours. The plaintext is returned once.

## Data plane

`POST /mcp` speaks Streamable HTTP MCP. `tools/list` returns only tools the task may use and that some allow or approval rule could match. Names are `toolId__toolName`, split on the first `__`.

`tools/call` canonicalizes the arguments, redacts secret-shaped values, evaluates policy, and writes an audit event before any upstream call.

Effects:

- `allow` forwards immediately.
- `deny`, and no matching rule, return a tool error and do not open an upstream connection.
- `require_approval` inserts a pending row bound to the argument hash and waits until a human approves, denies, or the rule timeout passes. A timeout or a process restart expires the wait. The call is not forwarded on the way back up.

The Postgres tool accepts one SQL statement. The classifier labels it `read`, `mutating`, or `destructive`. That label is a predicate the policy uses. Multi-statement batches are refused by the tool itself.

`mcp_http` keeps an upstream MCP session and injects the sealed headers. `mcp_stdio` is off unless `KEYWARDEN_ALLOW_STDIO=1`, and it is refused entirely in demo mode.

## Audit

A transaction-scoped advisory lock keeps a single writer. The genesis previous hash is 32 zero bytes. `GET /v1/audit/verify` and `keywarden audit verify` recompute the chain.

## Dashboard

The Next.js app checks a bootstrap password and sets an HMAC session cookie. Server actions call Keywarden with the admin token. Approvals, the audit chain, agents, tools, and policy are the five screens.

## One deployment, one tenant

There is no organization table. Run another Compose stack, or another database, for another team.
