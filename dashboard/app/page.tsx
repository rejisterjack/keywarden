import { ArchitectureDiagram } from "@/components/landing/ArchitectureDiagram";
import { AuditLedgerExplorer } from "@/components/landing/AuditLedgerExplorer";
import { ComparisonTable } from "@/components/landing/ComparisonTable";
import { FaqSection } from "@/components/landing/FaqSection";
import { Footer } from "@/components/landing/Footer";
import { HeroInterceptor } from "@/components/landing/HeroInterceptor";
import { Navbar } from "@/components/landing/Navbar";
import { PolicyVisualizer } from "@/components/landing/PolicyVisualizer";
import { QuickstartSection } from "@/components/landing/QuickstartSection";
import { TriadSection } from "@/components/landing/TriadSection";
import { loadMeta } from "@/lib/api";
import { hasSession } from "@/lib/session";
import Link from "next/link";

export default async function HomePage() {
  const [sessionActive, meta] = await Promise.all([
    hasSession(),
    loadMeta(),
  ]);

  return (
    <div className="kw-landing-root">
      {meta.demo ? (
        <div className="kw-session-banner">
          <span>Fictional demo. Do not register real credentials.</span>
        </div>
      ) : null}
      {sessionActive && (
        <div className="kw-session-banner">
          <span>Active operator session.</span>
          <Link href="/approvals" className="kw-session-link">
            Open approvals
          </Link>
        </div>
      )}

      <Navbar hasSession={sessionActive} />

      <main>
        <HeroInterceptor />
        <ComparisonTable />
        <ArchitectureDiagram />
        <PolicyVisualizer />
        <AuditLedgerExplorer />
        <TriadSection />
        <QuickstartSection />
        <FaqSection />
      </main>

      <Footer />
    </div>
  );
}
