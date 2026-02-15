import { useEffect, useState } from "react";
import { AIDecision, api, CanonicalEvent, PrecedentStep, PrecedentSuggestResponse } from "../api";

interface Props {
  event: CanonicalEvent;
  onClose: () => void;
  onRefresh: () => Promise<void>;
}

export default function PrecedentResolvePanel({ event, onClose, onRefresh }: Props) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<PrecedentSuggestResponse | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [operatorId, setOperatorId] = useState("demo-operator");
  const [action, setAction] = useState<"accept" | "override" | "annotate">("accept");
  const [annotation, setAnnotation] = useState("");
  const [busy, setBusy] = useState(false);

  const loadSuggestions = async () => {
    setLoading(true);
    setErr(null);
    try {
      const res = await api.operatorSuggestPrecedents(event.event_id);
      setResult(res);
      await onRefresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadSuggestions();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load once per opened event
  }, [event.event_id]);

  const submitResolve = async (decision: AIDecision) => {
    setBusy(true);
    setErr(null);
    try {
      await api.operatorResolve({
        event_id: decision.event_id,
        decision_audit_hash: decision.audit_chain_hash,
        operator_id: operatorId.trim() || "anonymous",
        action,
        annotation: annotation.trim(),
      });
      setResult(null);
      await onRefresh();
      onClose();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="precedent-panel" role="dialog" aria-labelledby={`resolve-${event.event_id}`}>
      <header className="precedent-panel-head">
        <h3 id={`resolve-${event.event_id}`}>Resolve with precedents</h3>
        <button type="button" className="precedent-close" onClick={onClose} aria-label="Close">
          ×
        </button>
      </header>
      <p className="precedent-meta">
        {event.severity} · {event.source_id} · {event.event_type}
      </p>
      {err && <div className="wedge-err">{err}</div>}
      {loading && !result && <p className="precedent-meta">Loading suggested actions…</p>}
      {!loading && !result && !err && (
        <button type="button" onClick={() => void loadSuggestions()}>
          Retry
        </button>
      )}
      {result && (
        <>
          <ol className="precedent-steps">
            {result.steps.map((step: PrecedentStep, i: number) => (
              <li key={`${step.cited_precedent_hash ?? "primary"}-${i}`}>
                {step.text}
                {step.cited_precedent_hash && (
                  <span className="precedent-cite" title="Cited precedent decision hash">
                    precedent {step.cited_precedent_hash.slice(0, 12)}…
                  </span>
                )}
              </li>
            ))}
          </ol>
          {result.precedents_used.length > 0 && (
            <p className="precedent-count">{result.precedents_used.length} matching past resolution(s) in audit chain.</p>
          )}
          <div className="wedge-resolve precedent-resolve-form">
            <p className="dec-suggestion">{result.decision.response.suggestion}</p>
            <label className="wedge-field">
              Operator ID
              <input value={operatorId} onChange={(e) => setOperatorId(e.target.value)} />
            </label>
            <fieldset className="wedge-radio">
              <legend>Recorded action</legend>
              <label>
                <input type="radio" name={`opact-${event.event_id}`} checked={action === "accept"} onChange={() => setAction("accept")} /> accept
              </label>
              <label>
                <input type="radio" name={`opact-${event.event_id}`} checked={action === "override"} onChange={() => setAction("override")} /> override
              </label>
              <label>
                <input type="radio" name={`opact-${event.event_id}`} checked={action === "annotate"} onChange={() => setAction("annotate")} /> annotate
              </label>
            </fieldset>
            <label className="wedge-field">
              Notes / rationale
              <textarea rows={2} value={annotation} onChange={(e) => setAnnotation(e.target.value)} />
            </label>
            <button type="button" disabled={busy} onClick={() => void submitResolve(result.decision)}>
              Submit operator decision (audited)
            </button>
          </div>
        </>
      )}
    </div>
  );
}
