CREATE TABLE IF NOT EXISTS decisions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_event_id TEXT        NOT NULL,
    tenant_id        TEXT        NOT NULL,
    outcome          TEXT        NOT NULL CHECK (outcome IN ('approve', 'flag', 'review', 'block')),
    reason_codes     TEXT[]      NOT NULL DEFAULT '{}',
    confidence_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    rule_results     JSONB       NOT NULL DEFAULT '[]',
    latency_ms       BIGINT      NOT NULL DEFAULT 0,
    overridden       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (payment_event_id, tenant_id)
);

CREATE TABLE IF NOT EXISTS decision_overrides (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id      UUID        NOT NULL REFERENCES decisions(id),
    analyst_id       TEXT        NOT NULL,
    previous_outcome TEXT        NOT NULL,
    new_outcome      TEXT        NOT NULL,
    reason           TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_decisions_tenant_id    ON decisions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_decisions_outcome      ON decisions (tenant_id, outcome);
CREATE INDEX IF NOT EXISTS idx_decisions_created_at   ON decisions (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_overrides_decision_id  ON decision_overrides (decision_id);
