"use client";

export function ComparisonTable() {
  const comparisons = [
    {
      feature: "Credential Exposure in Agent Prompt/Context",
      direct: "Permanent root DB passwords, AWS IAM keys, or API tokens stored directly in agent configuration.",
      directBad: true,
      keywarden: "Zero real credentials in LLM context. Agents hold ephemeral 15-minute task tokens (kwt_...).",
      keywardenGood: true,
    },
    {
      feature: "Destructive DDL & Prompt Injections",
      direct: "Full execution. Rogue LLMs can drop tables, alter schema, or truncate historical records without restriction.",
      directBad: true,
      keywarden: "Fast-Fail Termination (<0.2ms). DDL and dangerous batches are refused before reaching upstream.",
      keywardenGood: true,
    },
    {
      feature: "High-Risk State Mutations (Refunds, Deletes)",
      direct: "Executed automatically without oversight or verification.",
      directBad: true,
      keywarden: "Human-in-the-Loop Gateway. Pauses in proxy memory and binds decision strictly to argument hash.",
      keywardenGood: true,
    },
    {
      feature: "Audit & Non-Repudiation Trail",
      direct: "Scattered, incomplete application logs. No proof of LLM request integrity.",
      directBad: true,
      keywarden: "Append-only, SHA-256 hash-chained Merkle ledger with one-command mathematical verification.",
      keywardenGood: true,
    },
    {
      feature: "Protocol Standard",
      direct: "Ad-hoc SDKs, fragmented custom wrappers, inconsistent error handling.",
      directBad: false,
      keywarden: "Native Streamable HTTP Model Context Protocol (POST /mcp). Plug-and-play for all MCP clients.",
      keywardenGood: true,
    },
    {
      feature: "Infrastructure Footprint",
      direct: "Vendor lock-in SaaS dependencies or fragile manual proxies.",
      directBad: false,
      keywarden: "Self-hosted Go core. Compiles to a single lightweight binary, runs locally or in Docker Compose.",
      keywardenGood: true,
    },
  ];

  return (
    <section className="kw-section kw-comparison-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Security Architecture Comparison</span>
          <h2 className="kw-section-title">The Naked Credential Trap vs. Zero-Trust Governance</h2>
          <p className="kw-section-subtitle">
            Why enterprise security teams refuse to let autonomous agents communicate directly with production infrastructure.
          </p>
        </div>

        <div className="kw-table-card">
          <table className="kw-comparison-table">
            <thead>
              <tr>
                <th style={{ width: "28%" }}>Capabilities & Risks</th>
                <th style={{ width: "36%" }}>Direct Tool Access (Without Keywarden)</th>
                <th style={{ width: "36%" }}>Keywarden Zero-Trust Proxy</th>
              </tr>
            </thead>
            <tbody>
              {comparisons.map((item, idx) => (
                <tr key={idx}>
                  <td className="kw-feature-name">
                    <strong>{item.feature}</strong>
                  </td>
                  <td className={`kw-cell ${item.directBad ? "kw-cell-bad" : ""}`}>
                    <div className="kw-cell-inner">
                      {item.directBad && <span className="kw-icon-cross">✕</span>}
                      <span>{item.direct}</span>
                    </div>
                  </td>
                  <td className={`kw-cell ${item.keywardenGood ? "kw-cell-good" : ""}`}>
                    <div className="kw-cell-inner">
                      {item.keywardenGood && <span className="kw-icon-check">✓</span>}
                      <span>{item.keywarden}</span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}
