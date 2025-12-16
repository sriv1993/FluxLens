# Domain Pack — Retail Supply-Chain Resilience

> Reference domain pack for U.S. national-scale retailers operating in
> CISA-designated Commercial Facilities and Food and Agriculture
> critical-infrastructure sectors.

## Scope

Targets national-scale U.S. retailers with distributed retail and
distribution operations. Typical operators: national supermarket and
mass-retail chains, regional grocery chains, pharmacy chains,
distribution-network operators.

## Canonical event types

| Event type | Severity guidance | Description |
|---|---|---|
| `facility.opening.completed` | info | Facility opened for business |
| `facility.closing.completed` | info | Facility closed |
| `workforce.scheduling.deficit` | warn | Scheduled headcount below operational threshold |
| `workforce.attendance.no_show.surge` | warn → error | Above-baseline no-show rate detected |
| `inventory.stockout.imminent` | warn → error | Critical SKU stockout predicted within window |
| `inventory.stockout.actual` | error | Critical SKU stocked out |
| `supply_chain.delivery.late` | warn → error | Inbound delivery exceeds expected window |
| `supply_chain.disruption.detected` | warn → error | Upstream supply-chain anomaly detected |
| `safety.incident.minor` | warn | Minor on-site safety incident |
| `safety.incident.major` | critical | Major on-site safety incident |
| `safety.severe_weather.alert` | critical | Severe-weather alert for facility geofence |
| `safety.public_health.alert` | critical | Public-health alert for facility geofence |
| `safety.active_threat.alert` | critical | Active-threat alert for facility geofence |
| `payment.system.outage` | error | Payment system unavailable |
| `loss_prevention.shrinkage.anomaly` | warn | Shrinkage rate above baseline |

## Operator roles

| Role | Responsibility |
|---|---|
| Store manager | Per-facility operations, frontline associate coordination |
| Regional ops director | Multi-facility coordination, regional resource allocation |
| Workforce-planning analyst | Scheduling and labor-supply optimization |
| Supply-chain analyst | Inbound/outbound logistics monitoring |
| Loss-prevention analyst | Shrinkage and theft anomaly response |
| Safety officer | Worker safety, emergency response |
| Incident commander | Major-incident coordination across roles |

## Recommended curation configuration

| Role | Strategy | Diversity % | K |
|---|---|---|---|
| Store manager (per facility) | Hybrid latest + per-source | n/a | 20 |
| Regional ops director | Guaranteed min diversity | 80 | 40 |
| Workforce-planning analyst | Guaranteed min diversity | 70 | 30 |
| Supply-chain analyst | Preferred sources (critical suppliers) | 60 | 30 |
| Loss-prevention analyst | Guaranteed min diversity | 70 | 20 |
| Safety officer | Latest, filtered severity ≥ error | n/a | 10 |
| Incident commander | Latest, filtered severity == critical | n/a | 10 |

## Critical-infrastructure alignment

This domain pack directly supports operators meeting the resilience
priorities of CISA's Commercial Facilities and Food and Agriculture
sectors:

- Sub-second emergency-response routing via the safety-officer and
  incident-commander views
- Workforce continuity via the workforce-planning analyst view
- Supply-chain disruption-detection via the supply-chain analyst view
- Per-facility situational awareness via store-manager views

## Audit-trail considerations

Retail incidents are subject to regulatory, insurance, and
liability scrutiny. The FluxLens audit log preserves the full decision
trail for every safety incident, supply-chain disruption, and
workforce-action decision.

Recommended retention: 7 years (typical statute-of-limitations
window for retail liability), with optional WORM mirroring for
high-stakes deployments.
