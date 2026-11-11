"use client";

import { useState } from "react";

export function FaqSection() {
  const [openIdx, setOpenIdx] = useState<number | null>(0);

  const faqs = [
    {
      q: "Does Keywarden add noticeable latency to agent tool execution?",
      a: "No. Keywarden is written in Go and compiles to a single native binary. Unrestricted allow-path evaluations execute in under 0.4 milliseconds in-memory. Destructive queries fail fast in ~0.2 milliseconds before opening any socket to the upstream database."
    },
    {
      q: "What if an agent has credentials directly in its environment?",
      a: "Keywarden governs tool calls that actually route through it. The foundational best practice is to remove root passwords from the agent runner and issue ephemeral task tokens (kwt_...) pointing at Keywarden's POST /mcp endpoint. If an agent retains a direct database password, Keywarden cannot revoke that bypass."
    },
    {
      q: "How does human-in-the-loop approval work during long-running tasks?",
      a: "When a policy rule dictates 'require_approval', the proxy pauses the JSON-RPC call in memory and writes a pending approval row bound to the SHA-256 hash of the exact arguments. An operator reviews the redacted query in the dashboard and clicks Approve or Deny. A rule timeout (e.g. 120s) expires the request if not decided."
    },
    {
      q: "Can Keywarden support multiple teams and organizations?",
      a: "Yes. In multi-tenant mode (KEYWARDEN_SINGLE_TENANT=0), operators create organizations using the process admin token, returning dedicated kwa_... tokens. Each tenant has isolated policies, dedicated task tokens, isolated database advisory locks, and independent SHA-256 hash chains."
    },
    {
      q: "Is Keywarden a full replacement for IAM or secrets managers?",
      a: "No. Keywarden is specifically an agent tool governance proxy. It holds secrets for the tools it exposes and enforces runtime policy on tool calls. It is not an enterprise identity provider (IdP), SIEM, or replacement for AWS IAM."
    }
  ];

  return (
    <section className="kw-section kw-faq-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Frequently Answered Questions</span>
          <h2 className="kw-section-title">Security & Technical Boundaries</h2>
          <p className="kw-section-subtitle">
            Clear, honest documentation on what Keywarden protects and its architectural limits.
          </p>
        </div>

        <div className="kw-faq-list">
          {faqs.map((faq, idx) => (
            <div
              key={idx}
              className={`kw-faq-item ${openIdx === idx ? "kw-faq-open" : ""}`}
            >
              <button
                onClick={() => setOpenIdx(openIdx === idx ? null : idx)}
                className="kw-faq-question"
                aria-expanded={openIdx === idx}
              >
                <span>{faq.q}</span>
                <span className="kw-faq-chevron">{openIdx === idx ? "−" : "+"}</span>
              </button>
              {openIdx === idx && (
                <div className="kw-faq-answer">
                  <p>{faq.a}</p>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
