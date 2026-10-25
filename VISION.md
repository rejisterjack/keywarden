# Keywarden — Agent Permission and Governance Layer

> Agents should not hold the keys. Keywarden holds credentials, enforces policy, and records every decision.

**Home:** [keywarden.rejisterjack.com](https://keywarden.rejisterjack.com)

## One-line idea

A least-privilege proxy between AI agents and the tools they call (including MCP servers). Agents never store real credentials; the proxy decides what they may do, for how long, and what requires human approval.

## The problem

Agents reach production faster than security and governance processes can keep up. Many deployments run with broad credentials, weak logging, and no clear answer to “what can this agent do, and who approved it?” When an agent can drop tables, delete infrastructure, or revoke accounts because a developer’s personal access was wired in, that is a process failure—not an accident waiting to happen.

Keywarden addresses tool calls that flow through a controlled path: scoped access, explicit policy, optional human approval, and an auditable record.

## What Keywarden does

- Issues scoped, short-lived credentials per task
- Policy rules that block dangerous actions (e.g. destructive SQL, infra teardown)
- Human approval workflow for risky calls
- Tamper-evident audit log: who requested, what was decided, and why
- MCP-first integration, matching how many agents connect to tools today

## Who it's for

- Teams shipping agents that call databases, cloud APIs, or internal services and need security sign-off
- Platform and security engineers answering governance questions about agent behavior
- Developers using MCP who refuse to embed long-lived secrets in agent configuration

## How you use it

1. Register a tool or MCP server with Keywarden; store real credentials only in the proxy
2. Define policy: which agents may call which tools, what is denied, what needs approval
3. Point the agent at Keywarden instead of the tool; receive short-lived scoped tokens per task
4. Safe calls proceed; blocked calls fail fast; risky calls wait for approval in the dashboard
5. Review the audit log for compliance and incident response

## How it runs

- **Self-hosted (primary):** Docker Compose including Postgres—credentials stay in your environment
- **Optional hosted demo:** Fake data only, for trying the UX without real secrets
- Dashboard for approvals and log review (documented Vercel deployment for demo UI)

## Tech stack

- Go (proxy core—small, fast, easy to ship as a single binary)
- Declarative policy engine (simple rules first; OPA-style extensibility later if needed)
- PostgreSQL (policies, approvals)
- Append-only audit log with hash chaining
- Web dashboard for approvals and log viewing
- Docker packaging for local and server deploys

## Success criteria

- Demonstrable flow: dangerous action blocked, risky action approved/denied, full audit trail
- README and short architecture doc
- Clear documentation of what Keywarden does **not** protect (access outside the proxy)
- Open source at [github.com/rejisterjack/keywarden](https://github.com/rejisterjack/keywarden)

## Limitations & non-goals

- Governs traffic that routes through Keywarden; cannot revoke access the agent already has elsewhere
- Not a full identity provider, SIEM, or secrets manager replacement—focused on agent tool governance
- Enterprise policy languages and every cloud IAM integration are out of initial scope

## Deployment

| Component | Typical hosting |
|-----------|-----------------|
| Proxy | Railway or self-host (Docker) |
| Dashboard (demo) | Vercel |
| Database | Neon or Postgres in Compose |

## Related projects

Keywarden is the **permission** layer in a three-part story for safer agents:

| Project | Role |
|---------|------|
| **Keywarden** | What agents may do and who approved it |
| [Undoom](https://github.com/rejisterjack/undoom) | Replay failures and lock fixes in as regression tests |
| [Boxwren](https://github.com/rejisterjack/boxwren) | Sandboxed execution when code must still run |

Together: govern tool access, prove reliability after failures, and contain untrusted execution.
