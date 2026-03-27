ALTER TABLE evaluation_traces
    ADD COLUMN IF NOT EXISTS cache_status TEXT NOT NULL DEFAULT 'computed';

CREATE INDEX IF NOT EXISTS evaluation_traces_cache_status_idx
    ON evaluation_traces (workspace_key, feature_key, cache_status, created_at DESC);
