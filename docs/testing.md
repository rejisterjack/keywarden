# Testing

A change is releasable when CI is green and the release checks at the bottom of this page have been run on the commit that will be tagged. A bug fix adds a regression test in the package that owns the failure.

```
unit tests
  -> API tests against Postgres
    -> Compose quickstart and dashboard smoke
      -> manual release checks
```

CI runs `go test ./...`, staticcheck, govulncheck, the dashboard production build, the proxy image build, the Compose journey, and the Playwright smoke. Do not add a coverage percentage gate.

## Functional

| Surface | Pass condition | Where it is tested |
| --- | --- | --- |
| `POST /mcp` | A `kwt_` token can list and call tools. `tools/list` returns only `toolId__toolName` pairs a rule could allow or hold. Allow forwards. Deny returns a tool error and does not dial. `require_approval` waits. | `TestAllowDenyAndApproval` |
| MCP methods | `GET /mcp` is not an SSE stream. `DELETE /mcp` is a no-op session end. | `TestMCPMethods` |
| Agents, tools, policy | Create, list, and delete. A rules policy and a Rego policy apply. An invalid Rego module fails at apply time. | flow tests |
| Tasks | Issue and revoke. `GET /v1/tasks` does not contain the plaintext token. | `TestTaskListOmitsPlaintextToken` |
| Approvals | Pending calls can be approved or denied. The same decision is idempotent. The opposite decision conflicts. | `TestAllowDenyAndApproval` |
| Audit | `GET /v1/audit/verify` reports the chain. A tampered row fails verification. | `TestAllowDenyAndApproval` |
| Orgs | The process admin token creates orgs and `kwa_` tokens. A token for org A cannot read org B. | `TestOrgIsolation`, `scripts/quickstart.sh` |
| Tool kinds | Postgres accepts one statement. `http` enforces method and path. AWS signs. GCP and Azure inject a bearer token. Stdio does not inherit the proxy environment and is refused in demo mode. | upstream and secret tests |
| CLI | `version`, `task create`, `task revoke`, `policy apply`, `audit verify`, `org create`, `org token` call the admin API with the admin token. | `cmd/keywarden/cli_test.go` |
| Dashboard | Wrong password stays on `/login`. A task token is shown once and is not put in the address bar. The audit page says the chain is intact. A down proxy renders "Proxy unreachable". The session cookie is HttpOnly and SameSite=Lax, and Secure in production. | `dashboard/e2e/console.spec.ts` |

## End to end

`scripts/quickstart.sh` runs against Compose in CI.

1. `/healthz` and `/readyz` return ok. `/v1/meta` reports demo and a version.
2. Issue a task for the seeded `support` agent.
3. `SELECT` on `orders__query` returns the fictional row.
4. `DROP` is denied and the orders table still exists.
5. `DELETE` stays pending until that argument hash is approved. Repeating the same decision and rejecting the opposite decision are covered by `TestAllowDenyAndApproval`.
6. `keywarden audit verify` reports the chain intact. The script checks `GET /v1/audit/verify`.
7. Revoke the task. The next `POST /mcp` is 401.
8. Create two organizations with the process admin token. An agent created in the first is invisible to the second. The second token cannot decide an approval id from the first.

## Edge and boundary

- **Auth.** Missing bearer and a non-Bearer scheme are 401. A revoked task and an expired task are 401. An agent that still has a task cannot be deleted, and that task keeps working. Org isolation is 401 or not-found, never the other org's data.
- **Policy.** No rules means deny. An unknown effect is rejected at apply time. The first matching rule wins. A Rego module with no decision denies. `timeout_seconds` outside 1 to 600 is rejected. Approving one call does not approve a later call with different SQL.
- **SQL.** An empty statement, a comment-only statement, a multi-statement batch, and `SELECT ... INTO` are destructive or refused, and they do not open a connection. SQL longer than 10000 characters is refused.
- **HTTP.** A method off the allow list, a path outside the prefix, and `..` are refused. A body over 1 MiB is refused. An upstream 500 is recorded in the audit chain and returned to the caller with that status.
- **Approvals.** A wait that reaches the rule timeout is not forwarded. A process restart expires an in-flight wait. A decision after the deadline is rejected.
- **Dashboard.** Empty lists render the empty state. Non-JSON config or secret is rejected in the server action before a successful save. `ttl_seconds` of 0 is rejected.
- **Startup.** Missing vault key, an admin token under 16 characters, `KEYWARDEN_REQUIRE_DEMO` without demo, demo plus stdio, and demo lock without demo refuse to boot. Covered by `internal/config/config_test.go`.

## Regression

Every push and pull request runs the CI jobs in `.github/workflows/ci.yml`. A failing test, staticcheck, govulncheck, dashboard build, image build, Compose journey, or Playwright smoke blocks the release.

Name a new regression test after the failure it locks in. The deny-does-not-dial assertion and `TestTimeoutAndRestartDoNotForward` are the pattern.

## Non-functional

- **Security.** govulncheck runs on Go 1.27.4 or newer. The vault test fails if ciphertext is moved onto another tool id. Redaction tests mask secret-shaped log fields. Demo mode rejects a secret that is not marked `kw_demo=fake` and an upstream that is not the fixture or the orders database. Task and admin tokens are stored as SHA-256 only.
- **Reliability.** `/readyz` depends on Postgres. In-flight approvals from a previous process are expired on boot. The server write timeout stays above the 10 minute approval maximum.
- **Performance.** `go test -bench=. ./internal/policy` benchmarks rules and Rego evaluation. It does not block CI on a latency number. A manual check, not a CI gate, is 50 concurrent allow-path calls against Compose: zero 5xx, and 429 once the caller passes `KEYWARDEN_MCP_RPS`.
- **Usability.** On a release candidate, walk the dashboard at desktop width and at 390 pixels: landing, login, approvals, tasks, audit, agents, tools, and policy. The issued token is readable. The demo banner is visible when `/v1/meta` says demo.

## Release sign-off

Run this once on the commit that will be tagged.

- `docker compose -f deploy/compose/docker-compose.yml up --build`, then `KEYWARDEN_ADMIN_TOKEN=local-dev-admin-token ./scripts/quickstart.sh`.
- `keywarden version` inside the image matches the tag without the leading `v`.
- On a public demo, `/v1/meta` reports `demo` and `demo_lock` true before the hostname is pointed at it.
- A real self-host trial uses a vault key and admin token that were never committed, `KEYWARDEN_DEMO` unset, and stdio left off.
- The GitHub release has the four binaries and checksums, and `ghcr.io/rejisterjack/keywarden:<tag>` pulls.
