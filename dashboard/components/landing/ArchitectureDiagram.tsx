"use client";

import { useState } from "react";

export function ArchitectureDiagram() {
  const [activeStep, setActiveStep] = useState<number>(2);

  const steps = [
    {
      num: "01",
      name: "Ephemeral Task Scoping",
      desc: "Agent requests a temporary task token (kwt_...) scoped to specific tool IDs. Root keys never touch LLM memory. Default lifetime is 15 minutes.",
      tech: "POST /v1/tasks",
      code: "kwt_sup_8819af02b1c4 (Expires in 15m)"
    },
    {
      num: "02",
      name: "Streamable HTTP MCP Ingress",
      desc: "Agent calls Keywarden's POST /mcp endpoint. Tool calls arrive in standard Model Context Protocol JSON-RPC format with arguments canonicalized.",
      tech: "POST /mcp",
      code: "tools/call: orders__query { sql: '...' }"
    },
    {
      num: "03",
      name: "Deterministic Policy Evaluation",
      desc: "Ordered declarative JSON rules or compiled Rego evaluate the request. Fast allow, fast deny (no upstream connection opened), or paused for human sign-off.",
      tech: "Go Core Engine (OPA Rego)",
      code: "Effect: ALLOW | DENY | REQUIRE_APPROVAL"
    },
    {
      num: "04",
      name: "Encrypted Vault Substitution",
      desc: "Only upon verified policy clearance does Keywarden unlock the tool secret (AES-256-GCM) and sign or inject sealed headers into the upstream request.",
      tech: "In-Memory Vault",
      code: "Inject sealed DSN / AWS SigV4 / Bearer Token"
    },
    {
      num: "05",
      name: "SHA-256 Hash Chaining",
      desc: "Before returning the response to the agent, Keywarden writes an immutable audit record linking to the previous block's SHA-256 hash. Verifiable at any time.",
      tech: "Per-Org Advisory Lock",
      code: "GET /v1/audit/verify -> Intact"
    }
  ];

  return (
    <section id="architecture" className="kw-section kw-arch-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Zero-Trust Data Plane</span>
          <h2 className="kw-section-title">How Keywarden Enforces Least Privilege</h2>
          <p className="kw-section-subtitle">
            A hardened Go proxy sitting cleanly between autonomous agent runners and sensitive production resources.
          </p>
        </div>

        {/* Visual Step-by-Step Interactive Pipeline */}
        <div className="kw-arch-pipeline">
          {steps.map((step, idx) => (
            <button
              key={idx}
              onClick={() => setActiveStep(idx)}
              className={`kw-arch-node ${activeStep === idx ? "kw-arch-node-active" : ""}`}
            >
              <div className="kw-node-top">
                <span className="kw-node-num">{step.num}</span>
                <span className="kw-node-tech">{step.tech}</span>
              </div>
              <h3 className="kw-node-name">{step.name}</h3>
              <p className="kw-node-desc">{step.desc}</p>
              <div className="kw-node-code">
                <code>{step.code}</code>
              </div>
            </button>
          ))}
        </div>

        {/* Deep-Dive Card for Selected Step */}
        <div className="kw-arch-details-card">
          <div className="kw-details-header">
            <span className="kw-details-pill">Step {steps[activeStep].num} Deep Dive</span>
            <span className="kw-details-title">{steps[activeStep].name}</span>
          </div>
          <p className="kw-details-body">{steps[activeStep].desc}</p>
          <div className="kw-details-specs">
            <div className="kw-spec-item">
              <span className="kw-spec-label">Protocol Boundary:</span>
              <span className="kw-spec-val font-mono">Streamable HTTP (JSON-RPC)</span>
            </div>
            <div className="kw-spec-item">
              <span className="kw-spec-label">Latency Profile:</span>
              <span className="kw-spec-val font-mono kw-text-accent">&lt; 0.4 ms (in-memory)</span>
            </div>
            <div className="kw-spec-item">
              <span className="kw-spec-label">Supported Upstreams:</span>
              <span className="kw-spec-val font-mono">Postgres, MCP HTTP, AWS SigV4, GCP/Azure HTTPS</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
