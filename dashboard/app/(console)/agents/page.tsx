import { createAgent, deleteAgent } from "@/lib/actions";
import { kw, loadMeta, when, type Agent } from "@/lib/api";

export default async function AgentsPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const [meta, body] = await Promise.all([loadMeta(), kw<{ agents: Agent[] }>("/v1/agents")]);
  return (
    <>
      <h1>Agents</h1>
      <p className="lede">An agent is the identity a task token is issued to. Policy matches the id.</p>
      {error ? <p className="error">{error}</p> : null}
      {body.agents.map((agent) => (
        <article className="card" key={agent.id}>
          <h2>{agent.name}</h2>
          <div className="meta">
            <span>{agent.id}</span>
            <span>{when(agent.created_at)}</span>
          </div>
          {meta.demo_lock ? null : (
            <form action={deleteAgent}>
              <input type="hidden" name="id" value={agent.id} />
              <div className="row">
                <button className="ghost" type="submit">
                  Delete
                </button>
              </div>
            </form>
          )}
        </article>
      ))}
      {body.agents.length === 0 ? <p className="empty">No agents yet.</p> : null}
      {meta.demo_lock ? null : (
        <form className="card" action={createAgent}>
          <h2>Register an agent</h2>
          <div className="grid">
            <div>
              <label htmlFor="agent-id">Id</label>
              <input id="agent-id" name="id" className="mono" required placeholder="support" />
            </div>
            <div>
              <label htmlFor="agent-name">Name</label>
              <input id="agent-name" name="name" required placeholder="Support" />
            </div>
          </div>
          <div className="row">
            <button type="submit">Save agent</button>
          </div>
        </form>
      )}
    </>
  );
}
