import { ProxyDown } from "@/components/proxy-down";
import { loadList, pretty, when, type AuditEvent, type Verify } from "@/lib/api";

export default async function AuditPage() {
  const [verify, body] = await Promise.all([
    loadList<Verify>("/v1/audit/verify"),
    loadList<{ events: AuditEvent[] }>("/v1/audit?limit=100"),
  ]);
  if (verify.error || body.error || !verify.data || !body.data) {
    return <ProxyDown message={verify.error || body.error || "request failed"} />;
  }
  return (
    <>
      <h1>Audit</h1>
      <p className="lede">Every decision is hash-chained. A broken row means the record was changed after it was written.</p>
      <p className={verify.data.ok ? "banner" : "error"}>
        {verify.data.ok
          ? `Chain intact · ${verify.data.events} events`
          : `Chain broken at ${verify.data.broken_seq ?? "unknown"}: ${verify.data.reason ?? "mismatch"}`}
      </p>
      <table>
        <thead>
          <tr>
            <th>Seq</th>
            <th>When</th>
            <th>Event</th>
            <th>Record</th>
          </tr>
        </thead>
        <tbody>
          {body.data.events.length === 0 ? (
            <tr>
              <td colSpan={4}>No events yet.</td>
            </tr>
          ) : null}
          {body.data.events.map((event) => (
            <tr key={event.seq}>
              <td>{event.seq}</td>
              <td>{when(event.created_at)}</td>
              <td>{event.event_type}</td>
              <td>
                <pre>{pretty(event.payload)}</pre>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
