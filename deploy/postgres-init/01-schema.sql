-- FluxLens local-development schema bootstrap.
-- Production schemas should be managed via migrations (e.g., golang-migrate).

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS fluxlens_events (
    event_id        TEXT        PRIMARY KEY,
    source_id       TEXT        NOT NULL,
    source_type     TEXT        NOT NULL,
    event_type      TEXT        NOT NULL,
    severity        TEXT        NOT NULL,
    timestamp_utc   TIMESTAMPTZ NOT NULL,
    ingested_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload         JSONB       NOT NULL,
    metadata        JSONB       NOT NULL,
    tenant_id       TEXT        NOT NULL DEFAULT 'default'
);

SELECT create_hypertable('fluxlens_events', 'timestamp_utc', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_events_source       ON fluxlens_events (source_id, timestamp_utc DESC);
CREATE INDEX IF NOT EXISTS idx_events_type         ON fluxlens_events (event_type, timestamp_utc DESC);
CREATE INDEX IF NOT EXISTS idx_events_severity     ON fluxlens_events (severity, timestamp_utc DESC);
CREATE INDEX IF NOT EXISTS idx_events_tenant       ON fluxlens_events (tenant_id, timestamp_utc DESC);

CREATE TABLE IF NOT EXISTS fluxlens_decisions (
    decision_id          TEXT        PRIMARY KEY,
    event_id             TEXT        NOT NULL,
    model_provider       TEXT        NOT NULL,
    model_id             TEXT        NOT NULL,
    prompt_hash          TEXT        NOT NULL,
    response             JSONB       NOT NULL,
    guardrails_status    TEXT        NOT NULL,
    operator_action      TEXT,
    operator_id          TEXT,
    operator_action_at   TIMESTAMPTZ,
    audit_chain_prev     TEXT        NOT NULL,
    audit_chain_hash     TEXT        NOT NULL,
    decided_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id            TEXT        NOT NULL DEFAULT 'default'
);

CREATE INDEX IF NOT EXISTS idx_decisions_event   ON fluxlens_decisions (event_id);
CREATE INDEX IF NOT EXISTS idx_decisions_tenant  ON fluxlens_decisions (tenant_id, decided_at DESC);

CREATE TABLE IF NOT EXISTS fluxlens_audit_log (
    sequence       BIGINT      PRIMARY KEY,
    timestamp_utc  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    kind           TEXT        NOT NULL,
    payload        JSONB       NOT NULL,
    prev_hash      TEXT        NOT NULL,
    hash           TEXT        NOT NULL,
    tenant_id      TEXT        NOT NULL DEFAULT 'default'
);

CREATE INDEX IF NOT EXISTS idx_audit_kind   ON fluxlens_audit_log (kind, timestamp_utc DESC);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON fluxlens_audit_log (tenant_id, timestamp_utc DESC);
