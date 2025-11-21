# ADR 0002 — Apache Kafka as the event bus

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens decouples ingestion from curation and decision-support via an
event bus. Options: Apache Kafka, NATS JetStream, Apache Pulsar,
Redpanda (Kafka API-compatible), AWS Kinesis.

## Decision

FluxLens uses Apache Kafka (or any Kafka-API-compatible implementation
such as Redpanda or Strimzi) as the event bus.

## Rationale

- **Operational maturity at scale.** Kafka is production-proven at the
  trillion-record-per-month scale FluxLens targets (cited in Vanga &
  Buthalapalli, 2025). Alternatives are credible but have less production
  evidence at this scale.
- **Ecosystem.** Mature client libraries, monitoring tooling, and
  CDC-connector integrations (Maxwell, Debezium).
- **Operator familiarity.** Most large-scale operators already run Kafka;
  reuse of existing infrastructure reduces deployment friction.
- **Kafka-API compatibility.** Redpanda, Bitnami's Kafka image, and
  managed offerings (Confluent Cloud, AWS MSK) provide multiple
  deployment options without code changes.

## Consequences

- FluxLens requires a Kafka-compatible bus to function. This is an
  intentional dependency.
- Operators in environments without an existing Kafka deployment must
  stand one up. Phase 1 docker-compose ships a single-broker Kafka for
  local development.

## Alternatives considered

- **NATS JetStream.** Lighter and simpler but less production evidence
  at trillion-record scale.
- **Apache Pulsar.** Tiered-storage native; rejected for v1 due to
  ecosystem-maturity gap; may be reconsidered later.
- **AWS Kinesis.** Cloud-vendor lock-in; rejected for open-source.
