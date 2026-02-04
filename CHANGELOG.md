# Changelog

All notable changes to FluxLens will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- Six curation algorithms generalized from Buthalapalli & Vanga (2025):
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
- National-interest framing documentation:
  - IRA / CHIPS alignment
  - DOE alignment
  - CISA critical-infrastructure alignment
  - Workforce multiplier impact analysis
- Compliance documentation:
  - NIST AI Risk Management Framework mapping
  - NIST SP 800-53 control mapping
  - FedRAMP readiness assessment
- Domain pack format specification + reference domain pack for clean-energy battery manufacturing

### Notes

This is pre-1.0 software in active development. APIs and behavior may
change. The first tagged release (v0.1.0) is planned for end of July 2026.
