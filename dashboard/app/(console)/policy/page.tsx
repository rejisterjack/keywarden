import { applyPolicy } from "@/lib/actions";
import { kw, KwError, loadMeta, pretty, type Policy } from "@/lib/api";

const starter = `{
  "version": 1,
  "rules": [
    {
      "id": "orders-read",
      "match": { "agent": "support", "tool": "orders", "name": "query" },
      "when": { "sql": "read" },
      "effect": "allow"
    }
  ]
}`;

export default async function PolicyPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const meta = await loadMeta();
  let policy: Policy | null = null;
  try {
    policy = await kw<Policy>("/v1/policies/active");
  } catch (err) {
    if (!(err instanceof KwError)) throw err;
  }
  return (
    <>
      <h1>Policy</h1>
      <p className="lede">The first matching rule wins. If nothing matches, the call is denied.</p>
      {error ? <p className="error">{error}</p> : null}
      {policy ? (
        <p className="meta">
          Active version {policy.version}. Saving creates the next version.
        </p>
      ) : (
        <p className="empty">No policy is active, so every call is denied.</p>
      )}
      {meta.demo_lock ? (
        <pre>{pretty(policy?.document ?? {})}</pre>
      ) : (
        <form action={applyPolicy}>
          <label htmlFor="document">Policy document</label>
          <textarea id="document" name="document" className="mono" rows={18} defaultValue={policy ? pretty(policy.document) : starter} />
          <div className="row">
            <button type="submit">Apply policy</button>
          </div>
        </form>
      )}
    </>
  );
}
