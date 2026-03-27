package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// FeatureRepo implements feature.Repository using PostgreSQL.
type FeatureRepo struct {
	client *Client
}

// NewFeatureRepo creates a new FeatureRepo.
func NewFeatureRepo(client *Client) *FeatureRepo {
	return &FeatureRepo{client: client}
}

// Create inserts a feature and its tag/rule relations.
func (r *FeatureRepo) Create(ctx context.Context, f *feature.Feature) error { //nolint:gocognit,funlen // multi-step feature creation
	if f.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		f.ID = id
	}
	f.WorkspaceKey = wsKey(ctx)

	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		authProfileID, err := r.resolveAuthProfileID(txCtx, f.AuthProfileKey)
		if err != nil {
			return err
		}

		defaultValueJSON, err := jsonBytes(f.DefaultValue, "null")
		if err != nil {
			return fmt.Errorf("marshal feature default value: %w", err)
		}
		inputContractJSON, err := jsonBytes(f.InputContract, "{}")
		if err != nil {
			return fmt.Errorf("marshal feature input contract: %w", err)
		}
		metadataJSON, err := jsonBytes(f.Metadata, "{}")
		if err != nil {
			return fmt.Errorf("marshal feature metadata: %w", err)
		}
		trialValueJSON, err := jsonBytes(f.TrialValue, "null")
		if err != nil {
			return fmt.Errorf("marshal feature trial value: %w", err)
		}

		_, err = r.client.db(txCtx).Exec(txCtx, `
			INSERT INTO features (
				id, workspace_key, key, name, description, enabled, eval_cache_enabled, eval_cache_ttl_seconds,
				value_type, default_value, active_from, active_until, environments, access_policy, auth_profile_id, input_contract,
				metadata, rollout_salt, created_at, updated_at, created_by, updated_by,
				trial_until, trial_value
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10::jsonb, $11, $12, $13, $14, $15, $16::jsonb,
				$17::jsonb, $18, $19, $20, $21, $22,
				$23, $24::jsonb
			)
		`,
			f.ID,
			f.WorkspaceKey,
			f.Key,
			f.Name,
			f.Description,
			f.Enabled,
			f.EvalCacheEnabled,
			f.EvalCacheTTLSeconds,
			f.ValueType,
			defaultValueJSON,
			f.ActiveFrom,
			f.ActiveUntil,
			f.Environments,
			f.AccessPolicy,
			authProfileID,
			inputContractJSON,
			metadataJSON,
			f.RolloutSalt,
			f.CreatedAt,
			f.UpdatedAt,
			f.CreatedBy,
			f.UpdatedBy,
			f.TrialUntil,
			trialValueJSON,
		)
		if isUniqueViolation(err) {
			return apierror.NewConflict(
				fmt.Sprintf("feature with key %q already exists", f.Key),
				"error.featureKeyExists",
			)
		}
		if err != nil {
			return fmt.Errorf("insert feature: %w", err)
		}

		if err := r.replaceFeatureTags(txCtx, f.ID, f.Tags); err != nil {
			return err
		}
		if err := r.replaceFeatureRules(txCtx, f.ID, f.Rules); err != nil {
			return err
		}

		return nil
	})
}

// GetByKey finds a feature by key.
func (r *FeatureRepo) GetByKey(ctx context.Context, key string) (*feature.Feature, error) {
	row := r.client.db(ctx).QueryRow(ctx, featureSelectQuery(`
		WHERE f.workspace_key = $1 AND f.key = $2
	`), wsKey(ctx), key)

	f, err := scanFeature(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("feature %q not found", key),
				"error.featureNotFound",
			)
		}
		return nil, fmt.Errorf("find feature: %w", err)
	}

	items := []feature.Feature{*f}
	if err := r.hydrateFeatures(ctx, items); err != nil {
		return nil, err
	}
	*f = items[0]

	return f, nil
}

// Update updates a feature and replaces tag mappings.
func (r *FeatureRepo) Update(ctx context.Context, f *feature.Feature) error {
	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		authProfileID, err := r.resolveAuthProfileID(txCtx, f.AuthProfileKey)
		if err != nil {
			return err
		}

		defaultValueJSON, err := jsonBytes(f.DefaultValue, "null")
		if err != nil {
			return fmt.Errorf("marshal feature default value: %w", err)
		}
		inputContractJSON, err := jsonBytes(f.InputContract, "{}")
		if err != nil {
			return fmt.Errorf("marshal feature input contract: %w", err)
		}
		metadataJSON, err := jsonBytes(f.Metadata, "{}")
		if err != nil {
			return fmt.Errorf("marshal feature metadata: %w", err)
		}
		trialValueJSON, err := jsonBytes(f.TrialValue, "null")
		if err != nil {
			return fmt.Errorf("marshal feature trial value: %w", err)
		}

		var featureID string
		tag, err := r.client.db(txCtx).Exec(txCtx, `
			UPDATE features
			SET name = $3, description = $4, enabled = $5, eval_cache_enabled = $6,
			    eval_cache_ttl_seconds = $7, value_type = $8, default_value = $9::jsonb,
			    active_from = $10, active_until = $11, environments = $12, access_policy = $13,
			    auth_profile_id = $14, input_contract = $15::jsonb, metadata = $16::jsonb,
			    updated_at = $17, updated_by = $18,
			    trial_until = $19, trial_value = $20::jsonb
			WHERE workspace_key = $1 AND key = $2
		`,
			wsKey(txCtx),
			f.Key,
			f.Name,
			f.Description,
			f.Enabled,
			f.EvalCacheEnabled,
			f.EvalCacheTTLSeconds,
			f.ValueType,
			defaultValueJSON,
			f.ActiveFrom,
			f.ActiveUntil,
			f.Environments,
			f.AccessPolicy,
			authProfileID,
			inputContractJSON,
			metadataJSON,
			f.UpdatedAt,
			f.UpdatedBy,
			f.TrialUntil,
			trialValueJSON,
		)
		if err != nil {
			return fmt.Errorf("update feature: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apierror.NewNotFound(
				fmt.Sprintf("feature %q not found", f.Key),
				"error.featureNotFound",
			)
		}
		if err := r.client.db(txCtx).QueryRow(txCtx, `
			SELECT id
			FROM features
			WHERE workspace_key = $1 AND key = $2
		`, wsKey(txCtx), f.Key).Scan(&featureID); err != nil {
			return fmt.Errorf("find updated feature id: %w", err)
		}

		f.ID = featureID
		return r.replaceFeatureTags(txCtx, f.ID, f.Tags)
	})
}

// Delete removes a feature by key.
func (r *FeatureRepo) Delete(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM features
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("feature %q not found", key),
			"error.featureNotFound",
		)
	}

	return nil
}

// List returns paginated features with filters.
func (r *FeatureRepo) List(ctx context.Context, params feature.ListParams) (*feature.ListResult, error) {
	if params.View == feature.ListViewSummary {
		return r.listSummary(ctx, params)
	}

	predicate, args := buildFeaturePredicate(ctx, params)

	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM features f WHERE `+predicate, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count features: %w", err)
	}

	sortField := "f.updated_at"
	switch params.SortBy {
	case "key":
		sortField = "f.key"
	case "name":
		sortField = "f.name"
	case "createdAt":
		sortField = "f.created_at"
	case "updatedAt":
		sortField = "f.updated_at"
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	pageArgs := append(args, params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.client.db(ctx).Query(ctx, featureSelectQuery(`
		WHERE `+predicate+`
		ORDER BY `+sortField+` `+sortOrder+`
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)), pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	defer rows.Close()

	items := make([]feature.Feature, 0)
	for rows.Next() {
		item, err := scanFeature(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feature: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate features: %w", err)
	}

	if err := r.hydrateFeatures(ctx, items); err != nil {
		return nil, err
	}

	return &feature.ListResult{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

func (r *FeatureRepo) listSummary(ctx context.Context, params feature.ListParams) (*feature.ListResult, error) {
	predicate, args := buildFeaturePredicate(ctx, params)

	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM features f WHERE `+predicate, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count feature summaries: %w", err)
	}

	sortField := "f.updated_at"
	switch params.SortBy {
	case "key":
		sortField = "f.key"
	case "name":
		sortField = "f.name"
	case "createdAt":
		sortField = "f.created_at"
	case "updatedAt":
		sortField = "f.updated_at"
	}
	sortOrder := "DESC"
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	pageArgs := append(args, params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.client.db(ctx).Query(ctx, featureSummarySelectQuery(`
		WHERE `+predicate+`
		ORDER BY `+sortField+` `+sortOrder+`
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)), pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("list feature summaries: %w", err)
	}
	defer rows.Close()

	items := make([]feature.Feature, 0)
	for rows.Next() {
		item, err := scanFeatureSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feature summary: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature summaries: %w", err)
	}

	if err := r.hydrateFeatureSummary(ctx, items); err != nil {
		return nil, err
	}

	return &feature.ListResult{
		Data:       items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

// ListEnabled returns all enabled features.
func (r *FeatureRepo) ListEnabled(ctx context.Context) ([]feature.Feature, error) {
	rows, err := r.client.db(ctx).Query(ctx, featureSelectQuery(`
		WHERE f.workspace_key = $1 AND f.enabled = TRUE
		ORDER BY f.key ASC
	`), wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list enabled features: %w", err)
	}
	defer rows.Close()

	items := make([]feature.Feature, 0)
	for rows.Next() {
		item, err := scanFeature(rows)
		if err != nil {
			return nil, fmt.Errorf("scan enabled feature: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled features: %w", err)
	}
	if err := r.hydrateFeatures(ctx, items); err != nil {
		return nil, err
	}

	if items == nil {
		return []feature.Feature{}, nil
	}
	return items, nil
}

// Toggle updates the enabled state of a feature.
func (r *FeatureRepo) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE features
		SET enabled = $3, updated_by = $4, updated_at = NOW()
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key, enabled, updatedBy)
	if err != nil {
		return fmt.Errorf("toggle feature: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("feature %q not found", key),
			"error.featureNotFound",
		)
	}

	return nil
}

// AddRule appends a rule to a feature.
func (r *FeatureRepo) AddRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		featureID, err := r.featureIDByKey(txCtx, featureKey)
		if err != nil {
			return err
		}
		if err := r.insertRule(txCtx, featureID, rule); err != nil {
			return err
		}
		return r.touchFeature(txCtx, featureID)
	})
}

// UpdateRule updates a rule within a feature.
func (r *FeatureRepo) UpdateRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		featureID, err := r.featureIDByKey(txCtx, featureKey)
		if err != nil {
			return err
		}

		valueJSON, err := jsonBytes(rule.Value, "null")
		if err != nil {
			return fmt.Errorf("marshal rule value: %w", err)
		}
		sourceBindingsJSON, err := jsonBytes(rule.SourceBindings, "{}")
		if err != nil {
			return fmt.Errorf("marshal rule source bindings: %w", err)
		}
		externalAPIBindingsJSON, err := jsonBytes(rule.ExternalAPIBindings, "[]")
		if err != nil {
			return fmt.Errorf("marshal rule external api bindings: %w", err)
		}
		metadataJSON, err := jsonBytes(rule.Metadata, "{}")
		if err != nil {
			return fmt.Errorf("marshal rule metadata: %w", err)
		}

		tag, err := r.client.db(txCtx).Exec(txCtx, `
			UPDATE feature_rules
			SET name = $3, priority = $4, enabled = $5, expression = $6, value = $7::jsonb,
			    rollout_percentage = $8, source_bindings = $9::jsonb,
			    external_validation = $10::jsonb, metadata = $11::jsonb, updated_at = $12
			WHERE feature_id = $1 AND id = $2
		`,
			featureID,
			rule.ID,
			rule.Name,
			rule.Priority,
			rule.Enabled,
			rule.Expression,
			valueJSON,
			rule.RolloutPercentage,
			sourceBindingsJSON,
			externalAPIBindingsJSON,
			metadataJSON,
			rule.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("update rule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apierror.NewNotFound("rule not found", "error.ruleNotFound")
		}

		return r.touchFeature(txCtx, featureID)
	})
}

// DeleteRule removes a rule from a feature.
func (r *FeatureRepo) DeleteRule(ctx context.Context, featureKey string, ruleID string) error {
	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		featureID, err := r.featureIDByKey(txCtx, featureKey)
		if err != nil {
			return err
		}

		if _, err := r.client.db(txCtx).Exec(txCtx, `
			DELETE FROM feature_rules
			WHERE feature_id = $1 AND id = $2
		`, featureID, ruleID); err != nil {
			return fmt.Errorf("delete rule: %w", err)
		}

		return r.touchFeature(txCtx, featureID)
	})
}

// ReorderRules reorders a feature's rules using the provided rule list.
func (r *FeatureRepo) ReorderRules(ctx context.Context, featureKey string, ruleIDs []string) error {
	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		featureID, err := r.featureIDByKey(txCtx, featureKey)
		if err != nil {
			return err
		}

		rows, err := r.client.db(txCtx).Query(txCtx, `
			SELECT id, feature_id::text, name, priority, enabled, expression, value, rollout_percentage,
			       source_bindings, external_validation, metadata, created_at, updated_at
			FROM feature_rules
			WHERE feature_id = $1
		`, featureID)
		if err != nil {
			return fmt.Errorf("load rules for reorder: %w", err)
		}
		defer rows.Close()

		rules := make(map[string]feature.Rule)
		for rows.Next() {
			rule, featureIDValue, err := scanRule(rows)
			if err != nil {
				return fmt.Errorf("scan rule for reorder: %w", err)
			}
			_ = featureIDValue
			rules[rule.ID] = *rule
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate rules for reorder: %w", err)
		}

		reordered := make([]feature.Rule, 0, len(ruleIDs))
		for idx, id := range ruleIDs {
			rule, ok := rules[id]
			if !ok {
				return apierror.NewBadRequest(
					fmt.Sprintf("rule %q not found in feature", id),
					"error.ruleNotFound",
				)
			}
			rule.Priority = idx + 1
			reordered = append(reordered, rule)
		}

		if err := r.replaceFeatureRules(txCtx, featureID.String(), reordered); err != nil {
			return err
		}

		return r.touchFeature(txCtx, featureID)
	})
}

func buildFeaturePredicate(ctx context.Context, params feature.ListParams) (string, []any) {
	where := []string{"f.workspace_key = $1"}
	args := []any{wsKey(ctx)}
	arg := 2

	search := sanitizeSearch(params.Search)
	if search != "" {
		where = append(where, fmt.Sprintf("f.key ILIKE '%%' || $%d || '%%'", arg))
		args = append(args, search)
		arg++
	}
	if params.Enabled != nil {
		where = append(where, fmt.Sprintf("f.enabled = $%d", arg))
		args = append(args, *params.Enabled)
		arg++
	}
	if params.ValueType != nil {
		where = append(where, fmt.Sprintf("f.value_type = $%d", arg))
		args = append(args, *params.ValueType)
		arg++
	}
	if params.Environment != "" {
		where = append(where, fmt.Sprintf("$%d = ANY(f.environments)", arg))
		args = append(args, params.Environment)
		arg++
	}
	if len(params.Tags) > 0 {
		where = append(where, fmt.Sprintf(`
			f.id IN (
				SELECT ft.feature_id
				FROM feature_tags ft
				JOIN features tf ON tf.id = ft.feature_id
				WHERE tf.workspace_key = $1
				  AND ft.tag_key = ANY($%d)
				GROUP BY ft.feature_id
				HAVING COUNT(DISTINCT ft.tag_key) = $%d
			)
		`, arg, arg+1))
		args = append(args, params.Tags, len(params.Tags))
	}

	return strings.Join(where, " AND "), args
}

func featureSelectQuery(suffix string) string {
	return `
		SELECT f.id, f.workspace_key, f.key, f.name, f.description, f.enabled, f.eval_cache_enabled,
		       f.eval_cache_ttl_seconds, f.value_type, f.default_value, f.active_from, f.active_until, f.environments, f.access_policy,
		       COALESCE(ap.key, '') AS auth_profile_key, f.input_contract, f.metadata, f.rollout_salt,
		       f.created_at, f.updated_at, f.created_by, f.updated_by,
		       f.trial_until, f.trial_value
		FROM features f
		LEFT JOIN auth_profiles ap ON ap.id = f.auth_profile_id
	` + suffix
}

func featureSummarySelectQuery(suffix string) string {
	return `
		SELECT f.id, f.workspace_key, f.key, f.name, f.description, f.enabled, f.eval_cache_enabled,
		       f.eval_cache_ttl_seconds, f.value_type,
		       f.active_from, f.active_until, f.environments, f.access_policy,
		       COALESCE(ap.key, '') AS auth_profile_key,
		       f.created_at, f.updated_at, f.created_by, f.updated_by,
		       f.trial_until
		FROM features f
		LEFT JOIN auth_profiles ap ON ap.id = f.auth_profile_id
	` + suffix
}

func (r *FeatureRepo) resolveAuthProfileID(ctx context.Context, key string) (*uuid.UUID, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}

	var authProfileID uuid.UUID
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM auth_profiles
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key).Scan(&authProfileID); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("auth profile %q not found", key),
				"error.authProfileNotFound",
			)
		}
		return nil, fmt.Errorf("resolve auth profile id: %w", err)
	}

	return &authProfileID, nil
}

func (r *FeatureRepo) featureIDByKey(ctx context.Context, key string) (uuid.UUID, error) {
	var featureID uuid.UUID
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM features
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key).Scan(&featureID); err != nil {
		if isNoRows(err) {
			return uuid.Nil, apierror.NewNotFound(
				fmt.Sprintf("feature %q not found", key),
				"error.featureNotFound",
			)
		}
		return uuid.Nil, fmt.Errorf("resolve feature id: %w", err)
	}

	return featureID, nil
}

func (r *FeatureRepo) replaceFeatureTags(ctx context.Context, featureID string, tagKeys []string) error {
	if _, err := r.client.db(ctx).Exec(ctx, `DELETE FROM feature_tags WHERE feature_id = $1`, featureID); err != nil {
		return fmt.Errorf("delete feature tags: %w", err)
	}
	for idx, tagKey := range tagKeys {
		if _, err := r.client.db(ctx).Exec(ctx, `
			INSERT INTO feature_tags (feature_id, tag_key, position)
			VALUES ($1, $2, $3)
		`, featureID, tagKey, idx); err != nil {
			return fmt.Errorf("insert feature tag: %w", err)
		}
	}

	return nil
}

func (r *FeatureRepo) replaceFeatureRules(ctx context.Context, featureID string, rules []feature.Rule) error {
	if _, err := r.client.db(ctx).Exec(ctx, `DELETE FROM feature_rules WHERE feature_id = $1`, featureID); err != nil {
		return fmt.Errorf("delete existing feature rules: %w", err)
	}
	parsedFeatureID, err := parseUUID(featureID)
	if err != nil {
		return fmt.Errorf("parse feature id for rules: %w", err)
	}
	for _, rule := range rules {
		rule := rule
		if err := r.insertRule(ctx, parsedFeatureID, &rule); err != nil {
			return err
		}
	}
	return nil
}

func (r *FeatureRepo) insertRule(ctx context.Context, featureID uuid.UUID, rule *feature.Rule) error {
	if rule.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		rule.ID = id
	}

	valueJSON, err := jsonBytes(rule.Value, "null")
	if err != nil {
		return fmt.Errorf("marshal rule value: %w", err)
	}
	sourceBindingsJSON, err := jsonBytes(rule.SourceBindings, "{}")
	if err != nil {
		return fmt.Errorf("marshal rule source bindings: %w", err)
	}
	externalAPIBindingsJSON, err := jsonBytes(rule.ExternalAPIBindings, "[]")
	if err != nil {
		return fmt.Errorf("marshal rule external api bindings: %w", err)
	}
	metadataJSON, err := jsonBytes(rule.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal rule metadata: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO feature_rules (
			id, feature_id, name, priority, enabled, expression, value,
			rollout_percentage, source_bindings, external_validation, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::jsonb,
			$8, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13
		)
	`,
		rule.ID,
		featureID,
		rule.Name,
		rule.Priority,
		rule.Enabled,
		rule.Expression,
		valueJSON,
		rule.RolloutPercentage,
		sourceBindingsJSON,
		externalAPIBindingsJSON,
		metadataJSON,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert feature rule: %w", err)
	}

	return nil
}

func (r *FeatureRepo) touchFeature(ctx context.Context, featureID uuid.UUID) error {
	if _, err := r.client.db(ctx).Exec(ctx, `
		UPDATE features
		SET updated_at = NOW()
		WHERE id = $1
	`, featureID); err != nil {
		return fmt.Errorf("touch feature: %w", err)
	}

	return nil
}

func (r *FeatureRepo) hydrateFeatures(ctx context.Context, items []feature.Feature) error {
	if len(items) == 0 {
		return nil
	}

	featureIDs := make([]string, 0, len(items))
	indexByID := make(map[string]int, len(items))
	for i := range items {
		featureIDs = append(featureIDs, items[i].ID)
		indexByID[items[i].ID] = i
	}

	tagMap, err := r.loadFeatureTags(ctx, featureIDs)
	if err != nil {
		return err
	}
	ruleMap, err := r.loadFeatureRules(ctx, featureIDs)
	if err != nil {
		return err
	}

	for id, idx := range indexByID {
		items[idx].Tags = tagMap[id]
		items[idx].Rules = ruleMap[id]
	}

	return nil
}

func (r *FeatureRepo) hydrateFeatureSummary(ctx context.Context, items []feature.Feature) error {
	if len(items) == 0 {
		return nil
	}

	featureIDs := make([]string, 0, len(items))
	indexByID := make(map[string]int, len(items))
	for i := range items {
		featureIDs = append(featureIDs, items[i].ID)
		indexByID[items[i].ID] = i
	}

	tagMap, err := r.loadFeatureTags(ctx, featureIDs)
	if err != nil {
		return err
	}
	ruleCountMap, err := r.loadFeatureRuleCounts(ctx, featureIDs)
	if err != nil {
		return err
	}
	packCountMap, err := r.loadFeaturePackCounts(ctx, featureIDs)
	if err != nil {
		return err
	}

	for id, idx := range indexByID {
		items[idx].Tags = tagMap[id]
		items[idx].RuleCount = ruleCountMap[id]
		items[idx].PackCount = packCountMap[id]
		items[idx].Rules = []feature.Rule{}
	}

	return nil
}

func (r *FeatureRepo) loadFeatureTags(ctx context.Context, featureIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(featureIDs))
	if len(featureIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(featureIDs)
	if err != nil {
		return nil, fmt.Errorf("parse feature ids for tags: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT feature_id::text, tag_key
		FROM feature_tags
		WHERE feature_id = ANY($1)
		ORDER BY position ASC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("load feature tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var featureID string
		var tagKey string
		if err := rows.Scan(&featureID, &tagKey); err != nil {
			return nil, fmt.Errorf("scan feature tag: %w", err)
		}
		result[featureID] = append(result[featureID], tagKey)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature tags: %w", err)
	}

	for _, id := range featureIDs {
		if _, ok := result[id]; !ok {
			result[id] = []string{}
		}
	}

	return result, nil
}

func (r *FeatureRepo) loadFeatureRules(ctx context.Context, featureIDs []string) (map[string][]feature.Rule, error) {
	result := make(map[string][]feature.Rule, len(featureIDs))
	if len(featureIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(featureIDs)
	if err != nil {
		return nil, fmt.Errorf("parse feature ids for rules: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, feature_id::text, name, priority, enabled, expression, value,
		       rollout_percentage, source_bindings, external_validation,
		       metadata, created_at, updated_at
		FROM feature_rules
		WHERE feature_id = ANY($1)
		ORDER BY feature_id ASC, priority ASC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("load feature rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rule, featureID, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feature rule: %w", err)
		}
		result[featureID] = append(result[featureID], *rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature rules: %w", err)
	}

	for _, id := range featureIDs {
		if _, ok := result[id]; !ok {
			result[id] = []feature.Rule{}
		}
	}

	return result, nil
}

func (r *FeatureRepo) loadFeatureRuleCounts(ctx context.Context, featureIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(featureIDs))
	if len(featureIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(featureIDs)
	if err != nil {
		return nil, fmt.Errorf("parse feature ids for rule counts: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT feature_id::text, COUNT(*)::int
		FROM feature_rules
		WHERE feature_id = ANY($1)
		GROUP BY feature_id
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("load feature rule counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var featureID string
		var count int
		if err := rows.Scan(&featureID, &count); err != nil {
			return nil, fmt.Errorf("scan feature rule count: %w", err)
		}
		result[featureID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature rule counts: %w", err)
	}

	for _, id := range featureIDs {
		if _, ok := result[id]; !ok {
			result[id] = 0
		}
	}

	return result, nil
}

func (r *FeatureRepo) loadFeaturePackCounts(ctx context.Context, featureIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(featureIDs))
	if len(featureIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(featureIDs)
	if err != nil {
		return nil, fmt.Errorf("parse feature ids for pack counts: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT pf.feature_id::text, COUNT(*)::int
		FROM pack_features pf
		JOIN packs p ON p.id = pf.pack_id
		WHERE p.workspace_key = $1 AND pf.feature_id = ANY($2)
		GROUP BY pf.feature_id
	`, wsKey(ctx), ids)
	if err != nil {
		return nil, fmt.Errorf("load feature pack counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var featureID string
		var count int
		if err := rows.Scan(&featureID, &count); err != nil {
			return nil, fmt.Errorf("scan feature pack count: %w", err)
		}
		result[featureID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feature pack counts: %w", err)
	}

	for _, id := range featureIDs {
		if _, ok := result[id]; !ok {
			result[id] = 0
		}
	}

	return result, nil
}

type featureScanner interface {
	Scan(dest ...any) error
}

func scanFeature(scanner featureScanner) (*feature.Feature, error) {
	var f feature.Feature
	var defaultValueJSON []byte
	var inputContractJSON []byte
	var metadataJSON []byte
	var trialValueJSON []byte
	if err := scanner.Scan(
		&f.ID,
		&f.WorkspaceKey,
		&f.Key,
		&f.Name,
		&f.Description,
		&f.Enabled,
		&f.EvalCacheEnabled,
		&f.EvalCacheTTLSeconds,
		&f.ValueType,
		&defaultValueJSON,
		&f.ActiveFrom,
		&f.ActiveUntil,
		&f.Environments,
		&f.AccessPolicy,
		&f.AuthProfileKey,
		&inputContractJSON,
		&metadataJSON,
		&f.RolloutSalt,
		&f.CreatedAt,
		&f.UpdatedAt,
		&f.CreatedBy,
		&f.UpdatedBy,
		&f.TrialUntil,
		&trialValueJSON,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(defaultValueJSON, &f.DefaultValue); err != nil {
		return nil, err
	}
	if err := decodeJSON(inputContractJSON, &f.InputContract); err != nil {
		return nil, err
	}
	if err := decodeJSON(metadataJSON, &f.Metadata); err != nil {
		return nil, err
	}
	if err := decodeJSON(trialValueJSON, &f.TrialValue); err != nil {
		return nil, err
	}

	return &f, nil
}

func scanFeatureSummary(scanner featureScanner) (*feature.Feature, error) {
	var f feature.Feature
	if err := scanner.Scan(
		&f.ID,
		&f.WorkspaceKey,
		&f.Key,
		&f.Name,
		&f.Description,
		&f.Enabled,
		&f.EvalCacheEnabled,
		&f.EvalCacheTTLSeconds,
		&f.ValueType,
		&f.ActiveFrom,
		&f.ActiveUntil,
		&f.Environments,
		&f.AccessPolicy,
		&f.AuthProfileKey,
		&f.CreatedAt,
		&f.UpdatedAt,
		&f.CreatedBy,
		&f.UpdatedBy,
		&f.TrialUntil,
	); err != nil {
		return nil, err
	}

	return &f, nil
}

type ruleScanner interface {
	Scan(dest ...any) error
}

func scanRule(scanner ruleScanner) (*feature.Rule, string, error) {
	var rule feature.Rule
	var featureID string
	var valueJSON []byte
	var sourceBindingsJSON []byte
	var externalAPIBindingsJSON []byte
	var metadataJSON []byte
	var rolloutPercentage *int
	if err := scanner.Scan(
		&rule.ID,
		&featureID,
		&rule.Name,
		&rule.Priority,
		&rule.Enabled,
		&rule.Expression,
		&valueJSON,
		&rolloutPercentage,
		&sourceBindingsJSON,
		&externalAPIBindingsJSON,
		&metadataJSON,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		return nil, "", err
	}
	rule.RolloutPercentage = rolloutPercentage
	if err := decodeJSON(valueJSON, &rule.Value); err != nil {
		return nil, "", err
	}
	if err := decodeJSON(sourceBindingsJSON, &rule.SourceBindings); err != nil {
		return nil, "", err
	}
	if len(externalAPIBindingsJSON) > 0 && string(externalAPIBindingsJSON) != "null" && string(externalAPIBindingsJSON) != "[]" {
		if err := decodeJSON(externalAPIBindingsJSON, &rule.ExternalAPIBindings); err != nil {
			return nil, "", err
		}
	}
	if err := decodeJSON(metadataJSON, &rule.Metadata); err != nil {
		return nil, "", err
	}

	return &rule, featureID, nil
}
