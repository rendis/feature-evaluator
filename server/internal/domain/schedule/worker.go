package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
)

const workerInterval = 30 * time.Second

// Worker polls for pending scheduled changes and executes them.
type Worker struct {
	repo         Repository
	featureSvc   *feature.Service
	changelogSvc *changelog.Service
	stopCh       chan struct{}
	doneCh       chan struct{}
}

// NewWorker creates a new schedule worker.
func NewWorker(repo Repository, featureSvc *feature.Service, changelogSvc *changelog.Service) *Worker {
	return &Worker{
		repo:         repo,
		featureSvc:   featureSvc,
		changelogSvc: changelogSvc,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

// Start begins the background polling loop.
func (w *Worker) Start() {
	slog.Info("schedule worker started", "interval", workerInterval)
	go w.run()
}

// Stop signals the worker to shut down and waits for it to finish.
func (w *Worker) Stop() {
	slog.Info("stopping schedule worker")
	close(w.stopCh)
	<-w.doneCh
	slog.Info("schedule worker stopped")
}

func (w *Worker) run() {
	defer close(w.doneCh)

	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()

	// Run once immediately on start
	w.poll()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *Worker) poll() {
	for {
		sc, err := w.repo.ClaimNextPending(context.Background())
		if err != nil {
			slog.Error("claiming next pending schedule", "error", err)
			return
		}
		if sc == nil {
			return // no more pending changes
		}

		slog.Info("executing scheduled change",
			"id", sc.ID,
			"featureKey", sc.FeatureKey,
			"changeType", sc.ChangeType,
		)

		w.execute(sc)
	}
}

func (w *Worker) execute(sc *ScheduledChange) {
	// Build a context with the workspace key so all repo operations are scoped.
	ctx := workspace.WithKey(context.Background(), sc.WorkspaceKey)

	err := w.applyChange(ctx, sc)
	if err != nil {
		slog.Error("scheduled change failed",
			"id", sc.ID,
			"featureKey", sc.FeatureKey,
			"error", err,
		)
		if setErr := w.repo.SetFailed(context.Background(), sc.ID, err.Error()); setErr != nil {
			slog.Error("setting schedule failed status", "id", sc.ID, "error", setErr)
		}
		return
	}

	if setErr := w.repo.SetCompleted(context.Background(), sc.ID); setErr != nil {
		slog.Error("setting schedule completed status", "id", sc.ID, "error", setErr)
		return
	}

	// Record changelog entry
	w.recordChangelog(ctx, sc)

	slog.Info("scheduled change completed",
		"id", sc.ID,
		"featureKey", sc.FeatureKey,
		"changeType", sc.ChangeType,
	)
}

//nolint:cyclop,gocognit // Scheduled changes intentionally dispatch different mutation types from one worker entry point.
func (w *Worker) applyChange(ctx context.Context, sc *ScheduledChange) error {
	switch sc.ChangeType {
	case ChangeToggle:
		enabled, ok := sc.Payload["enabled"].(bool)
		if !ok {
			return errPayload("enabled must be a boolean")
		}
		return w.featureSvc.Toggle(ctx, sc.FeatureKey, enabled, "system:scheduler")

	case ChangeDefaultVal:
		f, err := w.featureSvc.GetByKey(ctx, sc.FeatureKey)
		if err != nil {
			return err
		}
		f.DefaultValue = sc.Payload["defaultValue"]
		f.UpdatedBy = "system:scheduler"
		return w.featureSvc.Update(ctx, f)

	case ChangeEnvironment:
		f, err := w.featureSvc.GetByKey(ctx, sc.FeatureKey)
		if err != nil {
			return err
		}
		if envs, ok := sc.Payload["environments"]; ok {
			if envSlice, ok := toStringSlice(envs); ok {
				f.Environments = envSlice
			}
		}
		f.UpdatedBy = "system:scheduler"
		return w.featureSvc.Update(ctx, f)

	case ChangeUpdate:
		f, err := w.featureSvc.GetByKey(ctx, sc.FeatureKey)
		if err != nil {
			return err
		}
		if name, ok := sc.Payload["name"].(string); ok && name != "" {
			f.Name = name
		}
		if desc, ok := sc.Payload["description"].(string); ok {
			f.Description = desc
		}
		if enabled, ok := sc.Payload["enabled"].(bool); ok {
			f.Enabled = enabled
		}
		if dv, ok := sc.Payload["defaultValue"]; ok {
			f.DefaultValue = dv
		}
		if envs, ok := sc.Payload["environments"]; ok {
			if envSlice, ok := toStringSlice(envs); ok {
				f.Environments = envSlice
			}
		}
		f.UpdatedBy = "system:scheduler"
		return w.featureSvc.Update(ctx, f)

	default:
		return errPayload("unsupported changeType: " + string(sc.ChangeType))
	}
}

func (w *Worker) recordChangelog(ctx context.Context, sc *ScheduledChange) {
	if w.changelogSvc == nil {
		return
	}

	action := changelog.ActionUpdate
	if sc.ChangeType == ChangeToggle {
		action = changelog.ActionToggle
	}

	entry := &changelog.ChangeEntry{
		EntityType: changelog.EntityFeature,
		EntityKey:  sc.FeatureKey,
		Action:     action,
		Actor:      "system:scheduler",
		ActorType:  changelog.ActorSystem,
		Metadata: map[string]any{
			"scheduleId": sc.ID,
			"changeType": sc.ChangeType,
		},
	}

	if err := w.changelogSvc.Record(ctx, entry); err != nil {
		slog.Error("recording scheduled change changelog", "error", err,
			"featureKey", sc.FeatureKey,
			"scheduleId", sc.ID,
		)
	}
}

type payloadError struct {
	msg string
}

func (e *payloadError) Error() string { return e.msg }

func errPayload(msg string) error { return &payloadError{msg: msg} }

func toStringSlice(v any) ([]string, bool) {
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		s := make([]string, 0, len(val))
		for _, item := range val {
			if str, ok := item.(string); ok {
				s = append(s, str)
			}
		}
		return s, true
	default:
		return nil, false
	}
}
