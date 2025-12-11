# FluxLens — NIST AI Risk Management Framework Mapping

> Mapping of FluxLens platform capabilities to the NIST AI Risk Management
> Framework (NIST AI 100-1, January 2023). This document is informational;
> it does not constitute a compliance attestation. Operators remain
> responsible for end-to-end compliance of their specific deployment.

## NIST AI RMF overview

The NIST AI RMF identifies seven characteristics of trustworthy AI:

1. Valid and reliable
2. Safe
3. Secure and resilient
4. Accountable and transparent
5. Explainable and interpretable
6. Privacy-enhanced
7. Fair, with harmful bias managed

It also defines four core functions: **GOVERN**, **MAP**, **MEASURE**,
**MANAGE**.

This document maps FluxLens platform features to each characteristic and
function.

## Trustworthiness characteristics

### Valid and reliable

| FluxLens capability | Where in code |
|---|---|
| Schema-validated LLM output | `internal/orchestrator/guardrails.go` (`ValidateOutput`) |
| Refusal-on-uncertainty (confidence floor with mandatory review flag) | `internal/orchestrator/guardrails.go` (`minConfidence`) |
| Shadow-mode evaluator for new models | `internal/orchestrator` (Phase 1 stub; full implementation milestone M1.8) |
| Reproducible deterministic curation (seeded RNG support) | `internal/curation/strategies.go` (`Request.Rand`) |
| Independent chain-verifier process | `cmd/audit-writer` (`--verify-every`) |

### Safe

| FluxLens capability | Where in code |
|---|---|
| Human-override enforced in code on every AI decision | `internal/orchestrator` orchestration path |
| Circuit-breaker degradation on LLM-provider failure | Phase 1: orchestrator falls back to no-AI mode |
| No autonomous action on workers | Architectural principle (see ADR 0006) |

### Secure and resilient

| FluxLens capability | Where in code |
|---|---|
| TLS in transit (TLS 1.3 minimum) | Deployment configuration |
| Encryption at rest for all stores | Deployment configuration |
| Prompt-injection input validation | `internal/orchestrator/guardrails.go` |
| Service mesh mTLS for service-to-service traffic | `docs/deployment/kubernetes.md` |
| Sigstore image signing + admission policy | `.github/workflows/release.yml` (Phase 2) |

### Accountable and transparent

| FluxLens capability | Where in code |
|---|---|
| Hash-chained append-only audit log | `internal/auditlog/chain.go` |
| Full input/output recording per decision | `internal/orchestrator` audit-write path |
| Operator action recording (accept/override/annotate) | `cmd/api-gateway` decision endpoints |
| OpenAPI 3.1 specification for all API surfaces | `api/openapi.yaml` (Phase 1 M1.9) |

### Explainable and interpretable

| FluxLens capability | Where in code |
|---|---|
| Reasons array returned with each LLM response | `internal/orchestrator/guardrails.go` (`Response.Reasons`) |
| Curation strategy and score breakdown returned to operators | `internal/curation/strategies.go` (`Result`) |
| Per-decision trace ID propagated through OpenTelemetry | `internal/observability` (Phase 1 stub) |

### Privacy-enhanced

| FluxLens capability | Where in code |
|---|---|
| Configurable PII redaction in audit log | Phase 2 milestone M2.x |
| Per-tenant data isolation | Phase 2 multi-tenancy |
| Local LLM provider option (no required outbound calls) | `internal/orchestrator` provider interface |
| Air-gapped deployment mode | `docs/deployment/air-gapped.md` |

### Fair, with harmful bias managed

| FluxLens capability | Where in code |
|---|---|
| Source-diversity-aware curation (no single source monopolizes attention) | `internal/curation/strategies.go` (algorithms 2, 4, 5, 6) |
| Operator preference weights configurable per role/team | `internal/curation/strategies.go` (`PreferredSources`) |
| Per-source rate-limiting and fairness controls | Phase 2 |

## Core functions

### GOVERN

Governance of FluxLens is defined by:

- `LICENSE` (Apache 2.0)
- `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1)
- `CONTRIBUTING.md` (PR process, DCO sign-off)
- `SECURITY.md` (vulnerability disclosure)
- `docs/adr/` (Architectural Decision Records)
- `CHANGELOG.md` (auditable change history)

### MAP

Operators using FluxLens map their AI-deployment context through:

- Domain packs (`pkg/domainpack/`) that define event types and risk
  classifications for the operator's specific domain
- Curation strategy selection (operator decides which of the six
  algorithms matches their context)
- Operator preference weights (operator-specified weighting per
  source, severity, event class)

### MEASURE

FluxLens emits OpenTelemetry traces, Prometheus metrics, and
structured logs for every system component. Operators measure:

- AI orchestrator latency distribution (per provider, per
  guardrails-rejection class)
- Curation freshness/diversity/redundancy scores per digest
- Audit chain length and verification status
- Per-operator override rate (signal of AI-suggestion quality)

### MANAGE

Operators manage AI risk in FluxLens via:

- Adjusting guardrails confidence floor (`Guardrails.SetMinConfidence`)
- Adjusting curation strategy and diversity floor
- Adding custom input/output validators (`Guardrails.AddInputValidator`,
  `AddOutputValidator`)
- Rolling back to shadow-mode for any model showing degraded
  behavior
- Reviewing the audit log for after-action analysis

## What this mapping does not provide

- Compliance certification or attestation
- Per-deployment risk assessment
- Operator-specific guidance on which controls to apply
- Substitute for operator's own compliance review

It provides a structured map between the platform's capabilities and the
NIST AI RMF framework, suitable as a starting point for operator
compliance review.
