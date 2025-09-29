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
available as an open-source platform you can adopt today.

FluxLens is the first attempt at building one.

## What FluxLens is

FluxLens is a composable platform with four layers:

1. **Hyper-scale CDC ingestion.** Read changes from your source
   systems (MySQL, Postgres, Kafka, webhooks) with zero meaningful
   impact on the sources themselves. The pattern is the same one I
   documented at trillion-record-per-month production scale in a
   recent technical paper.
2. **Curation that knows the operational tradeoffs.** Six
   configurable selection algorithms balance freshness (operators
   see the latest), source diversity (no single source monopolizes
   attention), and redundancy suppression (operators don't see the
   same event twice). The algorithms generalize work I co-authored
   on social-media digest systems; the math turns out to apply
   directly to operational events.
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

## What I built today

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
  (production readiness, multi-AZ, hardening) is the next ~3 months
  of work.
- Production case studies. There are none yet. If you deploy
  FluxLens, I'd be very interested in talking.
- Closed-source extensions or hosted SaaS. The full platform is
  Apache 2.0 and intended to remain so.

## Why I'm building this

I've spent a decade building these patterns privately at large U.S.
operators — federally funded research environments, clean-energy
vehicle manufacturing, national-scale retail and supply chain. The
patterns are real, they work, and they're nowhere available as
open source.

The U.S. is investing hundreds of billions of dollars into clean-
energy manufacturing capacity (IRA §45X), advanced manufacturing
R&D (CHIPS Act), and AI deployment that protects the American
workforce (EO 14110, NIST AI RMF). Capital creates capacity;
operational software determines whether capacity becomes output.

I want the operational software that converts that investment into
output to exist as open source. FluxLens is my contribution.

## How to help

If you've thought about these problems — at your employer, in
research, as a vendor, as an investor — I'd love to hear from you.

- **Try the demo.** `git clone`, `make demo`, tell me what
  surprised you, what confused you, what would have made you stop
  using it.
- **Open issues.** Anything that's broken, unclear, or missing.
- **Star the repo** if you'd like to see this exist.
- **Build a domain pack** for your sector. The three I've shipped
  cover clean-energy manufacturing, retail, and federal research.
  There are obvious gaps: utilities, transportation, healthcare,
  finance.
- **Contribute a CDC connector** for a source FluxLens doesn't yet
  support (Postgres logical replication, MongoDB change streams,
  CockroachDB CDC, etc.).
- **Write about it.** If FluxLens-shaped infrastructure helps your
  organization, a public mention is one of the most useful things
  you can do.

If you'd like to talk about a deployment, sponsorship, or
collaboration: sriharshav1@gmail.com.

## What's next

The roadmap is at `ROADMAP.md` in the repository. Phase 1 (the
MVP) is what shipped today. Phase 2 takes the platform to
production-readiness: full Postgres CDC, multi-AZ deployment with
chaos testing, OAuth2/OIDC and RBAC, Sigstore image signing,
Helm-chart hardening, comprehensive observability. Phase 3
(roughly mid-2027) is ecosystem and integrations — Helm chart in
public catalogs, AWS Marketplace listing, Splunk and Elastic
integrations, plugin SDK, formal community steering committee.

Thanks for reading.

— Sri Harsha Vanga
sriharshav1@gmail.com
linkedin.com/in/sriharshav1
