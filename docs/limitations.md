# Limitations

Keywarden governs tool calls that actually go through it. That boundary is the product, and it is also the limit.

## It cannot see around itself

If an agent still has a database password, a cloud key, or a direct MCP URL in its own configuration, Keywarden cannot revoke that path and will not have a record of it. Point the agent at Keywarden and remove the other credential.

## A bad policy still allows a bad call

No matching rule means deny. A rule you wrote as `allow` will allow the call, including a `SELECT` that invokes a dangerous function. The SQL classifier marks `DROP`, `TRUNCATE`, `ALTER`, `GRANT`, `REVOKE`, other DDL, and multi-statement batches as `destructive`, and `INSERT`, `UPDATE`, `DELETE`, and `MERGE` as `mutating`. It does not understand every function in every extension. Read the rule before you rely on it.

## A stolen admin token approves calls

The dashboard password and `KEYWARDEN_ADMIN_TOKEN` are the human gate. Keywarden is not an identity provider. Put the proxy on a private network, and do not publish the admin token.

## Approvals are for one call

An approval is bound to the hash of that call's arguments. It does not grant the agent the same tool for later calls. If the proxy restarts while a call is waiting, or after a human approved but before the upstream call finished, the call is expired and is not forwarded. The agent has to try again.

## Stdio is the host

An `mcp_stdio` tool runs on the proxy machine as the proxy's user. Demo mode disables it. Do not register a stdio server you would not run yourself.

## The public demo is fake

The hosted demo refuses secrets that are not part of the seeded fictional set, and the locked demo refuses agent, tool, and policy edits. Do not type a real password, DSN, or API token into it.

## Out of scope

Keywarden is not a secrets manager for the rest of your company, a SIEM, or a replacement for cloud IAM. An OPA-style policy engine can sit behind the current evaluator later. It is not in this version. Generic HTTP proxies and per-cloud IAM integrations are not in this version either.
