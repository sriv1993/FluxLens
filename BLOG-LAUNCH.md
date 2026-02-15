# FluxLens: An Open-Source Platform for AI-Augmented Industrial Event Curation

*Posted: [LAUNCH DATE]. By Sri Harsha Vanga.*

> **TL;DR.** FluxLens is a new open-source platform that combines
> hyper-scale Change Data Capture ingestion, freshness/diversity/
> redundancy-aware event curation, AI decision support with hard
> human-override guarantees, and tamper-evident audit logging. It's
> designed for the operational realities of clean-energy
> manufacturing, national-scale retail and supply-chain operations,
> and federally funded research environments. Apache 2.0. Repository:
> https://github.com/sriharshav1/fluxlens

---

## The problem

If you operate a modern industrial system — a battery gigafactory, a
nationwide retail network, a federally funded research facility —
you face the same composite problem:

- Your systems emit far more events than your operators can attend
  to.
- A few high-volume sources crowd out the lower-volume but
  operationally critical ones.
- AI assistants help, but you cannot deploy them in regulated or
  consequential contexts without verifiable human oversight and
  audit trails.
- And when an incident happens, you need a defensible record of
  every decision — auditable, tamper-evident, and exportable to
  regulators or insurers.

The architectural patterns that solve this problem exist. They have
been built — privately — at large operators. They have not been
widely packaged as an open-source platform you can adopt today.

FluxLens is an attempt at building one.

## What FluxLens is

FluxLens is a composable platform with four layers:

1. **Hyper-scale CDC ingestion.** Read changes from your source
   systems (MySQL, Postgres, Kafka, webhooks) with minimal impact
   on the sources themselves — the same CDC + Kafka idioms common at
   large industrial scale.
2. **Curation that knows the operational tradeoffs.** Six
   configurable selection algorithms balance freshness (operators see
   the latest), source diversity (no single source monopolizes
   the digest), and redundancy suppression.
3. **AI decision support with hard guarantees.** The orchestrator
   calls an LLM through a pluggable provider interface (OpenAI,
   Ollama, vLLM, mock for tests), validates input and output via
   guardrails, and surfaces a suggestion to the operator — never
   takes the action itself. Override is enforced in code, not in
   policy.
4. **Tamper-evident audit log.** Every decision and every operator
   action is hash-chained. Tampering breaks the chain detectably.
   Optional WORM mirroring is supported for high-stakes
   deployments.

## What shipped today

- A complete, working Phase 1 MVP. `make demo` brings up the stack
  and runs synthetic events through ingestion → curation → AI
  orchestration → audit, end-to-end.
- A polished React + TypeScript operator dashboard.
- All six curation algorithms with unit and end-to-end tests.
- An OpenAI-compatible LLM provider (works with OpenAI, Ollama,
  vLLM, WireMock for CI) and a mock provider for tests.
- A Helm chart for Kubernetes deployment.
- Three reference domain packs: clean-energy battery manufacturing,
  retail supply-chain resilience, federal research coordination.
- Ten Architecture Decision Records explaining the design choices.
- Mappings to NIST AI Risk Management Framework, NIST SP 800-53
  control families, and FedRAMP readiness posture.
- Substantial documentation of how the platform aligns with
  federally identified priorities (IRA §45X, CHIPS Act, DOE smart
  manufacturing, CISA critical-infrastructure framework, FEMA
  National Preparedness Goal).

## What's intentionally NOT in the launch

- A "trust me, it's production-ready" claim. It isn't. Phase 1 is
  an MVP that works end-to-end on a single machine. Phase 2
  (production readiness, multi-AZ, hardening) is the next stretch of work.
- Production case studies. There are none yet. Feedback from early
  adopters is welcome.
- Closed-source extensions or hosted SaaS. The full platform is
  Apache 2.0 and intended to remain so.

## Why FluxLens matters

Operational patterns matter for U.S.-critical sectors: clean-energy capacity (IRA §45X), advanced-manufacturing R&D (CHIPS Act), and AI deployment aligned with EO 14110 and the NIST AI RMF.

This repository exists so those operational patterns — hyper-scale CDC, intelligent event curation, AI with explicit human override, and auditable chains — have a permissively licensed implementation others can fork, harden, and deploy.

## How to help

- **Try the demo.** `git clone`, `make demo`, and send feedback via issues.
- **Open issues.** Anything that's broken, unclear, or missing.
- **Star the repo** if you want to see ongoing development.
- **Build a domain pack** for your sector. Reference packs shipped for clean-energy manufacturing, retail, and federal research; gaps remain in utilities, transportation, healthcare, finance, and elsewhere.
- **Contribute a CDC connector** for a source FluxLens doesn't yet
  support (Postgres logical replication, MongoDB change streams,
  CockroachDB CDC, etc.).
- **Write about it** if FluxLens-shaped infrastructure helps your organization.

Technical discussion: https://github.com/sriharshav1/fluxlens/discussions

## What's next

The roadmap is at `ROADMAP.md` in the repository. Phase 1 (the
MVP) is what shipped. Phase 2 takes the platform toward
production-readiness: full Postgres CDC, multi-AZ deployment with
chaos testing, OAuth2/OIDC and RBAC, Sigstore image signing,
Helm-chart hardening, comprehensive observability. Phase 3
covers ecosystem integrations — catalogs, Marketplace listings,
SIEM integrations, plugin SDKs, formal community steering.

Thanks for reading.

— Sri Harsha Vanga
