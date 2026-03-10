package segment

import "context"

// ListParams holds parameters for listing segments.
type ListParams struct {
	Search   string
	Page     int
	PageSize int
}

// ListResult holds a paginated list of segments.
type ListResult struct {
	Data       []Segment
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// RecordListParams holds parameters for listing segment records.
type RecordListParams struct {
	SegmentKey     string
	DatasetVersion string
	Query          string
	Page           int
	PageSize       int
}

// RecordListResult holds a paginated list of segment records.
type RecordListResult struct {
	Data       []Record
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// Repository defines the persistence interface for segments.
type Repository interface {
	Create(ctx context.Context, seg *Segment) error
	GetByKey(ctx context.Context, key string) (*Segment, error)
	Update(ctx context.Context, seg *Segment) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, params ListParams) (*ListResult, error)
}

// RecordRepository defines the persistence interface for segment records.
type RecordRepository interface {
	ListRecords(ctx context.Context, params RecordListParams) (*RecordListResult, error)
	GetRecordByKey(ctx context.Context, segmentKey, datasetVersion, recordKey string) (*Record, error)
	InsertMany(ctx context.Context, records []Record) error
	DeleteAllBySegment(ctx context.Context, segmentKey string) (int64, error)
	DeleteAllBySegmentExceptVersion(ctx context.Context, segmentKey, datasetVersion string) (int64, error)
	ExistsRecordKey(ctx context.Context, segmentKey, datasetVersion, recordKey string) (bool, error)
}
