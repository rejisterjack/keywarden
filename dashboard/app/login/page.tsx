import { login } from "@/lib/actions";
import { loadMeta } from "@/lib/api";
import { hasSession } from "@/lib/session";
import { redirect } from "next/navigation";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  if (await hasSession()) redirect("/");
  const { error } = await searchParams;
  const meta = await loadMeta();
  return (
    <main className="login">
      <section className="card">
        <h1>Keywarden</h1>
        <p className="lede">Sign in to review what agents are allowed to do.</p>
        {meta.demo ? (
          <p className="banner">This demo holds fictional data. Do not enter real credentials.</p>
        ) : null}
        {error ? <p className="error">{error}</p> : null}
        <form action={login}>
          <label htmlFor="password">Password</label>
          <input id="password" name="password" type="password" autoComplete="current-password" required />
          <div className="row">
            <button type="submit">Continue</button>
          </div>
        </form>
      </section>
    </main>
  );
}
