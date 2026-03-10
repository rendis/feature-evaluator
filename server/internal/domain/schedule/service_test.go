package schedule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// mockScheduleRepo implements Repository for testing.
type mockScheduleRepo struct {
	createFn           func(ctx context.Context, sc *ScheduledChange) error
	getByIDFn          func(ctx context.Context, id string) (*ScheduledChange, error)
	deleteFn           func(ctx context.Context, id string) error
	listByFeatureFn    func(ctx context.Context, featureKey string) ([]ScheduledChange, error)
	claimNextPendingFn func(ctx context.Context) (*ScheduledChange, error)
	setCompletedFn     func(ctx context.Context, id string) error
	setFailedFn        func(ctx context.Context, id string, errMsg string) error
}

func (m *mockScheduleRepo) Create(ctx context.Context, sc *ScheduledChange) error {
	if m.createFn != nil {
		return m.createFn(ctx, sc)
	}
	return nil
}

func (m *mockScheduleRepo) GetByID(ctx context.Context, id string) (*ScheduledChange, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockScheduleRepo) ListByFeature(ctx context.Context, featureKey string) ([]ScheduledChange, error) {
	if m.listByFeatureFn != nil {
		return m.listByFeatureFn(ctx, featureKey)
	}
	return nil, nil
}

func (m *mockScheduleRepo) ClaimNextPending(ctx context.Context) (*ScheduledChange, error) {
	if m.claimNextPendingFn != nil {
		return m.claimNextPendingFn(ctx)
	}
	return nil, nil
}

func (m *mockScheduleRepo) SetCompleted(ctx context.Context, id string) error {
	if m.setCompletedFn != nil {
		return m.setCompletedFn(ctx, id)
	}
	return nil
}

func (m *mockScheduleRepo) SetFailed(ctx context.Context, id string, errMsg string) error {
	if m.setFailedFn != nil {
		return m.setFailedFn(ctx, id, errMsg)
	}
	return nil
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	var created *ScheduledChange
	repo := &mockScheduleRepo{
		createFn: func(_ context.Context, sc *ScheduledChange) error {
			created = sc
			return nil
		},
	}

	svc := NewService(repo)
	sc := &ScheduledChange{
		FeatureKey:  "my-feature",
		ScheduledAt: time.Now().UTC().Add(1 * time.Hour),
		ChangeType:  ChangeToggle,
		Payload:     map[string]any{"enabled": true},
	}

	err := svc.Create(context.Background(), sc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if created.Status != StatusPending {
		t.Errorf("expected status %q, got %q", StatusPending, created.Status)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestCreate_EmptyFeatureKey(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockScheduleRepo{})
	sc := &ScheduledChange{
		FeatureKey:  "",
		ScheduledAt: time.Now().UTC().Add(1 * time.Hour),
		ChangeType:  ChangeToggle,
	}

	err := svc.Create(context.Background(), sc)
	if err == nil {
		t.Fatal("expected error for empty featureKey")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "featureKeyRequired") {
		t.Errorf("expected messageKey containing 'featureKeyRequired', got %q", apiErr.MessageKey)
	}
}

func TestCreate_ZeroScheduledAt(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockScheduleRepo{})
	sc := &ScheduledChange{
		FeatureKey: "my-feature",
		ChangeType: ChangeToggle,
	}

	err := svc.Create(context.Background(), sc)
	if err == nil {
		t.Fatal("expected error for zero scheduledAt")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "scheduledAtRequired") {
		t.Errorf("expected messageKey containing 'scheduledAtRequired', got %q", apiErr.MessageKey)
	}
}

func TestCreate_PastScheduledAt(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockScheduleRepo{})
	sc := &ScheduledChange{
		FeatureKey:  "my-feature",
		ScheduledAt: time.Now().UTC().Add(-1 * time.Hour),
		ChangeType:  ChangeToggle,
	}

	err := svc.Create(context.Background(), sc)
	if err == nil {
		t.Fatal("expected error for past scheduledAt")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "scheduledAtPast") {
		t.Errorf("expected messageKey containing 'scheduledAtPast', got %q", apiErr.MessageKey)
	}
}

func TestCreate_EmptyChangeType(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockScheduleRepo{})
	sc := &ScheduledChange{
		FeatureKey:  "my-feature",
		ScheduledAt: time.Now().UTC().Add(1 * time.Hour),
		ChangeType:  "",
	}

	err := svc.Create(context.Background(), sc)
	if err == nil {
		t.Fatal("expected error for empty changeType")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "changeTypeRequired") {
		t.Errorf("expected messageKey containing 'changeTypeRequired', got %q", apiErr.MessageKey)
	}
}

func TestCreate_InvalidChangeType(t *testing.T) {
	t.Parallel()

	svc := NewService(&mockScheduleRepo{})
	sc := &ScheduledChange{
		FeatureKey:  "my-feature",
		ScheduledAt: time.Now().UTC().Add(1 * time.Hour),
		ChangeType:  ChangeType("badtype"),
	}

	err := svc.Create(context.Background(), sc)
	if err == nil {
		t.Fatal("expected error for invalid changeType")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "invalidChangeType") {
		t.Errorf("expected messageKey containing 'invalidChangeType', got %q", apiErr.MessageKey)
	}
}

func TestCreate_AllValidChangeTypes(t *testing.T) {
	t.Parallel()

	types := []struct {
		name string
		ct   ChangeType
	}{
		{"toggle", ChangeToggle},
		{"update", ChangeUpdate},
		{"default_value", ChangeDefaultVal},
		{"environment", ChangeEnvironment},
	}

	for _, tc := range types {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockScheduleRepo{
				createFn: func(_ context.Context, _ *ScheduledChange) error {
					return nil
				},
			}
			svc := NewService(repo)
			sc := &ScheduledChange{
				FeatureKey:  "my-feature",
				ScheduledAt: time.Now().UTC().Add(1 * time.Hour),
				ChangeType:  tc.ct,
			}

			err := svc.Create(context.Background(), sc)
			if err != nil {
				t.Errorf("expected no error for changeType %q, got %v", tc.ct, err)
			}
		})
	}
}

func TestCancel_Success(t *testing.T) {
	t.Parallel()

	var deletedID string
	repo := &mockScheduleRepo{
		getByIDFn: func(_ context.Context, id string) (*ScheduledChange, error) {
			return &ScheduledChange{ID: id, Status: StatusPending}, nil
		},
		deleteFn: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}

	svc := NewService(repo)
	err := svc.Cancel(context.Background(), "sched-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if deletedID != "sched-1" {
		t.Errorf("expected repo.Delete called with 'sched-1', got %q", deletedID)
	}
}

func TestCancel_NotPending(t *testing.T) {
	t.Parallel()

	repo := &mockScheduleRepo{
		getByIDFn: func(_ context.Context, id string) (*ScheduledChange, error) {
			return &ScheduledChange{ID: id, Status: StatusCompleted}, nil
		},
	}

	svc := NewService(repo)
	err := svc.Cancel(context.Background(), "sched-1")
	if err == nil {
		t.Fatal("expected error for non-pending schedule")
	}
	apiErr, ok := err.(*apierror.APIError)
	if !ok {
		t.Fatalf("expected *apierror.APIError, got %T", err)
	}
	if !strings.Contains(apiErr.MessageKey, "scheduleNotCancellable") {
		t.Errorf("expected messageKey containing 'scheduleNotCancellable', got %q", apiErr.MessageKey)
	}
}

func TestCancel_NotFound(t *testing.T) {
	t.Parallel()

	wantErr := apierror.NewNotFound("not found", "error.notFound")
	repo := &mockScheduleRepo{
		getByIDFn: func(_ context.Context, _ string) (*ScheduledChange, error) {
			return nil, wantErr
		},
	}

	svc := NewService(repo)
	err := svc.Cancel(context.Background(), "missing")
	if err != wantErr {
		t.Errorf("expected error to be propagated, got %v", err)
	}
}

func TestListByFeature_Success(t *testing.T) {
	t.Parallel()

	want := []ScheduledChange{
		{ID: "1", FeatureKey: "feat-1", Status: StatusPending},
		{ID: "2", FeatureKey: "feat-1", Status: StatusCompleted},
	}

	repo := &mockScheduleRepo{
		listByFeatureFn: func(_ context.Context, _ string) ([]ScheduledChange, error) {
			return want, nil
		},
	}

	svc := NewService(repo)
	got, err := svc.ListByFeature(context.Background(), "feat-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != len(want) {
		t.Errorf("expected %d schedules, got %d", len(want), len(got))
	}
	for i, s := range got {
		if s.ID != want[i].ID {
			t.Errorf("schedule[%d]: expected ID %q, got %q", i, want[i].ID, s.ID)
		}
	}
}
