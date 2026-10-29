# Deployment

Self-hosting is the product. The public instance is a fictional demo and must stay that way.

## Local Compose

From the repository root:

```bash
docker compose -f deploy/compose/docker-compose.yml up --build
```

The proxy listens on port 8080 and the dashboard on port 3000. The Compose file turns on demo mode and seeds the fictional support agent, orders database, and catalog fixture. The dashboard password in that file is `local-demo-password`. Replace `KEYWARDEN_VAULT_KEY`, `KEYWARDEN_ADMIN_TOKEN`, `KEYWARDEN_DASHBOARD_PASSWORD`, and `KEYWARDEN_SESSION_SECRET` before you register anything that is not fake, and set `KEYWARDEN_DEMO=0` when you are ready to store real secrets.

Back up the Postgres volume. The vault key is not in the volume. A backup without the key cannot be opened.

```bash
docker compose -f deploy/compose/docker-compose.yml exec postgres \
  pg_dump -U keywarden keywarden > keywarden.sql
```

## Hosted demo

Use three services, and only fictional data.

| Piece | Where | Notes |
| --- | --- | --- |
| Proxy | Railway | `railway.toml` builds `deploy/Dockerfile`. Health check is `/healthz`. |
| Database | Neon | One database for Keywarden. A second database, or a restricted role, for the orders tool. |
| Dashboard | Vercel | Project root is `dashboard/`. |

Set these on the proxy and refuse to boot any other way:

```
KEYWARDEN_DEMO=1
KEYWARDEN_REQUIRE_DEMO=1
KEYWARDEN_DEMO_LOCK=1
KEYWARDEN_SEED=1
KEYWARDEN_DATABASE_URL=...        # Neon connection string, sslmode=require
KEYWARDEN_ORDERS_DSN=...          # fictional orders database only
KEYWARDEN_VAULT_KEY=...           # openssl rand -base64 32
KEYWARDEN_ADMIN_TOKEN=...         # at least 16 characters, not the example
KEYWARDEN_PUBLIC_URL=https://...  # the Railway URL
KEYWARDEN_FIXTURE_URL=http://127.0.0.1:8080/demo/mcp
```

`KEYWARDEN_REQUIRE_DEMO` stops the process if demo mode is off. Demo mode rejects secrets that are not marked `kw_demo=fake`, rejects MCP servers other than the fixture, and rejects any Postgres DSN that is not the orders database. `KEYWARDEN_DEMO_LOCK` rejects agent, tool, and policy edits after seed. Approvals and task tokens still work, which is the point of the demo.

On Vercel, set the root directory to `dashboard/` and:

```
KEYWARDEN_API_URL=https://<railway-host>
KEYWARDEN_ADMIN_TOKEN=<same admin token>
KEYWARDEN_DASHBOARD_PASSWORD=<a password you are willing to publish>
KEYWARDEN_SESSION_SECRET=<openssl rand -base64 32>
```

Publish the dashboard password in the demo instructions. Do not publish the admin token or the vault key. The browser only receives a session cookie.

Do not point `KEYWARDEN_ORDERS_DSN` at the Keywarden database. The orders role should own only the fictional `orders` table.

## Real self-host

Run Compose, or the same Docker image, with `KEYWARDEN_DEMO` unset. Generate a vault key and an admin token that have never been committed. Put the proxy on a private network. Terminate TLS in front of it. Keep stdio tools off unless you intend to run child processes as the proxy user.

The image listens with `/keywarden serve`. Migrations run on startup.
