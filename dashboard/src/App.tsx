import { useCallback, useEffect, useState } from "react";
import { api, AlertsResponse, AuditResponse, DigestResult, HealthResponse } from "./api";
import Header from "./components/Header";
import EventFeed from "./components/EventFeed";
import AuditPanel from "./components/AuditPanel";
import AlertsPanel from "./components/AlertsPanel";
import OperatorWedge from "./components/OperatorWedge";

const STRATEGY_LABELS: Record<number, string> = {
  1: "Latest",
  2: "Latest per source",
  3: "Hybrid latest + per source",
  4: "Guaranteed min diversity",
  5: "Guaranteed min diversity (random eviction)",
  6: "Preferred sources",
};

export default function App() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [digest, setDigest] = useState<DigestResult | null>(null);
  const [audit, setAudit] = useState<AuditResponse | null>(null);
  const [alertPack, setAlertPack] = useState<AlertsResponse | null>(null);
  const [strategy, setStrategy] = useState(4);
  const [diversity, setDiversity] = useState(80);
  const [k, setK] = useState(20);
  const [err, setErr] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const [h, d, a, al] = await Promise.all([
        api.health(),
        api.digest(strategy, diversity, k),
        api.audit(),
        api.alerts(),
      ]);
      setHealth(h);
      setDigest(d);
      setAudit(a);
      setAlertPack(al);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }, [strategy, diversity, k]);

  useEffect(() => {
    void refresh();
    const id = setInterval(refresh, 5000);
    return () => clearInterval(id);
  }, [refresh]);

  return (
    <div className="app">
      <Header health={health} />
      {err && <div className="error">FluxLens API: {err}</div>}

      <section className="controls">
        <label>
          Strategy
          <select value={strategy} onChange={(e) => setStrategy(parseInt(e.target.value, 10))}>
            {Object.entries(STRATEGY_LABELS).map(([v, label]) => (
              <option key={v} value={v}>
                {v} — {label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Diversity %
          <input
            type="number"
            min={0}
            max={100}
            value={diversity}
            onChange={(e) => setDiversity(parseInt(e.target.value, 10))}
          />
        </label>
        <label>
          Digest size (k)
          <input type="number" min={1} max={100} value={k} onChange={(e) => setK(parseInt(e.target.value, 10))} />
        </label>
        <button onClick={() => void refresh()}>Refresh now</button>
      </section>

      {digest && (
        <section className="scores">
          <div>
            <span className="label">Freshness</span>
            <span className="value">{digest.FreshnessScore.toFixed(3)}</span>
          </div>
          <div>
            <span className="label">Diversity</span>
            <span className="value">{digest.DiversityScore.toFixed(3)}</span>
          </div>
          <div>
            <span className="label">Redundancy</span>
            <span className="value">{digest.RedundancyScore.toFixed(3)}</span>
          </div>
        </section>
      )}

      <OperatorWedge events={digest?.Selected ?? []} onRefresh={refresh} />

      <main className="main">
        <EventFeed events={digest?.Selected ?? []} onRefresh={refresh} />
        <AuditPanel audit={audit} />
        <AlertsPanel
          alerts={alertPack?.alerts ?? []}
          onClear={async () => {
            await api.clearAlerts();
            await refresh();
          }}
        />
      </main>
    </div>
  );
}
