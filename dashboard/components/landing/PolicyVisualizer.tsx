"use client";

import { useState } from "react";

export function PolicyVisualizer() {
  const [format, setFormat] = useState<"rules" | "rego">("rules");
  const [testTool, setTestTool] = useState<"query_read" | "query_delete" | "catalog_refund">("query_read");

  const rulesCode = `{
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
    },
    {
      "id": "orders-destroy",
      "match": { "agent": "support", "tool": "orders", "name": "query" },
      "when": { "sql": "destructive" },
      "effect": "deny"
    },
    {
      "id": "catalog-lookup",
      "match": { "agent": "support", "tool": "catalog", "name": "lookup_order" },
      "effect": "allow"
    },
    {
      "id": "catalog-refund",
      "match": { "agent": "support", "tool": "catalog", "name": "refund_order" },
      "effect": "require_approval"
    }
  ]
}`;

  const regoCode = `package keywarden

import rego.v1

# Evaluated in-process via embedded Open Policy Agent engine
decision := {"effect": "allow", "rule_id": "orders-read"} if {
    input.tool_id == "orders"
    input.name == "query"
    input.sql_class == "read"
}

decision := {"effect": "require_approval", "rule_id": "orders-mutate", "timeout_seconds": 120} if {
    input.tool_id == "orders"
    input.name == "query"
    input.sql_class == "mutating"
}

decision := {"effect": "deny", "rule_id": "orders-destroy"} if {
    input.tool_id == "orders"
    input.name == "query"
    input.sql_class == "destructive"
}

visible if {
    input.tool_id == "orders"
    input.name == "query"
}`;

  const testCases = {
    query_read: {
      input: '{ "tool": "orders", "name": "query", "sql_class": "read" }',
      ruleMatch: "orders-read",
      effect: "allow",
      effectClass: "kw-status-allow",
      desc: "Safe read executes immediately; upstream PostgreSQL is queried."
    },
    query_delete: {
      input: '{ "tool": "orders", "name": "query", "sql_class": "destructive" }',
      ruleMatch: "orders-destroy",
      effect: "deny",
      effectClass: "kw-status-deny",
      desc: "Destructive DDL is rejected with code 403 before opening upstream socket."
    },
    catalog_refund: {
      input: '{ "tool": "catalog", "name": "refund_order" }',
      ruleMatch: "catalog-refund",
      effect: "require_approval",
      effectClass: "kw-status-pending",
      desc: "Pauses call in Keywarden proxy, dispatches approval item to operator dashboard."
    }
  };

  const activeTest = testCases[testTool];

  return (
    <section id="policy" className="kw-section kw-policy-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Expressive Policy Engine</span>
          <h2 className="kw-section-title">Ordered Rules or Embedded Rego Modules</h2>
          <p className="kw-section-subtitle">
            Define fine-grained capability policies in straightforward JSON or compile full Open Policy Agent modules.
            First matching rule wins. Unmatched calls are denied by default.
          </p>
        </div>

        <div className="kw-policy-grid">
          {/* Left: Code Editor View */}
          <div className="kw-policy-editor">
            <div className="kw-editor-header">
              <div className="kw-editor-title">
                <span className="kw-window-dot kw-dot-red" />
                <span className="kw-window-dot kw-dot-yellow" />
                <span className="kw-window-dot kw-dot-green" />
                <span className="kw-filename">{format === "rules" ? "examples/policy.json" : "examples/policy.rego"}</span>
              </div>
              <div className="kw-format-toggle">
                <button
                  onClick={() => setFormat("rules")}
                  className={`kw-toggle-btn ${format === "rules" ? "kw-toggle-active" : ""}`}
                >
                  Declarative JSON
                </button>
                <button
                  onClick={() => setFormat("rego")}
                  className={`kw-toggle-btn ${format === "rego" ? "kw-toggle-active" : ""}`}
                >
                  Rego Module
                </button>
              </div>
            </div>
            <pre className="kw-policy-pre">
              <code>{format === "rules" ? rulesCode : regoCode}</code>
            </pre>
          </div>

          {/* Right: Interactive Evaluation Simulator */}
          <div className="kw-policy-tester">
            <div className="kw-tester-header">
              <span className="kw-tester-title">Live Policy Evaluator</span>
              <span className="kw-badge-pill">In-Process Go Core</span>
            </div>

            <div className="kw-tester-controls">
              <label className="kw-test-label">Select Inbound Tool Call Input:</label>
              <div className="kw-test-buttons">
                <button
                  onClick={() => setTestTool("query_read")}
                  className={`kw-test-btn ${testTool === "query_read" ? "kw-test-btn-active" : ""}`}
                >
                  SELECT orders
                </button>
                <button
                  onClick={() => setTestTool("query_delete")}
                  className={`kw-test-btn ${testTool === "query_delete" ? "kw-test-btn-active" : ""}`}
                >
                  DROP orders
                </button>
                <button
                  onClick={() => setTestTool("catalog_refund")}
                  className={`kw-test-btn ${testTool === "catalog_refund" ? "kw-test-btn-active" : ""}`}
                >
                  POST refund
                </button>
              </div>
            </div>

            <div className="kw-eval-box">
              <div className="kw-eval-label">Evaluated Context:</div>
              <pre className="kw-eval-input">
                <code>{activeTest.input}</code>
              </pre>

              <div className="kw-eval-result">
                <div className="kw-result-row">
                  <span className="kw-label">Policy Rule Matched:</span>
                  <span className="kw-value font-mono">{activeTest.ruleMatch}</span>
                </div>
                <div className="kw-result-row">
                  <span className="kw-label">Computed Effect:</span>
                  <span className={`kw-status-badge ${activeTest.effectClass}`}>
                    {activeTest.effect.toUpperCase()}
                  </span>
                </div>
                <p className="kw-result-desc">{activeTest.desc}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
