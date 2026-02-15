import type { AIDecision } from "../api";

type Props = {
  decisions: AIDecision[];
  connected: boolean;
};

export default function DecisionsPanel({ decisions, connected }: Props) {
  const recent = [...decisions].reverse().slice(0, 12);
  return (
    <section className="decisions-panel">
      <h2>
        Pipeline decisions{" "}
        <span className={`pill ${connected ? "pill-ok" : "pill-warn"}`}>
          {connected ? "live" : "polling"}
        </span>
      </h2>
      {recent.length === 0 ? (
        <p className="muted">
          No decisions from Kafka yet. Run orchestrator with gateway{" "}
          <code>-kafka</code> enabled.
        </p>
      ) : (
        <ul className="decision-list">
          {recent.map((d) => (
            <li key={d.audit_chain_hash + d.event_id}>
              <strong>{d.event_id}</strong>
              <span className="tag">{d.response.classification}</span>
              <p>{d.response.suggestion}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
