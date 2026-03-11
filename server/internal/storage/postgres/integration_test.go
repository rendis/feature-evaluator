package postgres

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/schedule"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
)

const defaultIntegrationDatabaseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" //nolint:gosec // test-only local credential

func TestFeatureRepoStoresRulesAndTags(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	tagSvc := tag.NewService(NewTagRepo(client))
	featureRepo := NewFeatureRepo(client)
	featureSvc := feature.NewService(featureRepo)

	createdTag, err := tagSvc.Create(ctx, "Beta Access", "#3B82F6", "tester")
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	input := &feature.Feature{
		Key:          "checkout_v2",
		Name:         "Checkout v2",
		Description:  "new checkout flow",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyPublic,
		Tags:         []string{createdTag.Key},
		CreatedBy:    "tester",
		UpdatedBy:    "tester",
	}
	if err := featureSvc.Create(ctx, input); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	rule := &feature.Rule{
		Name:       "beta cohort",
		Priority:   1,
		Enabled:    true,
		Expression: `user.cohort == "beta"`,
		Value:      true,
	}
	if err := featureSvc.AddRule(ctx, input.Key, rule); err != nil {
		t.Fatalf("add rule: %v", err)
	}

	stored, err := featureRepo.GetByKey(ctx, input.Key)
	if err != nil {
		t.Fatalf("get feature: %v", err)
	}

	if !reflect.DeepEqual(stored.Tags, []string{createdTag.Key}) {
		t.Fatalf("tags = %#v, want %#v", stored.Tags, []string{createdTag.Key})
	}
	if len(stored.Rules) != 1 {
		t.Fatalf("rules length = %d, want 1", len(stored.Rules))
	}
	if stored.Rules[0].Name != rule.Name {
		t.Fatalf("rule name = %q, want %q", stored.Rules[0].Name, rule.Name)
	}
	if stored.RolloutSalt == "" {
		t.Fatal("expected rollout salt to be generated")
	}
}

func TestFeatureRepoListSummaryLoadsCountsWithoutHydratingRules(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	featureRepo := NewFeatureRepo(client)
	featureSvc := feature.NewService(featureRepo)
	packRepo := NewPackRepo(client)
	now := time.Now().UTC()

	input := &feature.Feature{
		Key:          "summary_flag",
		Name:         "Summary Flag",
		Description:  "summary listing",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyPublic,
		CreatedBy:    "tester",
		UpdatedBy:    "tester",
	}
	if err := featureSvc.Create(ctx, input); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	rule := &feature.Rule{
		Name:       "summary rule",
		Priority:   1,
		Enabled:    true,
		Expression: `user.cohort == "beta"`,
		Value:      true,
	}
	if err := featureSvc.AddRule(ctx, input.Key, rule); err != nil {
		t.Fatalf("add rule: %v", err)
	}

	if err := packRepo.Create(ctx, &pack.Pack{
		Key:         "summary_pack",
		Name:        "Summary Pack",
		Description: "contains the summary flag",
		FeatureKeys: []string{input.Key},
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "tester",
		UpdatedBy:   "tester",
	}); err != nil {
		t.Fatalf("create pack: %v", err)
	}

	result, err := featureRepo.List(ctx, feature.ListParams{
		Page:     1,
		PageSize: 20,
		View:     feature.ListViewSummary,
	})
	if err != nil {
		t.Fatalf("list summary: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("summary length = %d, want 1", len(result.Data))
	}

	item := result.Data[0]
	if item.RuleCount != 1 {
		t.Fatalf("ruleCount = %d, want 1", item.RuleCount)
	}
	if item.PackCount != 1 {
		t.Fatalf("packCount = %d, want 1", item.PackCount)
	}
	if len(item.Rules) != 0 {
		t.Fatalf("rules length = %d, want 0", len(item.Rules))
	}
	if item.DefaultValue != nil {
		t.Fatalf("defaultValue = %#v, want nil for summary view", item.DefaultValue)
	}
}

func TestSegmentServiceReplaceRecords(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	segmentSvc := segment.NewService(NewSegmentRepo(client), NewSegmentRecordRepo(client), nil)
	segmentSvc.SetTxManager(client)

	seg := &segment.Segment{
		Key:         "beta_users",
		Name:        "Beta Users",
		Description: "eligible users",
		CreatedBy:   "tester",
		UpdatedBy:   "tester",
	}
	if err := segmentSvc.Create(ctx, seg); err != nil {
		t.Fatalf("create segment: %v", err)
	}

	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type":                 "object",
			"additionalProperties": true,
			"properties": map[string]any{
				"id":     map[string]any{"type": "string"},
				"cohort": map[string]any{"type": "string"},
			},
			"required": []any{"id", "cohort"},
		},
	}

	count, err := segmentSvc.ReplaceRecords(ctx, seg.Key, segment.ReplaceInput{
		SourceType:    segment.SourceTypeJSON,
		RecordKeyPath: "id",
		Schema:        schema,
		Records: []map[string]any{
			{"id": "user-1", "cohort": "beta"},
			{"id": "user-2", "cohort": "beta"},
		},
		UpdatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("replace records: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	isMember, err := segmentSvc.IsMember(ctx, seg.Key, "user-1", "")
	if err != nil {
		t.Fatalf("is member: %v", err)
	}
	if !isMember {
		t.Fatal("expected user-1 to be part of the segment")
	}

	stored, err := segmentSvc.GetByKey(ctx, seg.Key)
	if err != nil {
		t.Fatalf("get segment: %v", err)
	}
	if stored.RecordCount != 2 {
		t.Fatalf("recordCount = %d, want 2", stored.RecordCount)
	}
	if stored.ActiveDatasetVersion == "" {
		t.Fatal("expected active dataset version to be set")
	}
}

func TestPackActivationRepoFindActiveFeatureKeys(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	featureSvc := feature.NewService(NewFeatureRepo(client))
	now := time.Now().UTC()

	feat := &feature.Feature{
		Key:          "pack_flag",
		Name:         "Pack flag",
		Description:  "granted by pack",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyPublic,
		CreatedBy:    "tester",
		UpdatedBy:    "tester",
	}
	if err := featureSvc.Create(ctx, feat); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	packRepo := NewPackRepo(client)
	if err := packRepo.Create(ctx, &pack.Pack{
		Key:         "starter_pack",
		Name:        "Starter Pack",
		Description: "base entitlement",
		FeatureKeys: []string{feat.Key},
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   "tester",
		UpdatedBy:   "tester",
	}); err != nil {
		t.Fatalf("create pack: %v", err)
	}

	activationRepo := NewPackActivationRepo(client)
	if err := activationRepo.Create(ctx, &pack.Activation{
		PackKey:     "starter_pack",
		TargetType:  pack.TargetTenant,
		TargetID:    "tenant-1",
		ActivatedAt: now,
		ActivatedBy: "tester",
	}); err != nil {
		t.Fatalf("create activation: %v", err)
	}

	keys, err := activationRepo.FindActiveFeatureKeys(ctx, "tenant-1", "", "")
	if err != nil {
		t.Fatalf("find active feature keys: %v", err)
	}
	if !reflect.DeepEqual(keys, []string{feat.Key}) {
		t.Fatalf("feature keys = %#v, want %#v", keys, []string{feat.Key})
	}
}

func TestScheduleRepoClaimNextPending(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	featureSvc := feature.NewService(NewFeatureRepo(client))
	feat := &feature.Feature{
		Key:          "scheduled_flag",
		Name:         "Scheduled Flag",
		Description:  "used by scheduler",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyPublic,
		CreatedBy:    "tester",
		UpdatedBy:    "tester",
	}
	if err := featureSvc.Create(ctx, feat); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	scheduleRepo := NewScheduleRepo(client)
	change := &schedule.ScheduledChange{
		FeatureKey:  feat.Key,
		ChangeType:  schedule.ChangeToggle,
		Payload:     map[string]any{"enabled": true},
		ScheduledAt: time.Now().UTC().Add(-time.Minute),
		Status:      schedule.StatusPending,
		CreatedBy:   "tester",
		CreatedAt:   time.Now().UTC(),
	}
	if err := scheduleRepo.Create(ctx, change); err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	claimed, err := scheduleRepo.ClaimNextPending(ctx)
	if err != nil {
		t.Fatalf("claim pending schedule: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a pending schedule to be claimed")
	}
	if claimed.Status != schedule.StatusExecuting {
		t.Fatalf("claimed status = %q, want %q", claimed.Status, schedule.StatusExecuting)
	}

	if err := scheduleRepo.SetCompleted(ctx, claimed.ID); err != nil {
		t.Fatalf("set completed: %v", err)
	}

	next, err := scheduleRepo.ClaimNextPending(ctx)
	if err != nil {
		t.Fatalf("claim after completion: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no more pending schedules, got %#v", next)
	}
}

func TestExperimentServiceDeclareWinner(t *testing.T) {
	client, ctx := newIntegrationClient(t)
	defer client.Close()

	featureRepo := NewFeatureRepo(client)
	featureSvc := feature.NewService(featureRepo)
	feat := &feature.Feature{
		Key:          "experiment_flag",
		Name:         "Experiment Flag",
		Description:  "winner should update this",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyPublic,
		CreatedBy:    "tester",
		UpdatedBy:    "tester",
	}
	if err := featureSvc.Create(ctx, feat); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	experimentRepo := NewExperimentRepo(client)
	exp := &experiment.Experiment{
		FeatureKey:  feat.Key,
		Name:        "Checkout Variant Test",
		Description: "winner application",
		Variants: []experiment.Variant{
			{Key: "control", Value: false, Weight: 50},
			{Key: "winner", Value: true, Weight: 50},
		},
		Metrics: []experiment.Metric{
			{Key: "conversion", Name: "Conversion"},
		},
		Status:      experiment.StatusCompleted,
		CompletedAt: ptrTime(time.Now().UTC()),
		CreatedBy:   "tester",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := experimentRepo.Create(ctx, exp); err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	expSvc := experiment.NewService(
		experimentRepo,
		NewExposureRepo(client),
		NewConversionRepo(client),
		featureSvc,
		nil,
	)
	expSvc.SetTxManager(client)

	if err := expSvc.DeclareWinner(ctx, exp.ID, "winner"); err != nil {
		t.Fatalf("declare winner: %v", err)
	}

	storedFeature, err := featureRepo.GetByKey(ctx, feat.Key)
	if err != nil {
		t.Fatalf("get feature after winner: %v", err)
	}
	updatedDefault, ok := storedFeature.DefaultValue.(bool)
	if !ok || !updatedDefault {
		t.Fatalf("defaultValue = %#v, want true", storedFeature.DefaultValue)
	}

	storedExperiment, err := experimentRepo.GetByID(ctx, exp.ID)
	if err != nil {
		t.Fatalf("get experiment after winner: %v", err)
	}
	if storedExperiment.WinnerKey != "winner" {
		t.Fatalf("winnerKey = %q, want winner", storedExperiment.WinnerKey)
	}
}

func newIntegrationClient(t *testing.T) (*Client, context.Context) {
	t.Helper()

	baseURL := os.Getenv("DATABASE_URL")
	if baseURL == "" {
		baseURL = defaultIntegrationDatabaseURL
	}

	adminConn, err := pgx.Connect(context.Background(), baseURL)
	if err != nil {
		t.Skipf("postgres is not available for integration tests: %v", err)
	}

	dbName := fmt.Sprintf("feature_evaluator_it_%d", time.Now().UnixNano())
	if _, err := adminConn.Exec(context.Background(), "CREATE DATABASE "+dbName); err != nil {
		_ = adminConn.Close(context.Background())
		t.Skipf("cannot create temporary database %q: %v", dbName, err)
	}

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(), "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", dbName)
		_, _ = adminConn.Exec(context.Background(), "DROP DATABASE IF EXISTS "+dbName)
		_ = adminConn.Close(context.Background())
	})

	testURL := databaseURLWithName(t, baseURL, dbName)
	client, err := NewClient(context.Background(), Config{
		DatabaseURL:       testURL,
		MaxConns:          4,
		MinConns:          1,
		MaxConnLifetime:   5 * time.Minute,
		MaxConnIdleTime:   time.Minute,
		HealthcheckPeriod: time.Minute,
		ConnectionTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("create postgres client: %v", err)
	}

	if err := RunMigrations(context.Background(), client); err != nil {
		client.Close()
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(client.Close)

	return client, workspace.WithKey(context.Background(), "default")
}

func databaseURLWithName(t *testing.T, baseURL, dbName string) string {
	t.Helper()

	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	parsed.Path = "/" + dbName
	return parsed.String()
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
