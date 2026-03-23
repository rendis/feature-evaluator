CREATE TABLE IF NOT EXISTS system_security_policies (
    singleton_key TEXT PRIMARY KEY DEFAULT 'global',
    cors_origins JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT NOT NULL DEFAULT '',
    CONSTRAINT system_security_policies_singleton CHECK (singleton_key = 'global')
);
