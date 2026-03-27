package experiment

import "time"

// Status represents the lifecycle state of an experiment.
type Status string

// Supported experiment statuses.
const (
	StatusDraft     Status = "draft"
	StatusRunning   Status = "running"
	StatusPaused    Status = "paused"
	StatusCompleted Status = "completed"
)

// Valid returns true if the status is one of the supported statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusRunning, StatusPaused, StatusCompleted:
		return true
	default:
		return false
	}
}

// Variant represents a single variant in an experiment.
type Variant struct {
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Weight int    `json:"weight"`
}

// Metric defines a tracked conversion metric.
type Metric struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Experiment represents an A/B test associated with a feature.
type Experiment struct {
	ID                    string     `json:"id"`
	WorkspaceKey          string     `json:"workspaceKey"`
	FeatureKey            string     `json:"featureKey"`
	Name                  string     `json:"name"`
	Description           string     `json:"description"`
	Variants              []Variant  `json:"variants"`
	Metrics               []Metric   `json:"metrics"`
	Status                Status     `json:"status"`
	LookupCacheEnabled    bool       `json:"lookupCacheEnabled,omitempty"`
	LookupCacheTTLSeconds int        `json:"lookupCacheTTLSeconds,omitempty"`
	WinnerKey             string     `json:"winnerKey,omitempty"`
	StartedAt             *time.Time `json:"startedAt,omitempty"`
	CompletedAt           *time.Time `json:"completedAt,omitempty"`
	CreatedBy             string     `json:"createdBy"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

// Exposure records that a user was exposed to a variant.
type Exposure struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	ExperimentID string    `json:"experimentId"`
	FeatureKey   string    `json:"featureKey"`
	UserID       string    `json:"userId"`
	VariantKey   string    `json:"variantKey"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Conversion records a user conversion event for a metric.
type Conversion struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	ExperimentID string    `json:"experimentId"`
	UserID       string    `json:"userId"`
	VariantKey   string    `json:"variantKey"`
	MetricKey    string    `json:"metricKey"`
	Value        float64   `json:"value"`
	CreatedAt    time.Time `json:"createdAt"`
}

// VariantStats holds computed stats for a variant.
type VariantStats struct {
	VariantKey     string  `json:"variantKey"`
	Exposures      int64   `json:"exposures"`
	Conversions    int64   `json:"conversions"`
	ConversionRate float64 `json:"conversionRate"`
	ConfidenceLow  float64 `json:"confidenceLow"`
	ConfidenceHigh float64 `json:"confidenceHigh"`
}

// Results holds aggregated results for an experiment.
type Results struct {
	ExperimentID     string         `json:"experimentId"`
	TotalExposures   int64          `json:"totalExposures"`
	TotalConversions int64          `json:"totalConversions"`
	Variants         []VariantStats `json:"variants"`
	IsSignificant    bool           `json:"isSignificant"`
}

const defaultExperimentLookupCacheTTLSeconds = 60

const (
	minCacheTTLSeconds = 30
	maxCacheTTLSeconds = 3600
)

// NormalizeCacheConfig applies defaults and bounds to experiment cache settings.
func (e *Experiment) NormalizeCacheConfig() {
	if e == nil {
		return
	}
	normalizeEnabledTTL(&e.LookupCacheEnabled, &e.LookupCacheTTLSeconds, defaultExperimentLookupCacheTTLSeconds)
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
