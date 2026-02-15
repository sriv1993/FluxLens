# ADR 0001 — Use CDC over polling for source-system ingestion

- **Status:** Accepted
- **Date:** 2025-11-20

## Context

FluxLens ingests change events from source systems (relational
databases, message buses, webhooks). For relational source systems
there are two common ingestion patterns:

1. **Polling.** The connector periodically queries the source for
   rows changed since the last poll, typically using a "last
   modified" column or a sequence counter.
2. **Change Data Capture (CDC).** The connector reads the source's
   replication log (MySQL binlog, Postgres WAL) and emits change
   events as they happen.

## Decision

FluxLens uses CDC for relational source-system ingestion. Polling
is supported only as a fallback for sources that do not expose a
replication log.

## Rationale

CDC has decisive advantages for FluxLens's use case:

- **Source-system non-impact.** CDC reads the replication log,
  which the source already writes; it does not impose query load
  on the source. Polling imposes recurring query load that
  scales with poll frequency and source size.
- **Latency.** CDC delivers changes within milliseconds of commit;
  polling latency is bounded below by poll frequency.
- **Completeness.** CDC sees every row change including updates and
  deletes; polling can miss intermediate states between polls.
- **Operational legibility.** CDC offsets are durable, recoverable,
  and observable; polling state is more fragile.

Large industrial operators routinely run CDC-based pipelines at
warehouse-to-petabyte monthly scale with minimal source impact; that
operational track record supports this choice for FluxLens.

## Consequences

- The MySQL connector requires `REPLICATION CLIENT` and
  `REPLICATION SLAVE` privileges on the source. Operators must
  provision these.
- The Postgres connector requires `wal_level = logical` and a
  logical replication slot. Operators must provision these.
- Sources without replication-log access (e.g., closed-source SaaS
  databases) cannot use CDC and must fall back to polling or
  webhooks.

## Alternatives considered

- **Polling only.** Rejected for the reasons above.
- **Dual-mode (CDC where possible, polling elsewhere).** Adopted;
  see Postgres-CDC milestone M2.1 for the dual-mode implementation.

## References

- Maxwell project: https://github.com/zendesk/maxwell
- Debezium project: https://debezium.io/
