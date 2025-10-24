import { AuditResponse } from "../api";

interface Props {
  audit: AuditResponse | null;
}

export default function AuditPanel({ audit }: Props) {
  if (!audit) return <section className="audit"><h2>Audit log</h2><p>Loading…</p></section>;

  return (
    <section className="audit">
      <h2>Audit log ({audit.records.length})</h2>
      <div className={`audit-status ${audit.verified ? "ok" : "bad"}`}>
        chain {audit.verified ? "verified ✓" : "FAILED VERIFICATION ✗"}{" "}
        {audit.verify_err && <span className="err">{audit.verify_err}</span>}
      </div>
      <ol>
        {[...audit.records].reverse().slice(0, 50).map((r) => (
          <li key={r.sequence} className="audit-record">
            <header>
              <span className="seq">#{r.sequence}</span>
              <span className="kind">{r.kind}</span>
              <time>{new Date(r.timestamp).toLocaleTimeString()}</time>
            </header>
            <pre>{JSON.stringify(r.payload, null, 2)}</pre>
            <footer>
              <code>hash: {r.hash.slice(0, 16)}…</code>
              <code>prev: {r.prev_hash ? r.prev_hash.slice(0, 16) + "…" : "(genesis)"}</code>
            </footer>
          </li>
        ))}
      </ol>
    </section>
  );
}
