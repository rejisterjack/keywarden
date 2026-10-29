"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  ["/", "Approvals"],
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
    </nav>
  );
}
