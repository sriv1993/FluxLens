# Quickstart — FluxLens in 10 minutes

This walks through bringing up the FluxLens dev stack, generating
synthetic events, watching them flow through the curator and AI
orchestrator, and inspecting the audit log — all on a single laptop.

## Prerequisites

- Docker + Docker Compose v2
- Go 1.22+
- Node.js 20+ (for the dashboard)
- `make`

## Optional — Operator dashboard only (no Docker)

Use this when you want to record the UI quickly without Kafka:

```bash
make build

./bin/fluxlens-api-gateway --addr :8090 &
./bin/fluxlens-synth-source --no-kafka --gateway http://localhost:8090 --rate 25 --source-count 12 &
cd dashboard && npm install && npm run dev
```

Then open http://localhost:5173 — you should see freshness/diversity/redundancy scores, the curated feed, and an auditable hash chain.

---

## Step 1 — Start the dependency stack

```bash
git clone https://github.com/sriharshav1/fluxlens.git
cd fluxlens
make dev
```

This starts:

- Apache Kafka (single broker, KRaft mode)
- Postgres with TimescaleDB
- Redis
- WireMock mock-LLM endpoint (responding to OpenAI `/v1/chat/completions`)
- Prometheus
- Grafana (admin / fluxlens-dev)

Check status:

```bash
make dev-status
```

## Step 2 — Build the FluxLens binaries

```bash
make tidy
make test           # all unit + e2e tests should pass
make build
```

The binaries land in `./bin/`.

## Step 3 — Start the FluxLens pipeline

In three separate terminals:

```bash
# Terminal 1 — curator
./bin/fluxlens-curator --kafka localhost:9092 --strategy 4 --diversity 80 --k 20

# Terminal 2 — orchestrator (uses the local mock-LLM by default)
./bin/fluxlens-orchestrator --kafka localhost:9092 --llm-base http://localhost:8080

# Terminal 3 — API gateway
./bin/fluxlens-api-gateway --addr :8090
```

## Step 4 — Generate synthetic events

In a fourth terminal:

```bash
./bin/fluxlens-synth-source --kafka localhost:9092 --rate 100 --source-count 20 \
  --gateway http://localhost:8090
```

The `--gateway` flag mirrors each synthetic event into the API gateway so the dashboard stays in sync with the Kafka pipeline (Phase 1 keeps gateway state in-memory).

You should now see in the curator log:

```
digest emitted: in=500 selected=20 freshness=0.821 diversity=0.950 redundancy=0.000
```

And in the orchestrator log:

```
digest processed: events=20 freshness=0.821 diversity=0.950
```

## Step 5 — Inspect via the API

```bash
curl -s localhost:8090/api/v1/health | jq
curl -s "localhost:8090/api/v1/digest?strategy=4&diversity=80&k=10" | jq
curl -s localhost:8090/api/v1/audit | jq '.verified, (.records | length)'
```

You should see `"verified": true` and a positive record count.

## Step 6 — Run the dashboard

```bash
cd dashboard
npm install
npm run dev
```

Open http://localhost:5173. You should see:

- Live curated event feed
- Real-time freshness / diversity / redundancy scores
- Audit log panel with the hash chain visible
- Header pill showing "audit chain: valid"

Change the strategy and diversity sliders; the digest updates within
5 seconds.

## Step 7 — Watch the audit log catch tampering

Stop the API gateway. Edit the in-memory chain via a manual API call
or by killing/restarting (the in-memory chain resets on restart, so
this step is illustrative). In a production deployment with the
Postgres-backed audit chain (Phase 2), the chain-verifier process
flags tampering within one verification cycle and emits a critical
alert.

## What you just demonstrated

- **Hyper-scale CDC-style ingestion** via synthetic source and Kafka.
- **All six curation algorithms** from Buthalapalli & Vanga (2025),
  generalized to operational events.
- **AI decision support** with guardrails enforced in code.
- **Hash-chained audit log** with end-to-end verification.
- **Operator dashboard** with real-time score telemetry.

This is the full FluxLens pipeline. Adopting it for a real source
system is the next step — see
[`02-ingest-from-mysql.md`](./02-ingest-from-mysql.md) (when published)
or jump straight to the Helm chart at `deploy/helm/fluxlens/`.

## Next reading

- [`docs/architecture/`](../../ARCHITECTURE.md) — the deep architecture
- [`docs/adr/`](../adr/) — the decisions behind the design
- [`docs/domain-packs/`](../domain-packs/) — reference packs for
  clean-energy manufacturing, retail, federal research
- [`docs/national-interest/`](../national-interest/) — why FluxLens
  matters for U.S. critical sectors
- [`docs/compliance/`](../compliance/) — NIST AI RMF, 800-53, FedRAMP
  posture
