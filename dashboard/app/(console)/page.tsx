import { decide } from "@/lib/actions";
import { kw, pretty, when, type Approval } from "@/lib/api";

export default async function ApprovalsPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const body = await kw<{ approvals: Approval[] }>("/v1/approvals?status=pending");
  return (
    <>
      <h1>Approvals</h1>
      <p className="lede">Risky calls wait here. A decision applies only to the argument hash shown.</p>
      {error ? <p className="error">{error}</p> : null}
      {body.approvals.length === 0 ? <p className="empty">No calls are waiting.</p> : null}
      {body.approvals.map((item) => (
        <article className="card" key={item.id}>
          <h2>{item.tool_name}</h2>
          <div className="meta">
            <span className="pill pending">pending</span>
            <span>agent {item.agent_id}</span>
            <span>tool {item.tool_id}</span>
            <span>rule {item.rule_id || "—"}</span>
            <span>expires {when(item.expires_at)}</span>
          </div>
          <p className="meta">{item.reason}</p>
          <pre>{pretty(item.args_redacted)}</pre>
          <p className="meta">hash {item.args_hash}</p>
          <form action={decide}>
            <input type="hidden" name="id" value={item.id} />
            <label htmlFor={`comment-${item.id}`}>Comment</label>
            <input id={`comment-${item.id}`} name="comment" placeholder="Why this decision" />
            <div className="row">
              <button name="decision" value="approve" type="submit">
                Approve
              </button>
              <button className="danger" name="decision" value="deny" type="submit">
                Deny
              </button>
            </div>
          </form>
        </article>
      ))}
    </>
  );
}
