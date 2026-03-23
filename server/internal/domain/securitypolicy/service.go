package securitypolicy

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const defaultPollInterval = 5 * time.Second

// Service manages the persisted global policy and the in-memory runtime snapshot.
type Service struct {
	repo      Repository
	inherited ManagedPolicy

	mu       sync.RWMutex
	snapshot Snapshot

	stopOnce sync.Once
	stopFn   context.CancelFunc
	wg       sync.WaitGroup
}

// NewService creates a new Service with the inherited env-managed values preloaded.
func NewService(repo Repository, inherited ManagedPolicy) *Service {
	inherited.CORSOrigins = cloneStrings(inherited.CORSOrigins)
	inherited.ExternalAPIAllowHosts = cloneStrings(inherited.ExternalAPIAllowHosts)

	return &Service{
		repo:      repo,
		inherited: inherited,
		snapshot:  buildSnapshot(inherited, ManagedPolicy{}),
	}
}

// Load refreshes the in-memory snapshot from the repository.
func (s *Service) Load(ctx context.Context) error {
	managed, err := s.repo.Get(ctx)
	if err != nil {
		return fmt.Errorf("loading security policy: %w", err)
	}

	normalizedManaged, err := normalizeManagedPolicy(managed)
	if err != nil {
		return err
	}

	s.setSnapshot(buildSnapshot(s.inherited, normalizedManaged))
	return nil
}

// Start begins periodic refresh polling.
func (s *Service) Start(interval time.Duration) {
	if interval <= 0 {
		interval = defaultPollInterval
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.stopFn = cancel
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Load(context.Background()); err != nil {
					slog.Warn("refreshing security policy snapshot", "error", err)
				}
			}
		}
	}()
}

// Stop halts background polling.
func (s *Service) Stop() {
	s.stopOnce.Do(func() {
		if s.stopFn != nil {
			s.stopFn()
		}
		s.wg.Wait()
	})
}

// Snapshot returns the current effective snapshot.
func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneSnapshot(s.snapshot)
}

// Update replaces the app-managed lists and refreshes the in-memory snapshot immediately.
func (s *Service) Update(ctx context.Context, policy ManagedPolicy) (Snapshot, error) {
	normalized, err := normalizeManagedPolicy(&policy)
	if err != nil {
		return Snapshot{}, apierror.NewBadRequest(err.Error(), "error.invalidSecurityPolicy")
	}

	normalized.UpdatedAt = time.Now().UTC()
	if err := s.repo.Upsert(ctx, &normalized); err != nil {
		return Snapshot{}, fmt.Errorf("upserting security policy: %w", err)
	}

	snapshot := buildSnapshot(s.inherited, normalized)
	s.setSnapshot(snapshot)

	return cloneSnapshot(snapshot), nil
}

func (s *Service) setSnapshot(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshot = cloneSnapshot(snapshot)
}

func normalizeManagedPolicy(policy *ManagedPolicy) (ManagedPolicy, error) {
	if policy == nil {
		return ManagedPolicy{}, nil
	}

	corsOrigins, err := NormalizeOrigins(policy.CORSOrigins)
	if err != nil {
		return ManagedPolicy{}, err
	}
	allowHosts, err := NormalizeHosts(policy.ExternalAPIAllowHosts)
	if err != nil {
		return ManagedPolicy{}, err
	}

	return ManagedPolicy{
		CORSOrigins:           corsOrigins,
		ExternalAPIAllowHosts: allowHosts,
		UpdatedAt:             policy.UpdatedAt,
		UpdatedBy:             policy.UpdatedBy,
	}, nil
}

func buildSnapshot(inherited ManagedPolicy, managed ManagedPolicy) Snapshot {
	return Snapshot{
		CORSOrigins: ListSnapshot{
			Managed:   cloneStrings(managed.CORSOrigins),
			Inherited: cloneStrings(inherited.CORSOrigins),
			Effective: unionStrings(inherited.CORSOrigins, managed.CORSOrigins),
		},
		ExternalAPIAllowHosts: ListSnapshot{
			Managed:   cloneStrings(managed.ExternalAPIAllowHosts),
			Inherited: cloneStrings(inherited.ExternalAPIAllowHosts),
			Effective: unionStrings(inherited.ExternalAPIAllowHosts, managed.ExternalAPIAllowHosts),
		},
		UpdatedAt: managed.UpdatedAt,
		UpdatedBy: managed.UpdatedBy,
	}
}

func unionStrings(left []string, right []string) []string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}

	result := make([]string, 0, len(left)+len(right))
	seen := make(map[string]struct{}, len(left)+len(right))
	for _, value := range append(cloneStrings(left), right...) {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	return Snapshot{
		CORSOrigins: ListSnapshot{
			Managed:   cloneStrings(snapshot.CORSOrigins.Managed),
			Inherited: cloneStrings(snapshot.CORSOrigins.Inherited),
			Effective: cloneStrings(snapshot.CORSOrigins.Effective),
		},
		ExternalAPIAllowHosts: ListSnapshot{
			Managed:   cloneStrings(snapshot.ExternalAPIAllowHosts.Managed),
			Inherited: cloneStrings(snapshot.ExternalAPIAllowHosts.Inherited),
			Effective: cloneStrings(snapshot.ExternalAPIAllowHosts.Effective),
		},
		UpdatedAt: snapshot.UpdatedAt,
		UpdatedBy: snapshot.UpdatedBy,
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}
