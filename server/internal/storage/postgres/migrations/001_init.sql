CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS workspaces (
    id UUID PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    archived_at TIMESTAMPTZ,
    archived_by TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS members (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    added_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (workspace_key, email)
);

CREATE TABLE IF NOT EXISTS auth_profiles (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    cache_ttl_seconds INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    secret_payload_encrypted TEXT NOT NULL DEFAULT '',
    has_secret BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS external_apis (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    request JSONB NOT NULL,
    params JSONB NOT NULL DEFAULT '[]'::jsonb,
    response_validation JSONB NOT NULL,
    secret_payload_encrypted TEXT NOT NULL DEFAULT '',
    has_secrets BOOLEAN NOT NULL DEFAULT FALSE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS features (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    value_type TEXT NOT NULL,
    default_value JSONB NOT NULL,
    active_from TIMESTAMPTZ,
    active_until TIMESTAMPTZ,
    environments TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    access_policy TEXT NOT NULL DEFAULT 'required',
    auth_profile_id UUID REFERENCES auth_profiles(id) ON DELETE SET NULL,
    input_contract JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    rollout_salt TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS feature_rules (
    id UUID PRIMARY KEY,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    priority INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    expression TEXT NOT NULL,
    value JSONB NOT NULL,
    requires_auth BOOLEAN NOT NULL DEFAULT FALSE,
    rollout_percentage INTEGER,
    source_bindings JSONB NOT NULL DEFAULT '{}'::jsonb,
    external_validation JSONB,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS feature_tags (
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    tag_key TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (feature_id, tag_key)
);

CREATE TABLE IF NOT EXISTS packs (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS pack_features (
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pack_id, feature_id)
);

CREATE TABLE IF NOT EXISTS pack_activations (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    activated_at TIMESTAMPTZ NOT NULL,
    activated_by TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (pack_id, target_type, target_id)
);

CREATE TABLE IF NOT EXISTS segments (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    record_key_path TEXT NOT NULL DEFAULT '',
    active_dataset_version TEXT NOT NULL DEFAULT '',
    preview_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_type TEXT NOT NULL DEFAULT '',
    record_count BIGINT NOT NULL DEFAULT 0,
    last_import_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    updated_by TEXT NOT NULL DEFAULT '',
    UNIQUE (workspace_key, key)
);

CREATE TABLE IF NOT EXISTS segment_records (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    segment_id UUID NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    dataset_version TEXT NOT NULL,
    record_key TEXT NOT NULL,
    attributes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (segment_id, dataset_version, record_key)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    name TEXT NOT NULL,
    hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by TEXT NOT NULL DEFAULT '',
    created_by_permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS evaluation_errors (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    feature_key TEXT NOT NULL,
    rule_id TEXT NOT NULL DEFAULT '',
    error_type TEXT NOT NULL,
    message TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT '',
    campus_id TEXT NOT NULL DEFAULT '',
    program_id TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS changelog (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_key TEXT NOT NULL,
    parent_key TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    field_changes JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    change_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    executed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS experiments (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variants JSONB NOT NULL,
    metrics JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL,
    winner_key TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS experiment_exposures (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    feature_id UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    variant_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (experiment_id, user_id)
);

CREATE TABLE IF NOT EXISTS experiment_conversions (
    id UUID PRIMARY KEY,
    workspace_key TEXT NOT NULL REFERENCES workspaces(key) ON DELETE CASCADE,
    experiment_id UUID NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    variant_key TEXT NOT NULL,
    metric_key TEXT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS tags_workspace_name_lower_uniq ON tags (workspace_key, lower(name));

CREATE INDEX IF NOT EXISTS members_workspace_role_idx ON members (workspace_key, role);
CREATE INDEX IF NOT EXISTS auth_profiles_workspace_updated_idx ON auth_profiles (workspace_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS external_apis_workspace_updated_idx ON external_apis (workspace_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS features_workspace_enabled_key_idx ON features (workspace_key, enabled, key);
CREATE INDEX IF NOT EXISTS features_workspace_updated_idx ON features (workspace_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS features_workspace_created_idx ON features (workspace_key, created_at DESC);
CREATE INDEX IF NOT EXISTS feature_rules_feature_priority_idx ON feature_rules (feature_id, priority ASC);
CREATE INDEX IF NOT EXISTS feature_tags_tag_key_idx ON feature_tags (tag_key);
CREATE INDEX IF NOT EXISTS packs_workspace_updated_idx ON packs (workspace_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS pack_features_feature_id_idx ON pack_features (feature_id);
CREATE INDEX IF NOT EXISTS pack_activations_target_idx ON pack_activations (workspace_key, target_type, target_id);
CREATE INDEX IF NOT EXISTS pack_activations_pack_expires_idx ON pack_activations (workspace_key, pack_id, expires_at);
CREATE INDEX IF NOT EXISTS segments_workspace_created_idx ON segments (workspace_key, created_at DESC);
CREATE INDEX IF NOT EXISTS segment_records_lookup_idx ON segment_records (segment_id, dataset_version, record_key);
CREATE INDEX IF NOT EXISTS segment_records_created_idx ON segment_records (segment_id, dataset_version, created_at DESC);
CREATE INDEX IF NOT EXISTS api_keys_workspace_type_idx ON api_keys (workspace_key, type);
CREATE INDEX IF NOT EXISTS api_keys_workspace_revoked_idx ON api_keys (workspace_key, revoked, type);
CREATE INDEX IF NOT EXISTS evaluation_errors_created_idx ON evaluation_errors (workspace_key, created_at DESC);
CREATE INDEX IF NOT EXISTS evaluation_errors_feature_created_idx ON evaluation_errors (workspace_key, feature_key, created_at DESC);
CREATE INDEX IF NOT EXISTS evaluation_errors_type_created_idx ON evaluation_errors (workspace_key, error_type, created_at DESC);
CREATE INDEX IF NOT EXISTS changelog_entity_created_idx ON changelog (workspace_key, entity_type, entity_key, created_at DESC);
CREATE INDEX IF NOT EXISTS changelog_actor_created_idx ON changelog (workspace_key, actor, created_at DESC);
CREATE INDEX IF NOT EXISTS changelog_created_idx ON changelog (workspace_key, created_at DESC);
CREATE INDEX IF NOT EXISTS changelog_action_created_idx ON changelog (workspace_key, entity_type, action, created_at DESC);
CREATE INDEX IF NOT EXISTS schedules_status_scheduled_idx ON schedules (status, scheduled_at ASC);
CREATE INDEX IF NOT EXISTS schedules_workspace_feature_scheduled_idx ON schedules (workspace_key, feature_id, scheduled_at ASC);
CREATE INDEX IF NOT EXISTS experiments_workspace_feature_status_idx ON experiments (workspace_key, feature_id, status);
CREATE INDEX IF NOT EXISTS experiments_feature_idx ON experiments (feature_id);
CREATE INDEX IF NOT EXISTS experiments_status_idx ON experiments (status);
CREATE INDEX IF NOT EXISTS experiments_workspace_updated_idx ON experiments (workspace_key, updated_at DESC);
CREATE INDEX IF NOT EXISTS experiment_exposures_variant_idx ON experiment_exposures (experiment_id, variant_key);
CREATE INDEX IF NOT EXISTS experiment_conversions_variant_idx ON experiment_conversions (experiment_id, metric_key, variant_key);
CREATE INDEX IF NOT EXISTS experiment_conversions_user_metric_idx ON experiment_conversions (experiment_id, user_id, metric_key);

CREATE INDEX IF NOT EXISTS features_key_trgm_idx ON features USING GIN (key gin_trgm_ops);
CREATE INDEX IF NOT EXISTS features_name_trgm_idx ON features USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS segments_key_trgm_idx ON segments USING GIN (key gin_trgm_ops);
CREATE INDEX IF NOT EXISTS tags_key_trgm_idx ON tags USING GIN (key gin_trgm_ops);
CREATE INDEX IF NOT EXISTS tags_name_trgm_idx ON tags USING GIN (name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS segment_records_record_key_trgm_idx ON segment_records USING GIN (record_key gin_trgm_ops);

INSERT INTO workspaces (
    id,
    key,
    name,
    description,
    metadata,
    created_at,
    updated_at,
    created_by,
    archived_by
) VALUES (
    '01956b3f-3a0d-7000-8000-000000000001',
    'default',
    'Default',
    'Default workspace',
    '{}'::jsonb,
    NOW(),
    NOW(),
    'system:migration',
    ''
) ON CONFLICT (key) DO NOTHING;
