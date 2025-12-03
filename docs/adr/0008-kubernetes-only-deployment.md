# ADR 0008 — Kubernetes-native deployment

- **Status:** Accepted
- **Date:** 2026-05-16

## Context

FluxLens deployments need consistent horizontal scaling, rolling
deployments, secrets management, observability, and service
discovery. Options: Kubernetes-native, plain Docker Compose, systemd
units, OS-specific package managers.

## Decision

FluxLens supports two deployment surfaces: (1) `docker-compose` for
local development and small single-node demos, and (2) Kubernetes (via
the Helm chart in `deploy/helm/fluxlens/`) for production. No
support is provided for systemd-based or package-manager-based
deployment.

## Rationale

- **Operator alignment.** Operators in FluxLens's target deployment
  contexts (clean-energy manufacturing IT, retail tech, federal labs)
  predominantly run Kubernetes for production workloads.
- **Reliability properties.** The reliability primitives FluxLens
  depends on (per-pod restart, HPA, multi-AZ scheduling, rolling
  upgrade, service mesh integration) are easiest to obtain on
  Kubernetes.
- **Air-gapped portability.** Kubernetes can be deployed in
  air-gapped environments (k3s, OpenShift on-prem, etc.).

## Consequences

- Operators without Kubernetes expertise have a higher initial
  learning curve. The Helm chart and quickstart attempt to minimize
  this.
- Future support for serverless deployment (e.g., AWS Fargate ECS)
  remains possible but is not in scope for v1 or v2.

## Alternatives considered

- **Plain Docker Compose only.** Insufficient for production
  reliability requirements.
- **Nomad.** Operator-familiarity gap in target sectors.
- **OS-specific packages.** Operationally fragmented; rejected.
