package segment

import (
	"context"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

type mockSegmentRepo struct {
	segments map[string]*Segment
}

func newMockSegmentRepo() *mockSegmentRepo {
	return &mockSegmentRepo{segments: make(map[string]*Segment)}
}

func (m *mockSegmentRepo) Create(_ context.Context, seg *Segment) error {
	m.segments[seg.Key] = seg
	return nil
}

func (m *mockSegmentRepo) GetByKey(_ context.Context, key string) (*Segment, error) {
	seg, ok := m.segments[key]
	if !ok {
		return nil, apierror.NewNotFound("segment not found", "error.segmentNotFound")
	}
	cp := *seg
	return &cp, nil
}

func (m *mockSegmentRepo) Update(_ context.Context, seg *Segment) error {
	m.segments[seg.Key] = seg
	return nil
}

func (m *mockSegmentRepo) Delete(_ context.Context, key string) error {
	delete(m.segments, key)
	return nil
}

func (m *mockSegmentRepo) List(_ context.Context, _ ListParams) (*ListResult, error) {
	return &ListResult{}, nil
}

type mockSegmentRecordRepo struct {
	records           map[string]*Record
	existsCalls       int
	getRecordCalls    int
	listCalls         int
	insertCalls       int
	deleteAllCalls    int
	deleteExceptCalls int
}

func newMockSegmentRecordRepo() *mockSegmentRecordRepo {
	return &mockSegmentRecordRepo{records: make(map[string]*Record)}
}

func segmentRecordKey(segmentKey, datasetVersion, recordKey string) string {
	return segmentKey + "|" + datasetVersion + "|" + recordKey
}

func (m *mockSegmentRecordRepo) InsertMany(_ context.Context, records []Record) error {
	m.insertCalls += len(records)
	return nil
}

func (m *mockSegmentRecordRepo) DeleteAllBySegment(_ context.Context, _ string) (int64, error) {
	m.deleteAllCalls++
	return 0, nil
}

func (m *mockSegmentRecordRepo) DeleteAllBySegmentExceptVersion(_ context.Context, _ string, _ string) (int64, error) {
	m.deleteExceptCalls++
	return 0, nil
}

func (m *mockSegmentRecordRepo) ListRecords(_ context.Context, _ RecordListParams) (*RecordListResult, error) {
	m.listCalls++
	return &RecordListResult{}, nil
}

func (m *mockSegmentRecordRepo) GetRecordByKey(_ context.Context, segmentKey, datasetVersion, recordKey string) (*Record, error) {
	m.getRecordCalls++
	record, ok := m.records[segmentRecordKey(segmentKey, datasetVersion, recordKey)]
	if !ok {
		return nil, apierror.NewNotFound("record not found", "error.segmentRecordNotFound")
	}
	cp := *record
	return &cp, nil
}

func (m *mockSegmentRecordRepo) ExistsRecordKey(_ context.Context, segmentKey, datasetVersion, recordKey string) (bool, error) {
	m.existsCalls++
	_, ok := m.records[segmentRecordKey(segmentKey, datasetVersion, recordKey)]
	return ok, nil
}

type mockSegmentCache struct {
	membershipCalls int
	recordCalls     int
	setMemberCalls  int
	setRecordCalls  int
	lastMemberTTL   time.Duration
	lastRecordTTL   time.Duration
	membership      map[string]bool
	records         map[string]*Record
}

func newMockSegmentCache() *mockSegmentCache {
	return &mockSegmentCache{
		membership: make(map[string]bool),
		records:    make(map[string]*Record),
	}
}

func (m *mockSegmentCache) GetMembership(_ context.Context, segmentKey, userID, tenantID string) (bool, bool) {
	m.membershipCalls++
	value, ok := m.membership[segmentKey+"|"+userID+"|"+tenantID]
	return value, ok
}

func (m *mockSegmentCache) SetMembership(_ context.Context, segmentKey, userID, tenantID string, isMember bool, ttl time.Duration) {
	m.setMemberCalls++
	m.lastMemberTTL = ttl
	m.membership[segmentKey+"|"+userID+"|"+tenantID] = isMember
}

func (m *mockSegmentCache) GetRecord(_ context.Context, segmentKey, datasetVersion, recordKey string) (*Record, bool) {
	m.recordCalls++
	record, ok := m.records[segmentKey+"|"+datasetVersion+"|"+recordKey]
	if !ok {
		return nil, false
	}
	cp := *record
	return &cp, true
}

func (m *mockSegmentCache) SetRecord(_ context.Context, segmentKey, datasetVersion, recordKey string, record *Record, ttl time.Duration) {
	m.setRecordCalls++
	m.lastRecordTTL = ttl
	cp := *record
	m.records[segmentKey+"|"+datasetVersion+"|"+recordKey] = &cp
}

func (m *mockSegmentCache) InvalidateSegment(_ context.Context, _ string) {}

func TestIsMember_SkipsCacheWhenDisabled(t *testing.T) {
	t.Parallel()

	repo := newMockSegmentRepo()
	repo.segments["beta-users"] = &Segment{
		Key:                       "beta-users",
		RecordKeyPath:             "userId",
		ActiveDatasetVersion:      "v1",
		MembershipCacheEnabled:    false,
		MembershipCacheTTLSeconds: 45,
	}
	recordRepo := newMockSegmentRecordRepo()
	recordRepo.records[segmentRecordKey("beta-users", "v1", "user-1")] = &Record{}
	cache := newMockSegmentCache()
	svc := NewService(repo, recordRepo, cache)

	got, err := svc.IsMember(context.Background(), "beta-users", "user-1", "")
	if err != nil {
		t.Fatalf("IsMember() error = %v", err)
	}
	if !got {
		t.Fatal("expected membership to resolve true")
	}
	if cache.membershipCalls != 0 || cache.setMemberCalls != 0 {
		t.Fatalf("expected cache to be bypassed, got get=%d set=%d", cache.membershipCalls, cache.setMemberCalls)
	}
}

func TestIsMember_UsesCacheWhenEnabled(t *testing.T) {
	t.Parallel()

	repo := newMockSegmentRepo()
	repo.segments["beta-users"] = &Segment{
		Key:                       "beta-users",
		RecordKeyPath:             "userId",
		ActiveDatasetVersion:      "v1",
		MembershipCacheEnabled:    true,
		MembershipCacheTTLSeconds: 45,
	}
	recordRepo := newMockSegmentRecordRepo()
	recordRepo.records[segmentRecordKey("beta-users", "v1", "user-1")] = &Record{}
	cache := newMockSegmentCache()
	svc := NewService(repo, recordRepo, cache)

	got, err := svc.IsMember(context.Background(), "beta-users", "user-1", "")
	if err != nil {
		t.Fatalf("IsMember() error = %v", err)
	}
	if !got {
		t.Fatal("expected membership to resolve true")
	}
	if cache.membershipCalls != 1 || cache.setMemberCalls != 1 {
		t.Fatalf("expected cache read/write, got get=%d set=%d", cache.membershipCalls, cache.setMemberCalls)
	}
	if cache.lastMemberTTL != 45*time.Second {
		t.Fatalf("lastMemberTTL = %s, want 45s", cache.lastMemberTTL)
	}
}

func TestGetRecordByKey_UsesRecordCacheHitWhenEnabled(t *testing.T) {
	t.Parallel()

	repo := newMockSegmentRepo()
	repo.segments["beta-users"] = &Segment{
		Key:                   "beta-users",
		ActiveDatasetVersion:  "v1",
		RecordCacheEnabled:    true,
		RecordCacheTTLSeconds: 90,
	}
	recordRepo := newMockSegmentRecordRepo()
	cache := newMockSegmentCache()
	cache.records[segmentRecordKey("beta-users", "v1", "user-1")] = &Record{
		SegmentKey:     "beta-users",
		DatasetVersion: "v1",
		RecordKey:      "user-1",
		Attributes:     map[string]any{"id": "user-1"},
	}
	svc := NewService(repo, recordRepo, cache)

	record, err := svc.GetRecordByKey(context.Background(), "beta-users", "user-1")
	if err != nil {
		t.Fatalf("GetRecordByKey() error = %v", err)
	}
	if record == nil || record.RecordKey != "user-1" {
		t.Fatalf("record = %#v, want user-1", record)
	}
	if cache.recordCalls != 1 || cache.setRecordCalls != 0 {
		t.Fatalf("expected cache hit only, got get=%d set=%d", cache.recordCalls, cache.setRecordCalls)
	}
	if recordRepo.getRecordCalls != 0 {
		t.Fatalf("expected record repo to be bypassed, got %d calls", recordRepo.getRecordCalls)
	}
}

func TestGetRecordByKey_SetsRecordCacheWhenEnabled(t *testing.T) {
	t.Parallel()

	repo := newMockSegmentRepo()
	repo.segments["beta-users"] = &Segment{
		Key:                   "beta-users",
		ActiveDatasetVersion:  "v1",
		RecordCacheEnabled:    true,
		RecordCacheTTLSeconds: 90,
	}
	recordRepo := newMockSegmentRecordRepo()
	recordRepo.records[segmentRecordKey("beta-users", "v1", "user-1")] = &Record{
		SegmentKey:     "beta-users",
		DatasetVersion: "v1",
		RecordKey:      "user-1",
		Attributes:     map[string]any{"id": "user-1"},
	}
	cache := newMockSegmentCache()
	svc := NewService(repo, recordRepo, cache)

	record, err := svc.GetRecordByKey(context.Background(), "beta-users", "user-1")
	if err != nil {
		t.Fatalf("GetRecordByKey() error = %v", err)
	}
	if record == nil || record.RecordKey != "user-1" {
		t.Fatalf("record = %#v, want user-1", record)
	}
	if cache.setRecordCalls != 1 {
		t.Fatalf("expected cache write, got %d", cache.setRecordCalls)
	}
	if cache.lastRecordTTL != 90*time.Second {
		t.Fatalf("lastRecordTTL = %s, want 90s", cache.lastRecordTTL)
	}
	if recordRepo.getRecordCalls != 1 {
		t.Fatalf("expected record repo to be called once, got %d", recordRepo.getRecordCalls)
	}
}

func TestGetRecordByKey_SkipsCacheWhenDisabled(t *testing.T) {
	t.Parallel()

	repo := newMockSegmentRepo()
	repo.segments["beta-users"] = &Segment{
		Key:                  "beta-users",
		ActiveDatasetVersion: "v1",
		RecordCacheEnabled:   false,
	}
	recordRepo := newMockSegmentRecordRepo()
	recordRepo.records[segmentRecordKey("beta-users", "v1", "user-1")] = &Record{
		SegmentKey:     "beta-users",
		DatasetVersion: "v1",
		RecordKey:      "user-1",
		Attributes:     map[string]any{"id": "user-1"},
	}
	cache := newMockSegmentCache()
	svc := NewService(repo, recordRepo, cache)

	record, err := svc.GetRecordByKey(context.Background(), "beta-users", "user-1")
	if err != nil {
		t.Fatalf("GetRecordByKey() error = %v", err)
	}
	if record == nil || record.RecordKey != "user-1" {
		t.Fatalf("record = %#v, want user-1", record)
	}
	if cache.recordCalls != 0 || cache.setRecordCalls != 0 {
		t.Fatalf("expected cache bypass, got get=%d set=%d", cache.recordCalls, cache.setRecordCalls)
	}
	if recordRepo.getRecordCalls != 1 {
		t.Fatalf("expected record repo to be called once, got %d", recordRepo.getRecordCalls)
	}
}
