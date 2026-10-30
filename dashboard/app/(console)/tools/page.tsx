import { createTool, deleteTool } from "@/lib/actions";
import { kw, loadMeta, pretty, when, type Tool } from "@/lib/api";

export default async function ToolsPage({
  searchParams,
}: {
  searchParams: Promise<{ error?: string }>;
}) {
  const { error } = await searchParams;
  const [meta, body] = await Promise.all([loadMeta(), kw<{ tools: Tool[] }>("/v1/tools")]);
  return (
    <>
      <h1>Tools</h1>
      <p className="lede">Real credentials are sealed here and never shown again. Agents receive a task token, not the secret.</p>
      {error ? <p className="error">{error}</p> : null}
      {body.tools.map((tool) => (
        <article className="card" key={tool.id}>
          <h2>{tool.name}</h2>
          <div className="meta">
            <span>{tool.id}</span>
            <span>{tool.kind}</span>
            {tool.seeded ? <span className="pill ok">seeded</span> : null}
            <span>{when(tool.created_at)}</span>
          </div>
          <pre>{pretty(tool.config)}</pre>
          {meta.demo_lock || tool.seeded ? null : (
            <form action={deleteTool}>
              <input type="hidden" name="id" value={tool.id} />
              <button className="ghost" type="submit">
                Delete
              </button>
            </form>
          )}
        </article>
      ))}
      {meta.demo_lock ? (
        <p className="empty">This demo does not accept new tools.</p>
      ) : (
        <form className="card" action={createTool}>
          <h2>Register a tool</h2>
          <div className="grid">
            <div>
              <label htmlFor="tool-id">Id</label>
              <input id="tool-id" name="id" className="mono" required placeholder="orders" />
            </div>
            <div>
              <label htmlFor="tool-name">Name</label>
              <input id="tool-name" name="name" required placeholder="Orders database" />
            </div>
          </div>
          <label htmlFor="kind">Kind</label>
          <select id="kind" name="kind" defaultValue="mcp_http">
            <option value="mcp_http">MCP over HTTP</option>
            <option value="postgres">Postgres query</option>
            <option value="mcp_stdio">MCP over stdio</option>
            <option value="http">HTTP API</option>
            <option value="aws_http">AWS SigV4 HTTP</option>
          </select>
          <label htmlFor="config">Config JSON</label>
          <textarea id="config" name="config" className="mono" defaultValue='{"url":"http://127.0.0.1:8090/mcp"}' />
          <label htmlFor="secret">Secret JSON</label>
          <textarea id="secret" name="secret" className="mono" required placeholder='{"headers":{"Authorization":"Bearer ..."}}' />
          <div className="row">
            <button type="submit">Save tool</button>
          </div>
        </form>
      )}
    </>
  );
}
