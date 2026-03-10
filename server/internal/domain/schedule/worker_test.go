package schedule

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
)

// --- feature.Repository mock ---

type mockFeatureRepo struct {
	createFn       func(ctx context.Context, f *feature.Feature) error
	getByKeyFn     func(ctx context.Context, key string) (*feature.Feature, error)
	updateFn       func(ctx context.Context, f *feature.Feature) error
	deleteFn       func(ctx context.Context, key string) error
	listFn         func(ctx context.Context, params feature.ListParams) (*feature.ListResult, error)
	listEnabledFn  func(ctx context.Context) ([]feature.Feature, error)
	toggleFn       func(ctx context.Context, key string, enabled bool, updatedBy string) error
	addRuleFn      func(ctx context.Context, featureKey string, rule *feature.Rule) error
	updateRuleFn   func(ctx context.Context, featureKey string, rule *feature.Rule) error
	deleteRuleFn   func(ctx context.Context, featureKey string, ruleID string) error
	reorderRulesFn func(ctx context.Context, featureKey string, ruleIDs []string) error
}

func (m *mockFeatureRepo) Create(ctx context.Context, f *feature.Feature) error {
	if m.createFn != nil {
		return m.createFn(ctx, f)
	}
	return nil
}

func (m *mockFeatureRepo) GetByKey(ctx context.Context, key string) (*feature.Feature, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return &feature.Feature{Key: key, ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}, nil
}

func (m *mockFeatureRepo) Update(ctx context.Context, f *feature.Feature) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, f)
	}
	return nil
}

func (m *mockFeatureRepo) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockFeatureRepo) List(ctx context.Context, params feature.ListParams) (*feature.ListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return &feature.ListResult{}, nil
}

func (m *mockFeatureRepo) ListEnabled(ctx context.Context) ([]feature.Feature, error) {
	if m.listEnabledFn != nil {
		return m.listEnabledFn(ctx)
	}
	return nil, nil
}

func (m *mockFeatureRepo) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	if m.toggleFn != nil {
		return m.toggleFn(ctx, key, enabled, updatedBy)
	}
	return nil
}

func (m *mockFeatureRepo) AddRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	if m.addRuleFn != nil {
		return m.addRuleFn(ctx, featureKey, rule)
	}
	return nil
}

func (m *mockFeatureRepo) UpdateRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	if m.updateRuleFn != nil {
		return m.updateRuleFn(ctx, featureKey, rule)
	}
	return nil
}

func (m *mockFeatureRepo) DeleteRule(ctx context.Context, featureKey string, ruleID string) error {
	if m.deleteRuleFn != nil {
		return m.deleteRuleFn(ctx, featureKey, ruleID)
	}
	return nil
}

func (m *mockFeatureRepo) ReorderRules(ctx context.Context, featureKey string, ruleIDs []string) error {
	if m.reorderRulesFn != nil {
		return m.reorderRulesFn(ctx, featureKey, ruleIDs)
	}
	return nil
}

// --- changelog.Repository mock ---

type mockChangelogRepo struct {
	createFn       func(ctx context.Context, entry *changelog.ChangeEntry) error
	listFn         func(ctx context.Context, params changelog.ListParams) (*changelog.ListResult, error)
	listByEntityFn func(ctx context.Context, entityType, entityKey string, params changelog.ListParams) (*changelog.ListResult, error)
}

func (m *mockChangelogRepo) Create(ctx context.Context, entry *changelog.ChangeEntry) error {
	if m.createFn != nil {
		return m.createFn(ctx, entry)
	}
	return nil
}

func (m *mockChangelogRepo) List(ctx context.Context, params changelog.ListParams) (*changelog.ListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return &changelog.ListResult{}, nil
}

func (m *mockChangelogRepo) ListByEntity(ctx context.Context, entityType, entityKey string, params changelog.ListParams) (*changelog.ListResult, error) {
	if m.listByEntityFn != nil {
		return m.listByEntityFn(ctx, entityType, entityKey, params)
	}
	return &changelog.ListResult{}, nil
}

// --- helpers to build services from mocks ---

func newFeatureSvc(repo *mockFeatureRepo) *feature.Service {
	return feature.NewService(repo)
}

func newChangelogSvc(repo *mockChangelogRepo) *changelog.Service {
	return changelog.NewService(repo)
}

// --- applyChange tests ---

func TestApplyChange_Toggle(t *testing.T) {
	t.Parallel()

	var toggledKey string
	var toggledEnabled bool
	featRepo := &mockFeatureRepo{
		toggleFn: func(_ context.Context, key string, enabled bool, _ string) error {
			toggledKey = key
			toggledEnabled = enabled
			return nil
		},
	}

	w := &Worker{featureSvc: newFeatureSvc(featRepo)}
	sc := &ScheduledChange{
		FeatureKey: "feat-toggle",
		ChangeType: ChangeToggle,
		Payload:    map[string]any{"enabled": true},
	}

	err := w.applyChange(context.Background(), sc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if toggledKey != "feat-toggle" {
		t.Errorf("expected toggle key 'feat-toggle', got %q", toggledKey)
	}
	if !toggledEnabled {
		t.Error("expected toggle enabled=true")
	}
}

func TestApplyChange_Toggle_InvalidPayload(t *testing.T) {
	t.Parallel()

	w := &Worker{featureSvc: newFeatureSvc(&mockFeatureRepo{})}
	sc := &ScheduledChange{
		FeatureKey: "feat-toggle",
		ChangeType: ChangeToggle,
		Payload:    map[string]any{"enabled": "not-bool"},
	}

	err := w.applyChange(context.Background(), sc)
	if err == nil {
		t.Fatal("expected payloadError for non-bool enabled")
	}
	var pe *payloadError
	if !errors.As(err, &pe) {
		t.Errorf("expected *payloadError, got %T: %v", err, err)
	}
}

func TestApplyChange_DefaultValue(t *testing.T) {
	t.Parallel()

	var updatedFeature *feature.Feature
	featRepo := &mockFeatureRepo{
		getByKeyFn: func(_ context.Context, key string) (*feature.Feature, error) {
			return &feature.Feature{
				Key:          key,
				DefaultValue: "old-val",
				ValueType:    feature.ValueTypeString,
				AccessPolicy: feature.AccessPolicyPublic,
			}, nil
		},
		updateFn: func(_ context.Context, f *feature.Feature) error {
			updatedFeature = f
			return nil
		},
	}

	w := &Worker{featureSvc: newFeatureSvc(featRepo)}
	sc := &ScheduledChange{
		FeatureKey: "feat-dv",
		ChangeType: ChangeDefaultVal,
		Payload:    map[string]any{"defaultValue": "new-val"},
	}

	err := w.applyChange(context.Background(), sc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected feature to be updated")
	}
	if updatedFeature.DefaultValue != "new-val" {
		t.Errorf("expected defaultValue 'new-val', got %v", updatedFeature.DefaultValue)
	}
	if updatedFeature.UpdatedBy != "system:scheduler" {
		t.Errorf("expected updatedBy 'system:scheduler', got %q", updatedFeature.UpdatedBy)
	}
}

func TestApplyChange_Environment(t *testing.T) {
	t.Parallel()

	var updatedFeature *feature.Feature
	featRepo := &mockFeatureRepo{
		getByKeyFn: func(_ context.Context, key string) (*feature.Feature, error) {
			return &feature.Feature{
				Key:          key,
				Environments: []string{"production"},
				ValueType:    feature.ValueTypeBoolean,
				AccessPolicy: feature.AccessPolicyPublic,
			}, nil
		},
		updateFn: func(_ context.Context, f *feature.Feature) error {
			updatedFeature = f
			return nil
		},
	}

	w := &Worker{featureSvc: newFeatureSvc(featRepo)}
	sc := &ScheduledChange{
		FeatureKey: "feat-env",
		ChangeType: ChangeEnvironment,
		Payload:    map[string]any{"environments": []any{"dev", "uat"}},
	}

	err := w.applyChange(context.Background(), sc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected feature to be updated")
	}
	if len(updatedFeature.Environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(updatedFeature.Environments))
	}
	if updatedFeature.Environments[0] != "dev" || updatedFeature.Environments[1] != "uat" {
		t.Errorf("expected environments [dev, uat], got %v", updatedFeature.Environments)
	}
}

func TestApplyChange_Update_MultipleFields(t *testing.T) {
	t.Parallel()

	var updatedFeature *feature.Feature
	featRepo := &mockFeatureRepo{
		getByKeyFn: func(_ context.Context, key string) (*feature.Feature, error) {
			return &feature.Feature{
				Key:          key,
				Name:         "Old Name",
				Description:  "Old Desc",
				Enabled:      false,
				DefaultValue: "old",
				Environments: []string{"production"},
				ValueType:    feature.ValueTypeString,
				AccessPolicy: feature.AccessPolicyPublic,
			}, nil
		},
		updateFn: func(_ context.Context, f *feature.Feature) error {
			updatedFeature = f
			return nil
		},
	}

	w := &Worker{featureSvc: newFeatureSvc(featRepo)}
	sc := &ScheduledChange{
		FeatureKey: "feat-update",
		ChangeType: ChangeUpdate,
		Payload: map[string]any{
			"name":         "New Name",
			"description":  "New Desc",
			"enabled":      true,
			"defaultValue": "new",
			"environments": []any{"dev", "uat"},
		},
	}

	err := w.applyChange(context.Background(), sc)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updatedFeature == nil {
		t.Fatal("expected feature to be updated")
	}
	if updatedFeature.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updatedFeature.Name)
	}
	if updatedFeature.Description != "New Desc" {
		t.Errorf("expected description 'New Desc', got %q", updatedFeature.Description)
	}
	if !updatedFeature.Enabled {
		t.Error("expected enabled=true")
	}
	if updatedFeature.DefaultValue != "new" {
		t.Errorf("expected defaultValue 'new', got %v", updatedFeature.DefaultValue)
	}
	if len(updatedFeature.Environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(updatedFeature.Environments))
	}
	if updatedFeature.UpdatedBy != "system:scheduler" {
		t.Errorf("expected updatedBy 'system:scheduler', got %q", updatedFeature.UpdatedBy)
	}
}

func TestApplyChange_UnsupportedType(t *testing.T) {
	t.Parallel()

	w := &Worker{featureSvc: newFeatureSvc(&mockFeatureRepo{})}
	sc := &ScheduledChange{
		FeatureKey: "feat-x",
		ChangeType: ChangeType("invalid"),
		Payload:    map[string]any{},
	}

	err := w.applyChange(context.Background(), sc)
	if err == nil {
		t.Fatal("expected payloadError for unsupported changeType")
	}
	var pe *payloadError
	if !errors.As(err, &pe) {
		t.Errorf("expected *payloadError, got %T: %v", err, err)
	}
}

// --- execute tests ---

func TestExecute_Success(t *testing.T) {
	t.Parallel()

	var completedID string
	schedRepo := &mockScheduleRepo{
		setCompletedFn: func(_ context.Context, id string) error {
			completedID = id
			return nil
		},
	}
	featRepo := &mockFeatureRepo{
		toggleFn: func(_ context.Context, _ string, _ bool, _ string) error {
			return nil
		},
	}

	w := &Worker{
		repo:       schedRepo,
		featureSvc: newFeatureSvc(featRepo),
	}

	sc := &ScheduledChange{
		ID:           "sched-ok",
		WorkspaceKey: "ws-1",
		FeatureKey:   "feat-1",
		ChangeType:   ChangeToggle,
		Payload:      map[string]any{"enabled": true},
	}

	w.execute(sc)

	if completedID != "sched-ok" {
		t.Errorf("expected SetCompleted called with 'sched-ok', got %q", completedID)
	}
}

func TestExecute_Failure(t *testing.T) {
	t.Parallel()

	var failedID, failedMsg string
	schedRepo := &mockScheduleRepo{
		setFailedFn: func(_ context.Context, id string, errMsg string) error {
			failedID = id
			failedMsg = errMsg
			return nil
		},
	}
	// Toggle with invalid payload to force applyChange failure
	w := &Worker{
		repo:       schedRepo,
		featureSvc: newFeatureSvc(&mockFeatureRepo{}),
	}

	sc := &ScheduledChange{
		ID:           "sched-fail",
		WorkspaceKey: "ws-1",
		FeatureKey:   "feat-1",
		ChangeType:   ChangeToggle,
		Payload:      map[string]any{"enabled": "not-a-bool"},
	}

	w.execute(sc)

	if failedID != "sched-fail" {
		t.Errorf("expected SetFailed called with 'sched-fail', got %q", failedID)
	}
	if failedMsg == "" {
		t.Error("expected non-empty error message in SetFailed")
	}
}

// --- poll tests ---

func TestPoll_ProcessesMultiple(t *testing.T) {
	t.Parallel()

	callCount := 0
	var mu sync.Mutex
	var executedIDs []string

	sc1 := &ScheduledChange{
		ID:           "sc-1",
		WorkspaceKey: "ws-1",
		FeatureKey:   "feat-1",
		ChangeType:   ChangeToggle,
		Payload:      map[string]any{"enabled": true},
	}
	sc2 := &ScheduledChange{
		ID:           "sc-2",
		WorkspaceKey: "ws-1",
		FeatureKey:   "feat-2",
		ChangeType:   ChangeToggle,
		Payload:      map[string]any{"enabled": false},
	}

	schedRepo := &mockScheduleRepo{
		claimNextPendingFn: func(_ context.Context) (*ScheduledChange, error) {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			switch callCount {
			case 1:
				return sc1, nil
			case 2:
				return sc2, nil
			default:
				return nil, nil
			}
		},
		setCompletedFn: func(_ context.Context, id string) error {
			mu.Lock()
			defer mu.Unlock()
			executedIDs = append(executedIDs, id)
			return nil
		},
	}

	featRepo := &mockFeatureRepo{
		toggleFn: func(_ context.Context, _ string, _ bool, _ string) error {
			return nil
		},
	}

	w := &Worker{
		repo:       schedRepo,
		featureSvc: newFeatureSvc(featRepo),
	}

	w.poll()

	mu.Lock()
	defer mu.Unlock()
	if len(executedIDs) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executedIDs))
	}
	if executedIDs[0] != "sc-1" || executedIDs[1] != "sc-2" {
		t.Errorf("expected executed IDs [sc-1, sc-2], got %v", executedIDs)
	}
}

func TestPoll_NoPending(t *testing.T) {
	t.Parallel()

	claimCalled := false
	schedRepo := &mockScheduleRepo{
		claimNextPendingFn: func(_ context.Context) (*ScheduledChange, error) {
			claimCalled = true
			return nil, nil
		},
	}

	w := &Worker{
		repo:       schedRepo,
		featureSvc: newFeatureSvc(&mockFeatureRepo{}),
	}

	w.poll()

	if !claimCalled {
		t.Error("expected ClaimNextPending to be called")
	}
}

// --- toStringSlice tests ---

func TestToStringSlice_FromStringSlice(t *testing.T) {
	t.Parallel()

	input := []string{"a", "b"}
	result, ok := toStringSlice(input)
	if !ok {
		t.Fatal("expected ok=true for []string")
	}
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Errorf("expected [a, b], got %v", result)
	}
}

func TestToStringSlice_FromAnySlice(t *testing.T) {
	t.Parallel()

	input := []any{"a", "b"}
	result, ok := toStringSlice(input)
	if !ok {
		t.Fatal("expected ok=true for []any{string...}")
	}
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Errorf("expected [a, b], got %v", result)
	}
}

func TestToStringSlice_Unsupported(t *testing.T) {
	t.Parallel()

	_, ok := toStringSlice(42)
	if ok {
		t.Error("expected ok=false for int input")
	}
}

// --- recordChangelog tests ---

func TestRecordChangelog_Toggle(t *testing.T) {
	t.Parallel()

	var recorded *changelog.ChangeEntry
	clRepo := &mockChangelogRepo{
		createFn: func(_ context.Context, entry *changelog.ChangeEntry) error {
			recorded = entry
			return nil
		},
	}

	w := &Worker{changelogSvc: newChangelogSvc(clRepo)}
	sc := &ScheduledChange{
		ID:         "sc-1",
		FeatureKey: "feat-1",
		ChangeType: ChangeToggle,
	}

	w.recordChangelog(context.Background(), sc)

	if recorded == nil {
		t.Fatal("expected changelog entry to be recorded")
	}
	if recorded.Action != changelog.ActionToggle {
		t.Errorf("expected action %q, got %q", changelog.ActionToggle, recorded.Action)
	}
	if recorded.EntityKey != "feat-1" {
		t.Errorf("expected entityKey 'feat-1', got %q", recorded.EntityKey)
	}
	if recorded.Actor != "system:scheduler" {
		t.Errorf("expected actor 'system:scheduler', got %q", recorded.Actor)
	}
}

func TestRecordChangelog_Update(t *testing.T) {
	t.Parallel()

	var recorded *changelog.ChangeEntry
	clRepo := &mockChangelogRepo{
		createFn: func(_ context.Context, entry *changelog.ChangeEntry) error {
			recorded = entry
			return nil
		},
	}

	w := &Worker{changelogSvc: newChangelogSvc(clRepo)}
	sc := &ScheduledChange{
		ID:         "sc-2",
		FeatureKey: "feat-2",
		ChangeType: ChangeUpdate,
	}

	w.recordChangelog(context.Background(), sc)

	if recorded == nil {
		t.Fatal("expected changelog entry to be recorded")
	}
	if recorded.Action != changelog.ActionUpdate {
		t.Errorf("expected action %q, got %q", changelog.ActionUpdate, recorded.Action)
	}
}

func TestRecordChangelog_NilService(t *testing.T) {
	t.Parallel()

	w := &Worker{changelogSvc: nil}
	sc := &ScheduledChange{
		ID:         "sc-3",
		FeatureKey: "feat-3",
		ChangeType: ChangeToggle,
	}

	// Should not panic
	w.recordChangelog(context.Background(), sc)
}

// --- lifecycle tests ---

func TestStartStop(t *testing.T) {
	t.Parallel()

	schedRepo := &mockScheduleRepo{
		claimNextPendingFn: func(_ context.Context) (*ScheduledChange, error) {
			return nil, nil
		},
	}

	w := NewWorker(schedRepo, newFeatureSvc(&mockFeatureRepo{}), nil)
	w.Start()

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within timeout")
	}
}
