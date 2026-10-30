import { applyPolicy } from "@/lib/actions";
import { kw, KwError, loadMeta, pretty, when, type Policy } from "@/lib/api";

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
  const [meta, history] = await Promise.all([
    loadMeta(),
    kw<{ policies: Policy[] }>("/v1/policies"),
  ]);
  const active = history.policies.find((item) => item.active) ?? null;
  return (
    <>
      <h1>Policy</h1>
      <p className="lede">The first matching rule wins. If nothing matches, the call is denied.</p>
      {error ? <p className="error">{error}</p> : null}
      {active ? (
        <p className="meta">
          Active version {active.version}. Saving creates the next version.
        </p>
      ) : (
        <p className="empty">No policy is active, so every call is denied.</p>
      )}
      {meta.demo_lock ? (
        <pre>{pretty(active?.document ?? {})}</pre>
      ) : (
        <form action={applyPolicy}>
          <label htmlFor="document">Policy document</label>
          <textarea id="document" name="document" className="mono" rows={18} defaultValue={active ? pretty(active.document) : starter} />
          <div className="row">
            <button type="submit">Apply policy</button>
          </div>
        </form>
      )}

      <h2>Version history</h2>
      <p className="lede">
        Every applied document is kept. Only one version is active and evaluated.
      </p>
      {history.policies.length === 0 ? <p className="empty">No versions yet.</p> : null}
      {history.policies.map((item) => (
        <details className="card" key={item.id} open={item.active}>
          <summary>
            <span className="meta">
              <span>version {item.version}</span>
              <span>{item.active ? <span className="pill ok">active</span> : <span className="pill">retired</span>}</span>
              <span>{when(item.created_at)}</span>
            </span>
          </summary>
          <pre>{pretty(item.document)}</pre>
        </details>
      ))}
    </>
  );
}
