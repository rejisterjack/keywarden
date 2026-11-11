import { ProxyDown } from "@/components/proxy-down";
import { createTask, revokeTask } from "@/lib/actions";
import { loadList, when, type Task } from "@/lib/api";
import { cookies } from "next/headers";

export default async function TasksPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const issued = (await cookies()).get("kw_issued_token")?.value ?? "";
  const loaded = await loadList<{ tasks: Task[] }>("/v1/tasks");
  if (loaded.error || !loaded.data) {
    return <ProxyDown message={loaded.error ?? "request failed"} />;
  }
  const body = loaded.data;
  return (
    <>
      <h1>Tasks</h1>
      <p className="lede">
        A task token is the short-lived credential an agent presents to{" "}
        <code>POST /mcp</code>. Revoking it stops new calls immediately.
      </p>
      {error ? <p className="error">{error}</p> : null}
      {issued ? (
        <article className="card">
          <h2>Issued token</h2>
          <p className="lede">Copy it now. It stays on this page for two minutes, and it is not put in the address bar.</p>
          <pre>{issued}</pre>
        </article>
      ) : null}
      {body.tasks.length === 0 ? <p className="empty">No tasks yet.</p> : null}
      {body.tasks.map((task) => {
        const revoked = Boolean(task.revoked_at);
        const expired = new Date(task.expires_at).getTime() <= Date.now();
        const status = revoked ? "revoked" : expired ? "expired" : "active";
        return (
          <article className="card" key={task.id}>
            <h2>{task.agent_id}</h2>
            <div className="meta">
              <span className={`pill ${status === "active" ? "ok" : status === "revoked" ? "pending" : ""}`}>
                {status}
              </span>
              <span className="mono">{task.id}</span>
              <span>expires {when(task.expires_at)}</span>
              <span>created {when(task.created_at)}</span>
              <span>
                {task.unrestricted
                  ? "all tools"
                  : `tools ${(task.tool_ids ?? []).join(", ") || "—"}`}
              </span>
            </div>
            {revoked || expired ? null : (
              <form action={revokeTask}>
                <input type="hidden" name="id" value={task.id} />
                <div className="row">
                  <button className="danger" type="submit">
                    Revoke
                  </button>
                </div>
              </form>
            )}
          </article>
        );
      })}
      <form className="card" action={createTask}>
        <h2>Issue a task token</h2>
        <div className="grid">
          <div>
            <label htmlFor="task-agent">Agent id</label>
            <input id="task-agent" name="agent_id" className="mono" required placeholder="support" />
          </div>
          <div>
            <label htmlFor="task-ttl">TTL seconds</label>
            <input id="task-ttl" name="ttl_seconds" type="number" min={1} max={86400} defaultValue={900} />
          </div>
        </div>
        <label htmlFor="task-tools">Tool ids (comma-separated, blank = all)</label>
        <input id="task-tools" name="tool_ids" className="mono" placeholder="orders, catalog" />
        <div className="row">
          <button type="submit">Create task</button>
        </div>
      </form>
    </>
  );
}
