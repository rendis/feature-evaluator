CREATE TABLE IF NOT EXISTS tiers (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    level INTEGER NOT NULL,
    color TEXT NOT NULL DEFAULT '#6B7280',
    icon TEXT NOT NULL DEFAULT 'builtin:star',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key),
    UNIQUE (workspace_key, level)
);

CREATE TABLE IF NOT EXISTS tier_icons (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, name)
);

ALTER TABLE packs ADD COLUMN IF NOT EXISTS tier_key TEXT;

ALTER TABLE features ADD COLUMN IF NOT EXISTS trial_until TIMESTAMPTZ;
ALTER TABLE features ADD COLUMN IF NOT EXISTS trial_value JSONB;

ALTER TABLE packs ADD COLUMN IF NOT EXISTS trial_until TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS pack_inheritance (
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    parent_pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pack_id, parent_pack_id),
    CHECK (pack_id != parent_pack_id)
);
