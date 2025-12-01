# ADR 0006 — Human-override enforced in code, not policy

- **Status:** Accepted
- **Date:** 2026-05-16
- **Author:** Sri Harsha Vanga

## Context

FluxLens deploys AI decision-support in operational contexts where
AI suggestions may inform consequential actions (manufacturing-line
adjustments, emergency-response routing, supply-chain decisions).
The NIST AI Risk Management Framework and Executive Order 14110 on
trustworthy AI both identify human oversight as a foundational
property of trustworthy AI deployment.

There are two distinct ways to "guarantee" human oversight:

1. **Policy-based.** Operators commit (in policy, in training, in
   procedure) that human operators will review AI suggestions
   before consequential actions are taken.
2. **Code-enforced.** The platform's decision pathway makes it
   architecturally impossible for an AI suggestion to result in a
   consequential action without an explicit human acceptance.

## Decision

FluxLens enforces human override **in code**. The decision pathway
is constructed such that:

- Every AI suggestion is logged before it is surfaced to the
  operator.
- Every consequential action requires an explicit operator action
  (accept, override, annotate) recorded against the suggestion.
- The platform exposes no API path that takes a consequential
  action directly from an AI suggestion without operator action.
- Operator action is recorded in the audit log.

This is not configurable; there is no "autonomous mode" toggle.

## Rationale

Policy-based oversight is sufficient when the cost of
non-compliance is acceptable and trust between operators is high.
Both conditions fail in FluxLens's target deployment contexts:

- **Cost of non-compliance is high.** In manufacturing, emergency
  response, and critical-infrastructure contexts, an autonomous AI
  action causing harm has severe operational, financial, and
  regulatory consequences.
- **Trust must be earned, not assumed.** A platform that can be
  configured to "autonomous mode" can have that configuration set
  in production by accident or by malicious insider. Architectural
  prohibition removes this failure mode.

Enforcing override in code also has a documentation benefit: the
platform's behavior matches its description. A FluxLens deployment
can be audited (including by adversarial auditors) to confirm that
the override property holds. This is significantly stronger than
operator self-attestation.

## Consequences

- FluxLens cannot be used in true closed-loop autonomous-control
  contexts where no human is in the loop. This is intentional and
  acceptable.
- The latency of operator response is bounded below by human
  reaction time. For some use cases this is unacceptable; FluxLens
  is not the right platform for those use cases.
- Operators cannot configure their way around this property. This
  is the point.

## Alternatives considered

- **Policy-based oversight only.** Rejected; insufficient for
  target deployment contexts.
- **Configurable autonomous mode.** Rejected; defeats the
  guarantee.
- **Per-event-class autonomy.** Rejected for v1; may be
  reconsidered in a future ADR if a clear, restricted use case
  emerges with adequate safety constraints.

## References

- NIST AI Risk Management Framework (NIST AI 100-1), section on
  human oversight
- Executive Order 14110 (October 30, 2023), provisions on AI
  augmenting and protecting the American workforce
