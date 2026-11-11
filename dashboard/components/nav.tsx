"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  ["/approvals", "Approvals"],
  ["/tasks", "Tasks"],
  ["/audit", "Audit"],
  ["/agents", "Agents"],
  ["/tools", "Tools"],
  ["/policy", "Policy"],
];

export function Nav() {
  const path = usePathname();
  return (
    <nav>
      {links.map(([href, label]) => (
        <Link key={href} href={href} aria-current={path === href ? "page" : undefined}>
          {label}
        </Link>
      ))}
      <div style={{ marginTop: "1rem", paddingTop: "1rem", borderTop: "1px solid rgba(255,255,255,0.1)" }}>
        <Link href="/" style={{ opacity: 0.75, fontSize: "13px" }}>
          ← View Landing Page
        </Link>
      </div>
    </nav>
  );
}
