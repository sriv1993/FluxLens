import { useEffect, useState } from "react";
import { AIDecision, api, CanonicalEvent, downloadAuditExport } from "../api";

interface Props {
  events: CanonicalEvent[];
  onRefresh: () => Promise<void>;
}

export default function OperatorWedge({ events, onRefresh }: Props) {
  const [selectedId, setSelectedId] = useState<string>("");
  const [instruction, setInstruction] = useState("");
  const [operatorId, setOperatorId] = useState("demo-operator");
  const [decision, setDecision] = useState<AIDecision | null>(null);
  const [action, setAction] = useState<"accept" | "override" | "annotate">("accept");
  const [annotation, setAnnotation] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedId && events.length > 0) {
      setSelectedId(events[0].event_id);
    }
    if (selectedId && events.every((e) => e.event_id !== selectedId) && events.length > 0) {
      setSelectedId(events[0].event_id);
    }
  }, [events, selectedId]);

  const runSuggest = async () => {
    if (!selectedId) return;
    setBusy(true);
    setErr(null);
    try {
      const iso = instruction.trim()
        ? instruction.trim()
        : undefined;
      const res = await api.operatorSuggest(selectedId, iso);
      setDecision(res.decision);
      await onRefresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const runResolve = async () => {
    if (!decision) return;
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
      setDecision(null);
      setAnnotation("");
      await onRefresh();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const exportBundle = async () => {
    setErr(null);
    try {
      await downloadAuditExport();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };

  return (
    <section className="operator-wedge">
      <header className="wedge-head">
        <div>
          <h2>Operator wedge</h2>
          <p className="wedge-blurb">
            End-to-end slice: curated event → mock LLM suggestion (shared audit chain) → human accept / override /
            annotate → exportable bundle.
          </p>
        </div>
        <button type="button" className="wedge-export" onClick={() => void exportBundle()}>
          Download audit bundle
        </button>
      </header>

      {err && <div className="wedge-err">{err}</div>}

      {events.length === 0 ? (
        <p className="empty">Ingest events (e.g. <code>make synth</code> or POST /api/v1/events) so the digest has something to review.</p>
      ) : (
        <div className="wedge-grid">
          <label className="wedge-field">
            Event from digest
            <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)}>
              {events.map((e) => (
                <option key={e.event_id} value={e.event_id}>
                  {e.severity} · {e.source_id} · {e.event_type}
                </option>
              ))}
            </select>
          </label>

          <label className="wedge-field wedge-instr">
            Optional custom instruction (defaults to bundled manufacturing supervisor prompt)
            <textarea
              rows={3}
              value={instruction}
              onChange={(e) => setInstruction(e.target.value)}
              placeholder="Leave blank to use the default wedge prompt…"
            />
          </label>

          <div className="wedge-actions">
            <button type="button" disabled={busy || !selectedId} onClick={() => void runSuggest()}>
              {busy ? "Working…" : "Get AI suggestion"}
            </button>
          </div>

          {decision && (
            <div className="wedge-decision">
              <h3>AI suggestion (audit hash {decision.audit_chain_hash.slice(0, 14)}…)</h3>
              <p className="dec-line">
                <span className="label">Classification</span> {decision.response.classification || "—"}
              </p>
              <p className="dec-line">
                <span className="label">Guardrails</span> {decision.guardrails}
              </p>
              <p className="dec-line">
                <span className="label">Operator review flag</span> {decision.operator_review ? "yes" : "no"}
              </p>
              <p className="dec-suggestion">{decision.response.suggestion}</p>
              {decision.response.reasons && decision.response.reasons.length > 0 && (
                <ul className="dec-reasons">
                  {decision.response.reasons.map((r) => (
                    <li key={r}>{r}</li>
                  ))}
                </ul>
              )}

              <div className="wedge-resolve">
                <label className="wedge-field">
                  Operator ID
                  <input value={operatorId} onChange={(e) => setOperatorId(e.target.value)} />
                </label>
                <fieldset className="wedge-radio">
                  <legend>Recorded action</legend>
                  <label>
                    <input
                      type="radio"
                      name="opact"
                      checked={action === "accept"}
                      onChange={() => setAction("accept")}
                    />{" "}
                    accept suggestion
                  </label>
                  <label>
                    <input
                      type="radio"
                      name="opact"
                      checked={action === "override"}
                      onChange={() => setAction("override")}
                    />{" "}
                    override
                  </label>
                  <label>
                    <input
                      type="radio"
                      name="opact"
                      checked={action === "annotate"}
                      onChange={() => setAction("annotate")}
                    />{" "}
                    annotate only
                  </label>
                </fieldset>
                <label className="wedge-field">
                  Notes / rationale
                  <textarea rows={2} value={annotation} onChange={(e) => setAnnotation(e.target.value)} />
                </label>
                <button type="button" disabled={busy} onClick={() => void runResolve()}>
                  Submit operator decision (audited)
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  );
}
