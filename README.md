# Keywarden

Agents should not hold the keys. Keywarden holds the credential, decides whether the call may proceed, and writes down who asked and what was decided.

It is a least-privilege proxy for tool calls, MCP first. The agent gets a short-lived task token. The real secret stays in the proxy. A safe call goes through. A denied call fails before the upstream is contacted. A risky call waits for a person.

Keywarden sits next to [Undoom](https://github.com/rejisterjack/undoom) (did a known failure come back) and [Boxwren](https://github.com/rejisterjack/boxwren) (where untrusted code runs). Use Keywarden when the question is permission.

## Run the demo

Docker Compose is the path that includes Postgres, the proxy, and the dashboard. It loads fictional data only.

```bash
docker compose -f deploy/compose/docker-compose.yml up --build
```

Open http://localhost:3000 and sign in with `local-demo-password`. The proxy is on http://localhost:8080.

Then, from another shell:

```bash
export KEYWARDEN_ADMIN_TOKEN=local-dev-admin-token
./scripts/quickstart.sh
```

You should see a `SELECT` succeed, a `DROP` come back denied, and a `DELETE` complete only after the script approves it. `audit verify` reports the chain intact. The same approval is waiting in the dashboard if you get there first.

The Compose defaults are public. They are for the fictional demo. Generate your own vault key, admin token, and dashboard password before you store anything real, and turn demo mode off. See [docs/deploy.md](docs/deploy.md).

## Use it

1. Register an agent and a tool. The tool secret is write-only.
2. Apply a policy. [examples/policy.json](examples/policy.json) is the one the demo loads: reads are allowed, mutations wait, destructive SQL is denied.
3. Issue a task token and point the agent at `POST /mcp` with `Authorization: Bearer kwt_…`.
4. Call tools as `toolId__toolName`. `orders__query` takes `{ "sql": "…" }`.
5. Approve or deny the waiting call in the dashboard. Read the audit log after.

```bash
export KEYWARDEN_URL=http://127.0.0.1:8080
export KEYWARDEN_ADMIN_TOKEN=local-dev-admin-token

keywarden task create --agent support --ttl 15m
keywarden policy apply examples/policy.json
keywarden audit verify
```

`serve` and `seed` talk to Postgres directly. `task`, `policy`, and `audit` talk to a running proxy.

## What a policy looks like

Rules are matched in order. The first match wins. If none match, the call is denied.

```json
{
  "version": 1,
  "rules": [
    {
      "id": "orders-read",
      "match": { "agent": "support", "tool": "orders", "name": "query" },
      "when": { "sql": "read" },
      "effect": "allow"
    },
    {
      "id": "orders-mutate",
      "match": { "agent": "support", "tool": "orders", "name": "query" },
      "when": { "sql": "mutating" },
      "effect": "require_approval",
      "timeout_seconds": 120
    }
  ]
}
```

`when.sql` is `read`, `mutating`, or `destructive`. An approval covers that call's argument hash and nothing else.

## Develop

Go 1.27.3. Integration tests need Postgres:

```bash
export KEYWARDEN_TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:5432/keywarden?sslmode=disable
go test ./...
```

The dashboard is the Next.js app in `dashboard/`. It needs `KEYWARDEN_API_URL`, `KEYWARDEN_ADMIN_TOKEN`, `KEYWARDEN_DASHBOARD_PASSWORD`, and `KEYWARDEN_SESSION_SECRET`.

## Read this before you trust it

Keywarden only governs calls that reach it. A credential the agent already has somewhere else is untouched. A rule you wrote as allow will allow the call. The public demo is fake data. The details are in [docs/limitations.md](docs/limitations.md), and the shape of the system is in [docs/architecture.md](docs/architecture.md). Operational notes are in [SECURITY.md](SECURITY.md).
