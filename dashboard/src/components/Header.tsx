import { HealthResponse } from "../api";

interface Props {
  health: HealthResponse | null;
}

export default function Header({ health }: Props) {
  return (
    <header className="header">
      <h1>FluxLens</h1>
      <div className="header-meta">
        {health ? (
          <>
            <span className={`pill ${health.audit_chain_valid ? "ok" : "bad"}`}>
              audit chain: {health.audit_chain_valid ? "valid" : "TAMPERED"} ({health.audit_chain_length})
            </span>
            <span className="pill">events: {health.recent_events}</span>
            {typeof health.alerts_buffered === "number" && (
              <span className={`pill ${health.alerts_buffered > 0 ? "warn-pill" : ""}`}>
                alerts: {health.alerts_buffered}
              </span>
            )}
            <span className="pill">head: {health.audit_chain_head.slice(0, 12)}…</span>
          </>
        ) : (
          <span className="pill">connecting…</span>
        )}
      </div>
    </header>
  );
}
