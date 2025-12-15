# Domain Pack — Federal Research Coordination

> Reference domain pack for U.S. Department of Energy national
> laboratories and other federally funded research and development
> centers (FFRDCs).

## Scope

Targets federally funded research environments where research-
coordination events, instrument telemetry, and security-audit events
must be processed under federal IT compliance discipline. Typical
operators: DOE national laboratories (PNNL, ORNL, Argonne, LBNL, LLNL,
Sandia, NREL), NIST, NASA centers, NSF-funded research facilities.

## Canonical event types

| Event type | Severity guidance | Description |
|---|---|---|
| `research.coordination.meeting.created` | info | New research-coordination meeting scheduled |
| `research.coordination.meeting.attendance.recorded` | info | Attendance recorded |
| `research.coordination.proposal.submitted` | info | Research proposal submitted |
| `research.computing.job.completed` | info | HPC/HTC job completed |
| `research.computing.job.failed` | warn | HPC/HTC job failed |
| `instrument.run.started` | info | Instrument run began |
| `instrument.run.completed` | info | Instrument run completed |
| `instrument.malfunction.detected` | warn → error | Instrument malfunction detected |
| `data.export.requested` | info | Data export request submitted |
| `data.export.approved` | info | Data export approved by data steward |
| `data.export.rejected` | warn | Data export denied (compliance) |
| `security.access.granted` | info | Authorization granted |
| `security.access.revoked` | info | Authorization revoked |
| `security.access.unauthorized_attempt` | warn → critical | Unauthorized access attempt |
| `security.audit.event` | varies | Generic security-audit event |
| `compliance.policy.violation.detected` | error | Compliance-policy violation detected |

## Operator roles

| Role | Responsibility |
|---|---|
| Research-coordination manager | Cross-team coordination, meeting and proposal flow |
| Research-computing operations manager | HPC/HTC system operations |
| Instrument operations manager | Instrument health and run scheduling |
| Data steward | Data-export approval and compliance |
| Information security officer | Security audit and access control |
| Compliance officer | Policy violations and reporting |

## Recommended curation configuration

| Role | Strategy | Diversity % | K |
|---|---|---|---|
| Research-coordination manager | Guaranteed min diversity | 80 | 30 |
| Research-computing operations manager | Guaranteed min diversity | 70 | 30 |
| Instrument operations manager | Preferred sources (assigned instruments) | 70 | 20 |
| Data steward | Latest, filtered to data.* events | n/a | 20 |
| Information security officer | Guaranteed min diversity, severity ≥ warn | 90 | 30 |
| Compliance officer | Latest, filtered to compliance.* and security.audit.* | n/a | 20 |

## Federal-compliance alignment

This domain pack directly supports DOE national-laboratory IT
operations meeting federal IT compliance requirements:

- Audit-log retention aligned with DOE retention schedules
- RBAC structured to align with federal role separations
- Air-gapped deployment supported via local LLM providers
- FedRAMP-baseline-compatible configuration values

See `docs/compliance/fedramp-readiness.md` for the full FedRAMP
posture.

## Audit-trail considerations

Federally funded research operations are subject to varied retention
requirements depending on funding source, classification level, and
research domain. Operators should configure retention to the
strictest applicable requirement (typically 7–10 years).

WORM mirroring (S3 Object Lock compliance mode, or equivalent) is
recommended for deployments handling Controlled Unclassified
Information (CUI) or higher classification levels.

## What this domain pack does not include

- Classification-level tagging (operators define per their environment)
- Specific federal classification schemes (NIST SP 800-171, NIST SP
  800-53 control mappings are in `docs/compliance/`)
- DOE-specific data-sharing agreements (operator-specific)
