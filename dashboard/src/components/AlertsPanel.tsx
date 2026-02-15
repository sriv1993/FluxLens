import { AlertRecord } from "../api";

interface Props {
  alerts: AlertRecord[];
  onClear: () => void;
}

export default function AlertsPanel({ alerts, onClear }: Props) {
  return (
    <aside className="alerts">
      <header className="alerts-head">
        <h2>Alerts</h2>
        <button type="button" className="alerts-clear" onClick={() => void onClear()}>
          Clear
        </button>
      </header>
      {alerts.length === 0 ? (
        <p className="empty">No operator alerts.</p>
      ) : (
        <ul className="alert-list">
          {alerts.map((a) => (
            <li key={a.id} className={`alert-card sev-${a.severity}`}>
              <div className="alert-meta">
                <span className="alert-rule">{a.rule_id}</span>
                <time dateTime={a.created_at}>{new Date(a.created_at).toLocaleString()}</time>
              </div>
              <h3>{a.title}</h3>
              <p className="alert-body">{a.body}</p>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
