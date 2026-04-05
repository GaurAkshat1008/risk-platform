CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS rules (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    name       TEXT        NOT NULL,
    version    INT         NOT NULL DEFAULT 1,
    expression JSONB       NOT NULL,
    action     TEXT        NOT NULL DEFAULT 'flag',
    priority   INT         NOT NULL DEFAULT 0,
    enabled    BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_rules_tenant_id ON rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rules_tenant_enabled ON rules(tenant_id, enabled);

CREATE TABLE IF NOT EXISTS rule_versions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id    UUID        NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    version    INT         NOT NULL,
    expression JSONB       NOT NULL,
    action     TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rule_versions_rule_id ON rule_versions(rule_id);