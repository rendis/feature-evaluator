package segment

import "time"

// SourceType identifies the original import source.
type SourceType string

// Supported import source types.
const (
	SourceTypeCSV  SourceType = "csv"
	SourceTypeJSON SourceType = "json"
)

// ImportMode defines how segment record imports are handled.
type ImportMode string

// Supported import modes.
const (
	ImportModeReplace ImportMode = "replace"
)

// Segment describes a dynamic dataset that can later be referenced by rules.
type Segment struct {
	ID                        string         `json:"id"`
	WorkspaceKey              string         `json:"workspaceKey"`
	Key                       string         `json:"key"`
	Name                      string         `json:"name"`
	Description               string         `json:"description"`
	Metadata                  map[string]any `json:"metadata,omitempty"`
	Schema                    map[string]any `json:"schema,omitempty"`
	RecordKeyPath             string         `json:"recordKeyPath,omitempty"`
	ActiveDatasetVersion      string         `json:"activeDatasetVersion,omitempty"`
	PreviewFields             []string       `json:"previewFields,omitempty"`
	SourceType                SourceType     `json:"sourceType,omitempty"`
	RecordCount               int64          `json:"recordCount"`
	MembershipCacheEnabled    bool           `json:"membershipCacheEnabled,omitempty"`
	MembershipCacheTTLSeconds int            `json:"membershipCacheTTLSeconds,omitempty"`
	RecordCacheEnabled        bool           `json:"recordCacheEnabled,omitempty"`
	RecordCacheTTLSeconds     int            `json:"recordCacheTTLSeconds,omitempty"`
	LastImportAt              *time.Time     `json:"lastImportAt,omitempty"`
	CreatedAt                 time.Time      `json:"createdAt"`
	UpdatedAt                 time.Time      `json:"updatedAt"`
	CreatedBy                 string         `json:"createdBy"`
	UpdatedBy                 string         `json:"updatedBy"`
}

// Record is a single JSON document within a segment dataset version.
type Record struct {
	ID             string         `json:"id"`
	WorkspaceKey   string         `json:"workspaceKey"`
	SegmentKey     string         `json:"segmentKey"`
	DatasetVersion string         `json:"datasetVersion"`
	RecordKey      string         `json:"recordKey"`
	Attributes     map[string]any `json:"attributes"`
	CreatedAt      time.Time      `json:"createdAt"`
}

// ReplaceInput captures the full payload for a replace import.
type ReplaceInput struct {
	SourceType    SourceType
	RecordKeyPath string
	Schema        map[string]any
	Records       []map[string]any
	UpdatedBy     string
}

const defaultSegmentCacheTTLSeconds = 300

const (
	minCacheTTLSeconds = 30
	maxCacheTTLSeconds = 3600
)

// NormalizeCacheConfig applies defaults and bounds to segment cache settings.
func (s *Segment) NormalizeCacheConfig() {
	if s == nil {
		return
	}
	normalizeSegmentCache(&s.MembershipCacheEnabled, &s.MembershipCacheTTLSeconds)
	normalizeSegmentCache(&s.RecordCacheEnabled, &s.RecordCacheTTLSeconds)
}

func normalizeSegmentCache(enabled *bool, ttl *int) {
	normalizeEnabledTTL(enabled, ttl, defaultSegmentCacheTTLSeconds)
}

func normalizeEnabledTTL(enabled *bool, ttl *int, defaultTTL int) {
	if enabled == nil || ttl == nil {
		return
	}
	if !*enabled {
		*ttl = 0
		return
	}
	if *ttl <= 0 {
		*ttl = defaultTTL
	}
	if *ttl < minCacheTTLSeconds {
		*ttl = minCacheTTLSeconds
	}
	if *ttl > maxCacheTTLSeconds {
		*ttl = maxCacheTTLSeconds
	}
}
