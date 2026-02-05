# FluxLens — Roadmap

> **Status:** Draft v0.1
> **Companion to:** [PRD.md](./PRD.md), [ARCHITECTURE.md](./ARCHITECTURE.md)

## Phase 1 — Minimum Viable Platform

**Goal.** Deliver a working end-to-end platform supporting the core
flow (ingest → curate → AI decision-support → audit → serve) on a
single Kubernetes cluster with one CDC source and one LLM provider.

### Milestones

| Milestone | Target | Detail |
|---|---|---|
| M1.1 Project scaffolding | Week 1 | Repository structure, CI, license, code of conduct, contributor guidelines, ADR template |
| M1.2 Canonical event schema | Week 1 | JSON-schema for canonical event format; codegen for Go and Python |
| M1.3 MySQL CDC connector | Week 2 | Maxwell-style binlog reader, JSON serialization to Kafka |
| M1.4 Normalizer | Week 2 | Source-shape to canonical-event translation |
| M1.5 Curator (algorithms 1–3) | Week 3 | Latest, latest-per-source, hybrid strategies |
| M1.6 Curator (algorithms 4–6) | Week 4 | Guaranteed-min-diversity, randomized eviction, preferred sources |
| M1.7 Audit log (write path) | Week 5 | Append-only writer service with hash-chain |
| M1.8 AI orchestrator (stub) | Week 5 | OpenAI integration with guardrails, override hook, shadow-mode eval |
| M1.9 REST + WebSocket API | Week 6 | All endpoints in PRD §6.5 |
| M1.10 Reference dashboard | Week 7 | React UI with live feed, AI panel, override controls, audit viewer |
| M1.11 docker-compose local dev | Week 7 | Single-machine stack for contributors |
| M1.12 End-to-end demo | Week 8 | Documented happy-path demo with sample MySQL source |
| M1.13 Phase 1 release (v0.1.0) | End of July 2026 | Tagged release with documented APIs and demo |

### Phase 1 deliverable criteria

- ✅ MVP runs end-to-end on docker-compose
- ✅ One CDC source ingested at ≥1,000 events/sec sustained
- ✅ All six curation strategies pass unit and integration tests
- ✅ AI orchestrator integrates with OpenAI and demonstrates
  override + audit
- ✅ Audit log writes with hash-chain verified by chain-verifier
- ✅ REST + WebSocket API documented in OpenAPI 3.1
- ✅ Reference dashboard renders curated events live
- ✅ Documentation: README, PRD, ARCHITECTURE, ROADMAP, CONTRIBUTING,
  ADRs for all major decisions
- ✅ CI: lint, test, build, image-sign, vulnerability scan on every PR
- ✅ Test coverage ≥70% lines (target ≥80% by Phase 2)

## Phase 2 — Production Readiness

**Goal.** Harden FluxLens for production deployment in
operator-controlled environments. Add Postgres CDC, Kafka source,
webhook source, MyRocks archive tier, full observability, and the
reliability properties documented in ARCHITECTURE §5.

### Milestones

| Milestone | 
|---|
| M2.1 Postgres logical-replication CDC connector | 
| M2.2 Kafka source bridge |
| M2.3 Webhook gateway |
| M2.4 MyRocks archive tier and partition-drop purge |
| M2.5 OpenTelemetry full coverage + Grafana dashboards |
| M2.6 Multi-AZ deployment with failover testing | 
| M2.7 Local LLM provider (Ollama / vLLM) | 
| M2.8 OAuth2 / OIDC authentication and RBAC | 
| M2.9 Sigstore / cosign image signing + admission policy | 
| M2.10 Comprehensive load testing + capacity planning docs | 
| M2.11 v1.0.0 release (production-ready) | 

### Phase 2 deliverable criteria

- ✅ All sources documented in PRD §6.1 supported
- ✅ Multi-AZ deployment validated under chaos testing
- ✅ Audit log chain-verifier running continuously in CI
- ✅ Sigstore image signing enforced
- ✅ Test coverage ≥80% lines
- ✅ API availability target 99.9% verified under load
- ✅ Helm chart published

## Phase 3 — Ecosystem and Integrations

**Goal.** Build the ecosystem around FluxLens — integrations,
deployment automation, plugin marketplace, and operator-community
governance.

### Themes

- **Integrations.** Splunk, Elastic, PagerDuty, Opsgenie, ServiceNow,
  Slack, Microsoft Teams, custom webhooks.
- **Deployment automation.** Helm charts, Terraform modules, AWS
  Marketplace listing, Azure Marketplace listing.
- **Plugin model.** Curation-algorithm plugins, source-connector
  plugins, LLM-provider plugins.
- **Standards engagement.** Participation in OASIS / IEEE / CNCF
  working groups relevant to operational AI and event-driven
  architectures.
- **Community.** Public roadmap, monthly community calls, formal
  steering committee, contributor mentorship program.

### Indicative milestones

| Milestone | Target |
|---|---|
| M3.1 Helm chart and AWS Marketplace listing | Q1 2026 |
| M3.2 Splunk and Elastic integrations | Q1 2026 |
| M3.3 Plugin SDK and reference plugins | Q2 2026 |
| M3.4 First external production deployment (case study) | Q2 2026 |
| M3.5 v2.0.0 release (extended ecosystem) | Mid-2026 |
| M3.6 Steering committee formation | Mid-2026 |

## Out-of-scope (explicitly)

- Direct control of plant equipment (SCADA, OT protocols)
- Domain-specific event ontologies (operators define their own)
- Replacement of general-purpose paging platforms
- Autonomous high-stakes decision-making without human override

## How this roadmap is maintained

- Updated monthly by the project lead in `ROADMAP.md` on the `main`
  branch.
- GitHub Projects board reflects current sprint state.
- Phase boundaries reviewed at the end of each phase; subsequent
  phase reviewed in light of lessons learned.
