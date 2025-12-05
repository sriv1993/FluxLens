# ADR 0010 — Apache 2.0 license

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens needs a license that maximizes adoption across the target
operator base (U.S. clean-energy manufacturers, national retailers,
federal labs, regulated industries) while preserving the project lead's
ability to contribute to the project's long-term direction.

## Decision

FluxLens is licensed under the Apache License 2.0.

## Rationale

- **Adoption maximization.** Apache 2.0 is the most widely adopted
  permissive license in the operator categories FluxLens targets. It
  has no copyleft viral effects that would discourage commercial or
  proprietary internal modifications.
- **Patent grant.** Apache 2.0 includes an explicit patent grant
  protecting downstream users — important for sectors with active
  patent activity (clean-energy manufacturing).
- **Federal adoption.** Apache 2.0 is acceptable for adoption by U.S.
  federal agencies and federally funded research and development
  centers.
- **Contributor protection.** Section 9 protects individual contributors
  from extended liability claims.

## Alternatives considered

- **MIT.** Acceptable but lacks the explicit patent grant.
- **BSD-3-Clause.** Acceptable but slightly less common in the target
  operator base.
- **AGPL-3.0.** Discourages operator adoption due to copyleft viral
  effects in network-service deployments.
- **Business Source License (BSL).** Discourages adoption due to
  commercial-restriction provisions.

## Consequences

- Operators are free to modify and redistribute FluxLens internally
  without contribution-back obligations.
- The project benefits from external contributions only when
  contributors choose to upstream them. This is intentional for v1;
  community-building activities will encourage upstreaming.

## References

- Apache License 2.0: https://www.apache.org/licenses/LICENSE-2.0
- OpenSSF License Compliance Guidelines
