"use client";

import { useState } from "react";

export function QuickstartSection() {
  const [copiedCompose, setCopiedCompose] = useState(false);
  const [copiedCli, setCopiedCli] = useState(false);

  const composeCmd = `docker compose -f deploy/compose/docker-compose.yml up --build`;

  const cliSnippet = `# Mint short-lived 15m task token for agent 'support'
keywarden task create --agent support --ttl 15m

# Apply declarative policy rules
keywarden policy apply examples/policy.json

# Mathematically verify the SHA-256 audit chain
keywarden audit verify`;

  const copy = (text: string, setFn: (v: boolean) => void) => {
    navigator.clipboard?.writeText(text);
    setFn(true);
    setTimeout(() => setFn(false), 2000);
  };

  return (
    <section id="quickstart" className="kw-section kw-quickstart-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Production Quickstart</span>
          <h2 className="kw-section-title">Up and Running in 60 Seconds</h2>
          <p className="kw-section-subtitle">
            Docker Compose boots PostgreSQL, the Keywarden Go proxy on <code>:8080</code>, and the operator dashboard on <code>:3000</code>.
          </p>
        </div>

        <div className="kw-quickstart-grid">
          {/* Card 1: Docker Compose */}
          <div className="kw-card kw-code-card">
            <div className="kw-code-header">
              <span className="kw-terminal-label">Option 1: Complete Docker Stack</span>
              <button
                onClick={() => copy(composeCmd, setCopiedCompose)}
                className="kw-copy-btn"
              >
                {copiedCompose ? "Copied!" : "Copy Command"}
              </button>
            </div>
            <pre className="kw-terminal-code">
              <code>{composeCmd}</code>
            </pre>
            <div className="kw-card-foot">
              <span className="kw-hint">Includes pre-seeded fictional support agent, orders DB, and demo policy.</span>
            </div>
          </div>

          {/* Card 2: CLI Operations */}
          <div className="kw-card kw-code-card">
            <div className="kw-code-header">
              <span className="kw-terminal-label">Option 2: Native CLI Workflow</span>
              <button
                onClick={() => copy(cliSnippet, setCopiedCli)}
                className="kw-copy-btn"
              >
                {copiedCli ? "Copied!" : "Copy Commands"}
              </button>
            </div>
            <pre className="kw-terminal-code">
              <code>{cliSnippet}</code>
            </pre>
            <div className="kw-card-foot">
              <span className="kw-hint">Talks directly to running Keywarden proxy on <code>KEYWARDEN_URL</code>.</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
