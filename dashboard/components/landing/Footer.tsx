"use client";

import Link from "next/link";

export function Footer() {
  return (
    <footer className="kw-footer">
      <div className="kw-container kw-footer-container">
        <div className="kw-footer-brand">
          <div className="kw-footer-logo">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <path d="M12 8v4" />
              <path d="M12 16h.01" />
            </svg>
            <span className="kw-footer-title">Keywarden</span>
          </div>
          <p className="kw-footer-tagline">
            Least-privilege permission and governance proxy for AI agent tool calls.
          </p>
          <div className="kw-footer-copy">
            &copy; {new Date().getFullYear()} Keywarden Project. Released under MIT License.
          </div>
        </div>

        <div className="kw-footer-cols">
          <div className="kw-footer-col">
            <span className="kw-col-title">Navigation</span>
            <a href="#interceptor">Live Sandbox</a>
            <a href="#architecture">Architecture</a>
            <a href="#policy">Policy Engine</a>
            <a href="#audit">Audit Ledger</a>
            <a href="#quickstart">Quickstart</a>
          </div>

          <div className="kw-footer-col">
            <span className="kw-col-title">Console</span>
            <Link href="/login">Operator Login</Link>
            <Link href="/approvals">Pending Approvals</Link>
            <Link href="/tasks">Mint Task Tokens</Link>
            <Link href="/audit">Verify Chain</Link>
          </div>

          <div className="kw-footer-col">
            <span className="kw-col-title">Ecosystem</span>
            <a href="https://github.com/rejisterjack/keywarden" target="_blank" rel="noopener noreferrer">
              Keywarden GitHub
            </a>
            <a href="https://github.com/rejisterjack/boxwren" target="_blank" rel="noopener noreferrer">
              Boxwren (Wasm SFI)
            </a>
            <a href="https://github.com/rejisterjack/undoom" target="_blank" rel="noopener noreferrer">
              Undoom (Replay Tests)
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
