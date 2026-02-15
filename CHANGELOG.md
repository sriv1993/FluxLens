# Changelog

All notable changes to FluxLens will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- `docs/archive/national-interest/` and its Markdown drafts (IRA/CHIPS,
  DOE, CISA, workforce, EO 14110 narratives).
- `docs/national-interest/` stub README (national-interest framing is only
  in this repository README).

### Added

- Precedent-informed resolution: `POST /api/v1/operator/suggest-precedents`, `internal/precedents`, orchestrator `SuggestWithPrecedents` (`decision_with_precedents` audit kind), dashboard **Suggested actions** on critical/error feed rows
- Real MySQL binlog CDC (`internal/cdc/mysql` with go-mysql replication syncer)
- `ingest-mysql` flags: `-tables`, `-binlog-file`, `-binlog-pos`
- Postgres-backed audit log (`internal/auditlog/postgres`, `-postgres-dsn` / `FLUXLENS_POSTGRES_DSN`)
- `cmd/chain-verifier` for scheduled audit-chain validation
- Prometheus metrics on api-gateway (`/metrics`)
- OpenAPI 3.1 spec (`api/openapi.yaml`, `GET /api/openapi.yaml`)
- API-key auth middleware (`-api-keys`, `FLUXLENS_API_KEYS`)
- Domain pack loader (`pkg/domainpack`, `-domain-pack` on gateway)
- Production tracking doc: `docs/PRODUCTION_CHECKLIST.md`

### Changed

- `PRD.md`: removed enumerated Phase 2–3 backlog (see `ROADMAP.md`);
  References renumbered accordingly.
- `auditlog.Store` interface; orchestrator uses interface not concrete chain

### Fixed

- CI `gofmt` check failing on several command entrypoints and `internal/orchestrator`.

### Added

- Initial project scaffolding (README, PRD, ARCHITECTURE, ROADMAP, LICENSE)
- Apache 2.0 license
- Contributor Covenant 2.1 code of conduct
- DCO sign-off requirement for contributions
- GitHub Actions CI (lint, test, build)
- Makefile with standard developer targets
- docker-compose development stack (Kafka, Postgres+Timescale, Redis, mock LLM, Prometheus, Grafana)
- Canonical event schema (Go package + JSON schema)
- Six curation algorithms balancing freshness, source diversity, and redundancy:
  - Latest events
  - Latest per source
  - Hybrid latest + per-source
  - Guaranteed minimum source diversity
  - Guaranteed minimum source diversity with randomized eviction
  - Preferred sources
- Append-only hash-chained audit log (Go package + tests)
- MySQL CDC connector skeleton (Maxwell-style binlog reader interface)
- AI orchestrator skeleton with guardrails and human-override hooks
- Synthetic event generator for local development and demos
- REST + WebSocket API gateway skeleton
- ADRs 0001–0010 documenting major architectural decisions
- Compliance documentation:
  - NIST AI Risk Management Framework mapping
  - NIST SP 800-53 control mapping
  - FedRAMP readiness assessment
- Domain pack format specification + reference domain pack for clean-energy battery manufacturing

### Notes

This is pre-1.0 software in active development. APIs and behavior may
change. The first tagged release (v0.1.0) is planned for end of July 2026.
