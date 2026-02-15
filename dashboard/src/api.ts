export type Severity = "info" | "warn" | "error" | "critical";

export interface CanonicalEvent {
  event_id: string;
  source_id: string;
  source_type: string;
  event_type: string;
  severity: Severity;
  timestamp: string;
  ingested_at: string;
  payload: unknown;
  metadata: {
    schema_version: string;
    trace_id?: string;
    ingestion_pod?: string;
    tenant_id?: string;
  };
}

export interface DigestResult {
  Selected: CanonicalEvent[];
  Strategy: number;
  FreshnessScore: number;
  DiversityScore: number;
  RedundancyScore: number;
}

export interface HealthResponse {
  status: string;
  audit_chain_length: number;
  audit_chain_head: string;
  audit_chain_valid: boolean;
  audit_chain_error: string;
  recent_events: number;
  alerts_buffered?: number;
}

export interface AuditRecord {
  sequence: number;
  timestamp: string;
  kind: string;
  payload: unknown;
  prev_hash: string;
  hash: string;
}

export interface AuditResponse {
  verified: boolean;
  verify_err: string;
  head_hash: string;
  records: AuditRecord[];
}

export interface AlertRecord {
  id: string;
  title: string;
  body: string;
  severity: Severity;
  rule_id: string;
  created_at: string;
  ref?: Record<string, string>;
}

export interface AlertsResponse {
  alerts: AlertRecord[];
  count: number;
}

export interface AIDecision {
  event_id: string;
  provider: string;
  model_id: string;
  prompt_hash: string;
  response: {
    classification: string;
    suggestion: string;
    confidence: number;
    requires_review: boolean;
    reasons?: string[];
  };
  guardrails: string;
  operator_review: boolean;
  audit_chain_hash: string;
  audit_chain_prev: string;
}

export interface OperatorSuggestResponse {
  decision: AIDecision;
}

export interface PrecedentStep {
  text: string;
  cited_precedent_hash?: string;
}

export interface PrecedentUsed {
  decision_hash: string;
  event_type: string;
  source_id: string;
  severity: string;
  operator_action: string;
  annotation?: string;
}

export interface PrecedentSuggestResponse {
  steps: PrecedentStep[];
  precedents_used: PrecedentUsed[];
  decision: AIDecision;
}

export interface OperatorResolveResponse {
  operator_audit_hash: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

async function postJSON<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const raw = await res.text();
  if (!res.ok) throw new Error(raw || `${res.status} ${res.statusText}`);
  return JSON.parse(raw) as T;
}

async function deleteJSON(url: string): Promise<void> {
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
}

export async function downloadAuditExport(): Promise<void> {
  const res = await fetch("/api/v1/operator/export");
  const raw = await res.text();
  if (!res.ok) throw new Error(raw || `${res.status}`);
  const blob = new Blob([raw], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `fluxlens-audit-bundle-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
}

export const api = {
  health: () => getJSON<HealthResponse>("/api/v1/health"),
  digest: (strategy: number, diversity: number, k: number) =>
    getJSON<DigestResult>(
      `/api/v1/digest?strategy=${strategy}&diversity=${diversity}&k=${k}`,
    ),
  audit: () => getJSON<AuditResponse>("/api/v1/audit"),
  alerts: () => getJSON<AlertsResponse>("/api/v1/alerts"),
  clearAlerts: () => deleteJSON("/api/v1/alerts"),
  operatorSuggest: (event_id: string, instruction?: string) => {
    const body: Record<string, string> = { event_id };
    if (instruction != null && instruction.trim() !== "") {
      body.instruction = instruction.trim();
    }
    return postJSON<OperatorSuggestResponse>("/api/v1/operator/suggest", body);
  },
  operatorResolve: (body: {
    event_id: string;
    decision_audit_hash: string;
    operator_id: string;
    action: "accept" | "override" | "annotate";
    annotation: string;
  }) => postJSON<OperatorResolveResponse>("/api/v1/operator/resolve", body),
  operatorSuggestPrecedents: (
    event_id: string,
    opts?: { instruction?: string; max_precedents?: number },
  ) => {
    const body: Record<string, string | number> = { event_id };
    if (opts?.instruction?.trim()) body.instruction = opts.instruction.trim();
    if (opts?.max_precedents != null && opts.max_precedents > 0) {
      body.max_precedents = opts.max_precedents;
    }
    return postJSON<PrecedentSuggestResponse>("/api/v1/operator/suggest-precedents", body);
  },
};
