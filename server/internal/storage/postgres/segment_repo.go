package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// SegmentRepo implements segment.Repository using PostgreSQL.
type SegmentRepo struct {
	client *Client
}

// NewSegmentRepo creates a new SegmentRepo.
func NewSegmentRepo(client *Client) *SegmentRepo {
	return &SegmentRepo{client: client}
}

// Create inserts a new segment.
func (r *SegmentRepo) Create(ctx context.Context, seg *segment.Segment) error {
	if seg.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		seg.ID = id
	}
	seg.WorkspaceKey = wsKey(ctx)

	metadataJSON, err := jsonBytes(seg.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal segment metadata: %w", err)
	}
	schemaJSON, err := jsonBytes(seg.Schema, "{}")
	if err != nil {
		return fmt.Errorf("marshal segment schema: %w", err)
	}
	previewFieldsJSON, err := jsonBytes(seg.PreviewFields, "[]")
	if err != nil {
		return fmt.Errorf("marshal segment preview fields: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO segments (
			id, workspace_key, key, name, description, metadata, schema, record_key_path,
			active_dataset_version, preview_fields, source_type, record_count,
			membership_cache_enabled, membership_cache_ttl_seconds,
			record_cache_enabled, record_cache_ttl_seconds,
			last_import_at, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8,
			$9, $10::jsonb, $11, $12, $13, $14,
			$15, $16,
			$17, $18, $19, $20, $21
		)
	`,
		seg.ID,
		seg.WorkspaceKey,
		seg.Key,
		seg.Name,
		seg.Description,
		metadataJSON,
		schemaJSON,
		seg.RecordKeyPath,
		seg.ActiveDatasetVersion,
		previewFieldsJSON,
		seg.SourceType,
		seg.RecordCount,
		seg.MembershipCacheEnabled,
		seg.MembershipCacheTTLSeconds,
		seg.RecordCacheEnabled,
		seg.RecordCacheTTLSeconds,
		seg.LastImportAt,
		seg.CreatedAt,
		seg.UpdatedAt,
		seg.CreatedBy,
		seg.UpdatedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("segment with key %q already exists", seg.Key),
			"error.segmentKeyExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert segment: %w", err)
	}

	return nil
}

// GetByKey finds a segment by key.
func (r *SegmentRepo) GetByKey(ctx context.Context, key string) (*segment.Segment, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, description, metadata, schema, record_key_path,
		       active_dataset_version, preview_fields, source_type, record_count,
		       membership_cache_enabled, membership_cache_ttl_seconds,
		       record_cache_enabled, record_cache_ttl_seconds,
		       last_import_at, created_at, updated_at, created_by, updated_by
		FROM segments
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	seg, err := scanSegment(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("segment %q not found", key),
				"error.segmentNotFound",
			)
		}
		return nil, fmt.Errorf("find segment: %w", err)
	}

	return seg, nil
}

// Update updates mutable segment fields.
func (r *SegmentRepo) Update(ctx context.Context, seg *segment.Segment) error {
	metadataJSON, err := jsonBytes(seg.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal segment metadata: %w", err)
	}
	schemaJSON, err := jsonBytes(seg.Schema, "{}")
	if err != nil {
		return fmt.Errorf("marshal segment schema: %w", err)
	}
	previewFieldsJSON, err := jsonBytes(seg.PreviewFields, "[]")
	if err != nil {
		return fmt.Errorf("marshal segment preview fields: %w", err)
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE segments
		SET name = $3, description = $4, metadata = $5::jsonb, schema = $6::jsonb,
		    record_key_path = $7, active_dataset_version = $8, preview_fields = $9::jsonb,
		    source_type = $10, record_count = $11,
		    membership_cache_enabled = $12, membership_cache_ttl_seconds = $13,
		    record_cache_enabled = $14, record_cache_ttl_seconds = $15,
		    last_import_at = $16, updated_at = $17, updated_by = $18
		WHERE workspace_key = $1 AND key = $2
	`,
		wsKey(ctx),
		seg.Key,
		seg.Name,
		seg.Description,
		metadataJSON,
		schemaJSON,
		seg.RecordKeyPath,
		seg.ActiveDatasetVersion,
		previewFieldsJSON,
		seg.SourceType,
		seg.RecordCount,
		seg.MembershipCacheEnabled,
		seg.MembershipCacheTTLSeconds,
		seg.RecordCacheEnabled,
		seg.RecordCacheTTLSeconds,
		seg.LastImportAt,
		seg.UpdatedAt,
		seg.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("update segment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("segment %q not found", seg.Key),
			"error.segmentNotFound",
		)
	}

	return nil
}

// Delete removes a segment by key.
func (r *SegmentRepo) Delete(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM segments
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete segment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("segment %q not found", key),
			"error.segmentNotFound",
		)
	}

	return nil
}

// List returns paginated segments.
func (r *SegmentRepo) List(ctx context.Context, params segment.ListParams) (*segment.ListResult, error) {
	search := sanitizeSearch(params.Search)

	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM segments
		WHERE workspace_key = $1
		  AND ($2 = '' OR key ILIKE '%' || $2 || '%')
	`, wsKey(ctx), search).Scan(&total); err != nil {
		return nil, fmt.Errorf("count segments: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, description, metadata, schema, record_key_path,
		       active_dataset_version, preview_fields, source_type, record_count,
		       membership_cache_enabled, membership_cache_ttl_seconds,
		       record_cache_enabled, record_cache_ttl_seconds,
		       last_import_at, created_at, updated_at, created_by, updated_by
		FROM segments
		WHERE workspace_key = $1
		  AND ($2 = '' OR key ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, wsKey(ctx), search, params.PageSize, (params.Page-1)*params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list segments: %w", err)
	}
	defer rows.Close()

	segments := make([]segment.Segment, 0)
	for rows.Next() {
		item, err := scanSegment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan segment: %w", err)
		}
		segments = append(segments, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate segments: %w", err)
	}

	return &segment.ListResult{
		Data:       segments,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

type segmentScanner interface {
	Scan(dest ...any) error
}

func scanSegment(scanner segmentScanner) (*segment.Segment, error) {
	var seg segment.Segment
	var metadataJSON []byte
	var schemaJSON []byte
	var previewFieldsJSON []byte
	if err := scanner.Scan(
		&seg.ID,
		&seg.WorkspaceKey,
		&seg.Key,
		&seg.Name,
		&seg.Description,
		&metadataJSON,
		&schemaJSON,
		&seg.RecordKeyPath,
		&seg.ActiveDatasetVersion,
		&previewFieldsJSON,
		&seg.SourceType,
		&seg.RecordCount,
		&seg.MembershipCacheEnabled,
		&seg.MembershipCacheTTLSeconds,
		&seg.RecordCacheEnabled,
		&seg.RecordCacheTTLSeconds,
		&seg.LastImportAt,
		&seg.CreatedAt,
		&seg.UpdatedAt,
		&seg.CreatedBy,
		&seg.UpdatedBy,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(metadataJSON, &seg.Metadata); err != nil {
		return nil, err
	}
	if err := decodeJSON(schemaJSON, &seg.Schema); err != nil {
		return nil, err
	}
	if err := decodeJSON(previewFieldsJSON, &seg.PreviewFields); err != nil {
		return nil, err
	}

	return &seg, nil
}
