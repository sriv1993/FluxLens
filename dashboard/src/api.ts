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

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return (await res.json()) as T;
}

export const api = {
  health: () => getJSON<HealthResponse>("/api/v1/health"),
  digest: (strategy: number, diversity: number, k: number) =>
    getJSON<DigestResult>(
      `/api/v1/digest?strategy=${strategy}&diversity=${diversity}&k=${k}`,
    ),
  audit: () => getJSON<AuditResponse>("/api/v1/audit"),
};
