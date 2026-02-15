# Operator Guide: Getting Started with FluxLens

This guide is for operators evaluating or deploying FluxLens to a real
operational environment.

## Decide your deployment posture

| Posture | When to choose | Notes |
|---|---|---|
| Local dev (docker-compose) | Evaluating FluxLens, building a domain pack | Not for production |
| Single-region Kubernetes | Single-site operator, low-medium HA needs | Default Phase 1 production posture |
| Multi-AZ Kubernetes | Production at scale | See [`ROADMAP.md`](../../ROADMAP.md) Phase 2 |
| Multi-region federated | Multi-site / multi-region operators | Phase 3 |
| Air-gapped | Federally sensitive / regulated environments | Local LLM provider (Ollama or vLLM) |

## Choose your LLM provider

| Provider | When to choose |
|---|---|
| OpenAI | General-purpose, network-egress permitted, latency-sensitive |
| Anthropic Claude (via OpenAI-compatible proxies) | Same as OpenAI |
| Ollama (local) | Air-gapped, cost-sensitive, latency-permissive |
| vLLM (local) | High-throughput local serving, large-model deployment |
| MockProvider | CI/CD, demo, smoke testing only |

The choice is configurable; all providers implement the same
`llm.Provider` interface.

## Choose your curation strategy

| Strategy | When to use | Role example |
|---|---|---|
| 1. Latest | Pure freshness; small focused operator views | Safety officer (severity >= error) |
| 2. Latest per source | Pure diversity; per-source health views | Multi-line manager |
| 3. Hybrid | Mixed coverage of all sources + global freshness | Default for operator dashboards |
| 4. Guaranteed min diversity | Default for analyst views | Quality engineer |
| 5. Min diversity random eviction | Comparison baseline; analytical workloads | Algorithm A/B testing |
| 6. Preferred sources | When specific sources must always appear | Supply-chain analyst (critical suppliers) |

## Author a domain pack

Domain packs encode operator domain knowledge into FluxLens:

1. Define event types and severity defaults
2. Define operator roles and recommended curation per role
3. Define retention and audit-trail requirements
4. (Optional) Define escalation rules

See `pkg/domainpack/examples/` for three reference packs:

- `clean-energy-battery.yaml`
- `retail-supply-chain.yaml`
- `federal-research.yaml`

## Configure ingestion sources

1. **MySQL.** Use `fluxlens-ingest-mysql`; ensure `binlog_format=ROW` and
   replication privileges on the source DB.
2. **Postgres.** Use `fluxlens-ingest-postgres`; ensure `wal_level=logical`
   and a replication slot/publication for your tables.
3. **Kafka.** Run `fluxlens-curator` and `fluxlens-orchestrator` on your
   topics; enable the gateway `-kafka` bridge for the dashboard.
4. **Webhook.** `POST /api/v1/webhook` on the gateway or `fluxlens-webhook-gateway`.

## Operate

- Monitor curator throughput and freshness/diversity scores; tune
  strategy and parameters per role until operators report the digest
  is useful.
- Monitor orchestrator latency p99; tune LLM provider, model, and
  guardrails confidence floor based on operator override rate.
- Monitor audit-chain verification; investigate any verification
  failure immediately.

## Operate safely

- Always run the audit chain verifier on a separate process/node
  from the writer.
- Always mirror the audit log to WORM storage for high-stakes
  deployments.
- There is no autonomous LLM mode. Operator override is enforced in code ([ADR 0006](../adr/0006-human-override-in-code.md)).
- Always run shadow-mode validation when switching LLM providers or
  models.

## Get help

- Issues: https://github.com/sriharshav1/fluxlens/issues
- Security: see `SECURITY.md`
- Contributing: see `CONTRIBUTING.md`
