package experiment

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ===========================================================================
// Mock: experiment Repository
// ===========================================================================

type mockExperimentRepo struct {
	experiments map[string]*Experiment
	running     map[string]*Experiment // featureKey -> experiment
	createErr   error
	getErr      error
	updateErr   error
}

func newMockExperimentRepo() *mockExperimentRepo {
	return &mockExperimentRepo{
		experiments: make(map[string]*Experiment),
		running:     make(map[string]*Experiment),
	}
}

func (m *mockExperimentRepo) Create(_ context.Context, exp *Experiment) error {
	if m.createErr != nil {
		return m.createErr
	}
	if exp.ID == "" {
		exp.ID = "generated-id"
	}
	m.experiments[exp.ID] = exp
	return nil
}

func (m *mockExperimentRepo) GetByID(_ context.Context, id string) (*Experiment, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	exp, ok := m.experiments[id]
	if !ok {
		return nil, apierror.NewNotFound("experiment not found", "error.experimentNotFound")
	}
	// Return a copy to avoid mutation issues.
	cp := *exp
	return &cp, nil
}

func (m *mockExperimentRepo) Update(_ context.Context, exp *Experiment) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.experiments[exp.ID] = exp
	return nil
}

func (m *mockExperimentRepo) List(_ context.Context) ([]Experiment, error) {
	out := make([]Experiment, 0, len(m.experiments))
	for _, e := range m.experiments {
		out = append(out, *e)
	}
	return out, nil
}

func (m *mockExperimentRepo) FindRunningByFeatureKey(_ context.Context, featureKey string) (*Experiment, error) {
	exp, ok := m.running[featureKey]
	if !ok {
		return nil, nil
	}
	return exp, nil
}

// ===========================================================================
// Mock: ExposureRepository
// ===========================================================================

type mockExposureRepo struct {
	exposures      map[string]*Exposure // "expID:userID" -> exposure
	upsertErr      error
	countByVariant map[string]int64 // variantKey -> count
}

func newMockExposureRepo() *mockExposureRepo {
	return &mockExposureRepo{
		exposures:      make(map[string]*Exposure),
		countByVariant: make(map[string]int64),
	}
}

func (m *mockExposureRepo) Upsert(_ context.Context, exp *Exposure) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	key := exp.ExperimentID + ":" + exp.UserID
	m.exposures[key] = exp
	return nil
}

func (m *mockExposureRepo) Find(_ context.Context, experimentID, userID string) (*Exposure, error) {
	key := experimentID + ":" + userID
	exp, ok := m.exposures[key]
	if !ok {
		return nil, nil
	}
	return exp, nil
}

func (m *mockExposureRepo) CountByVariant(_ context.Context, _ string) (map[string]int64, error) {
	return m.countByVariant, nil
}

// ===========================================================================
// Mock: ConversionRepository
// ===========================================================================

type mockConversionRepo struct {
	conversions    []*Conversion
	createErr      error
	countByVariant map[string]int64 // variantKey -> count
}

func newMockConversionRepo() *mockConversionRepo {
	return &mockConversionRepo{
		countByVariant: make(map[string]int64),
	}
}

func (m *mockConversionRepo) Create(_ context.Context, conv *Conversion) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.conversions = append(m.conversions, conv)
	return nil
}

func (m *mockConversionRepo) CountByVariant(_ context.Context, _, _ string) (map[string]int64, error) {
	return m.countByVariant, nil
}

// ===========================================================================
// Mock: Cache
// ===========================================================================

type mockCache struct {
	data      map[string]*Experiment // "wsKey:featureKey" -> exp
	getCalled bool
	setCalled bool
	lastTTL   time.Duration
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]*Experiment),
	}
}

func (m *mockCache) GetRunning(_ context.Context, workspaceKey, featureKey string) (*Experiment, bool) {
	m.getCalled = true
	exp, ok := m.data[workspaceKey+":"+featureKey]
	return exp, ok
}

func (m *mockCache) SetRunning(_ context.Context, workspaceKey, featureKey string, exp *Experiment, ttl time.Duration) {
	m.setCalled = true
	m.lastTTL = ttl
	m.data[workspaceKey+":"+featureKey] = exp
}

func (m *mockCache) Invalidate(_ context.Context, workspaceKey, featureKey string) {
	delete(m.data, workspaceKey+":"+featureKey)
}

// ===========================================================================
// Mock: feature.Repository (for building *feature.Service)
// ===========================================================================

type mockFeatureRepo struct {
	features map[string]*feature.Feature // key -> feature
	getErr   error
	updateFn func(ctx context.Context, f *feature.Feature) error
}

func newMockFeatureRepo() *mockFeatureRepo {
	return &mockFeatureRepo{
		features: make(map[string]*feature.Feature),
	}
}

func (m *mockFeatureRepo) Create(_ context.Context, f *feature.Feature) error {
	m.features[f.Key] = f
	return nil
}

func (m *mockFeatureRepo) GetByKey(_ context.Context, key string) (*feature.Feature, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	f, ok := m.features[key]
	if !ok {
		return nil, apierror.NewNotFound("feature not found", "error.featureNotFound")
	}
	cp := *f
	return &cp, nil
}

func (m *mockFeatureRepo) Update(ctx context.Context, f *feature.Feature) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, f)
	}
	m.features[f.Key] = f
	return nil
}

func (m *mockFeatureRepo) Delete(_ context.Context, key string) error {
	delete(m.features, key)
	return nil
}

func (m *mockFeatureRepo) List(_ context.Context, _ feature.ListParams) (*feature.ListResult, error) {
	return &feature.ListResult{}, nil
}

func (m *mockFeatureRepo) ListEnabled(_ context.Context) ([]feature.Feature, error) {
	return nil, nil
}

func (m *mockFeatureRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}

func (m *mockFeatureRepo) AddRule(_ context.Context, _ string, _ *feature.Rule) error {
	return nil
}

func (m *mockFeatureRepo) UpdateRule(_ context.Context, _ string, _ *feature.Rule) error {
	return nil
}

func (m *mockFeatureRepo) DeleteRule(_ context.Context, _ string, _ string) error {
	return nil
}

func (m *mockFeatureRepo) ReorderRules(_ context.Context, _ string, _ []string) error {
	return nil
}

// ===========================================================================
// Helpers
// ===========================================================================

func validVariants() []Variant {
	return []Variant{
		{Key: "control", Value: false, Weight: 50},
		{Key: "treatment", Value: true, Weight: 50},
	}
}

func newTestService(
	repo *mockExperimentRepo,
	expRepo *mockExposureRepo,
	convRepo *mockConversionRepo,
	featureRepo *mockFeatureRepo,
	cache Cache,
) *Service {
	featureSvc := feature.NewService(featureRepo)
	return NewService(repo, expRepo, convRepo, featureSvc, cache)
}

func ctxWithWorkspace() context.Context {
	return workspace.WithKey(context.Background(), "ws-test")
}

// requireAPIError asserts the error is an *apierror.APIError whose MessageKey contains wantKey.
func requireAPIError(t *testing.T, err error, wantKey string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantKey)
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *apierror.APIError, got %T: %v", err, err)
	}
	if !strings.Contains(apiErr.MessageKey, wantKey) {
		t.Fatalf("expected MessageKey containing %q, got %q", wantKey, apiErr.MessageKey)
	}
}

// ===========================================================================
// Create tests
// ===========================================================================

func TestCreate_Success(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{Key: "feat-1", ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:         "Test Experiment",
		FeatureKey:   "feat-1",
		WorkspaceKey: "ws-test",
		Variants:     validVariants(),
	}
	if err := svc.Create(context.Background(), exp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.Status != StatusDraft {
		t.Fatalf("expected status draft, got %s", exp.Status)
	}
	if exp.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if exp.Metrics == nil {
		t.Fatal("expected Metrics to be non-nil (empty slice)")
	}
	if len(exp.Metrics) != 0 {
		t.Fatalf("expected empty Metrics, got %d", len(exp.Metrics))
	}
}

func TestCreate_EmptyName(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{Key: "feat-1", ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:       "",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
	}
	err := svc.Create(context.Background(), exp)
	requireAPIError(t, err, "experimentNameRequired")
}

func TestCreate_EmptyFeatureKey(t *testing.T) {
	t.Parallel()
	svc := newTestService(newMockExperimentRepo(), newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	exp := &Experiment{
		Name:     "Test",
		Variants: validVariants(),
	}
	err := svc.Create(context.Background(), exp)
	requireAPIError(t, err, "featureKeyRequired")
}

func TestCreate_TooFewVariants(t *testing.T) {
	t.Parallel()
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{Key: "feat-1", ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}
	svc := newTestService(newMockExperimentRepo(), newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   []Variant{{Key: "only", Weight: 100}},
	}
	err := svc.Create(context.Background(), exp)
	requireAPIError(t, err, "experimentMinVariants")
}

func TestCreate_InvalidVariants_DuplicateKeys(t *testing.T) {
	t.Parallel()
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{Key: "feat-1", ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}
	svc := newTestService(newMockExperimentRepo(), newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants: []Variant{
			{Key: "a", Weight: 50},
			{Key: "a", Weight: 50},
		},
	}
	err := svc.Create(context.Background(), exp)
	if err == nil {
		t.Fatal("expected error for duplicate variant keys")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected 'duplicate' in error, got: %s", err.Error())
	}
}

func TestCreate_FeatureNotFound(t *testing.T) {
	t.Parallel()
	fr := newMockFeatureRepo() // empty — no features
	svc := newTestService(newMockExperimentRepo(), newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:       "Test",
		FeatureKey: "nonexistent",
		Variants:   validVariants(),
	}
	err := svc.Create(context.Background(), exp)
	if err == nil {
		t.Fatal("expected error when feature not found")
	}
}

func TestCreate_AlreadyRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.running["feat-1"] = &Experiment{ID: "existing-exp", FeatureKey: "feat-1", Status: StatusRunning}
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{Key: "feat-1", ValueType: feature.ValueTypeBoolean, AccessPolicy: feature.AccessPolicyPublic}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	exp := &Experiment{
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
	}
	err := svc.Create(context.Background(), exp)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 409 {
		t.Fatalf("expected 409 conflict, got: %v", err)
	}
}

// ===========================================================================
// State transition tests
// ===========================================================================

func seedDraftExperiment(repo *mockExperimentRepo) {
	exp := &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusDraft,
		CreatedAt:  time.Now().UTC(),
	}
	repo.experiments[exp.ID] = exp
}

func TestStart_FromDraft(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	seedDraftExperiment(repo)
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	if err := svc.Start(ctxWithWorkspace(), "exp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exp := repo.experiments["exp-1"]
	if exp.Status != StatusRunning {
		t.Fatalf("expected running, got %s", exp.Status)
	}
	if exp.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
}

func TestStart_FromPaused(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	startedAt := time.Now().UTC().Add(-1 * time.Hour)
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusPaused,
		StartedAt:  &startedAt,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	if err := svc.Start(ctxWithWorkspace(), "exp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exp := repo.experiments["exp-1"]
	if exp.Status != StatusRunning {
		t.Fatalf("expected running, got %s", exp.Status)
	}
	// StartedAt should NOT be overwritten.
	if !exp.StartedAt.Equal(startedAt) {
		t.Fatalf("StartedAt should not be overwritten: want %v, got %v", startedAt, *exp.StartedAt)
	}
}

func TestStart_FromCompleted(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusCompleted,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.Start(ctxWithWorkspace(), "exp-1")
	requireAPIError(t, err, "experimentInvalidTransition")
}

func TestStart_AnotherRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	seedDraftExperiment(repo)
	repo.running["feat-1"] = &Experiment{ID: "other-exp", FeatureKey: "feat-1", Status: StatusRunning}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.Start(ctxWithWorkspace(), "exp-1")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) || apiErr.HTTPStatus != 409 {
		t.Fatalf("expected 409 conflict, got: %v", err)
	}
}

func TestPause_FromRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusRunning,
		StartedAt:  &now,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	if err := svc.Pause(ctxWithWorkspace(), "exp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.experiments["exp-1"].Status != StatusPaused {
		t.Fatalf("expected paused, got %s", repo.experiments["exp-1"].Status)
	}
}

func TestPause_NotRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	seedDraftExperiment(repo)
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.Pause(ctxWithWorkspace(), "exp-1")
	if err == nil {
		t.Fatal("expected error pausing non-running experiment")
	}
}

func TestComplete_FromRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusRunning,
		StartedAt:  &now,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	if err := svc.Complete(ctxWithWorkspace(), "exp-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exp := repo.experiments["exp-1"]
	if exp.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", exp.Status)
	}
	if exp.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestComplete_FromDraft(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	seedDraftExperiment(repo)
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.Complete(ctxWithWorkspace(), "exp-1")
	requireAPIError(t, err, "experimentInvalidTransition")
}

// ===========================================================================
// DeclareWinner tests
// ===========================================================================

func TestDeclareWinner_Success(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	completedAt := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:          "exp-1",
		Name:        "Test",
		FeatureKey:  "feat-1",
		Variants:    validVariants(),
		Status:      StatusCompleted,
		CompletedAt: &completedAt,
	}
	fr := newMockFeatureRepo()
	fr.features["feat-1"] = &feature.Feature{
		Key:          "feat-1",
		ValueType:    feature.ValueTypeBoolean,
		AccessPolicy: feature.AccessPolicyPublic,
		DefaultValue: false,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), fr, nil)

	if err := svc.DeclareWinner(context.Background(), "exp-1", "treatment"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check experiment winner key.
	exp := repo.experiments["exp-1"]
	if exp.WinnerKey != "treatment" {
		t.Fatalf("expected WinnerKey='treatment', got %q", exp.WinnerKey)
	}
	// Check feature default value was updated.
	f := fr.features["feat-1"]
	if f.DefaultValue != true {
		t.Fatalf("expected feature DefaultValue=true, got %v", f.DefaultValue)
	}
}

func TestDeclareWinner_NotCompleted(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusRunning,
		StartedAt:  &now,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.DeclareWinner(context.Background(), "exp-1", "treatment")
	requireAPIError(t, err, "experimentNotCompleted")
}

func TestDeclareWinner_VariantNotFound(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	completedAt := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:          "exp-1",
		Name:        "Test",
		FeatureKey:  "feat-1",
		Variants:    validVariants(),
		Status:      StatusCompleted,
		CompletedAt: &completedAt,
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.DeclareWinner(context.Background(), "exp-1", "nonexistent")
	requireAPIError(t, err, "experimentVariantNotFound")
}

// ===========================================================================
// RecordConversion tests
// ===========================================================================

func TestRecordConversion_Success(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:         "exp-1",
		Name:       "Test",
		FeatureKey: "feat-1",
		Variants:   validVariants(),
		Status:     StatusRunning,
		StartedAt:  &now,
		Metrics:    []Metric{{Key: "click", Name: "Click Rate"}},
	}

	exposureRepo := newMockExposureRepo()
	exposureRepo.exposures["exp-1:user-42"] = &Exposure{
		ExperimentID: "exp-1",
		UserID:       "user-42",
		VariantKey:   "treatment",
	}

	convRepo := newMockConversionRepo()
	svc := newTestService(repo, exposureRepo, convRepo, newMockFeatureRepo(), nil)

	conv := &Conversion{
		ExperimentID: "exp-1",
		UserID:       "user-42",
		MetricKey:    "click",
		Value:        1.0,
	}
	if err := svc.RecordConversion(context.Background(), conv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conv.VariantKey != "treatment" {
		t.Fatalf("expected VariantKey='treatment', got %q", conv.VariantKey)
	}
	if len(convRepo.conversions) != 1 {
		t.Fatalf("expected 1 conversion recorded, got %d", len(convRepo.conversions))
	}
}

func TestRecordConversion_NotRunning(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.experiments["exp-1"] = &Experiment{
		ID:       "exp-1",
		Status:   StatusDraft,
		Metrics:  []Metric{{Key: "click", Name: "Click Rate"}},
		Variants: validVariants(),
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.RecordConversion(context.Background(), &Conversion{
		ExperimentID: "exp-1",
		UserID:       "user-42",
		MetricKey:    "click",
	})
	requireAPIError(t, err, "experimentNotRunning")
}

func TestRecordConversion_MetricNotFound(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:        "exp-1",
		Status:    StatusRunning,
		StartedAt: &now,
		Metrics:   []Metric{{Key: "click", Name: "Click Rate"}},
		Variants:  validVariants(),
	}
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.RecordConversion(context.Background(), &Conversion{
		ExperimentID: "exp-1",
		UserID:       "user-42",
		MetricKey:    "nonexistent",
	})
	requireAPIError(t, err, "experimentMetricNotFound")
}

func TestRecordConversion_NoExposure(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	now := time.Now().UTC()
	repo.experiments["exp-1"] = &Experiment{
		ID:        "exp-1",
		Status:    StatusRunning,
		StartedAt: &now,
		Metrics:   []Metric{{Key: "click", Name: "Click Rate"}},
		Variants:  validVariants(),
	}
	// Empty exposure repo — no exposure for user.
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	err := svc.RecordConversion(context.Background(), &Conversion{
		ExperimentID: "exp-1",
		UserID:       "user-42",
		MetricKey:    "click",
	})
	requireAPIError(t, err, "experimentNoExposure")
}

// ===========================================================================
// GetResults tests
// ===========================================================================

func TestGetResults_WithConversions(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.experiments["exp-1"] = &Experiment{
		ID:       "exp-1",
		Variants: validVariants(),
		Metrics:  []Metric{{Key: "click", Name: "Click Rate"}},
	}
	exposureRepo := newMockExposureRepo()
	exposureRepo.countByVariant = map[string]int64{
		"control":   500,
		"treatment": 500,
	}
	convRepo := newMockConversionRepo()
	convRepo.countByVariant = map[string]int64{
		"control":   50,
		"treatment": 100,
	}
	svc := newTestService(repo, exposureRepo, convRepo, newMockFeatureRepo(), nil)

	results, err := svc.GetResults(context.Background(), "exp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results.TotalExposures != 1000 {
		t.Fatalf("expected 1000 total exposures, got %d", results.TotalExposures)
	}
	if results.TotalConversions != 150 {
		t.Fatalf("expected 150 total conversions, got %d", results.TotalConversions)
	}
	if len(results.Variants) != 2 {
		t.Fatalf("expected 2 variant stats, got %d", len(results.Variants))
	}
	// Check conversion rates.
	for _, vs := range results.Variants {
		switch vs.VariantKey {
		case "control":
			if vs.ConversionRate != 0.1 {
				t.Errorf("control: expected rate 0.1, got %f", vs.ConversionRate)
			}
		case "treatment":
			if vs.ConversionRate != 0.2 {
				t.Errorf("treatment: expected rate 0.2, got %f", vs.ConversionRate)
			}
		}
	}
}

func TestGetResults_Significance(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.experiments["exp-1"] = &Experiment{
		ID:       "exp-1",
		Variants: validVariants(),
		Metrics:  []Metric{{Key: "click", Name: "Click Rate"}},
	}
	exposureRepo := newMockExposureRepo()
	exposureRepo.countByVariant = map[string]int64{
		"control":   10000,
		"treatment": 10000,
	}
	// Large difference: 1% vs 10% with 10K samples each -> non-overlapping CIs.
	convRepo := newMockConversionRepo()
	convRepo.countByVariant = map[string]int64{
		"control":   100,
		"treatment": 1000,
	}
	svc := newTestService(repo, exposureRepo, convRepo, newMockFeatureRepo(), nil)

	results, err := svc.GetResults(context.Background(), "exp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !results.IsSignificant {
		t.Fatal("expected significant result for large difference with large sample")
	}
}

func TestGetResults_NotSignificant(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.experiments["exp-1"] = &Experiment{
		ID:       "exp-1",
		Variants: validVariants(),
		Metrics:  []Metric{{Key: "click", Name: "Click Rate"}},
	}
	exposureRepo := newMockExposureRepo()
	exposureRepo.countByVariant = map[string]int64{
		"control":   10,
		"treatment": 10,
	}
	// Small sample, similar rates -> overlapping CIs.
	convRepo := newMockConversionRepo()
	convRepo.countByVariant = map[string]int64{
		"control":   5,
		"treatment": 6,
	}
	svc := newTestService(repo, exposureRepo, convRepo, newMockFeatureRepo(), nil)

	results, err := svc.GetResults(context.Background(), "exp-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results.IsSignificant {
		t.Fatal("expected not significant for small sample with similar rates")
	}
}

// ===========================================================================
// Cache tests
// ===========================================================================

func TestFindRunning_CacheHit(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	cache := newMockCache()
	cachedExp := &Experiment{ID: "cached-exp", FeatureKey: "feat-1", Status: StatusRunning, LookupCacheEnabled: true, LookupCacheTTLSeconds: 60}
	cache.data["ws-test:feat-1"] = cachedExp

	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), cache)

	ctx := ctxWithWorkspace()
	exp, err := svc.FindRunningByFeatureKey(ctx, "feat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil || exp.ID != "cached-exp" {
		t.Fatalf("expected cached experiment, got %v", exp)
	}
	if !cache.getCalled {
		t.Fatal("expected cache.GetRunning to be called")
	}
	// Repo should NOT have been called — verify by checking that the repo has no experiments
	// but we still got a result from cache.
	if len(repo.experiments) != 0 {
		t.Fatal("expected repo to not be populated (cache hit)")
	}
}

func TestFindRunning_CacheMiss(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.running["feat-1"] = &Experiment{ID: "repo-exp", FeatureKey: "feat-1", Status: StatusRunning, LookupCacheEnabled: true, LookupCacheTTLSeconds: 60}
	cache := newMockCache()

	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), cache)

	ctx := ctxWithWorkspace()
	exp, err := svc.FindRunningByFeatureKey(ctx, "feat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil || exp.ID != "repo-exp" {
		t.Fatalf("expected repo experiment, got %v", exp)
	}
	if !cache.getCalled {
		t.Fatal("expected cache.GetRunning to be called")
	}
	if !cache.setCalled {
		t.Fatal("expected cache.SetRunning to be called after miss")
	}
	if cache.lastTTL != 60*time.Second {
		t.Fatalf("lastTTL = %s, want 60s", cache.lastTTL)
	}
}

func TestFindRunning_NilCache(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.running["feat-1"] = &Experiment{ID: "repo-exp", FeatureKey: "feat-1", Status: StatusRunning}

	// Pass nil for cache.
	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), nil)

	ctx := ctxWithWorkspace()
	exp, err := svc.FindRunningByFeatureKey(ctx, "feat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil || exp.ID != "repo-exp" {
		t.Fatalf("expected repo experiment, got %v", exp)
	}
}

func TestFindRunning_DoesNotWriteCacheWhenDisabled(t *testing.T) {
	t.Parallel()
	repo := newMockExperimentRepo()
	repo.running["feat-1"] = &Experiment{
		ID:                    "repo-exp",
		FeatureKey:            "feat-1",
		Status:                StatusRunning,
		LookupCacheEnabled:    false,
		LookupCacheTTLSeconds: 60,
	}
	cache := newMockCache()

	svc := newTestService(repo, newMockExposureRepo(), newMockConversionRepo(), newMockFeatureRepo(), cache)

	ctx := ctxWithWorkspace()
	exp, err := svc.FindRunningByFeatureKey(ctx, "feat-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp == nil || exp.ID != "repo-exp" {
		t.Fatalf("expected repo experiment, got %v", exp)
	}
	if !cache.getCalled {
		t.Fatal("expected cache.GetRunning to be called")
	}
	if cache.setCalled {
		t.Fatal("expected cache.SetRunning to be skipped when cache disabled")
	}
}
