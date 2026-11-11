"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

export function Navbar({ hasSession }: { hasSession: boolean }) {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  return (
    <header className={`kw-navbar ${scrolled ? "kw-navbar-scrolled" : ""}`}>
      <div className="kw-nav-container">
        <Link href="/" className="kw-nav-brand">
          <div className="kw-brand-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <path d="M12 8v4" />
              <path d="M12 16h.01" />
            </svg>
          </div>
          <span className="kw-brand-name">Keywarden</span>
          <span className="kw-badge-pill">v0.1.1</span>
        </Link>

        <nav className="kw-nav-links" aria-label="Main Navigation">
          <a href="#interceptor">Live Sandbox</a>
          <a href="#architecture">Architecture</a>
          <a href="#policy">Policy Engine</a>
          <a href="#audit">Audit Ledger</a>
          <a href="#triad">Safety Triad</a>
          <a href="#quickstart">Quickstart</a>
        </nav>

        <div className="kw-nav-actions">
          <a
            href="https://github.com/rejisterjack/keywarden"
            target="_blank"
            rel="noopener noreferrer"
            className="kw-btn-ghost"
            title="View on GitHub"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.53 1.032 1.53 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
            </svg>
            <span>GitHub</span>
          </a>

          {hasSession ? (
            <Link href="/approvals" className="kw-btn-primary">
              <span>Open Console</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </Link>
          ) : (
            <Link href="/login" className="kw-btn-primary">
              <span>Live Demo Console</span>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
