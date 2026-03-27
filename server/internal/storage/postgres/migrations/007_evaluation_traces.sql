CREATE TABLE IF NOT EXISTS evaluation_traces (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    rule_id TEXT NOT NULL DEFAULT '',
    used_redis BOOLEAN NOT NULL DEFAULT FALSE,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    result_reason TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    trace JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS evaluation_traces_feature_created_idx ON evaluation_traces (workspace_key, feature_key, created_at DESC);
CREATE INDEX IF NOT EXISTS evaluation_traces_rule_created_idx ON evaluation_traces (workspace_key, rule_id, created_at DESC);
