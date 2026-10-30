import { Nav } from "@/components/nav";
import { logout } from "@/lib/actions";
import { loadMeta } from "@/lib/api";
import { requireSession } from "@/lib/session";

export default async function ConsoleLayout({ children }: { children: React.ReactNode }) {
  await requireSession();
  const meta = await loadMeta();
  return (
    <div className="shell">
      <aside className="rail">
        <p className="brand">
          Keywarden
          <span>Permission proxy</span>
        </p>
        <Nav />
        <form action={logout}>
          <button type="submit">Sign out</button>
        </form>
      </aside>
      <div className="main">
        {meta.demo ? (
          <p className="banner">Demo mode. The data here is fictional. Do not register real credentials.</p>
        ) : null}
        {meta.single_tenant === false ? (
          <p className="banner">Multi-tenant mode. This session is scoped to one organization admin token.</p>
        ) : null}
        {children}
      </div>
    </div>
  );
}
