package securitypolicy

import "time"

// ManagedPolicy stores only the mutable values persisted by the application.
type ManagedPolicy struct {
	CORSOrigins []string
	UpdatedAt   time.Time
	UpdatedBy   string
}

// ListSnapshot describes managed, inherited, and effective values for one policy list.
type ListSnapshot struct {
	Managed   []string
	Inherited []string
	Effective []string
}

// Snapshot is the full runtime view consumed by handlers and middleware.
type Snapshot struct {
	CORSOrigins ListSnapshot
	UpdatedAt   time.Time
	UpdatedBy   string
}

// Reader exposes the effective runtime policy snapshot.
type Reader interface {
	Snapshot() Snapshot
}

// StaticReader is a fixed snapshot reader used by tests and simple callers.
type StaticReader struct {
	snapshot Snapshot
}

// NewStaticReader creates a new immutable policy reader.
func NewStaticReader(snapshot Snapshot) *StaticReader {
	return &StaticReader{snapshot: snapshot}
}

// Snapshot returns a copy of the configured static snapshot.
func (r *StaticReader) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{}
	}

	return cloneSnapshot(r.snapshot)
}
