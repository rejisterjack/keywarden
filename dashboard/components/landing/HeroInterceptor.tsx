"use client";

import { useState } from "react";

type Scenario = "read" | "mutate" | "destroy";

export function HeroInterceptor() {
  const [activeScenario, setActiveScenario] = useState<Scenario>("destroy");
  const [isApproved, setIsApproved] = useState(false);
  const [copiedToken, setCopiedToken] = useState(false);

  const scenarios = {
    read: {
      badge: "ALLOW",
      badgeClass: "kw-status-allow",
      title: "Routine Order Query",
      callPayload: `{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "orders__query",
    "arguments": {
      "sql": "SELECT id, status, customer FROM orders WHERE id = 1042"
    }
  }
}`,
      classification: "READ ONLY (SAFE)",
      ruleId: "orders-read",
      decisionText: "Forwarded directly to PostgreSQL (orders_owner)",
      latency: "0.38 ms",
      hash: "7f4c9a8201bfa2...d19b",
      upstreamResult: `{
  "rows": [
    { "id": 1042, "status": "shipped", "customer": "Ada Lovelace" }
  ]
}`
    },
    mutate: {
      badge: isApproved ? "OPERATOR APPROVED" : "REQUIRE_APPROVAL",
      badgeClass: isApproved ? "kw-status-allow" : "kw-status-pending",
      title: "High-Risk Customer Refund",
      callPayload: `{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "orders__query",
    "arguments": {
      "sql": "UPDATE orders SET status = 'refunded' WHERE id = 1042"
    }
  }
}`,
      classification: "MUTATING SQL",
      ruleId: "orders-mutate",
      decisionText: isApproved
        ? "Human verified argument hash. Executed with sealed credentials."
        : "Paused in Keywarden Memory. Zero database traffic until human signs off.",
      latency: isApproved ? "1.12 s (Human Gate)" : "Waiting (timeout: 120s)",
      hash: "a93e1104e7c388...8bf2",
      upstreamResult: isApproved
        ? `{ "rows_affected": 1, "status": "refunded", "decided_by": "secops_admin" }`
        : `{ "status": "pending_approval", "request_id": "app_982", "expires_in": "118s" }`
    },
    destroy: {
      badge: "DENY (BLOCKED FAST)",
      badgeClass: "kw-status-deny",
      title: "Prompt Injection Attack",
      callPayload: `{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "orders__query",
    "arguments": {
      "sql": "DROP TABLE orders CASCADE; -- bypass"
    }
  }
}`,
      classification: "DESTRUCTIVE DDL",
      ruleId: "orders-destroy",
      decisionText: "Connection refused at proxy boundary. Upstream database NEVER touched.",
      latency: "0.19 ms",
      hash: "c04d88e14fa620...33a0",
      upstreamResult: `{
  "error": {
    "code": -32603,
    "message": "Keywarden policy violation: destructive actions are strictly denied"
  }
}`
    }
  };

  const current = scenarios[activeScenario];

  const handleCopyToken = () => {
    navigator.clipboard?.writeText("kwt_sup_8819af02b1c4");
    setCopiedToken(true);
    setTimeout(() => setCopiedToken(false), 2000);
  };

  return (
    <section id="interceptor" className="kw-section kw-hero-section">
      <div className="kw-hero-glow" aria-hidden="true" />
      
      <div className="kw-container kw-hero-container">
        {/* Hero Copy */}
        <div className="kw-hero-copy">
          <div className="kw-hero-pill">
            <span className="kw-pulse-dot" />
            <span>Zero-Trust Permission Layer for Autonomous Agents</span>
          </div>

          <h1 className="kw-hero-title">
            Agents should <span className="kw-text-gradient">not hold the keys.</span>
          </h1>

          <p className="kw-hero-subtitle">
            Keywarden is a least-privilege proxy for AI agent tool calls. Agents get short-lived 
            15-minute task tokens; real credentials never leave the vault. Safe calls pass instantly, 
            destructive DDL fails fast, and risky mutations pause for human sign-off.
          </p>

          <div className="kw-hero-ctas">
            <a href="/login" className="kw-btn-primary kw-btn-lg">
              <span>Launch Live Demo</span>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
            <a href="#quickstart" className="kw-btn-secondary kw-btn-lg">
              <span>Deploy in 60s</span>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M19 14l-7 7m0 0l-7-7m7 7V3" />
              </svg>
            </a>
          </div>

          {/* Quick Stats / Micro-proof */}
          <div className="kw-hero-metrics">
            <div className="kw-metric-item">
              <span className="kw-metric-num">&lt; 0.4 ms</span>
              <span className="kw-metric-label">Allow-Path Overhead</span>
            </div>
            <div className="kw-metric-divider" />
            <div className="kw-metric-item">
              <span className="kw-metric-num">15 min</span>
              <span className="kw-metric-label">Default Token TTL</span>
            </div>
            <div className="kw-metric-divider" />
            <div className="kw-metric-item">
              <span className="kw-metric-num">SHA-256</span>
              <span className="kw-metric-label">Merkle Audit Chain</span>
            </div>
          </div>
        </div>

        {/* Interactive Interceptor Terminal Component */}
        <div className="kw-interceptor-card">
          {/* Header Bar */}
          <div className="kw-card-header">
            <div className="kw-card-title-group">
              <span className="kw-dot-indicator" />
              <span className="kw-card-title">Real-Time Permission Interceptor</span>
            </div>
            
            <div className="kw-scenario-tabs" role="tablist">
              <button
                role="tab"
                aria-selected={activeScenario === "read"}
                onClick={() => { setActiveScenario("read"); setIsApproved(false); }}
                className={`kw-tab ${activeScenario === "read" ? "kw-tab-active" : ""}`}
              >
                Safe Read
              </button>
              <button
                role="tab"
                aria-selected={activeScenario === "mutate"}
                onClick={() => { setActiveScenario("mutate"); setIsApproved(false); }}
                className={`kw-tab ${activeScenario === "mutate" ? "kw-tab-active" : ""}`}
              >
                Risky Mutation (HITL)
              </button>
              <button
                role="tab"
                aria-selected={activeScenario === "destroy"}
                onClick={() => { setActiveScenario("destroy"); setIsApproved(false); }}
                className={`kw-tab ${activeScenario === "destroy" ? "kw-tab-active" : ""}`}
              >
                Malicious DDL Attack
              </button>
            </div>
          </div>

          {/* Two-Pane Terminal Interface */}
          <div className="kw-interceptor-grid">
            {/* Left: Agent Side */}
            <div className="kw-pane kw-pane-left">
              <div className="kw-pane-header">
                <span className="kw-pane-title">1. Untrusted Agent Request (POST /mcp)</span>
                <button
                  onClick={handleCopyToken}
                  className="kw-token-chip"
                  title="Click to copy token string"
                >
                  <span className="kw-token-text">Bearer kwt_sup_8819...</span>
                  <span className="kw-token-copy">{copiedToken ? "Copied!" : "Copy"}</span>
                </button>
              </div>

              <div className="kw-code-wrap">
                <pre className="kw-code-block">
                  <code>{current.callPayload}</code>
                </pre>
              </div>

              <div className="kw-pane-footer">
                <div className="kw-footer-tag">
                  <span className="kw-label">Classification:</span>
                  <span className="kw-value">{current.classification}</span>
                </div>
                <div className="kw-footer-tag">
                  <span className="kw-label">Policy Rule:</span>
                  <span className="kw-value font-mono">{current.ruleId}</span>
                </div>
              </div>
            </div>

            {/* Right: Keywarden Enforcement */}
            <div className="kw-pane kw-pane-right">
              <div className="kw-pane-header">
                <span className="kw-pane-title">2. Keywarden Proxy Decision</span>
                <span className={`kw-status-badge ${current.badgeClass}`}>
                  {current.badge}
                </span>
              </div>

              <div className="kw-decision-body">
                <div className="kw-decision-narrative">
                  <span className="kw-narrative-label">Enforcement Action:</span>
                  <p className="kw-narrative-text">{current.decisionText}</p>
                </div>

                {activeScenario === "mutate" && !isApproved && (
                  <div className="kw-approval-prompt">
                    <div className="kw-prompt-info">
                      <strong>Operator Gate Required:</strong>
                      <span>Argument hash <code>{current.hash.slice(0, 12)}</code> paused for operator confirmation.</span>
                    </div>
                    <button
                      onClick={() => setIsApproved(true)}
                      className="kw-btn-approve"
                    >
                      Authorize Call →
                    </button>
                  </div>
                )}

                <div className="kw-code-wrap kw-upstream-code">
                  <div className="kw-code-label">Agent Tool Response:</div>
                  <pre className="kw-code-block">
                    <code>{current.upstreamResult}</code>
                  </pre>
                </div>
              </div>

              <div className="kw-pane-footer">
                <div className="kw-footer-tag">
                  <span className="kw-label">Evaluation:</span>
                  <span className="kw-value kw-text-accent">{current.latency}</span>
                </div>
                <div className="kw-footer-tag">
                  <span className="kw-label">Chain Link:</span>
                  <span className="kw-value kw-text-cyan font-mono">{current.hash}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
