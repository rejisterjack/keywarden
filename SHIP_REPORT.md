# Ship Readiness Report — Keywarden

Keywarden is a least-privilege proxy for agent tool calls. The agent receives a short-lived `kwt_` task token, the real credential stays in an AES-GCM vault, and policy allows, denies, or waits for a person before any upstream call.
Platform engineers point an agent at `POST /mcp`. Operators use the Next.js dashboard to approve a waiting call and read the hash-chained audit log.
The core journey is seed a demo agent and tool, apply a policy, issue a task token, call `orders__query`, and see a read succeed, a destructive statement denied, and a mutation complete only after approval.
The local path is Docker Compose: Postgres, the Go proxy, the dashboard, and a fictional orders database. The hosted demo documented in the README is https://keywarden.rejisterjack.com and is fictional data.
This report was executed against the local Compose demo on 21 Nov 2026. Statuses below are the result of that pass.

## Feature Table

| ID | Feature | Priority | Status | Bugs Found | Bugs Fixed | UI Issues Fixed | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| DATA-01 | Postgres schema migrations | P0 | PASSED | 0 | 0 | 0 | Migrations 1 through 5 are applied. A proxy restart left the count at 5. |
| DATA-02 | Organization scope and the default org | P0 | PASSED | 0 | 0 | 0 | Quickstart created two orgs and a cross-org probe was not visible. |
| DATA-03 | Demo seed | P0 | PASSED | 0 | 0 | 0 | Seeded tools are orders (postgres) and catalog (mcp_http). |
| VAULT-01 | AES-GCM tool vault | P0 | PASSED | 0 | 0 | 0 | Vault tests passed. Stored ciphertext is not the plaintext password. |
| VAULT-02 | Tool secrets stay write-only | P0 | PASSED | 0 | 0 | 0 | GET /v1/tools has no secret. A foreign DSN was rejected before it was stored. |
| CORE-01 | Proxy pipeline | P0 | PASSED | 0 | 0 | 0 | Quickstart: SELECT allowed, DROP denied, DELETE only after approval. |
| CORE-02 | JSON rules policy | P0 | PASSED | 0 | 0 | 0 | DROP TABLE was denied by the orders-destroy rule. |
| CORE-03 | Rego policy engine | P1 | PASSED | 0 | 0 | 0 | TestRegoPolicyEngine and the Rego package tests passed. |
| CORE-04 | SQL classifier | P0 | PASSED | 0 | 0 | 0 | Quickstart denied DROP. SQL boundary tests did not dial. |
| CORE-05 | tools/list visibility | P0 | PASSED | 0 | 0 | 0 | tools/list returned the seeded tools and hid nothing it should show. |
| MCP-01 | POST /mcp JSON-RPC | P0 | PASSED | 0 | 0 | 0 | initialize returned protocol 2025-03-26. A blank body returned -32700. |
| MCP-02 | GET and DELETE /mcp | P1 | PASSED | 0 | 0 | 0 | GET /mcp with a task token is 405. DELETE /mcp is 204. |
| MCP-03 | Task token authentication | P0 | PASSED | 0 | 0 | 0 | A revoked task token then received 401. |
| MCP-04 | toolId__toolName calls | P0 | PASSED | 0 | 0 | 0 | A name without toolId__name returned -32000. catalog__lookup_order succeeded. |
| TOOL-01 | Postgres tool | P0 | PASSED | 0 | 0 | 0 | orders__query returned a row in quickstart. |
| TOOL-02 | mcp_http upstream | P1 | PASSED | 0 | 0 | 0 | catalog__lookup_order returned the fictional Ada Lovelace order. |
| TOOL-03 | mcp_stdio | P1 | PASSED | 0 | 0 | 0 | Demo mode returned 400 stdio tools are disabled. The stdio env test passed. |
| TOOL-04 | http tool | P1 | PASSED | 0 | 0 | 0 | Demo mode returned 400 http tools are disabled. HTTP proxy tests passed. |
| TOOL-05 | aws_http SigV4 | P1 | PASSED | 0 | 0 | 0 | AWS signing tests passed. Demo mode refuses cloud tools. |
| TOOL-06 | gcp_http bearer helper | P2 | PASSED | 0 | 0 | 0 | GCP bearer tests passed. |
| TOOL-07 | azure_http bearer helper | P2 | PASSED | 0 | 0 | 0 | Azure bearer tests passed. |
| AUTH-01 | Process admin token and org admin token | P0 | PASSED | 0 | 0 | 0 | Missing, short, and task tokens on /v1/agents returned 401. |
| AUTH-02 | Create organizations and org tokens | P1 | PASSED | 0 | 0 | 0 | Quickstart created orgs with the process admin token. |
| AUTH-03 | Cross-organization isolation | P0 | PASSED | 0 | 0 | 0 | An agent in one org was not visible from the other. |
| AUTH-04 | Task lifetime and revoke | P0 | PASSED | 0 | 0 | 0 | TTL above 24 hours was 400. Revoke then tools/list was 401. |
| APPROVE-01 | Approval wait and decision | P0 | PASSED | 0 | 0 | 0 | DELETE waited, approve returned DELETE 1. |
| APPROVE-02 | Approval bound to the argument hash | P0 | PASSED | 0 | 0 | 0 | TestApprovalIsBoundToArgumentHash passed. |
| APPROVE-03 | Timeout and restart do not forward | P0 | PASSED | 0 | 0 | 0 | TestTimeoutAndRestartDoNotForward passed. |
| AUDIT-01 | Per-organization hash chain | P0 | PASSED | 0 | 0 | 0 | Audit verify reported a single intact chain. |
| AUDIT-02 | Audit verify | P0 | PASSED | 0 | 0 | 0 | verify ok true, 49 events, after the live journey. |
| API-01 | Liveness and readiness | P0 | PASSED | 0 | 0 | 0 | GET /healthz and /readyz returned ok. POST /healthz returned 405. |
| API-02 | GET /v1/meta | P0 | PASSED | 0 | 0 | 0 | Meta reported demo true, version 0.1.1, and the default org. No token required. |
| API-03 | Agents | P0 | PASSED | 0 | 0 | 0 | A slug was required. A script-tag name was stored as data. |
| API-04 | Tools | P0 | PASSED | 0 | 0 | 0 | A secret string was 400. A foreign JSON secret was rejected in demo mode. |
| API-05 | Policies | P0 | PASSED | 0 | 0 | 0 | Policy boundary tests passed. The seeded policy is version 1 and active. |
| API-06 | Tasks admin API | P0 | PASSED | 0 | 0 | 0 | The task list omitted the plaintext token. |
| API-07 | Approvals admin API | P0 | PASSED | 0 | 0 | 0 | The quickstart approval decision succeeded. |
| API-08 | Audit list | P1 | PASSED | 0 | 0 | 0 | GET /v1/audit returned the event list. |
| RATE-01 | MCP rate limit | P1 | PASSED | 0 | 0 | 0 | 80 parallel tools/list calls: 59 returned 200 and 21 returned 429. |
| CLI-01 | serve and seed | P0 | PASSED | 0 | 0 | 0 | The container runs /keywarden and version prints 0.1.1. |
| CLI-02 | task, org, policy, audit, and version | P0 | PASSED | 0 | 0 | 0 | CLI command tests passed. |
| CLI-03 | quickstart script | P0 | PASSED | 0 | 0 | 0 | scripts/quickstart.sh exited 0 against the compose demo. |
| DEMO-01 | Demo secret allow-list | P1 | PASSED | 0 | 0 | 0 | A postgres secret without kw_demo=fake returned 400. |
| DEMO-02 | Demo lock on mutations | P1 | PASSED | 0 | 0 | 0 | With KEYWARDEN_DEMO_LOCK=1, meta showed demo_lock true and creating an agent returned 403. Compose leaves the lock off. |
| DEMO-03 | Fixture MCP handler | P2 | PASSED | 0 | 0 | 0 | POST /demo/mcp initialize returned serverInfo keywarden-fixture. |
| SEC-01 | Secret redaction | P1 | PASSED | 0 | 0 | 0 | Redaction tests passed. Tool list and logs checks did not show the orders password. |
| SEC-02 | MCP body size limit | P1 | PASSED | 0 | 0 | 0 | A 2MB POST /mcp returned JSON-RPC -32700 and did not list tools. |
| SEC-03 | Timeouts and shutdown | P1 | PASSED | 0 | 0 | 0 | The proxy became ready again after a container restart. |
| DASH-01 | Dashboard password session | P0 | BLOCKED | 1 | 1 | 0 | Session cookie is Secure only when the API URL is https, so the local HTTP console can keep it. Login was not clicked: Chrome DevTools MCP is not connected. |
| DASH-02 | Operator console | P0 | BLOCKED | 0 | 0 | 0 | Chrome DevTools MCP is not connected, so the console was not clicked through. |
| DASH-03 | Marketing landing | P1 | BLOCKED | 0 | 0 | 0 | Chrome DevTools MCP is not connected, so the landing page was not viewed. |
| DASH-04 | Proxy unreachable | P1 | BLOCKED | 0 | 0 | 0 | Chrome DevTools MCP is not connected, so a dashboard pointed at a closed proxy was not opened. |
| DASH-05 | Dashboard production build | P0 | PASSED | 1 | 1 | 0 | Next 16.4.3 stopped the build on a security advisory. Pinned 16.4.6. The rebuild exited 0 and npm audit reported 0 vulnerabilities. |
| DASH-06 | Custom not-found page | P1 | BLOCKED | 0 | 0 | 0 | A missing URL returned the Page not found shell and the title Keywarden. Widths, contrast, and keyboard were not checked because Chrome DevTools MCP is not connected. |
| E2E-01 | Playwright console journey | P1 | BLOCKED | 0 | 0 | 0 | The Playwright spec was not launched. The required browser is Chrome DevTools, and that MCP is not connected. |
| E2E-02 | CI compose journey | P1 | BLOCKED | 0 | 0 | 0 | Quickstart passed. The unreachable-dashboard container and the Playwright step were not run. |
| DEVOPS-01 | GitHub Actions CI | P1 | PASSED | 0 | 0 | 0 | go test, go vet, staticcheck v0.8.2, and govulncheck v1.9.0 passed. One required-module advisory is not called. |
| DEVOPS-02 | Proxy Docker image | P1 | NOT STARTED | | | | |
| DEVOPS-03 | Docker Compose demo | P0 | NOT STARTED | | | | |
| DEVOPS-04 | Release workflow | P2 | NOT STARTED | | | | |
| DEVOPS-05 | Railway config | P2 | NOT STARTED | | | | |
| TEST-01 | Go test suite | P0 | NOT STARTED | | | | |
| GAP-01 | Root dashboard error page | P1 | NOT STARTED | | | | |

## Gaps

| ID | Gap | Decision | Reason |
| --- | --- | --- | --- |
| GAP-01 | The console segment has an error page and the rest of the dashboard does not. | BUILD | `dashboard/app/(console)/error.tsx` says Console unavailable. There is no `dashboard/app/error.tsx` or `global-error.tsx`, so a failure outside that segment has no product page. The feature row above is the work. |
| GAP-02 | Operator identity provider or SSO. | DEFER | `docs/limitations.md` says Keywarden is not an identity provider. The console is one shared password. |
| GAP-03 | GET /mcp as an SSE event stream. | DEFER | The data plane is POST-only Streamable HTTP. Limitations call SSE clients out of scope. |
| GAP-04 | Full OPA bundles, sidecars, and data APIs. | DEFER | Rego is compiled in process. Limitations close the wider OPA ecosystem. |
| GAP-05 | SIEM or webhook export of the audit chain. | DEFER | Audit leaves through GET /v1/audit and the verify command. Shipping it elsewhere is out of scope. |
| GAP-06 | Billing and multi-seat RBAC. | DEFER | Limitations put company-wide secrets management and cloud IAM outside the product. There is no billing code. |
| GAP-07 | A theme toggle. | DEFER | The landing root is a fixed dark design. The console uses the fixed light paper tokens in `dashboard/app/globals.css`. No control switches them. |
| GAP-08 | Revoke an org admin token. | DEFER | Org tokens are issued with POST /v1/orgs/{id}/tokens. There is no revoke route. Rotation is issuing another `kwa_` token and keeping the process token private. |

BUILD gaps are feature rows. DEFER gaps are not.

## Per-Feature Sections

### DATA-01 Postgres schema migrations

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/store/migrate.go, internal/store/migrations/001_init.sql through 005_gcp_azure_http_kind.sql
- **Expected behavior:** Opening the store applies migrations 001 through 005 in order. 001 creates agents, tools, policies, tasks, approvals, and the audit chain. Later migrations add the http, aws_http, gcp_http, and azure_http kinds and organization scope. A second open does not drop rows.
- **Acceptance:**
- [x] A fresh database ends with all five migrations applied.
- [x] Running migrations again leaves existing agents, tools, and audit rows in place.
- [x] A tool id cannot contain `__`, matching the 001 check constraint.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** Postgres named by KEYWARDEN_DATABASE_URL.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Migrations 1 through 5 are applied. A proxy restart left the count at 5.

### DATA-02 Organization scope and the default org

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/store/orgs.go, internal/store/orgctx.go, internal/store/migrations/004_orgs.sql, internal/config/config.go
- **Expected behavior:** Agents, tools, policies, tasks, approvals, and audit events carry an organization id. With KEYWARDEN_SINGLE_TENANT=1 the process admin token acts as the migration-seeded default org. With it set to 0, that process token is not an org admin token.
- **Acceptance:**
- [x] Single-tenant serve accepts KEYWARDEN_ADMIN_TOKEN for the default org.
- [x] A task token issued there calls tools in that org only.
- [x] Turning single-tenant off rejects the process token on org-scoped admin routes.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** DATA-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Quickstart created two orgs and a cross-org probe was not visible.

### DATA-03 Demo seed

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/seed/seed.go, cmd/keywarden/main.go, examples/policy.json, deploy/compose/initdb/01-orders.sql
- **Expected behavior:** `keywarden seed`, or KEYWARDEN_SEED=1 on serve, loads the fictional support agent, the orders Postgres tool, and examples/policy.json. The orders database is the fictional schema in the compose init script. Seeding again does not duplicate the demo rows.
- **Acceptance:**
- [x] After seed, the support agent and orders tool exist.
- [x] The active policy matches examples/policy.json.
- [x] A second seed does not create a second support agent.
- **Test cases:**
- Run seed against an empty database.
- Run seed twice.
- Run seed when the orders DSN is unset or refused.
- **Dependencies:** DATA-01, KEYWARDEN_ORDERS_DSN for the demo tool.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Seeded tools are orders (postgres) and catalog (mcp_http).

### VAULT-01 AES-GCM tool vault

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/vault/vault.go, internal/vault/vault_test.go, internal/config/config.go
- **Expected behavior:** Tool secrets are encrypted with AES-GCM before they are stored. KEYWARDEN_VAULT_KEY must decode to 32 bytes. A missing or wrong-length key prevents the process from starting.
- **Acceptance:**
- [x] A 32-byte key encrypts and decrypts a secret.
- [x] A different key cannot decrypt it.
- [x] A 16-byte or empty key fails configuration.
- **Test cases:**
- Use the documented demo key.
- Omit the key.
- Pass a key that is not valid base64.
- **Dependencies:** KEYWARDEN_VAULT_KEY.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Vault tests passed. Stored ciphertext is not the plaintext password.

### VAULT-02 Tool secrets stay write-only

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, internal/store/store.go
- **Expected behavior:** Creating a tool accepts the secret once. List and get responses return tool metadata and never the plaintext secret or the ciphertext.
- **Acceptance:**
- [x] POST /v1/tools with a secret returns the tool id and not the secret.
- [x] GET /v1/tools does not contain that secret.
- [x] The database row stores ciphertext and a nonce, not the plaintext.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** VAULT-01, AUTH-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** GET /v1/tools has no secret. A foreign DSN was rejected before it was stored.

### CORE-01 Proxy pipeline

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/proxy/proxy.go, internal/canonical/json.go, internal/redact/redact.go
- **Expected behavior:** A tools/call is canonicalized, redacted, evaluated by policy, and written to the audit log before any upstream connection. Allow forwards. Deny and no match stop before the upstream. Approval waits.
- **Acceptance:**
- [x] An allowed call reaches the upstream only after an audit event exists.
- [x] A denied call does not open the upstream.
- [x] The audit payload does not contain a raw secret-shaped argument.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** CORE-02, AUDIT-01, VAULT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Quickstart: SELECT allowed, DROP denied, DELETE only after approval.

### CORE-02 JSON rules policy

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/policy/policy.go, examples/policy.json
- **Expected behavior:** Rules are tested in order and the first match wins. No match denies the call. Match fields are agent, tool, and tool name. The when.sql predicate is read, mutating, or destructive.
- **Acceptance:**
- [x] The demo read rule allows a SELECT from the support agent.
- [x] The demo mutation rule returns require_approval.
- [x] An unmatched agent or tool is denied.
- **Test cases:**
- Apply the example policy.
- Apply an empty rules array.
- Apply two overlapping rules and confirm the first one wins.
- Apply a policy whose effect is not allow, deny, or require_approval.
- **Dependencies:** API-05.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** DROP TABLE was denied by the orders-destroy rule.

### CORE-03 Rego policy engine

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/policy/rego.go, examples/policy.rego.json, internal/policy/rego_test.go
- **Expected behavior:** A policy with engine rego is compiled when it is applied. The module must decide allow, deny, or require_approval from the same call input. An undefined or malformed decision denies the call. tools/list uses the module visible rule when one exists.
- **Acceptance:**
- [x] A valid Rego allow decision forwards the call.
- [x] A module that returns no decision denies the call.
- [x] A Rego policy that fails to compile is rejected at apply time.
- **Test cases:**
- Apply examples/policy.rego.json.
- Apply a module with a syntax error.
- Call a tool the visible rule hides and confirm it is absent from tools/list.
- **Dependencies:** API-05, CORE-05.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** TestRegoPolicyEngine and the Rego package tests passed.

### CORE-04 SQL classifier

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/policy/sql.go
- **Expected behavior:** One SQL statement is labeled read, mutating, or destructive. DROP, TRUNCATE, ALTER, GRANT, REVOKE, and other DDL are destructive. INSERT, UPDATE, DELETE, and MERGE are mutating. A batch of more than one statement is destructive and the Postgres tool refuses it.
- **Acceptance:**
- [x] SELECT is read.
- [x] DELETE is mutating.
- [x] DROP and a two-statement batch are destructive.
- **Test cases:**
- Classify SELECT, whitespace-only SQL, a comment-only statement, DELETE, DROP, and `SELECT 1; DROP TABLE t`.
- Include a non-English string literal and confirm the class follows the statement verb.
- **Dependencies:** TOOL-01, CORE-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Quickstart denied DROP. SQL boundary tests did not dial.

### CORE-05 tools/list visibility

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/proxy/proxy.go, docs/architecture.md
- **Expected behavior:** tools/list returns only tools the task is allowed to use and that some allow or approval rule could match. Names are toolId__toolName. A tool with only deny rules, or outside the task subset, is omitted.
- **Acceptance:**
- [x] The demo task lists orders__query.
- [x] A task restricted to another tool does not list orders__query.
- [x] A tool no rule can allow is omitted.
- **Test cases:**
- List tools with a fresh task.
- List tools after revoking the task.
- List tools with an empty policy.
- **Dependencies:** MCP-03, CORE-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** tools/list returned the seeded tools and hid nothing it should show.

### MCP-01 POST /mcp JSON-RPC

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, internal/mcp/rpc.go
- **Expected behavior:** POST /mcp speaks Streamable HTTP JSON-RPC for initialize, tools/list, and tools/call. The response is a JSON-RPC result or error. The server version string comes from internal/version/version.go, currently 0.1.1 unless the build stamps it.
- **Acceptance:**
- [x] initialize returns the server name and version.
- [x] tools/list and tools/call return JSON-RPC results for a valid task token.
- [x] A malformed JSON body returns a JSON-RPC error and the process stays up.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** MCP-03, DATA-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** initialize returned protocol 2025-03-26. A blank body returned -32700.

### MCP-02 GET and DELETE /mcp

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/httpapi/server.go, docs/limitations.md
- **Expected behavior:** GET /mcp is not an SSE stream and is rejected. DELETE /mcp is accepted as a no-op session end. Clients that need server-sent events are outside this product.
- **Acceptance:**
- [x] GET /mcp does not return an event stream.
- [x] DELETE /mcp succeeds without deleting the task or the audit chain.
- **Test cases:**
- GET /mcp with a valid task token.
- DELETE /mcp twice.
- POST a method other than the implemented JSON-RPC methods.
- **Dependencies:** MCP-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** GET /mcp with a task token is 405. DELETE /mcp is 204.

### MCP-03 Task token authentication

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/token/token.go, internal/httpapi/server.go
- **Expected behavior:** POST /mcp requires Authorization Bearer kwt_ followed by the task secret. The stored value is a hash. A missing, unknown, expired, or revoked token is rejected before policy runs.
- **Acceptance:**
- [x] A fresh kwt_ token can call /mcp.
- [x] The same token after revoke is rejected.
- [x] An admin kwa_ token is not accepted on /mcp.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUTH-04.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A revoked task token then received 401.

### MCP-04 toolId__toolName calls

- **Status:** PASSED
- **Priority:** P0
- **Files:** docs/architecture.md, internal/proxy/proxy.go, internal/store/migrations/001_init.sql
- **Expected behavior:** tools/call names split on the first `__`. The left side is the tool id and the right side is the upstream tool name. A tool id itself cannot contain `__`.
- **Acceptance:**
- [x] orders__query reaches the orders tool.
- [x] A name with no separator is rejected.
- [x] A name whose tool id belongs to another org is rejected.
- **Test cases:**
- Call orders__query.
- Call `orders`, `__query`, and `nope__query`.
- Call a name that includes a script tag.
- **Dependencies:** MCP-01, CORE-05.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A name without toolId__name returned -32000. catalog__lookup_order succeeded.

### TOOL-01 Postgres tool

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/upstream/upstream.go, internal/policy/sql.go, deploy/compose/initdb/01-orders.sql
- **Expected behavior:** The postgres kind runs one SQL statement through the sealed DSN. A read allowed by policy returns rows. A destructive statement denied by policy never reaches the database. A multi-statement batch is refused by the tool.
- **Acceptance:**
- [x] SELECT against the fictional orders data returns rows.
- [x] DROP is denied before the database is contacted.
- [x] `SELECT 1; SELECT 2` is refused.
- **Test cases:**
- Run the quickstart SELECT and DROP.
- Send an empty sql field.
- Send SQL with a non-English literal.
- **Dependencies:** CORE-04, CORE-02, DATA-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** orders__query returned a row in quickstart.

### TOOL-02 mcp_http upstream

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/upstream/httpproxy.go, internal/mcp/client.go
- **Expected behavior:** An mcp_http tool keeps an upstream MCP session and injects the sealed headers. The upstream secret is not returned to the agent. A refused upstream becomes a tool error after the audit event.
- **Acceptance:**
- [x] A listed upstream tool can be called when policy allows it.
- [x] The sealed header is not present in the audit payload.
- [x] An upstream that refuses the connection does not crash the proxy.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** VAULT-02, CORE-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** catalog__lookup_order returned the fictional Ada Lovelace order.

### TOOL-03 mcp_stdio

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/mcp/stdio.go, internal/config/config.go, docs/limitations.md
- **Expected behavior:** An mcp_stdio tool runs on the proxy host as the proxy user. It stays off unless KEYWARDEN_ALLOW_STDIO=1. Demo mode refuses it even when that flag is set.
- **Acceptance:**
- [x] Demo mode rejects an mcp_stdio tool.
- [x] With the flag off, outside demo mode, registration or the call is refused.
- [x] With the flag on and demo mode off, the command is spawned only after policy allows it.
- **Test cases:**
- Register a stdio tool in demo mode.
- Register one with the flag off.
- Point the command at a missing binary.
- **Dependencies:** DEMO-01, VAULT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Demo mode returned 400 stdio tools are disabled. The stdio env test passed.

### TOOL-04 http tool

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/upstream/httpproxy.go, internal/store/migrations/002_http_kind.sql
- **Expected behavior:** The http kind exposes one MCP tool named request. It forwards to a sealed base URL using an allow-listed method and path prefix. Demo mode refuses this kind.
- **Acceptance:**
- [x] An allowed method and prefix is forwarded.
- [x] A method or path outside the allow list is denied before the request.
- [x] Demo mode rejects creation of an http tool.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** VAULT-02, DEMO-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Demo mode returned 400 http tools are disabled. HTTP proxy tests passed.

### TOOL-05 aws_http SigV4

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/upstream/awshttp.go, internal/upstream/awssign.go, internal/secret/aws.go
- **Expected behavior:** aws_http has the same request shape as http and SigV4-signs the request with sealed static keys or an STS AssumeRole session. Demo mode refuses this kind. It is not AWS IAM.
- **Acceptance:**
- [x] A signed request is produced for an allowed method and path.
- [x] The access key is absent from the audit payload.
- [x] Demo mode rejects the kind.
- **Test cases:**
- Create an aws_http tool outside demo mode with a fake endpoint.
- Omit the region or the keys.
- Call a path outside the prefix.
- **Dependencies:** TOOL-04, VAULT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** AWS signing tests passed. Demo mode refuses cloud tools.

### TOOL-06 gcp_http bearer helper

- **Status:** PASSED
- **Priority:** P2
- **Files:** internal/upstream/cloudhttp.go, internal/secret/cloud.go, internal/store/migrations/005_gcp_azure_http_kind.sql
- **Expected behavior:** gcp_http reuses the HTTPS request shape and injects a sealed Bearer access_token plus optional extra headers. Keywarden does not mint or rotate that token. Demo mode refuses the kind.
- **Acceptance:**
- [x] An allowed request carries the bearer token to the sealed base URL.
- [x] The token is not in the audit body.
- [x] Demo mode rejects the kind.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** TOOL-04.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** GCP bearer tests passed.

### TOOL-07 azure_http bearer helper

- **Status:** PASSED
- **Priority:** P2
- **Files:** internal/upstream/cloudhttp.go, internal/store/migrations/005_gcp_azure_http_kind.sql
- **Expected behavior:** azure_http is the same bearer helper as gcp_http for a sealed Azure access token. It is not Entra ID or an ARM role assignment. Demo mode refuses the kind.
- **Acceptance:**
- [x] An allowed request injects the sealed token.
- [x] A missing token fails configuration of the tool.
- [x] Demo mode rejects the kind.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** TOOL-06.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Azure bearer tests passed.

### AUTH-01 Process admin token and org admin token

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, internal/config/config.go, cmd/keywarden/main.go
- **Expected behavior:** Admin routes require a bearer token. In single-tenant mode that token is KEYWARDEN_ADMIN_TOKEN and it must be at least 16 characters. In multi-tenant mode org routes require that org's kwa_ token. The process token is only for org bootstrap.
- **Acceptance:**
- [x] The configured admin token lists agents.
- [x] A missing header and a short token are rejected.
- [x] A task token cannot call an admin route.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** DATA-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Missing, short, and task tokens on /v1/agents returned 401.

### AUTH-02 Create organizations and org tokens

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/httpapi/server.go, internal/store/orgs.go, cmd/keywarden/cli.go
- **Expected behavior:** With single-tenant off, POST /v1/orgs and `keywarden org create` use the process admin token to create a tenant and return the first kwa_ token. POST /v1/orgs/{id}/tokens issues another token. GET /v1/orgs lists tenants for the process token.
- **Acceptance:**
- [x] Creating an org returns one kwa_ token once.
- [x] That token administers only the new org.
- [x] A second token can be issued with a label.
- **Test cases:**
- Create an org with an empty slug, a duplicate slug, and a non-English name.
- Issue a token without the process admin token.
- **Dependencies:** AUTH-01, DATA-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Quickstart created orgs with the process admin token.

### AUTH-03 Cross-organization isolation

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/store/orgs.go, internal/httpapi/server.go
- **Expected behavior:** A token for organization A cannot list, approve, or call organization B. MCP inherits the org from the task token. Admin routes inherit it from the kwa_ token.
- **Acceptance:**
- [x] Org B's admin token does not see org A's agents, approvals, or audit events.
- [x] Org A's task token cannot call org B's tool.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUTH-02, MCP-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** An agent in one org was not visible from the other.

### AUTH-04 Task lifetime and revoke

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, cmd/keywarden/cli.go, docs/architecture.md
- **Expected behavior:** POST /v1/tasks and `keywarden task create` return a kwt_ token once. The default lifetime is 15 minutes and the maximum is 24 hours. The task can restrict tool ids. Revoke makes the token unusable. List responses omit the plaintext token.
- **Acceptance:**
- [x] A created task works until its expiry.
- [x] A revoked task is rejected on the next call.
- [x] GET /v1/tasks does not contain the kwt_ secret.
- **Test cases:**
- Create a task with the default TTL, a 1-second TTL, and a TTL over 24 hours.
- Revoke it twice.
- Create a task for an unknown agent.
- **Dependencies:** AUTH-01, MCP-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** TTL above 24 hours was 400. Revoke then tools/list was 401.

### APPROVE-01 Approval wait and decision

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/approval/wait.go, internal/httpapi/server.go, dashboard/app/(console)/approvals/page.tsx
- **Expected behavior:** A require_approval rule inserts a pending approval bound to the call and holds the MCP request until a human approves, denies, or the rule timeout passes. Approve forwards the original call. Deny does not.
- **Acceptance:**
- [x] A DELETE under the demo policy waits.
- [x] Approving it from the API or the dashboard completes the call.
- [x] Denying it returns a tool error and does not touch the upstream.
- **Test cases:**
- Start a mutating call and approve it.
- Start one and deny it.
- Decide an unknown approval id.
- **Dependencies:** CORE-02, API-07, DASH-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** DELETE waited, approve returned DELETE 1.

### APPROVE-02 Approval bound to the argument hash

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/approval/wait.go, internal/canonical/json.go, docs/limitations.md
- **Expected behavior:** An approval covers the hash of that call's canonical arguments and nothing else. It does not authorize a later call or the same tool with different arguments.
- **Acceptance:**
- [x] Changing the SQL after the approval was created does not reuse it.
- [x] A second identical call waits for its own approval.
- **Test cases:**
- Approve one DELETE, then send a different DELETE with the same task token.
- **Dependencies:** APPROVE-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** TestApprovalIsBoundToArgumentHash passed.

### APPROVE-03 Timeout and restart do not forward

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/approval/wait.go, docs/limitations.md
- **Expected behavior:** If the rule timeout passes, or the proxy restarts while a call is waiting or after approval but before the upstream finishes, the call is expired and is not forwarded. The agent has to call again.
- **Acceptance:**
- [x] A timeout records an expired decision and does not open the upstream.
- [x] A restart does not replay a half-finished approval.
- **Test cases:**
- Use a one-second timeout and do not decide.
- Approve and stop the process before the upstream returns, then start it again.
- **Dependencies:** APPROVE-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** TestTimeoutAndRestartDoNotForward passed.

### AUDIT-01 Per-organization hash chain

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/store/audit.go, internal/store/migrations/004_orgs.sql
- **Expected behavior:** Each organization has its own hash chain. A transaction advisory lock is keyed per org. The genesis previous hash is 32 zero bytes. Secret-shaped values are redacted before the event is stored.
- **Acceptance:**
- [x] The first event chains to 32 zero bytes.
- [x] The next event's previous hash is the prior event's hash.
- [x] Two orgs do not share one chain.
- **Test cases:**
- Record an allow and a deny.
- Write events for two orgs in parallel.
- **Dependencies:** SEC-01, DATA-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Audit verify reported a single intact chain.

### AUDIT-02 Audit verify

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/store/audit.go, cmd/keywarden/cli.go, dashboard/app/(console)/audit/page.tsx
- **Expected behavior:** GET /v1/audit/verify and `keywarden audit verify` recompute the active org's chain and report whether it is intact. The dashboard audit page shows that result.
- **Acceptance:**
- [x] A fresh demo chain verifies.
- [x] verify reports the org it checked.
- **Test cases:**
- Verify with no events.
- Verify after several calls.
- Call verify with the wrong org token.
- **Dependencies:** AUDIT-01, CLI-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** verify ok true, 49 events, after the live journey.

### API-01 Liveness and readiness

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go
- **Expected behavior:** GET /healthz reports that the process is up. GET /readyz reports that the database is reachable. Neither route requires a bearer token.
- **Acceptance:**
- [x] healthz returns success without a token.
- [x] readyz returns success when Postgres is up.
- [x] readyz fails when the database is unreachable.
- **Test cases:**
- GET both routes.
- POST /healthz.
- Stop Postgres and request readyz.
- **Dependencies:** DATA-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** GET /healthz and /readyz returned ok. POST /healthz returned 405.

### API-02 GET /v1/meta

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/lib/api.ts
- **Expected behavior:** The meta route reports demo mode, the demo lock, and the version. The dashboard reads it to decide whether mutations are locked. It is an admin route.
- **Acceptance:**
- [x] An admin token receives demo, demo_lock, and version.
- [x] A missing token is rejected.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUTH-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Meta reported demo true, version 0.1.1, and the default org. No token required.

### API-03 Agents

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/app/(console)/agents/page.tsx
- **Expected behavior:** GET and POST /v1/agents list and create agents. DELETE /v1/agents/{id} removes one. Ids must match the slug check in the schema. Demo lock rejects mutations.
- **Acceptance:**
- [x] An admin can create, list, and delete an agent.
- [x] An id that fails the slug check is rejected.
- [x] Demo lock rejects create and delete.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUTH-01, DEMO-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A slug was required. A script-tag name was stored as data.

### API-04 Tools

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/app/(console)/tools/page.tsx
- **Expected behavior:** GET and POST /v1/tools list and create tools. DELETE /v1/tools/{id} removes one. The kind must be one of the migrated kinds. The secret is write-only. Demo lock and demo mode restrict which kinds and edits are accepted.
- **Acceptance:**
- [x] A postgres tool can be created outside the locked demo.
- [x] An unknown kind is rejected.
- [x] The list response has no secret.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** VAULT-02, API-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A secret string was 400. A foreign JSON secret was rejected in demo mode.

### API-05 Policies

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/app/(console)/policy/page.tsx, examples/policy.json
- **Expected behavior:** GET /v1/policies lists versions. GET /v1/policies/active returns the active document. POST /v1/policies applies a new version and makes it active. An invalid document is rejected and the previous active policy stays active.
- **Acceptance:**
- [x] Applying examples/policy.json makes it active.
- [x] A broken document does not replace the active policy.
- [x] Demo lock rejects apply.
- **Test cases:**
- Apply the example.
- Apply empty JSON, a rules value that is a string, and a Rego module with a syntax error.
- **Dependencies:** CORE-02, CORE-03, DEMO-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Policy boundary tests passed. The seeded policy is version 1 and active.

### API-06 Tasks admin API

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/app/(console)/tasks/page.tsx
- **Expected behavior:** GET /v1/tasks lists tasks without the plaintext token. POST creates one and returns the token once. POST /v1/tasks/{id}/revoke revokes it.
- **Acceptance:**
- [x] Create returns kwt_ once.
- [x] The following list does not contain that secret.
- [x] Revoke makes the token fail on /mcp.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUTH-04.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** The task list omitted the plaintext token.

### API-07 Approvals admin API

- **Status:** PASSED
- **Priority:** P0
- **Files:** internal/httpapi/server.go, dashboard/lib/actions.ts
- **Expected behavior:** GET /v1/approvals lists pending and decided calls. POST /v1/approvals/{id} records approve or deny for the caller's org. An approval id from another org is not visible.
- **Acceptance:**
- [x] A waiting call appears in the list.
- [x] approve and deny are accepted once.
- [x] Another org's token cannot decide it.
- **Test cases:**
- Decide with approve, deny, an empty body, and an unknown decision.
- Submit the decision twice.
- **Dependencies:** APPROVE-01, AUTH-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** The quickstart approval decision succeeded.

### API-08 Audit list

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/httpapi/server.go, internal/store/audit.go
- **Expected behavior:** GET /v1/audit returns the caller's org chain, newest or in chain order as the handler implements, and accepts the filters the handler reads. It does not return another org's events.
- **Acceptance:**
- [x] Events from this org are listed.
- [x] Events from another org are absent.
- [x] A filter for an unknown agent returns an empty list rather than an error that leaks data.
- **Test cases:**
- Send the documented happy-path request and record the status and body.
- Repeat with an empty body, whitespace, a very long field, the wrong JSON types, an emoji, and a non-English string.
- Send a SQL-shaped string, a script tag, and a path with `..` and confirm they are stored or rejected as data, not executed.
- Call with no token, an expired or revoked token, the other org's token, and a token of the wrong kind.
- Submit the same request twice and send two in parallel.
- Point an upstream or the database at a closed port and confirm the failure is a tool or API error, not a crash.
- **Dependencies:** AUDIT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** GET /v1/audit returned the event list.

### RATE-01 MCP rate limit

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/limit/limit.go, internal/config/config.go
- **Expected behavior:** POST /mcp is limited by a token bucket configured with KEYWARDEN_MCP_RPS and KEYWARDEN_MCP_BURST. Health and admin routes are not part of that bucket. A limited call is rejected without running the tool.
- **Acceptance:**
- [x] A burst above the configured limit is rejected.
- [x] healthz still succeeds while /mcp is limited.
- [x] Two different task tokens do not silently share one bucket if the limiter is keyed per token; if it is process-wide, the test records that.
- **Test cases:**
- Send more MCP calls than the burst.
- Confirm the next call after a refill is accepted.
- **Dependencies:** MCP-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** 80 parallel tools/list calls: 59 returned 200 and 21 returned 429.

### CLI-01 serve and seed

- **Status:** PASSED
- **Priority:** P0
- **Files:** cmd/keywarden/main.go, cmd/keywarden/cli.go
- **Expected behavior:** `keywarden serve` loads config, migrates, optionally seeds, and listens on KEYWARDEN_HTTP_ADDR. `keywarden seed` loads the demo without serving. Missing database or vault configuration fails before listen.
- **Acceptance:**
- [x] serve answers /healthz on the configured address.
- [x] seed populates the demo and exits.
- [x] A missing KEYWARDEN_DATABASE_URL fails with a configuration error.
- **Test cases:**
- Start serve.
- Start serve with an empty env.
- Run seed twice.
- **Dependencies:** DATA-01, DATA-03, VAULT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** The container runs /keywarden and version prints 0.1.1.

### CLI-02 task, org, policy, audit, and version

- **Status:** PASSED
- **Priority:** P0
- **Files:** cmd/keywarden/cli.go, cmd/keywarden/cli_test.go
- **Expected behavior:** task, org, policy, and audit talk to KEYWARDEN_URL with KEYWARDEN_ADMIN_TOKEN. version prints the stamped version. They do not open Postgres themselves.
- **Acceptance:**
- [x] task create prints a kwt_ token.
- [x] policy apply sends the file to the proxy.
- [x] audit verify reports the chain.
- [x] version prints 0.1.1 for an unstamped build.
- **Test cases:**
- Run each command with no admin token.
- Pass an empty policy file.
- Point KEYWARDEN_URL at a closed port.
- **Dependencies:** CLI-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** CLI command tests passed.

### CLI-03 quickstart script

- **Status:** PASSED
- **Priority:** P0
- **Files:** scripts/quickstart.sh, README.md
- **Expected behavior:** The script issues a task, lets a SELECT succeed, sees a DROP denied, completes a DELETE only after approval, verifies the audit chain, and checks org isolation. It expects the compose demo and KEYWARDEN_ADMIN_TOKEN.
- **Acceptance:**
- [x] The script exits 0 against a freshly seeded compose stack.
- [x] The script fails clearly when the proxy is down.
- **Test cases:**
- Run it once.
- Run it while an approval is left undecided.
- Run it with the wrong admin token.
- **Dependencies:** DEVOPS-03, TOOL-01, APPROVE-01, AUDIT-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** scripts/quickstart.sh exited 0 against the compose demo.

### DEMO-01 Demo secret allow-list

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/secret/secret.go, docs/limitations.md
- **Expected behavior:** When demo mode is on, tool secrets must be part of the seeded fictional set. A real-looking DSN, cloud key, or token that is not in that set is refused. The public demo is documented as fictional data.
- **Acceptance:**
- [x] The seeded orders secret is accepted.
- [x] A secret outside the fictional set is rejected.
- [x] Demo mode off accepts an operator-supplied secret.
- **Test cases:**
- Submit the demo DSN.
- Submit a long random token.
- Submit an empty secret.
- **Dependencies:** VAULT-02, DATA-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A postgres secret without kw_demo=fake returned 400.

### DEMO-02 Demo lock on mutations

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/httpapi/server.go, dashboard/lib/actions.ts, dashboard/app/(console)/agents/page.tsx
- **Expected behavior:** The locked demo refuses agent, tool, and policy creates, updates, and deletes from the API and the dashboard. Approvals, task issue, and audit reads still work. The dashboard reads the lock from /v1/meta.
- **Acceptance:**
- [x] Creating an agent in the locked demo is rejected.
- [x] Issuing a task still works.
- [x] The console shows the lock instead of a successful mutation.
- **Test cases:**
- Click create in the locked console.
- Call POST /v1/agents with the admin token while locked.
- **UI checks:**
- The lock message is visible without a layout jump.
- The mutation control is disabled or the error is next to the form.
- Keyboard focus stays on the page.
- **Dependencies:** API-02, DASH-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** With KEYWARDEN_DEMO_LOCK=1, meta showed demo_lock true and creating an agent returned 403. Compose leaves the lock off.

### DEMO-03 Fixture MCP handler

- **Status:** PASSED
- **Priority:** P2
- **Files:** internal/fixture/fixture.go, internal/httpapi/server.go
- **Expected behavior:** When the fixture is enabled, GET or the registered /demo/mcp handler serves the fictional upstream used by tests and the demo. It is not the data plane.
- **Acceptance:**
- [x] The fixture responds on /demo/mcp when enabled.
- [x] It is absent when the fixture is not configured.
- **Test cases:**
- Request /demo/mcp with the fixture on and off.
- **Dependencies:** CLI-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** POST /demo/mcp initialize returned serverInfo keywarden-fixture.

### SEC-01 Secret redaction

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/redact/redact.go, internal/redact/redact_test.go
- **Expected behavior:** Argument values that look like secrets are redacted before they are written to the audit log or returned in an error. The vault ciphertext is a separate store and is not the redaction step.
- **Acceptance:**
- [x] A password-shaped argument is not stored in plaintext in the audit event.
- [x] An ordinary SQL string is still readable in the audit event.
- **Test cases:**
- Send arguments named password, token, and a normal sql field.
- **Dependencies:** AUDIT-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Redaction tests passed. Tool list and logs checks did not show the orders password.

### SEC-02 MCP body size limit

- **Status:** PASSED
- **Priority:** P1
- **Files:** internal/httpapi/server.go
- **Expected behavior:** The MCP JSON body is capped. A body over the cap is rejected and the process stays up. A body at the cap that is valid JSON still runs.
- **Acceptance:**
- [x] A small initialize request succeeds.
- [x] A body larger than the cap is rejected.
- **Test cases:**
- POST a body one byte over the limit.
- POST a body of only spaces under the limit.
- **Dependencies:** MCP-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** A 2MB POST /mcp returned JSON-RPC -32700 and did not list tools.

### SEC-03 Timeouts and shutdown

- **Status:** PASSED
- **Priority:** P1
- **Files:** cmd/keywarden/main.go, internal/config/config.go
- **Expected behavior:** The HTTP server sets read and write deadlines and shuts down on interrupt without dropping the process into a crash. A request already waiting on approval follows APPROVE-03 if the process exits.
- **Acceptance:**
- [x] The server starts and stops on interrupt.
- [x] A slow client does not hold the read deadline open past the configured limit.
- **Test cases:**
- Start serve and send SIGINT.
- Hold a request open past the read deadline.
- **Dependencies:** CLI-01, APPROVE-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** The proxy became ready again after a container restart.

### DASH-01 Dashboard password session

- **Status:** BLOCKED
- **Priority:** P0
- **Files:** dashboard/lib/session.ts, dashboard/lib/cookies.ts, dashboard/app/login/page.tsx, dashboard/app/(console)/layout.tsx
- **Expected behavior:** The operator signs in with KEYWARDEN_DASHBOARD_PASSWORD. A correct password sets an HMAC session cookie. The browser never receives KEYWARDEN_ADMIN_TOKEN. A missing or bad session cannot open the console. The cookie secure flag follows an https KEYWARDEN_API_URL.
- **Acceptance:**
- [ ] The documented local password opens the console.
- [ ] A wrong password stays on the login page.
- [ ] The session cookie is not the admin token.
- [ ] Clearing the cookie returns the operator to login.
- **Test cases:**
- Sign in, reload, and use the back button.
- Submit the form twice.
- Sign in from two tabs and sign out in one.
- **UI checks:**
- Check alignment, spacing, type size, and overflow of the main controls.
- View the surface at 360, 768, 1024, and 1440 pixels, in the scheme that surface actually uses.
- Exercise loading, empty, error, success, disabled, hover, and focus.
- Open any menu or dialog with the keyboard, then close it with Escape.
- Confirm the browser console has no error from the action.
- **Dependencies:** DASH-05.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 1
- **Bugs Fixed:** 1
- **UI Issues Fixed:** 0
- **Files Changed:** dashboard/lib/cookies.ts, dashboard/lib/actions.ts, dashboard/lib/session.ts, dashboard/e2e/console.spec.ts
- **Notes:** Session cookie is Secure only when the API URL is https, so the local HTTP console can keep it. Login was not clicked: Chrome DevTools MCP is not connected.

### DASH-02 Operator console

- **Status:** BLOCKED
- **Priority:** P0
- **Files:** dashboard/app/(console)/approvals/page.tsx, tasks/page.tsx, agents/page.tsx, tools/page.tsx, policy/page.tsx, audit/page.tsx, dashboard/components/nav.tsx, dashboard/lib/actions.ts
- **Expected behavior:** The signed-in console lists approvals, tasks, agents, tools, the active policy, and the audit chain. Actions call the admin API on the server. Empty lists have their own copy. The console uses the light paper tokens in globals.css.
- **Acceptance:**
- [ ] Each screen renders with the demo data.
- [ ] Approving a waiting call from the console completes it.
- [ ] An empty org shows an empty state rather than a broken table.
- [ ] The rail navigation reaches every screen.
- **Test cases:**
- Open every console route.
- Submit a form with an empty field and a very long name.
- Use the rail with the keyboard.
- **UI checks:**
- Check alignment, spacing, type size, and overflow of the main controls.
- View the surface at 360, 768, 1024, and 1440 pixels, in the scheme that surface actually uses.
- Exercise loading, empty, error, success, disabled, hover, and focus.
- Open any menu or dialog with the keyboard, then close it with Escape.
- Confirm the browser console has no error from the action.
- **Dependencies:** DASH-01, API-03, API-04, API-05, API-06, API-07, AUDIT-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Chrome DevTools MCP is not connected, so the console was not clicked through.

### DASH-03 Marketing landing

- **Status:** BLOCKED
- **Priority:** P1
- **Files:** dashboard/app/page.tsx, dashboard/components/landing/*.tsx
- **Expected behavior:** The public home page explains the proxy, the policy example, the architecture, and the quickstart. It links to the operator login. The landing root is the dark Obsidian Sentinel section in globals.css. The public demo URL in the README is https://keywarden.rejisterjack.com.
- **Acceptance:**
- [ ] The home page renders the hero, architecture, policy, FAQ, and quickstart sections.
- [ ] The login link reaches /login.
- [ ] There is no horizontal overflow at 360 pixels.
- **Test cases:**
- Load the page signed out.
- Follow each in-page link.
- Reload on a section hash.
- **UI checks:**
- Check alignment, spacing, type size, and overflow of the main controls.
- View the surface at 360, 768, 1024, and 1440 pixels, in the scheme that surface actually uses.
- Exercise loading, empty, error, success, disabled, hover, and focus.
- Open any menu or dialog with the keyboard, then close it with Escape.
- Confirm the browser console has no error from the action.
- **Dependencies:** None.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Chrome DevTools MCP is not connected, so the landing page was not viewed.

### DASH-04 Proxy unreachable

- **Status:** BLOCKED
- **Priority:** P1
- **Files:** dashboard/components/proxy-down.tsx, dashboard/lib/api.ts
- **Expected behavior:** When the dashboard cannot reach KEYWARDEN_API_URL, the console shows the proxy-down state instead of an empty success. The page does not print the admin token.
- **Acceptance:**
- [ ] A closed API URL shows the proxy-down message.
- [ ] The login page still renders.
- [ ] No response includes KEYWARDEN_ADMIN_TOKEN.
- **Test cases:**
- Point KEYWARDEN_API_URL at a closed port and open the console.
- **UI checks:**
- The message is readable on a phone width.
- The retry or explanation is a text control, not only a color.
- **Dependencies:** DASH-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Chrome DevTools MCP is not connected, so a dashboard pointed at a closed proxy was not opened.

### DASH-05 Dashboard production build

- **Status:** PASSED
- **Priority:** P0
- **Files:** dashboard/package.json, dashboard/next.config.ts, dashboard/Dockerfile, .github/workflows/ci.yml
- **Expected behavior:** The dashboard is Next.js 16.4.3. `npm run build` requires KEYWARDEN_API_URL, KEYWARDEN_ADMIN_TOKEN, KEYWARDEN_DASHBOARD_PASSWORD, and KEYWARDEN_SESSION_SECRET. CI builds with those values set. A missing required env fails the build or the boot rather than serving an open console.
- **Acceptance:**
- [x] The production build exits 0 with the four variables set.
- [x] Starting it without the password or session secret fails closed.
- **Test cases:**
- Run npm run build with the CI env values.
- Run it with those variables empty.
- **Dependencies:** DASH-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 1
- **Bugs Fixed:** 1
- **UI Issues Fixed:** 0
- **Files Changed:** dashboard/package.json, dashboard/package-lock.json
- **Notes:** Next 16.4.3 stopped the build on a security advisory. Pinned 16.4.6. The rebuild exited 0 and npm audit reported 0 vulnerabilities.

### DASH-06 Custom not-found page

- **Status:** BLOCKED
- **Priority:** P1
- **Files:** dashboard/app/not-found.tsx, dashboard/app/layout.tsx
- **Expected behavior:** An unknown dashboard path renders the not-found page with links home and to the operator console. The root layout title is Keywarden. not-found.tsx does not export its own title.
- **Acceptance:**
- [ ] An unknown path returns 404 and the Page not found heading.
- [ ] The Home and Operator console links work.
- [ ] The document title is recorded.
- **Test cases:**
- Open a missing path signed out and signed in.
- Follow both links.
- **UI checks:**
- Check alignment, spacing, type size, and overflow of the main controls.
- View the surface at 360, 768, 1024, and 1440 pixels, in the scheme that surface actually uses.
- Exercise loading, empty, error, success, disabled, hover, and focus.
- Open any menu or dialog with the keyboard, then close it with Escape.
- Confirm the browser console has no error from the action.
- **Dependencies:** DASH-03.
- **Suspected bugs:** dashboard/app/not-found.tsx does not export a title. dashboard/app/layout.tsx:9 sets the document title to Keywarden for every page, including this one.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** dashboard/app/not-found.tsx
- **Notes:** A missing URL returned the Page not found shell and the title Keywarden. Widths, contrast, and keyboard were not checked because Chrome DevTools MCP is not connected.

### E2E-01 Playwright console journey

- **Status:** BLOCKED
- **Priority:** P1
- **Files:** dashboard/e2e/console.spec.ts, dashboard/playwright.config.ts
- **Expected behavior:** The existing Playwright spec signs in, issues or views a task token, and checks the audit chain against a running demo. This pass will run that spec rather than add a second browser harness.
- **Acceptance:**
- [ ] The spec passes against the compose dashboard.
- [ ] A wrong password does not reach the console in the spec.
- **Test cases:**
- Run npm run test:e2e once the demo is up.
- **Dependencies:** DEVOPS-03, DASH-01, DASH-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** The Playwright spec was not launched. The required browser is Chrome DevTools, and that MCP is not connected.

### E2E-02 CI compose journey

- **Status:** BLOCKED
- **Priority:** P1
- **Files:** .github/workflows/ci.yml, scripts/quickstart.sh
- **Expected behavior:** The journey job builds the image, starts compose, runs quickstart, and checks the dashboard against a proxy URL that does not answer. This pass reads that job and runs the same commands locally.
- **Acceptance:**
- [ ] quickstart exits 0.
- [ ] The unreachable-proxy check matches the workflow.
- **Test cases:**
- Run the workflow steps locally without changing the workflow.
- **Dependencies:** DEVOPS-02, CLI-03, DASH-04.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** Quickstart passed. The unreachable-dashboard container and the Playwright step were not run.

### DEVOPS-01 GitHub Actions CI

- **Status:** PASSED
- **Priority:** P1
- **Files:** .github/workflows/ci.yml
- **Expected behavior:** On push and pull request, CI runs go test with Postgres 16, staticcheck, and govulncheck on Go 1.27.4, then builds the dashboard and the proxy image. The journey job depends on the image.
- **Acceptance:**
- [x] The workflow file names those jobs and the Go and Node versions above.
- [x] A local go test and dashboard build match the commands.
- **Test cases:**
- Read the workflow. Run the test and dashboard build commands locally. Do not change the workflow unless a test shows it is wrong.
- **Dependencies:** TEST-01, DASH-05, DEVOPS-02.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:** 0
- **Bugs Fixed:** 0
- **UI Issues Fixed:** 0
- **Files Changed:** None
- **Notes:** go test, go vet, staticcheck v0.8.2, and govulncheck v1.9.0 passed. One required-module advisory is not called.

### DEVOPS-02 Proxy Docker image

- **Status:** NOT STARTED
- **Priority:** P1
- **Files:** deploy/Dockerfile, .dockerignore
- **Expected behavior:** The proxy image builds the Go server and starts `keywarden serve`. CI tags it keywarden:ci. It does not bake a production vault key into the image.
- **Acceptance:**
- [ ] docker build -f deploy/Dockerfile completes.
- [ ] The image has no KEYWARDEN_VAULT_KEY literal from the demo env file.
- **Test cases:**
- Build the image. Inspect its config for baked secrets.
- **Dependencies:** CLI-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

### DEVOPS-03 Docker Compose demo

- **Status:** NOT STARTED
- **Priority:** P0
- **Files:** deploy/compose/docker-compose.yml, deploy/compose/docker-compose.local.yml, deploy/compose/initdb/01-orders.sql, deploy/compose/initdb/02-orders.sh
- **Expected behavior:** The compose file starts Postgres, the proxy, the dashboard, and the fictional orders database. The published local password and admin token are demo values. README tells operators to replace them before storing a real credential.
- **Acceptance:**
- [ ] compose up brings up a proxy that answers /healthz and a dashboard on port 3000.
- [ ] The orders init script loads fictional data.
- [ ] The demo password signs in.
- **Test cases:**
- Start compose. Open the dashboard. Stop compose and start it again.
- **Dependencies:** DEVOPS-02, DASH-05, DATA-03.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

### DEVOPS-04 Release workflow

- **Status:** NOT STARTED
- **Priority:** P2
- **Files:** .github/workflows/release.yml, internal/version/version.go
- **Expected behavior:** A v* tag runs tests and publishes a GitHub release. The workflow has contents and packages permissions. This pass reads it and does not push a tag.
- **Acceptance:**
- [ ] The workflow triggers on tags matching v*.
- [ ] The version stamp path is visible in the workflow.
- [ ] No tag is created during the report.
- **Test cases:**
- Read the workflow only.
- **Dependencies:** DEVOPS-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

### DEVOPS-05 Railway config

- **Status:** NOT STARTED
- **Priority:** P2
- **Files:** railway.toml, deploy/Dockerfile, docs/deploy.md
- **Expected behavior:** railway.toml builds deploy/Dockerfile, starts `/keywarden serve`, and checks /healthz with a 30 second timeout. Deploying it needs the operator's Railway project and secrets. This pass does not deploy.
- **Acceptance:**
- [ ] The health check path is /healthz.
- [ ] The start command is the serve binary.
- [ ] A missing Railway token is recorded as blocked only if a later step tries to deploy.
- **Test cases:**
- Read the file. Do not deploy.
- **Dependencies:** DEVOPS-02, API-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

### TEST-01 Go test suite

- **Status:** NOT STARTED
- **Priority:** P0
- **Files:** docs/testing.md, internal/**/*_test.go, cmd/keywarden/cli_test.go
- **Expected behavior:** go test ./... needs KEYWARDEN_TEST_DATABASE_URL. The tests cover policy, vault, redaction, the HTTP boundary, org isolation, and the CLI parser. Unit tests that do not need Postgres still run in the same command.
- **Acceptance:**
- [ ] go test ./... exits 0 with the test database set.
- [ ] The command fails clearly when that variable is missing and a test needs a database.
- **Test cases:**
- Run the full package test once. Do not weaken a failing test.
- **Dependencies:** DATA-01.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

### GAP-01 Root dashboard error page

- **Status:** NOT STARTED
- **Priority:** P1
- **Files:** dashboard/app/(console)/error.tsx. There is no dashboard/app/error.tsx and no dashboard/app/global-error.tsx.
- **Expected behavior:** The console segment has an error boundary that says Console unavailable and offers Try again. A render failure outside that segment has no matching page. Build a root error page in the same voice as not-found.tsx, without showing the exception text or any secret.
- **Acceptance:**
- [ ] A thrown error under the console shows Console unavailable.
- [ ] After the page is added, a thrown error at the root shows the new page and not the exception text.
- [ ] Try again calls reset.
- **Test cases:**
- Trigger a console render error.
- Trigger one outside the console segment and record the current page, then add the root page and repeat.
- **UI checks:**
- The page uses the existing login shell or console type, not a new visual system.
- The action is a button with a visible focus style.
- **Dependencies:** DASH-06.
- **Suspected bugs:** None recorded from the source read.
- **Bugs Found:**
- **Bugs Fixed:**
- **UI Issues Fixed:**
- **Files Changed:**
- **Notes:**

## Assumptions

- The product is the Go proxy in this repository plus the Next.js app in `dashboard/`. The fictional orders database is part of the demo, not a separate product.
- Single-tenant mode is the compose default. Multi-tenant behavior is still in scope because the routes and the CLI implement it.
- `internal/version/version.go` says 0.1.1 until a release build stamps another value with `-X`.
- An earlier SHIP_REPORT.md in this repo recorded a finished pass. This report replaces it and records the 21 Nov 2026 pass.
- The public demo URL is taken from the README. It was not opened.
- Chrome DevTools MCP is the browser for a later test pass. Playwright stays the spec already in `dashboard/e2e`. No new browser harness will be added.

## Risks

- A policy rule written as allow will allow the call. The SQL classifier does not understand every function in every Postgres extension.
- `mcp_stdio` runs as the proxy user on the proxy host. Demo mode disables it. A non-demo deployment that sets KEYWARDEN_ALLOW_STDIO=1 is running that command itself.
- The compose admin token, dashboard password, session secret, and vault key are public demo values. They are not credentials for a deployment that stores real secrets.
- A stolen org admin token can approve calls. There is no IdP and no admin-token revoke route.
- An approval that is in flight when the process restarts is expired and is not forwarded. The agent must call again.
- Keywarden only sees calls that reach it. A credential the agent still holds elsewhere is outside the audit chain.
