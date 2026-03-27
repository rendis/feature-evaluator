package segment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/observability"
	"github.com/rendis/feature-evaluator/pkg/apierror"
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
	SetMembership(ctx context.Context, segmentKey, userID, tenantID string, isMember bool, ttl time.Duration)
	// GetRecord returns the cached record and whether it was found.
	GetRecord(ctx context.Context, segmentKey, datasetVersion, recordKey string) (*Record, bool)
	// SetRecord caches a record with TTL.
	SetRecord(ctx context.Context, segmentKey, datasetVersion, recordKey string, record *Record, ttl time.Duration)
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
	seg.NormalizeCacheConfig()
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
	seg.NormalizeCacheConfig()
	seg.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, seg); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.InvalidateSegment(ctx, seg.Key)
	}
	return nil
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
	trace, _ := observability.TraceRecorderFromContext(ctx)
	stepStart := time.Now()
	seg, err := s.GetByKey(ctx, segmentKey)
	if err != nil {
		return nil, err
	}
	if record, found := s.cachedRecord(ctx, seg, segmentKey, recordKey); found {
		s.recordRecordTrace(trace, segmentKey, seg, stepStart, observability.CacheStatusHit, "cached")
		return record, nil
	}
	record, err := s.recordRepo.GetRecordByKey(ctx, segmentKey, seg.ActiveDatasetVersion, recordKey)
	if err != nil {
		s.recordRecordTrace(trace, segmentKey, seg, stepStart, recordCacheStatusForError(seg, err), recordCacheOutcomeForError(seg, err))
		return nil, err
	}
	if s.shouldCacheRecord(seg) && record != nil {
		s.cache.SetRecord(ctx, segmentKey, seg.ActiveDatasetVersion, recordKey, record, time.Duration(seg.RecordCacheTTLSeconds)*time.Second)
	}
	s.recordRecordTrace(trace, segmentKey, seg, stepStart, recordCacheStatusForComputed(seg), "computed")
	return record, nil
}

// ReplaceRecords imports a new dataset version for a segment.
func (s *Service) ReplaceRecords(ctx context.Context, segmentKey string, input ReplaceInput) (int64, error) { //nolint:gocognit,cyclop,funlen // batch record replacement
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
	trace, _ := observability.TraceRecorderFromContext(ctx)
	stepStart := time.Now()
	seg, err := s.GetByKey(ctx, segmentKey)
	if err != nil {
		return false, err
	}

	// Check cache first, but only when explicitly enabled for the segment.
	if isMember, found := s.cachedMembership(ctx, seg, segmentKey, userID, tenantID); found {
		s.recordMembershipTrace(trace, segmentKey, seg, stepStart, observability.CacheStatusHit, "cached")
		return isMember, nil
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

	// Store in cache when enabled.
	if s.shouldCacheMembership(seg) {
		s.cache.SetMembership(ctx, segmentKey, userID, tenantID, result, time.Duration(seg.MembershipCacheTTLSeconds)*time.Second)
	}
	s.recordMembershipTrace(trace, segmentKey, seg, stepStart, recordMembershipCacheStatus(seg), "computed")

	return result, nil
}

func (s *Service) cachedRecord(ctx context.Context, seg *Segment, segmentKey, recordKey string) (*Record, bool) {
	if !s.shouldCacheRecord(seg) {
		return nil, false
	}
	return s.cache.GetRecord(ctx, segmentKey, seg.ActiveDatasetVersion, recordKey)
}

func (s *Service) shouldCacheRecord(seg *Segment) bool {
	return s.cache != nil && seg.RecordCacheEnabled && seg.RecordCacheTTLSeconds > 0
}

func (s *Service) cachedMembership(ctx context.Context, seg *Segment, segmentKey, userID, tenantID string) (bool, bool) {
	if !s.shouldCacheMembership(seg) {
		return false, false
	}
	return s.cache.GetMembership(ctx, segmentKey, userID, tenantID)
}

func (s *Service) shouldCacheMembership(seg *Segment) bool {
	return s.cache != nil && seg.MembershipCacheEnabled && seg.MembershipCacheTTLSeconds > 0
}

func (s *Service) recordRecordTrace(trace observability.TraceRecorder, segmentKey string, seg *Segment, stepStart time.Time, status observability.CacheStatus, outcome string) {
	if trace == nil {
		return
	}
	trace.RecordComponent(observability.ComponentTrace{
		Name:         "segment_record:" + segmentKey,
		CacheBackend: observability.CacheBackendRedis,
		CacheEnabled: seg.RecordCacheEnabled,
		CacheStatus:  status,
		TTLSeconds:   seg.RecordCacheTTLSeconds,
		DurationMs:   time.Since(stepStart).Milliseconds(),
		Outcome:      outcome,
	})
}

func (s *Service) recordMembershipTrace(trace observability.TraceRecorder, segmentKey string, seg *Segment, stepStart time.Time, status observability.CacheStatus, outcome string) {
	if trace == nil {
		return
	}
	trace.RecordComponent(observability.ComponentTrace{
		Name:         "segment_membership:" + segmentKey,
		CacheBackend: observability.CacheBackendRedis,
		CacheEnabled: seg.MembershipCacheEnabled,
		CacheStatus:  status,
		TTLSeconds:   seg.MembershipCacheTTLSeconds,
		DurationMs:   time.Since(stepStart).Milliseconds(),
		Outcome:      outcome,
	})
}

func recordCacheStatusForError(seg *Segment, err error) observability.CacheStatus {
	if !seg.RecordCacheEnabled || seg.RecordCacheTTLSeconds <= 0 {
		return observability.CacheStatusDisabled
	}
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.Code == apierror.CodeNotFound {
		return observability.CacheStatusMiss
	}
	return observability.CacheStatusComputed
}

func recordCacheOutcomeForError(seg *Segment, err error) string {
	if !seg.RecordCacheEnabled || seg.RecordCacheTTLSeconds <= 0 {
		return "error"
	}
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.Code == apierror.CodeNotFound {
		return "not_found"
	}
	return "error"
}

func recordCacheStatusForComputed(seg *Segment) observability.CacheStatus {
	if !seg.RecordCacheEnabled || seg.RecordCacheTTLSeconds <= 0 {
		return observability.CacheStatusDisabled
	}
	return observability.CacheStatusMiss
}

func recordMembershipCacheStatus(seg *Segment) observability.CacheStatus {
	if !seg.MembershipCacheEnabled || seg.MembershipCacheTTLSeconds <= 0 {
		return observability.CacheStatusDisabled
	}
	return observability.CacheStatusMiss
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
