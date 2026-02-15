# FluxLens — Architecture

> **Author:** Sri Harsha Vanga  
> **Companion to:** [PRD.md](./PRD.md)

This document records the technical architecture and key design
decisions for FluxLens. It is intended for engineers building or
contributing to the platform.

## 1. Architectural principles

FluxLens follows five architectural principles, in order of
precedence:

1. **Source-system non-impact.** Ingestion must not meaningfully
   affect the performance of source systems. CDC-style log-following
   is preferred over polling. Backpressure must be handled at the
   FluxLens layer, never propagated to source.
2. **Verifiable human override.** No AI-suggested action is taken
   without a human-override path. The path is enforced in code, not
   in policy.
3. **Append-only auditability.** Every decision and operator action
   is recorded in a tamper-evident, hash-chained, append-only log.
4. **Pluggable provider interfaces.** LLM providers, storage
   backends, source connectors, and authentication providers are
   pluggable behind stable interfaces.
5. **Operational legibility.** OpenTelemetry tracing, Prometheus
   metrics, and structured logging are first-class concerns. The
   platform must be debuggable from the outside.

## 2. Layered architecture

```mermaid
flowchart TB
    L1[Layer 1: Ingestion<br/>CDC + webhook + Kafka consumers]
    L2[Layer 2: Event bus<br/>Apache Kafka]
    L3[Layer 3: Curation<br/>Freshness/diversity/redundancy]
    L4[Layer 4: AI decision-support<br/>LLM orchestrator + guardrails]
    L5[Layer 5: Storage<br/>Hot + archive + audit]
    L6[Layer 6: Serving<br/>REST + WebSocket + dashboard]
    L7[Layer 7: Observability<br/>OTEL + Prom + Grafana]

    L1 --> L2
    L2 --> L3
    L3 --> L4
    L3 --> L5
    L4 --> L5
    L4 --> L6
    L7 -.-> L1
    L7 -.-> L2
    L7 -.-> L3
    L7 -.-> L4
    L7 -.-> L5
    L7 -.-> L6
```

## 3. Service decomposition

```mermaid
flowchart LR
    subgraph SVC_INGEST[fluxlens-ingest]
        IN1[mysql-cdc]
        IN2[postgres-cdc]
        IN3[kafka-bridge]
        IN4[webhook-gateway]
        IN5[normalizer]
    end

    subgraph SVC_CORE[fluxlens-core]
        CO1[curator]
        CO2[ai-orchestrator]
        CO3[audit-writer]
        CO4[archive-mover]
    end

    subgraph SVC_SERVE[fluxlens-serve]
        SR1[api-gateway]
        SR2[websocket-fanout]
        SR3[dashboard]
        SR4[admin-api]
    end

    subgraph SVC_DATA[fluxlens-data]
        DA1[(postgres-timescale)]
        DA2[(myrocks-archive)]
        DA3[(audit-log)]
        DA4[(kafka-cluster)]
    end

    IN1 & IN2 & IN3 & IN4 --> IN5
    IN5 --> DA4
    DA4 --> CO1
    CO1 --> CO2
    CO1 --> DA1
    CO2 --> CO3
    CO3 --> DA3
    DA1 --> CO4
    CO4 --> DA2
    CO2 --> SR1
    SR1 --> SR2
    SR1 --> SR3
    SR4 --> CO1
```

**Phase 1 note:** `SR2` (WebSocket fan-out) and the Kafka decisions bridge
are implemented **inside** `cmd/api-gateway` (`internal/stream`,
`internal/kafkabridge`). A separate `websocket-fanout` service remains the
target for high-scale Phase 2 deployments.

### 3.1 Phase 1 reference topology (`cmd/api-gateway`)

Production retains the decomposition above. For **depth-first demos**
(and CI), the `api-gateway` binary optionally **folds** orchestrator,
recent-events RAM buffer, digest scoring, alert buffering, and audit
verification into **one OS process** sharing a single `auditlog.Chain`.

```mermaid
flowchart LR
  subgraph GW[api-gateway single process]
    REST[HTTP mux]
    WS[WebSocket hub]
    KBR[Kafka bridge optional]
    BUF[recent events buffer]
    CHAIN[auditlog.Chain]
    CUR[curation.Select]
    ORCH[orchestrator]
    MOCK[Mock LLM provider]
    ALRT[alerts.Store]
  end
  UI[dashboard static UI]
  KAFKA[(Kafka topics)]

  KBR -.->|decisions curated raw| KAFKA
  KBR --> BUF
  KBR --> WS
  REST --> WS

  REST --> BUF
  REST --> CUR
  CUR --> BUF
  CUR -->|append digest_selection| CHAIN
  REST --> ORCH
  ORCH --> MOCK
  ORCH -->|append decision variants| CHAIN
  REST -->|operator resolve| ORCH
  ORCH -->|append operator_action| CHAIN
  REST --> ALRT
  CHAIN -->|Verify on reads| REST
  UI --> REST
```

This topology trades HA isolation for **legibility**: reviewers can
execute ingest → digest → AI suggestion → human resolution → JSON export
without Kafka or Postgres. With `-kafka`, the same process also consumes
orchestrator decisions and curated digests for the live dashboard. Horizontal
deployments MUST split writers per ADR backlog once throughput exceeds
single-node limits.

## 4. Data architecture

### 4.1 Storage tiers

```mermaid
flowchart LR
    SRC[Source events] -->|CDC| K[(Kafka)]
    K -->|consume| CUR[Curator]
    CUR -->|recent events| HOT[(Hot store<br/>Postgres + Timescale<br/>retention: 30 days)]
    HOT -->|partition rollover| ARCH[(Archive<br/>MyRocks<br/>retention: 5 years)]
    ARCH -->|TTL drop| TRASH[Purged]

    CUR -->|decisions| AUD[(Audit log<br/>append-only<br/>retention: 7 years)]
```

### 4.2 Hot store schema (Postgres + TimescaleDB)

- `events` — hypertable partitioned by `ingested_at`, 1-day chunks
- `digests` — recent curated digests per strategy
- `decisions` — recent AI decisions (last 30 days)
- `operator_actions` — recent operator overrides/accepts

### 4.3 Archive store schema (MyRocks)

- `events_archive` — partitioned by `ingested_at`, 30-day chunks;
  partition drops for purge
- `decisions_archive` — partitioned similarly
- Compression ratio target: 2:1 vs. equivalent InnoDB (typical for
  MyRocks on similar write-heavy archive workloads)

### 4.4 Audit log architecture

The audit log is the most security-sensitive component. Properties:

```mermaid
flowchart LR
    APP[Application writer] -->|append-only RPC| WRT[Audit writer service]
    WRT -->|hash-chain<br/>compute current hash<br/>= sha256(prev_hash + payload)| LOG[(Audit log<br/>append-only)]
    WRT -->|optional| WORM[(WORM storage<br/>S3 Object Lock)]
    AUD[Auditor reads] -->|read-only| LOG
    VER[Chain verifier] -->|periodic| LOG
```

- Append-only via dedicated writer service; no application has direct
  write access to the underlying storage.
- Hash chain: each record contains `prev_hash`; current record's hash
  is `sha256(prev_hash || canonical_serialization(payload))`.
- Optional WORM (Write-Once-Read-Many) mirroring via S3 Object Lock
  in compliance mode for operators that require it.
- Periodic chain verifier process recomputes hashes and alerts on
  divergence.

**Phase 1 gateway note.** `cmd/api-gateway` keeps `auditlog.Chain` in-process,
calls `Verify()` after ingest/digest/audit/operator/export mutations, and maps
verification regressions into buffered alerts (`internal/alerts`). This makes
tampering visible during demos but **does not** satisfy separated-duties or WORM
requirements until the standalone audit-writer path is exercised.

## 5. Reliability architecture

### 5.1 Failure-domain isolation

```mermaid
flowchart TB
    subgraph AZ1[Availability Zone 1]
        K11[Kafka broker 1]
        ING1[Ingest pods]
        CORE1[Core pods]
        PG1[(Postgres replica 1<br/>primary)]
    end
    subgraph AZ2[Availability Zone 2]
        K12[Kafka broker 2]
        ING2[Ingest pods]
        CORE2[Core pods]
        PG2[(Postgres replica 2)]
    end
    subgraph AZ3[Availability Zone 3]
        K13[Kafka broker 3]
        ING3[Ingest pods]
        CORE3[Core pods]
        PG3[(Postgres replica 3)]
    end

    K11 <-->|replication| K12
    K12 <-->|replication| K13
    PG1 <-->|sync streaming<br/>replication| PG2
    PG2 <-->|sync streaming<br/>replication| PG3
```

### 5.2 Failure handling matrix

| Failure mode | Detection | Response | RPO | RTO |
|---|---|---|---|---|
| Ingest pod crash | Liveness probe | Kubernetes restart | 0 | <60s |
| Kafka broker loss | ISR shrink alert | Auto-leadership transfer | 0 | <30s |
| Postgres primary loss | Replication lag alert | Standby promotion | 0 | <120s |
| LLM provider outage | Timeout + circuit breaker | Failover to local model or degraded mode | 0 (events buffered) | <10s circuit-open |
| Network partition | OTEL trace gap | Per-AZ continued operation | 0 | depends on partition |
| Audit log write failure | Synchronous write check | Block decision pathway; alert | 0 | <5s |

## 6. Security architecture

### 6.0 Phase 1 reference auth (shipped)

The reference gateway enforces **API keys** (`Authorization: Bearer` or
`X-API-Key`) and optional **role bindings** (`FLUXLENS_API_KEY_ROLES`:
`secret:operator+admin`). Roles: `operator`, `reviewer`, `admin`, `auditor`.
WebSocket `/api/v1/stream` accepts the same keys when configured. When no
keys are set, middleware runs in local-dev mode (all roles). This is not a
substitute for enterprise OIDC; see target model below.

### 6.1 Target production auth (Phase 2+)

```mermaid
flowchart TB
    USER[Browser / service principal] -->|TLS + OAuth2| GW[API Gateway]
    GW -->|JWT validation| AUTHZ[Authz middleware]
    AUTHZ -->|RBAC check<br/>operator / reviewer / admin / auditor| ROUTE[Route handler]
    ROUTE -->|service mesh mTLS| INT[Internal services]
    INT -->|encrypted-at-rest| DATA[(Data stores)]

    ADM[Cluster admin] -->|short-lived OIDC tokens| K8S[K8s API]
    SECRET[Secrets backend<br/>Vault / SOPS] -->|mounted as files| INT
```

- **Authentication:** OAuth 2.0 / OIDC for human users; service-
  account JWTs for service-to-service.
- **Authorization:** Role-based (operator / reviewer / admin /
  auditor) with per-namespace scoping.
- **Encryption:** TLS in transit (1.3 minimum); encryption at rest
  for all data stores.
- **Secrets:** External secret backend (Vault or cloud-native KMS);
  never committed to repo.
- **Image provenance:** Sigstore / cosign signed container images;
  admission controller enforces signed-image policy.

## 7. Architecture Decision Records (ADRs)

Maintained under `/docs/adr/`. Index:

- **ADR-001:** Use CDC over polling for source-system ingestion.
- **ADR-002:** Apache Kafka as the event bus (vs. NATS, Pulsar).
- **ADR-003:** Postgres + TimescaleDB for hot store (vs. ClickHouse,
  InfluxDB).
- **ADR-004:** MyRocks for archive tier (vs. plain InnoDB, S3 Glacier).
- **ADR-005:** Append-only hash-chained audit log (vs. blockchain-
  based or signed batches).
- **ADR-006:** Human-override enforced in code, not policy.
- **ADR-007:** Pluggable LLM provider interface (vs. OpenAI-only).
- **ADR-008:** Kubernetes-native deployment; no support for non-
  containerized deployment.
- **ADR-009:** OpenTelemetry as the sole observability standard.
- **ADR-010:** Apache 2.0 license (vs. AGPL, BSL).

Each ADR follows the [Michael Nygard ADR
template](https://github.com/joelparkerhenderson/architecture-decision-record)
and is added before any non-trivial architectural change.

## 8. Technology stack rationale

| Component | Choice | Rationale |
|---|---|---|
| Ingestion services | Go | High-throughput, low-memory, mature CDC ecosystem |
| ML services | Python | Standard ML/LLM ecosystem |
| Event bus | Kafka | Production-proven at very high throughput in large operators |
| Hot store | Postgres + TimescaleDB | ACID guarantees + time-series partitioning |
| Archive store | MyRocks | LSM-tree compression for cold data |
| Audit log | Append-only on Postgres with WORM mirror | Tamper-evidence via hash chain |
| Orchestration | Kubernetes | Standard horizontal scale + failover |
| API gateway | Custom Go service behind nginx-ingress | Lightweight, OpenTelemetry-native |
| Dashboard | TypeScript + React | Modern, accessible UI |
| Observability | OpenTelemetry + Prometheus + Grafana | Vendor-neutral |
| Service mesh (optional) | Linkerd | Lighter than Istio; mTLS by default |
| Secrets | HashiCorp Vault or cloud-native KMS | Pluggable |
| CI | GitHub Actions | Standard, free for OSS |

## 9. Local development architecture

```mermaid
flowchart LR
    DEV[Developer laptop] --> DC[docker-compose stack]
    DC --> K[Kafka single-broker]
    DC --> PG[Postgres + Timescale]
    DC --> RD[Redis]
    DC --> MOCK[Mock LLM endpoint]
    DC --> FX[FluxLens services<br/>built from source]
```

The single-machine docker-compose stack starts Kafka, Postgres, Redis,
a mock LLM endpoint, and all FluxLens services from source for
developer iteration. `make dev` brings the stack up.

### 9.1 Operator wedge sequence (reference UI path)

When hitting only `api-gateway` + static dashboard assets, the dominant
sequence reduces to synchronous REST calls—still respecting guardrails +
explicit operator acknowledgement semantics:

```mermaid
sequenceDiagram
    autonumber
    participant UI as Dashboard
    participant GW as api-gateway
    participant BUF as Recent buffer
    participant CUR as Curation pkg
    participant OR as Orchestrator
    participant LG as Mock LLM
    participant CH as auditlog.Chain

    UI->>GW: POST /api/v1/events
    GW->>BUF: append canonical event
    GW->>CH: Append ingest
    GW->>CH: Verify()
    UI->>GW: GET /api/v1/digest
    GW->>CUR: Select(...)
    CUR->>BUF: snapshot inputs
    GW->>CH: Append digest_selection
    GW->>CH: Verify()
    UI->>GW: POST /operator/suggest
    GW->>OR: Decide(...)
    OR->>LG: Decide prompt
    LG-->>OR: structured response
    OR->>CH: Append decision / rejected_* variants
    GW->>CH: Verify()
    GW-->>UI: JSON decision + audit_chain_hash
    UI->>GW: POST /operator/resolve
    OR->>CH: Append operator_action
    GW->>CH: Verify()
    GW-->>UI: operator_audit_hash
    UI->>GW: GET /operator/export
    GW-->>UI: tamper-evident bundle JSON
```

Buffered alerts populate via the same handlers whenever ingest severity,
digest heuristics, chain verification, AI review flags, or resolutions fire—see
PRD §9.1.

### 9.2 Precedent retrieval and operator UX wedge extension

The operator wedge (§9.1) is extended by **`POST /api/v1/operator/suggest-precedents`**
and the dashboard **Suggested actions** control on critical/error feed rows.
`internal/precedents` scans the same `auditlog.Store` the gateway and orchestrator
share (in-memory `Chain` or Postgres when `FLUXLENS_POSTGRES_DSN` is set). Matching
requires a completed past resolution: a guardrails-passing decision record plus a
linked `operator_action`. The orchestrator appends **`decision_with_precedents`** on
success; human resolution still flows through **`operator_action`** only. See
[PRD — Precedent-informed resolution](PRD.md#precedent-informed-resolution).

```mermaid
flowchart TB
    subgraph UI[dashboard]
        FE[EventFeed]
        PP[PrecedentResolvePanel]
        OW[OperatorWedge]
    end
    subgraph GW[api-gateway]
        H[suggest-precedents handler]
    end
    subgraph Core[internal]
        PR[precedents.FindMatches]
        OR[orchestrator.SuggestWithPrecedents]
        GR[guardrails]
        LM[llm.Provider]
    end
    subgraph Store[auditlog.Store]
        CH[(Hash chain<br/>decision*, operator_action)]
    end

    FE -->|critical/error| PP
    PP -->|POST suggest-precedents| H
    OW -->|POST suggest| H
    H --> PR
    PR -->|Snapshot| CH
    H --> OR
    OR --> GR
    OR --> LM
    OR -->|Append decision_with_precedents| CH
    PP -->|POST resolve| H
    H -->|RecordOperatorAction| OR
    OR -->|Append operator_action| CH
```

### 9.3 Kafka-connected dashboard path

When `api-gateway` runs with `-kafka` alongside `curator`, `orchestrator`,
and a synthetic or CDC ingest path:

```mermaid
sequenceDiagram
    autonumber
    participant ING as ingest / synth
    participant K as Kafka
    participant CUR as curator
    participant OR as orchestrator
    participant GW as api-gateway
    participant WS as WebSocket hub
    participant UI as Dashboard

    ING->>K: fluxlens.events.raw
    K->>CUR: consume raw
    CUR->>K: fluxlens.events.curated
    K->>OR: consume curated
    OR->>K: fluxlens.decisions
    K->>GW: bridge consumers
    GW->>WS: digest / decision messages
    WS-->>UI: live feed
    UI->>GW: REST health / audit / operator resolve
```

The UI still uses REST for operator accept/override/annotate and audit
export; WebSocket carries incremental digest and decision visibility.

## 10. Observability architecture

Every service emits OpenTelemetry traces (W3C trace context), metrics
(Prometheus exposition format), and structured logs (JSON, line-
delimited). The OpenTelemetry Collector sidecar aggregates and
exports to operator-configured backends.

Standard dashboards (Grafana JSON committed to `/dashboards/`):

- Ingestion throughput and lag per source
- Curation engine throughput and selection-strategy mix
- AI orchestrator latency distribution and guardrails-rejection rate
- Audit log write rate and chain-verifier status
- API latency p50/p95/p99 per endpoint
- Pod and node resource utilization
