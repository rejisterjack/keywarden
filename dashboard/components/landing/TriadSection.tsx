"use client";

export function TriadSection() {
  const triad = [
    {
      name: "Keywarden",
      badge: "Permission & Governance",
      role: "What agents may do and who approved it",
      desc: "Least-privilege proxy, task token scoping, real-time policy interception, and cryptographic audit logging.",
      linkText: "Active Runtime",
      isCurrent: true,
      href: "#interceptor",
    },
    {
      name: "Boxwren",
      badge: "In-Process Containment",
      role: "Where untrusted code runs safely",
      desc: "Microsecond WebAssembly SFI runtime. Zero Docker daemons, deny-by-default capabilities, instant kill switch.",
      linkText: "Explore Boxwren →",
      isCurrent: false,
      href: "https://github.com/rejisterjack/boxwren",
    },
    {
      name: "Undoom",
      badge: "Regression Immunity",
      role: "Did a known failure come back",
      desc: "Replay agent failure trajectories, verify fixes, and lock them in as deterministic regression tests.",
      linkText: "Explore Undoom →",
      isCurrent: false,
      href: "https://github.com/rejisterjack/undoom",
    },
  ];

  return (
    <section id="triad" className="kw-section kw-triad-section">
      <div className="kw-container">
        <div className="kw-section-header">
          <span className="kw-section-tag">Defense in Depth</span>
          <h2 className="kw-section-title">The Three Pillars of Agent Reliability</h2>
          <p className="kw-section-subtitle">
            Autonomous systems require multiple safety boundaries. Keywarden forms the critical access control layer.
          </p>
        </div>

        <div className="kw-triad-grid">
          {triad.map((item, idx) => (
            <div key={idx} className={`kw-triad-card ${item.isCurrent ? "kw-triad-current" : ""}`}>
              <div className="kw-triad-top">
                <span className="kw-triad-badge">{item.badge}</span>
                {item.isCurrent && <span className="kw-current-indicator">Current Sentinel</span>}
              </div>
              <h3 className="kw-triad-title">{item.name}</h3>
              <p className="kw-triad-role">{item.role}</p>
              <p className="kw-triad-desc">{item.desc}</p>
              <div className="kw-triad-footer">
                <a
                  href={item.href}
                  target={item.isCurrent ? undefined : "_blank"}
                  rel={item.isCurrent ? undefined : "noopener noreferrer"}
                  className={`kw-triad-link ${item.isCurrent ? "kw-triad-link-active" : ""}`}
                >
                  {item.linkText}
                </a>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
