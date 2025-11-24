# ADR 0003 — Postgres + TimescaleDB for the hot store

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens needs a hot store for recent events (default 30-day retention)
that supports time-series indexing, ACID guarantees, and operator-facing
ad-hoc queries.

## Decision

Use PostgreSQL with the TimescaleDB extension for the hot store.

## Rationale

- **ACID + time-series.** TimescaleDB layers time-series partitioning
  on top of standard Postgres, preserving ACID guarantees while giving
  range-query performance comparable to purpose-built TSDBs.
- **Ad-hoc query support.** SQL is the lingua franca of operator
  analytics. Operators can write their own queries without learning a
  proprietary query language.
- **Mature ecosystem.** Postgres has a deep ecosystem of monitoring,
  backup, replication, and migration tooling.
- **License clarity.** Both Postgres (PostgreSQL License) and TimescaleDB
  (Apache 2.0 for the open-source community edition) are open-source-
  friendly.

## Alternatives considered

- **ClickHouse.** Excellent at columnar analytics but transactional
  guarantees are weaker; deferred for v1.
- **InfluxDB.** Purpose-built TSDB but smaller ecosystem and the v2/v3
  transition created operator uncertainty.
- **Plain Postgres without TimescaleDB.** Range-query performance on
  long time-series degrades without hypertable partitioning.

## Consequences

- Hot-store performance depends on appropriate hypertable chunking
  (default 1-day chunks; tunable per deployment).
- Operators must install the TimescaleDB extension; the Helm chart and
  docker-compose stack include it by default.
