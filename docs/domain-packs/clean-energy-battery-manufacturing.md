# Domain Pack — Clean-Energy Battery Manufacturing

> Reference domain pack for U.S. clean-energy battery manufacturers
> (cell, module, and pack producers). Defines canonical event types,
> severity classifications, operator roles, and recommended curation
> configuration for FluxLens deployment in this domain.

## Scope

This domain pack targets U.S. battery manufacturers in the segments
incentivized by IRA §45X(b) (battery cells and modules, applicable
critical minerals). Typical operators:

- Gigafactory operators producing lithium-ion cells and modules at
  multi-GWh annual capacity
- Battery pack assemblers supplying EV OEMs
- Specialized battery chemistries (Sila Nano, QuantumScape, Solidion,
  Form Energy, etc.)

## Canonical event types

| Event type | Severity guidance | Description |
|---|---|---|
| `cell.formation.cycle.completed` | info | A cell completed a formation-cycle step |
| `cell.formation.cycle.failed` | error | A cell failed a formation-cycle step |
| `cell.quality.electrochemical.anomaly` | warn → error | Electrochemical signature outside specification |
| `cell.quality.dimensional.deviation` | warn → error | Dimensional inspection outside tolerance |
| `module.assembly.weld.failure` | error | Module welding inspection failure |
| `module.assembly.completed` | info | Module assembly step completed |
| `pack.test.thermal.runaway.precursor` | critical | Thermal-runaway precursor detected |
| `pack.test.completed` | info | Pack-level test step completed |
| `line.throughput.drop` | warn | Line throughput dropped below threshold |
| `line.equipment.downtime` | warn → error | Equipment downtime detected |
| `materials.supply.delivery.late` | warn → error | Critical-material delivery is late |
| `materials.supply.disruption.predicted` | warn | AI-driven supply-disruption prediction |
| `safety.evacuation.required` | critical | Plant-floor evacuation conditions detected |

## Severity escalation guidance

Severity levels in this domain pack escalate based on:

- Frequency thresholds (multiple `warn` events from the same source
  within a window escalate to `error`)
- Proximity to safety-critical equipment
- Time-to-detection windows (a defect detected at end-of-line is
  more severe than one detected upstream because the cost of
  remediation is higher)

## Operator roles

| Role | Typical responsibilities |
|---|---|
| Line operator | Frontline associate responsible for one production line |
| Cell-quality engineer | Reviews electrochemical and dimensional anomalies |
| Process engineer | Owns line-throughput and equipment-downtime response |
| Materials/supply-chain engineer | Owns supply-disruption prediction and response |
| Safety officer | Owns evacuation and safety-critical events |
| Compliance/recall engineer | Owns audit-trail review for warranty and recall scenarios |

## Recommended curation configuration

### Per role

| Role | Strategy | Diversity % | K | Preferred sources |
|---|---|---|---|---|
| Line operator (per line) | StrategyHybridLatestAndPerSource | n/a | 20 | The operator's own line |
| Cell-quality engineer | StrategyGuaranteedMinDiversity | 80 | 40 | All formation/quality sources |
| Process engineer | StrategyGuaranteedMinDiversity | 70 | 30 | All line-throughput and equipment sources |
| Materials/supply-chain engineer | StrategyPreferredSources | 60 | 20 | Critical-materials supplier sources |
| Safety officer | StrategyLatest | n/a | 10 | All safety-critical sources (filtered to severity ≥ error) |
| Compliance/recall engineer | (audit log view; not curated stream) | — | — | — |

### Suppression windows

Suppress within 60 seconds (suppression set bounded at 5,000 IDs per
operator) to prevent flapping events from dominating attention.

## Curation rationale

This domain pack uses a mix of all six FluxLens curation strategies
across roles, demonstrating the value of the algorithm catalog:

- **StrategyHybridLatestAndPerSource** for line operators ensures
  the operator sees their own line in detail while also catching
  the latest events from feeder stations.
- **StrategyGuaranteedMinDiversity** for cell-quality and process
  engineers ensures no single high-volume source crowds out
  lower-volume but operationally critical sources.
- **StrategyPreferredSources** for materials/supply-chain engineers
  ensures critical-materials supplier sources are surfaced even
  when other event traffic dominates.
- **StrategyLatest** filtered to severity ≥ error for safety
  officers ensures the freshest critical events reach the safety
  function without latency.

## Integration with existing manufacturing systems

| External system | Recommended FluxLens connector | Notes |
|---|---|---|
| MES | `mysql-cdc` or `postgres-cdc` | Capture work-order state changes |
| SCADA gateway | `kafka` or `webhook` | High-volume sensor telemetry; configure source-side downsampling |
| Quality database | `mysql-cdc` or `postgres-cdc` | Capture quality-test outcomes |
| Supply-chain platform | `webhook` or `kafka` | Capture supplier-delivery events |
| Plant-floor PLCs | (not directly; via SCADA gateway) | FluxLens does not directly interface with PLCs |

## Audit-trail considerations

Battery manufacturing is subject to warranty, recall, and
consumer-product-safety regulation. The FluxLens audit log
preserves the full decision history for every cell, module, and
pack — supporting after-the-fact warranty claim investigation,
recall scoping, and regulatory submission.

Operators should:

- Tag every event with the relevant cell/module/pack serial
  number in the `payload`
- Configure retention to meet warranty + regulatory minimums
  (typically 8–10 years for U.S. battery products)
- Replicate the audit log to WORM storage (S3 Object Lock in
  compliance mode) for tamper-evidence beyond the in-cluster hash
  chain

## What this domain pack does not include

- Cell-chemistry-specific event ontologies (operators define these
  for their own cell chemistry)
- Equipment-vendor-specific event ontologies (operators map these
  from their MES / SCADA gateway)
- Recall-management workflow (FluxLens preserves the trail; recall
  workflow is operator-specific)
