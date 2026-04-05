-- Explanation Service: initial schema
-- Stores cached explanations for decisions.
-- decision_id is UNIQUE: each decision has exactly one explanation record.

CREATE TABLE IF NOT EXISTS explanations (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id         TEXT        NOT NULL,
    tenant_id           TEXT        NOT NULL,
    payment_event_id    TEXT        NOT NULL DEFAULT '',
    outcome             TEXT        NOT NULL,
    confidence_score    DOUBLE PRECISION NOT NULL DEFAULT 0,
    rule_contributions  JSONB       NOT NULL DEFAULT '[]',
    feature_values      JSONB       NOT NULL DEFAULT '[]',
    narrative           TEXT        NOT NULL DEFAULT '',
    policy_version      TEXT        NOT NULL DEFAULT '',
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT explanations_decision_id_unique UNIQUE (decision_id),
    CONSTRAINT explanations_payment_event_id_unique UNIQUE (payment_event_id)
);

CREATE INDEX IF NOT EXISTS idx_explanations_tenant_id       ON explanations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_explanations_generated_at    ON explanations (generated_at DESC);
