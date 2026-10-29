import { kw, pretty, when, type AuditEvent, type Verify } from "@/lib/api";

export default async function AuditPage() {
  const [verify, body] = await Promise.all([
    kw<Verify>("/v1/audit/verify"),
    kw<{ events: AuditEvent[] }>("/v1/audit?limit=100"),
  ]);
  return (
    <>
      <h1>Audit</h1>
      <p className="lede">Every decision is hash-chained. A broken row means the record was changed after it was written.</p>
      <p className={verify.ok ? "banner" : "error"}>
        {verify.ok
          ? `Chain intact · ${verify.events} events`
          : `Chain broken at ${verify.broken_seq ?? "unknown"}: ${verify.reason ?? "mismatch"}`}
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
          {body.events.length === 0 ? (
            <tr>
              <td colSpan={4}>No events yet.</td>
            </tr>
          ) : null}
          {body.events.map((event) => (
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
