# FluxLens — FedRAMP Readiness

> This document describes FluxLens's architectural posture relative to
> FedRAMP authorization requirements. It is informational; FluxLens is
> not currently FedRAMP-authorized. Operators pursuing FedRAMP ATO
> remain responsible for their specific deployment package, 3PAO
> assessment, and continuous-monitoring program.

## Posture summary

FluxLens is designed to be **deployable in environments pursuing
FedRAMP Moderate or High authorization**. The platform implements
architectural controls and operational discipline appropriate for
those baselines. FluxLens does not pursue platform-level FedRAMP
authorization at this time; operators incorporate FluxLens as a
component within their own authorization boundary.

## Baseline alignment

### FedRAMP Moderate baseline

FedRAMP Moderate inherits NIST SP 800-53 Rev. 5 Moderate baseline
controls. See `docs/compliance/nist-800-53-mapping.md` for the
control-family mapping. FluxLens-implemented controls align
strongly across the families most relevant to a data-and-AI
decision-support platform.

### FedRAMP High baseline

FluxLens's hash-chained audit log, optional WORM mirroring, FIPS-
compatible cryptography, and air-gapped deployment mode are
oriented toward FedRAMP High deployments. Additional Phase 2
hardening (KMS-backed signing of audit records; mTLS-everywhere
defaults) further supports High-baseline deployment.

## Deployment patterns supported

### Cloud-region restriction

FluxLens has no architectural dependency on any specific cloud
region. Operators pursuing FedRAMP authorization typically restrict
deployment to U.S.-region cloud providers (e.g., AWS GovCloud
(US), Azure Government, Google Cloud Assured Workloads for U.S.
Public Sector). FluxLens deploys identically in those regions.

### Air-gapped / restricted-egress deployment

FluxLens supports fully air-gapped deployment with no required
outbound network calls:

- Local LLM providers (Ollama, vLLM) replace external LLM APIs.
- No telemetry is sent off-cluster by default.
- Container images can be mirrored to an internal registry.

See `docs/deployment/air-gapped.md` for the full air-gapped
deployment guide.

### FIPS-compatible cryptography

FluxLens's audit-log hashing uses SHA-256, which is FIPS 140-3
approved. For deployments requiring strict FIPS validation,
operators should build with a FIPS-validated Go toolchain (e.g.,
Microsoft's Go FIPS builds, or BoringSSL-backed Go).

## What FluxLens contributes to a FedRAMP package

When an operator includes FluxLens in a system pursuing FedRAMP
authorization, the platform contributes:

- Architectural primitives and control implementations that map to
  NIST 800-53 controls (see `nist-800-53-mapping.md`).
- Open-source code that the 3PAO can review (no opaque binary).
- Architecture and design documentation suitable for inclusion in
  System Security Plan (SSP) sections.
- Continuous-monitoring artifacts (Prometheus metrics, audit log
  exports) suitable for FedRAMP continuous-monitoring submissions.

## What FluxLens does not provide

- Authority to Operate (ATO)
- 3PAO assessment
- POA&M management
- Continuous monitoring as a service
- Substitute for operator-specific compliance review

FluxLens provides the architectural primitives. Compliance is
achieved through the operator's specific deployment, configuration,
and operational program.

## Recommended steps for operators pursuing FedRAMP

1. Confirm FedRAMP baseline (Moderate, High, or Tailored).
2. Deploy in a FedRAMP-authorized cloud region.
3. Configure FluxLens with FedRAMP-aligned baseline values
   (TLS 1.3 minimum, encryption at rest, OAuth2/OIDC, RBAC
   enabled, audit log replicated to WORM storage).
4. Build container images from source using a FIPS-validated Go
   toolchain if FIPS validation is required.
5. Map FluxLens-implemented controls into the System Security Plan
   using `nist-800-53-mapping.md` as a starting point.
6. Include FluxLens in the 3PAO assessment scope.
7. Establish continuous-monitoring routines around the FluxLens
   audit log and metrics.
