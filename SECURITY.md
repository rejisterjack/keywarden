# Security

Keywarden holds the credentials agents are not supposed to hold. Treat the proxy, its database, and `KEYWARDEN_VAULT_KEY` as production secrets.

## Reporting

Open a private security advisory on [github.com/rejisterjack/keywarden](https://github.com/rejisterjack/keywarden) or email the maintainer. Please include a way to reproduce the issue and the version you ran. Give us a chance to ship a fix before writing about it in public.

## What the vault does

Tool secrets are sealed with AES-256-GCM before they are written. The tool id is additional authenticated data, so a ciphertext cannot be moved onto another tool. The key is `KEYWARDEN_VAULT_KEY` (32 bytes, base64) and it never goes in the database. Losing the key makes stored secrets unreadable. Anyone who has the key and the database can open every secret.

Task tokens are shown once. The database stores SHA-256 only. Admin authentication compares SHA-256 digests in constant time.

The audit log is append-only with a hash chain: each row's hash is SHA-256 of the previous hash and the canonical JSON payload. `keywarden audit verify` walks the chain. The payload is redacted. The argument hash commits to the original arguments, so a reviewer can prove what was decided without keeping a raw secret in the log.

## Demo mode

`KEYWARDEN_DEMO=1` refuses secrets that are not marked `{"kw_demo":"fake"}`, refuses MCP URLs that are not the configured fixture, and refuses Postgres DSNs that are not the configured orders database. `KEYWARDEN_DEMO_LOCK=1` also refuses agent, tool, and policy changes. Use both on a public demo. The hosted demo is fictional data only.

`KEYWARDEN_REQUIRE_DEMO=1` refuses to start unless demo mode is on. Set it on the public deployment so a config mistake cannot boot a real-credential proxy in that environment.

## Stdio tools

`mcp_stdio` runs a process on the proxy host. The child does not inherit Keywarden's environment (`KEYWARDEN_VAULT_KEY` and the database URL are not passed through). It still runs as the proxy's user. Leave `KEYWARDEN_ALLOW_STDIO` unset unless you mean to do that. Demo mode will not start if stdio is enabled.

## What this does not fix

A policy you wrote can still allow a bad call. A stolen admin token can approve one. A credential the agent already has, outside Keywarden, is unaffected. See [docs/limitations.md](docs/limitations.md).
