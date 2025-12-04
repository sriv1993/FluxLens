# ADR 0009 — OpenTelemetry as the sole observability standard

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens services need to emit traces, metrics, and structured logs in
a vendor-neutral format that operators can route to their existing
observability stack.

## Decision

FluxLens services emit OpenTelemetry traces (W3C trace context),
metrics in Prometheus exposition format (the OpenMetrics standard
backed by the OpenTelemetry community), and structured logs (JSON,
line-delimited). The OpenTelemetry Collector sidecar aggregates and
exports to operator-configured backends.

## Rationale

- **Vendor neutrality.** OpenTelemetry is the CNCF-incubated standard
  for observability; operators retain backend choice (Datadog, Honeycomb,
  Grafana, AWS X-Ray, Jaeger).
- **Trace context propagation.** W3C trace context is supported across
  modern HTTP, gRPC, and Kafka clients, allowing end-to-end traces
  across the FluxLens pipeline.
- **Reduced code duplication.** A single observability initialization
  per service rather than multiple per-backend SDKs.

## Consequences

- All FluxLens services depend on the OpenTelemetry Go SDK; this
  imposes a dependency surface but is justified by the vendor-neutrality
  benefit.
- Operators wanting a single backend must run an OpenTelemetry
  Collector or compatible receiver. The Helm chart supports a sidecar
  collector out of the box.

## Alternatives considered

- **Per-backend SDKs (Datadog, Honeycomb, etc.).** Vendor lock-in.
- **Plain logging only.** Insufficient for distributed tracing
  requirements.
- **Custom telemetry format.** Rejected; reinvention of OpenTelemetry.
