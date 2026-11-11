"use client";

import { useState } from "react";

export function AuditLedgerExplorer() {
  const [verifying, setVerifying] = useState(false);
  const [verified, setVerified] = useState(true);

  const handleVerify = () => {
    setVerifying(true);
    setTimeout(() => {
      setVerifying(false);
      setVerified(true);
    }, 600);
  };

  const blocks = [
    {
      seq: 104,
      eventType: "task.created",
      timestamp: "2026-11-09 01:23:50 UTC",
      actor: "agent:support",
      tool: "orders, catalog",
      prevHash: "0000000000000000000000000000000000000000000000000000000000000000 (genesis)",
      currentHash: "3b91a7c02b88f19da4c05e32187f4c9a8201bfa27d04e31899bf918231e4a102",
      status: "SEALED",
    },
    {
      seq: 105,
      eventType: "tool.intercepted",
      timestamp: "2026-11-09 01:23:52 UTC",
      actor: "agent:support",
      tool: "orders__query (mutating)",
      prevHash: "3b91a7c02b88f19da4c05e32187f4c9a8201bfa27d04e31899bf918231e4a102",
      currentHash: "7f4c9a8201bfa27d04e31899bf918231e4a102a93e1104e7c388bc61109a8bf2",
      status: "SEALED",
    },
    {
      seq: 106,
      eventType: "approval.granted",
      timestamp: "2026-11-09 01:23:58 UTC",
      actor: "secops:jack",
      tool: "orders__query (id: 1042)",
      prevHash: "7f4c9a8201bfa27d04e31899bf918231e4a102a93e1104e7c388bc61109a8bf2",
      currentHash: "a93e1104e7c388bc61109a8bf2c04d88e14fa620f4c01823d04e31899bf91823",
      status: "SEALED",
    },
  ];

  return (
    <section id="audit" className="kw-section kw-audit-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Tamper-Evident Ledger</span>
          <h2 className="kw-section-title">Cryptographically Chained Audit Trail</h2>
          <p className="kw-section-subtitle">
            Every task token minted, tool call intercepted, policy match evaluated, and human sign-off recorded
            is locked into an append-only SHA-256 hash chain per organization.
          </p>
        </div>

        <div className="kw-audit-card">
          <div className="kw-audit-toolbar">
            <div className="kw-audit-status">
              <span className={`kw-verify-badge ${verified ? "kw-verify-ok" : ""}`}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
                  <path d="M20 6L9 17l-5-5" />
                </svg>
                <span>keywarden audit verify: CHAIN INTACT (3/3 BLOCKS)</span>
              </span>
            </div>

            <button
              onClick={handleVerify}
              disabled={verifying}
              className="kw-btn-secondary kw-btn-sm"
            >
              {verifying ? "Recomputing Hashes..." : "Recompute SHA-256 Chain"}
            </button>
          </div>

          <div className="kw-audit-blocks">
            {blocks.map((b) => (
              <div key={b.seq} className="kw-block-item">
                <div className="kw-block-header">
                  <div className="kw-block-seq">
                    <span className="kw-seq-num">#{b.seq}</span>
                    <span className="kw-event-type">{b.eventType}</span>
                  </div>
                  <span className="kw-block-timestamp">{b.timestamp}</span>
                </div>

                <div className="kw-block-fields">
                  <div className="kw-field-row">
                    <span className="kw-field-label">Actor & Tool:</span>
                    <span className="kw-field-val font-mono">{b.actor} &rarr; {b.tool}</span>
                  </div>
                  <div className="kw-field-row">
                    <span className="kw-field-label">Prev Hash:</span>
                    <span className="kw-field-hash font-mono">{b.prevHash}</span>
                  </div>
                  <div className="kw-field-row">
                    <span className="kw-field-label">Block Hash:</span>
                    <span className="kw-field-hash font-mono kw-text-cyan">{b.currentHash}</span>
                  </div>
                </div>

                <div className="kw-block-footer">
                  <span className="kw-seal-pill">HMAC Sealed</span>
                  <span className="kw-advisory-pill">Advisory Locked Per-Org</span>
                </div>
              </div>
            ))}
          </div>

          <div className="kw-audit-footer-note">
            <span>Verify anytime from CLI:</span>
            <code className="kw-inline-code">keywarden audit verify --org 00000000-0000-4000-8000-000000000001</code>
          </div>
        </div>
      </div>
    </section>
  );
}
