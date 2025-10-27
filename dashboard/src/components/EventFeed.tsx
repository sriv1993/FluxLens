import { CanonicalEvent, Severity } from "../api";

interface Props {
  events: CanonicalEvent[];
}

const SEVERITY_RANK: Record<Severity, number> = {
  info: 0,
  warn: 1,
  error: 2,
  critical: 3,
};

export default function EventFeed({ events }: Props) {
  const sorted = [...events].sort((a, b) => {
    const r = SEVERITY_RANK[b.severity] - SEVERITY_RANK[a.severity];
    if (r !== 0) return r;
    return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
  });

  return (
    <section className="feed">
      <h2>Curated event feed ({events.length})</h2>
      {sorted.length === 0 && (
        <div className="empty">No events yet. Run <code>make synth</code> to generate sample events, or POST to /api/v1/events.</div>
      )}
      <ul>
        {sorted.map((e) => (
          <li key={e.event_id} className={`event sev-${e.severity}`}>
            <header>
              <span className="sev">{e.severity}</span>
              <span className="src">{e.source_id}</span>
              <span className="type">{e.event_type}</span>
              <time>{new Date(e.timestamp).toLocaleTimeString()}</time>
            </header>
            <pre className="payload">{JSON.stringify(e.payload, null, 2)}</pre>
          </li>
        ))}
      </ul>
    </section>
  );
}
