package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// SegmentRecordRepo implements segment.RecordRepository using PostgreSQL.
type SegmentRecordRepo struct {
	client *Client
}

// NewSegmentRecordRepo creates a new SegmentRecordRepo.
func NewSegmentRecordRepo(client *Client) *SegmentRecordRepo {
	return &SegmentRecordRepo{client: client}
}

// ListRecords returns paginated records for a dataset version.
func (r *SegmentRecordRepo) ListRecords(ctx context.Context, params segment.RecordListParams) (*segment.RecordListResult, error) {
	segmentID, err := r.segmentIDByKey(ctx, params.SegmentKey)
	if err != nil {
		return nil, err
	}

	query := sanitizeSearch(params.Query)
	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM segment_records
		WHERE workspace_key = $1
		  AND segment_id = $2
		  AND dataset_version = $3
		  AND ($4 = '' OR record_key ILIKE '%' || $4 || '%')
	`, wsKey(ctx), segmentID, params.DatasetVersion, query).Scan(&total); err != nil {
		return nil, fmt.Errorf("count segment records: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, $4, dataset_version, record_key, attributes, created_at
		FROM segment_records
		WHERE workspace_key = $1
		  AND segment_id = $2
		  AND dataset_version = $3
		  AND ($5 = '' OR record_key ILIKE '%' || $5 || '%')
		ORDER BY created_at DESC
		LIMIT $6 OFFSET $7
	`, wsKey(ctx), segmentID, params.DatasetVersion, params.SegmentKey, query, params.PageSize, (params.Page-1)*params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("list segment records: %w", err)
	}
	defer rows.Close()

	records := make([]segment.Record, 0)
	for rows.Next() {
		item, err := scanSegmentRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan segment record: %w", err)
		}
		records = append(records, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate segment records: %w", err)
	}

	return &segment.RecordListResult{
		Data:       records,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

// GetRecordByKey returns a single record by record key.
func (r *SegmentRecordRepo) GetRecordByKey(ctx context.Context, segmentKey, datasetVersion, recordKey string) (*segment.Record, error) {
	segmentID, err := r.segmentIDByKey(ctx, segmentKey)
	if err != nil {
		return nil, err
	}

	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, $4, dataset_version, record_key, attributes, created_at
		FROM segment_records
		WHERE workspace_key = $1
		  AND segment_id = $2
		  AND dataset_version = $3
		  AND record_key = $5
	`, wsKey(ctx), segmentID, datasetVersion, segmentKey, recordKey)

	record, err := scanSegmentRecord(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("segment record %q not found", recordKey),
				"error.segmentRecordNotFound",
			)
		}
		return nil, fmt.Errorf("find segment record: %w", err)
	}

	return record, nil
}

// InsertMany imports a full dataset version in bulk.
func (r *SegmentRecordRepo) InsertMany(ctx context.Context, records []segment.Record) error {
	if len(records) == 0 {
		return nil
	}

	segmentKey := records[0].SegmentKey
	segmentID, err := r.segmentIDByKey(ctx, segmentKey)
	if err != nil {
		return err
	}

	rows := make([][]any, 0, len(records))
	workspaceKey := wsKey(ctx)
	for _, record := range records {
		id := record.ID
		if id == "" {
			generated, err := newID()
			if err != nil {
				return err
			}
			id = generated
		}
		attributesJSON, err := jsonBytes(record.Attributes, "{}")
		if err != nil {
			return fmt.Errorf("marshal segment record attributes: %w", err)
		}
		rows = append(rows, []any{id, workspaceKey, segmentID, record.DatasetVersion, record.RecordKey, attributesJSON, record.CreatedAt})
	}

	_, err = r.client.db(ctx).CopyFrom(
		ctx,
		pgx.Identifier{"segment_records"},
		[]string{"id", "workspace_key", "segment_id", "dataset_version", "record_key", "attributes", "created_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("copy segment records: %w", err)
	}

	return nil
}

// DeleteAllBySegment deletes all records for a segment.
func (r *SegmentRecordRepo) DeleteAllBySegment(ctx context.Context, segmentKey string) (int64, error) {
	segmentID, err := r.segmentIDByKey(ctx, segmentKey)
	if err != nil {
		if apiErr, ok := err.(*apierror.APIError); ok && apiErr.Code == apierror.CodeNotFound {
			return 0, nil
		}
		return 0, err
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM segment_records
		WHERE workspace_key = $1 AND segment_id = $2
	`, wsKey(ctx), segmentID)
	if err != nil {
		return 0, fmt.Errorf("delete segment records: %w", err)
	}

	return tag.RowsAffected(), nil
}

// DeleteAllBySegmentExceptVersion deletes stale dataset versions.
func (r *SegmentRecordRepo) DeleteAllBySegmentExceptVersion(ctx context.Context, segmentKey, datasetVersion string) (int64, error) {
	segmentID, err := r.segmentIDByKey(ctx, segmentKey)
	if err != nil {
		return 0, err
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM segment_records
		WHERE workspace_key = $1
		  AND segment_id = $2
		  AND dataset_version <> $3
	`, wsKey(ctx), segmentID, datasetVersion)
	if err != nil {
		return 0, fmt.Errorf("delete stale segment records: %w", err)
	}

	return tag.RowsAffected(), nil
}

// ExistsRecordKey checks if a record exists in the dataset version.
func (r *SegmentRecordRepo) ExistsRecordKey(ctx context.Context, segmentKey, datasetVersion, recordKey string) (bool, error) {
	segmentID, err := r.segmentIDByKey(ctx, segmentKey)
	if err != nil {
		return false, err
	}

	var exists bool
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM segment_records
			WHERE workspace_key = $1
			  AND segment_id = $2
			  AND dataset_version = $3
			  AND record_key = $4
		)
	`, wsKey(ctx), segmentID, datasetVersion, recordKey).Scan(&exists); err != nil {
		return false, fmt.Errorf("exists segment record key: %w", err)
	}

	return exists, nil
}

func (r *SegmentRecordRepo) segmentIDByKey(ctx context.Context, segmentKey string) (uuid.UUID, error) {
	var segmentID uuid.UUID
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM segments
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), segmentKey).Scan(&segmentID); err != nil {
		if isNoRows(err) {
			return uuid.Nil, apierror.NewNotFound(
				fmt.Sprintf("segment %q not found", segmentKey),
				"error.segmentNotFound",
			)
		}
		return uuid.Nil, fmt.Errorf("find segment id: %w", err)
	}

	return segmentID, nil
}

type segmentRecordScanner interface {
	Scan(dest ...any) error
}

func scanSegmentRecord(scanner segmentRecordScanner) (*segment.Record, error) {
	var record segment.Record
	var attributesJSON []byte
	if err := scanner.Scan(
		&record.ID,
		&record.WorkspaceKey,
		&record.SegmentKey,
		&record.DatasetVersion,
		&record.RecordKey,
		&attributesJSON,
		&record.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(attributesJSON, &record.Attributes); err != nil {
		return nil, err
	}

	return &record, nil
}
