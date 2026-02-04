# ADR 0004 — MyRocks for the archive tier

- **Status:** Accepted
- **Date:** 2025-11-25

## Context

FluxLens needs an archive tier for events that have aged out of the hot
store (default 30 days) and must be retained for warranty, regulatory,
and audit purposes (typically 5–10 years).

## Decision

Use MyRocks (Facebook's RocksDB storage engine for MySQL) as the
default archive backend.

## Rationale

- **Compression.** MyRocks achieves roughly 2:1 storage reduction
  vs. equivalent InnoDB on FluxLens-shaped workloads, consistent with
  Vanga & Buthalapalli (2025).
- **Write amplification.** LSM-tree write amplification is lower than
  B-tree InnoDB for the write-heavy archive workload.
- **Partition-based purge.** MyRocks supports efficient partition
  dropping, allowing time-based retention purge in seconds rather than
  hours.
- **Operator familiarity.** MyRocks is MySQL-protocol-compatible;
  operators with MySQL operational experience can run it without
  retraining.

## Alternatives considered

- **Plain InnoDB.** No compression benefit; higher storage cost.
- **S3 / object storage with Parquet.** Cheaper at very long retention
  but query latency is high; deferred to a future tiering ADR.
- **Cassandra.** Operator-familiarity gap; not Phase 1.

## Consequences

- Operators must install MyRocks (typically via MariaDB or Percona
  Server). The Helm chart documents the requirement.
- For deployments without MyRocks expertise, FluxLens supports
  configuring InnoDB as the archive tier at the cost of higher storage.
