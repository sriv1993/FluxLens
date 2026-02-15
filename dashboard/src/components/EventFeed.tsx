import { useState } from "react";
import { CanonicalEvent, Severity } from "../api";
import PrecedentResolvePanel from "./PrecedentResolvePanel";

interface Props {
  events: CanonicalEvent[];
  onRefresh: () => Promise<void>;
}

const SEVERITY_RANK: Record<Severity, number> = {
  info: 0,
  warn: 1,
  error: 2,
  critical: 3,
};

function needsResolveButton(sev: Severity): boolean {
  return sev === "critical" || sev === "error";
}

export default function EventFeed({ events, onRefresh }: Props) {
  const [resolveEventId, setResolveEventId] = useState<string | null>(null);

  const sorted = [...events].sort((a, b) => {
    const r = SEVERITY_RANK[b.severity] - SEVERITY_RANK[a.severity];
    if (r !== 0) return r;
    return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime();
  });

  const resolveEvent = resolveEventId ? sorted.find((e) => e.event_id === resolveEventId) : undefined;

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
              {needsResolveButton(e.severity) && (
                <button
                  type="button"
                  className="feed-resolve-btn"
                  onClick={() => setResolveEventId(e.event_id)}
                >
                  Suggested actions
                </button>
              )}
            </header>
            <pre className="payload">{JSON.stringify(e.payload, null, 2)}</pre>
            {resolveEventId === e.event_id && resolveEvent && (
              <PrecedentResolvePanel
                event={resolveEvent}
                onClose={() => setResolveEventId(null)}
                onRefresh={onRefresh}
              />
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
