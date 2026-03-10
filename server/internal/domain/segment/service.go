package segment

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// TxManager coordinates multi-step persistence operations under one transaction.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
}

// Cache defines the caching interface for segment membership lookups.
type Cache interface {
	// GetMembership returns the cached membership value ("1"/"0") and whether it was found.
	GetMembership(ctx context.Context, segmentKey, userID, tenantID string) (isMember bool, found bool)
	// SetMembership caches the membership result with TTL.
	SetMembership(ctx context.Context, segmentKey, userID, tenantID string, isMember bool)
	// InvalidateSegment removes all cached membership entries for a segment.
	InvalidateSegment(ctx context.Context, segmentKey string)
}

// Service handles segment business logic.
type Service struct {
	repo       Repository
	recordRepo RecordRepository
	cache      Cache
	txManager  TxManager
}

// NewService creates a new segment service.
func NewService(repo Repository, recordRepo RecordRepository, cache Cache) *Service {
	return &Service{repo: repo, recordRepo: recordRepo, cache: cache}
}

// SetTxManager configures transactional execution for multi-step imports.
func (s *Service) SetTxManager(txManager TxManager) {
	s.txManager = txManager
}

// Create creates a new segment.
func (s *Service) Create(ctx context.Context, seg *Segment) error {
	now := time.Now().UTC()
	seg.Key = NormalizeKey(seg.Key)
	if err := ValidateNormalizedKey(seg.Key); err != nil {
		return err
	}
	seg.CreatedAt = now
	seg.UpdatedAt = now
	seg.RecordCount = 0
	if seg.PreviewFields == nil {
		seg.PreviewFields = []string{}
	}
	return s.repo.Create(ctx, seg)
}

// GetByKey retrieves a segment by its key, including member count.
func (s *Service) GetByKey(ctx context.Context, key string) (*Segment, error) {
	seg, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("getting segment %s: %w", key, err)
	}
	return seg, nil
}

// Update updates an existing segment.
func (s *Service) Update(ctx context.Context, seg *Segment) error {
	seg.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, seg)
}

// Delete removes a segment and its records.
func (s *Service) Delete(ctx context.Context, key string) error {
	if _, err := s.recordRepo.DeleteAllBySegment(ctx, key); err != nil {
		return fmt.Errorf("deleting segment records for %s: %w", key, err)
	}
	if err := s.repo.Delete(ctx, key); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.InvalidateSegment(ctx, key)
	}
	return nil
}

// List returns a paginated list of segments.
func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, params)
}

// ListRecords returns paginated records from the active dataset version.
func (s *Service) ListRecords(ctx context.Context, params RecordListParams) (*RecordListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	seg, err := s.GetByKey(ctx, params.SegmentKey)
	if err != nil {
		return nil, err
	}
	params.DatasetVersion = seg.ActiveDatasetVersion
	return s.recordRepo.ListRecords(ctx, params)
}

// GetRecordByKey returns a single record from the active dataset version.
func (s *Service) GetRecordByKey(ctx context.Context, segmentKey, recordKey string) (*Record, error) {
	seg, err := s.GetByKey(ctx, segmentKey)
	if err != nil {
		return nil, err
	}
	return s.recordRepo.GetRecordByKey(ctx, segmentKey, seg.ActiveDatasetVersion, recordKey)
}

// ReplaceRecords imports a new dataset version for a segment.
func (s *Service) ReplaceRecords(ctx context.Context, segmentKey string, input ReplaceInput) (int64, error) {
	if input.SourceType != SourceTypeCSV && input.SourceType != SourceTypeJSON {
		return 0, fmt.Errorf("invalid source type %q", input.SourceType)
	}
	if input.RecordKeyPath == "" {
		return 0, fmt.Errorf("record key path is required")
	}
	if len(input.Records) == 0 {
		return 0, fmt.Errorf("records are required")
	}
	if len(input.Schema) == 0 {
		return 0, fmt.Errorf("schema is required")
	}
	normalizedSchema, err := validateSchemaRecords(input.Schema, input.Records)
	if err != nil {
		return 0, err
	}

	seg, err := s.GetByKey(ctx, segmentKey)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	datasetVersion := NewDatasetVersion()
	records := make([]Record, 0, len(input.Records))
	seen := make(map[string]int, len(input.Records))
	for idx, record := range input.Records {
		recordKey, err := ExtractRecordKey(record, input.RecordKeyPath)
		if err != nil {
			return 0, fmt.Errorf("record %d: %w", idx+1, err)
		}
		if _, exists := seen[recordKey]; exists {
			return 0, fmt.Errorf("record %d: duplicate record key %q", idx+1, recordKey)
		}
		seen[recordKey] = idx + 1
		records = append(records, Record{
			SegmentKey:     segmentKey,
			DatasetVersion: datasetVersion,
			RecordKey:      recordKey,
			Attributes:     record,
			CreatedAt:      now,
		})
	}

	apply := func(txCtx context.Context) error {
		if err := s.recordRepo.InsertMany(txCtx, records); err != nil {
			return fmt.Errorf("importing records to segment %s: %w", segmentKey, err)
		}

		previewFields := DerivePreviewFields(input.Records)
		seg.Schema = normalizedSchema
		seg.RecordKeyPath = input.RecordKeyPath
		seg.ActiveDatasetVersion = datasetVersion
		seg.PreviewFields = previewFields
		seg.SourceType = input.SourceType
		seg.RecordCount = int64(len(records))
		seg.LastImportAt = &now
		seg.UpdatedAt = now
		seg.UpdatedBy = input.UpdatedBy

		if err := s.repo.Update(txCtx, seg); err != nil {
			return fmt.Errorf("updating segment %s import metadata: %w", segmentKey, err)
		}

		if _, err := s.recordRepo.DeleteAllBySegmentExceptVersion(txCtx, segmentKey, datasetVersion); err != nil {
			return fmt.Errorf("cleaning old records for segment %s: %w", segmentKey, err)
		}

		return nil
	}

	if s.txManager != nil {
		if err := s.txManager.WithinTx(ctx, apply); err != nil {
			return int64(len(records)), err
		}
	} else if err := apply(ctx); err != nil {
		return int64(len(records)), err
	}

	if s.cache != nil {
		s.cache.InvalidateSegment(ctx, segmentKey)
	}

	return int64(len(records)), nil
}

// IsMember keeps current rule evaluation compiling by treating userID as the lookup key.
func (s *Service) IsMember(ctx context.Context, segmentKey, userID, tenantID string) (bool, error) {
	// Check cache first
	if s.cache != nil {
		if isMember, found := s.cache.GetMembership(ctx, segmentKey, userID, tenantID); found {
			return isMember, nil
		}
	}

	seg, err := s.GetByKey(ctx, segmentKey)
	if err != nil {
		return false, err
	}

	if seg.RecordKeyPath != "" && !isUserIDCompatibleKeyPath(seg.RecordKeyPath) {
		slog.Warn("inSegment() lookup uses userId but segment is keyed by a different field",
			"segmentKey", segmentKey,
			"recordKeyPath", seg.RecordKeyPath,
			"userID", userID,
		)
	}

	result, err := s.recordRepo.ExistsRecordKey(ctx, segmentKey, seg.ActiveDatasetVersion, userID)
	if err != nil {
		return false, err
	}

	// Store in cache
	if s.cache != nil {
		s.cache.SetMembership(ctx, segmentKey, userID, tenantID, result)
	}

	return result, nil
}

// isUserIDCompatibleKeyPath returns true if the segment's recordKeyPath
// is compatible with userId-based lookups from inSegment().
func isUserIDCompatibleKeyPath(path string) bool {
	switch path {
	case "userId", "user_id", "id", "user.id", "sub":
		return true
	}
	return false
}
