import { login } from "@/lib/actions";
import { loadMeta } from "@/lib/api";
import { hasSession } from "@/lib/session";
import Link from "next/link";
import { redirect } from "next/navigation";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  if (await hasSession()) redirect("/approvals");
  const { error } = await searchParams;
  const meta = await loadMeta();
  const demoPw = process.env.KEYWARDEN_DASHBOARD_PASSWORD || "local-demo-password";

  return (
    <main className="login">
      <section className="card" style={{ maxWidth: "440px", width: "100%" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
          <h1 style={{ margin: 0 }}>Keywarden</h1>
          <Link href="/" style={{ fontSize: "13px", color: "var(--muted)", textDecoration: "none" }}>
            ← Back to Home
          </Link>
        </div>
        <p className="lede">Sign in to review what agents are allowed to do.</p>
        {meta.demo ? (
          <p className="banner" style={{ fontSize: "13px" }}>
            This demo holds fictional data. Do not enter real credentials.
            <br />
            <strong>Demo Password:</strong> <code>{demoPw}</code>
          </p>
        ) : null}
        {error ? <p className="error">{error}</p> : null}
        <form action={login}>
          <label htmlFor="password">Dashboard Password</label>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            defaultValue={meta.demo ? demoPw : ""}
            required
            autoFocus
          />
          <div className="row" style={{ marginTop: "1rem" }}>
            <button type="submit" style={{ width: "100%", justifyContent: "center" }}>
              Sign In to Operator Console →
            </button>
          </div>
        </form>
      </section>
    </main>
  );
}
