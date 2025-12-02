# ADR 0007 — Pluggable LLM provider interface

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens deploys in environments with diverse LLM-provider preferences:
some operators use OpenAI; some use self-hosted models (Ollama, vLLM);
some require air-gapped deployment with no external LLM access; some
want to evaluate multiple providers in parallel.

## Decision

The orchestrator interacts with LLMs through the `llm.Provider`
interface (`internal/llm/provider.go`). Concrete providers are
selected at startup via configuration. Adding a new provider requires
implementing four methods.

## Rationale

- **Avoid vendor lock-in.** Operators are not bound to a single LLM
  vendor.
- **Air-gapped deployment.** A local Ollama or vLLM endpoint plugs in
  with no orchestrator code changes.
- **Testability.** The `MockProvider` allows orchestrator and guardrails
  testing without any network calls.
- **Future ensembling.** A multi-provider ensemble (ADR-pending for
  Phase 2) is implemented as a Provider that fans out to multiple
  underlying Providers.

## Consequences

- All providers must produce JSON-mode output parseable into
  `DecisionResponse`. Provider implementations are responsible for
  prompt construction that elicits this format from the model.
- Operators evaluating provider quality should use the shadow-mode
  evaluator (Phase 1 stub, full implementation Phase 1 milestone M1.8)
  to compare providers against current production logic before
  cutover.

## Alternatives considered

- **OpenAI-only.** Rejected; locks out air-gapped and self-hosted
  deployments.
- **One-provider-per-deployment with no abstraction.** Rejected;
  makes future ensembling and shadow-mode evaluation harder.
