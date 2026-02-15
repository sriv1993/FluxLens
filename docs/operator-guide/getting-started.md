# Operator Guide ΓÇö Getting Started with FluxLens

This guide is for operators evaluating or deploying FluxLens to a real
operational environment.

## Decide your deployment posture

| Posture | When to choose | Notes |
|---|---|---|
| Local dev (docker-compose) | Evaluating FluxLens, building a domain pack | Not for production |
| Single-region Kubernetes | Single-site operator, low-medium HA needs | Default Phase 1 production posture |
| Multi-AZ Kubernetes | Production at scale | See `docs/deployment/kubernetes.md` |
| Multi-region federated | Multi-site / multi-region operators | Phase 3 |
| Air-gapped | Federally sensitive / regulated environments | Local LLM provider required; see `docs/deployment/air-gapped.md` |

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
| 1 ΓÇö Latest | Pure freshness; small focused operator views | Safety officer (filtered to severity ΓëÑ error) |
| 2 ΓÇö Latest per source | Pure diversity; per-source health views | Multi-line manager |
| 3 ΓÇö Hybrid | Mixed coverage of all sources + global freshness | Default for operator dashboards |
| 4 ΓÇö Guaranteed min diversity | Default for analyst views | Quality engineer |
| 5 ΓÇö Min diversity random eviction | Comparison baseline; analytical workloads | Algorithm A/B testing |
| 6 ΓÇö Preferred sources | When specific sources must always appear | Supply-chain analyst (critical suppliers) |

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

1. **MySQL.** Use the binlog CDC connector; ensure the source DB has
   `binlog_format=ROW` and the FluxLens user has `REPLICATION CLIENT`
   and `REPLICATION SLAVE` privileges.
2. **Postgres.** Use the logical-replication CDC connector; ensure
   `wal_level=logical` and create a replication slot.
3. **Kafka.** Use the Kafka bridge to re-curate events already on a
   Kafka topic.
4. **Webhook.** Use the webhook gateway for HTTP-pushable sources.

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
- Never set the LLM provider to "autonomous" ΓÇö there is no such mode.
  Operator override is enforced in code (ADR 0006).
- Always run shadow-mode validation when switching LLM providers or
  models.

## Get help

- Issues: https://github.com/sriharshav1/fluxlens/issues
- Security: see `SECURITY.md`
- Contributing: see `CONTRIBUTING.md`
