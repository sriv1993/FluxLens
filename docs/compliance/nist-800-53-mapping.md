# FluxLens — NIST SP 800-53 Control Family Mapping

> Mapping of FluxLens platform capabilities to selected NIST Special
> Publication 800-53 (Rev. 5) control families. This document is
> informational; it does not constitute a compliance attestation.
> Operators remain responsible for end-to-end compliance of their
> specific deployment.

This mapping focuses on the control families most directly relevant
to a data-and-AI decision-support platform deployed in regulated
environments. It does not enumerate every applicable control.

## Access Control (AC)

| Control | FluxLens implementation |
|---|---|
| AC-2 Account Management | OAuth2/OIDC integration; role-based identity (operator / reviewer / admin / auditor) |
| AC-3 Access Enforcement | Authz middleware in API gateway; per-endpoint RBAC checks |
| AC-6 Least Privilege | Per-role minimal privilege sets; Maxwell connector uses dedicated MySQL user with REPLICATION CLIENT/SLAVE only |
| AC-17 Remote Access | All API access via TLS 1.3; service mesh mTLS for internal traffic |

## Audit and Accountability (AU)

| Control | FluxLens implementation |
|---|---|
| AU-2 Event Logging | Every event ingest, decision, operator action recorded |
| AU-3 Content of Audit Records | Audit record schema includes who/what/when/where (operator ID, decision ID, timestamp, input hash, response, action) |
| AU-9 Protection of Audit Information | Hash-chained append-only chain; chain verifier process; optional WORM mirroring to S3 Object Lock |
| AU-10 Non-Repudiation | Hash chain provides tamper evidence; signed-commit policy for code (DCO) |
| AU-12 Audit Record Generation | All services emit audit records; coverage tested in CI |

## Configuration Management (CM)

| Control | FluxLens implementation |
|---|---|
| CM-2 Baseline Configuration | All configuration version-controlled in repo |
| CM-6 Configuration Settings | Helm chart values define operator-controllable configuration |
| CM-7 Least Functionality | Air-gapped deployment mode disables all non-essential network egress |
| CM-8 System Component Inventory | Container images signed via Sigstore; inventory enforced by admission controller |

## Contingency Planning (CP)

| Control | FluxLens implementation |
|---|---|
| CP-9 System Backup | Audit log replicated to WORM storage; Kafka topic replication factor 3+ |
| CP-10 System Recovery | Kafka offset checkpointing supports recovery from any prior point; Postgres synchronous streaming replication |

## Identification and Authentication (IA)

| Control | FluxLens implementation |
|---|---|
| IA-2 User Identification | OAuth2/OIDC; per-user authentication |
| IA-5 Authenticator Management | Externally managed via OIDC provider |
| IA-8 Identification (non-organizational users) | Service-account JWTs for service-to-service authentication |

## Incident Response (IR)

| Control | FluxLens implementation |
|---|---|
| IR-4 Incident Handling | Per-decision audit trail supports after-action review |
| IR-6 Incident Reporting | Audit log export supports SIEM integration (Splunk, Elastic) |

## Risk Assessment (RA)

| Control | FluxLens implementation |
|---|---|
| RA-5 Vulnerability Monitoring and Scanning | CI runs CodeQL and dependency vulnerability scans; results gated on PR merge |

## System and Communications Protection (SC)

| Control | FluxLens implementation |
|---|---|
| SC-8 Transmission Confidentiality and Integrity | TLS 1.3 in transit; service mesh mTLS |
| SC-13 Cryptographic Protection | SHA-256 for audit chain hashing; KMS-backed signing in Phase 2 |
| SC-28 Protection of Information at Rest | Encryption at rest on all data stores |

## System and Information Integrity (SI)

| Control | FluxLens implementation |
|---|---|
| SI-2 Flaw Remediation | Dependabot enabled; release cadence supports security patches |
| SI-3 Malicious Code Protection | Container image scanning in CI |
| SI-7 Software, Firmware, and Information Integrity | Hash-chained audit log; signed container images |
| SI-10 Information Input Validation | Guardrails input validation; canonical event schema validation |

## What this mapping does not provide

- Compliance certification or attestation
- ATO (Authority to Operate) for any specific deployment
- Substitute for FedRAMP / FISMA / SOC2 audit
- Operator-specific guidance on which controls to apply

It provides a structured starting point for operator compliance review.
