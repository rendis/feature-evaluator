package workspace

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// mockWorkspaceRepo is an inline mock of Repository.
type mockWorkspaceRepo struct {
	createFn      func(ctx context.Context, w *Workspace) error
	getByKeyFn    func(ctx context.Context, key string) (*Workspace, error)
	updateFn      func(ctx context.Context, w *Workspace) error
	archiveFn     func(ctx context.Context, key, archivedBy string) error
	restoreFn     func(ctx context.Context, key string) error
	listFn        func(ctx context.Context, includeArchived bool) ([]Workspace, error)
	countActiveFn func(ctx context.Context) (int64, error)
	createCalls   int
	updateCalls   int
	archiveCalls  int
	restoreCalls  int
}

func (m *mockWorkspaceRepo) Create(ctx context.Context, w *Workspace) error {
	m.createCalls++
	if m.createFn != nil {
		return m.createFn(ctx, w)
	}
	return nil
}

func (m *mockWorkspaceRepo) GetByKey(ctx context.Context, key string) (*Workspace, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, errors.New("not found")
}

func (m *mockWorkspaceRepo) Update(ctx context.Context, w *Workspace) error {
	m.updateCalls++
	if m.updateFn != nil {
		return m.updateFn(ctx, w)
	}
	return nil
}

func (m *mockWorkspaceRepo) Archive(ctx context.Context, key, archivedBy string) error {
	m.archiveCalls++
	if m.archiveFn != nil {
		return m.archiveFn(ctx, key, archivedBy)
	}
	return nil
}

func (m *mockWorkspaceRepo) Restore(ctx context.Context, key string) error {
	m.restoreCalls++
	if m.restoreFn != nil {
		return m.restoreFn(ctx, key)
	}
	return nil
}

func (m *mockWorkspaceRepo) List(ctx context.Context, includeArchived bool) ([]Workspace, error) {
	if m.listFn != nil {
		return m.listFn(ctx, includeArchived)
	}
	return nil, nil
}

func (m *mockWorkspaceRepo) CountActive(ctx context.Context) (int64, error) {
	if m.countActiveFn != nil {
		return m.countActiveFn(ctx)
	}
	return 0, nil
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	w := &Workspace{Key: "my-ws", Name: "My Workspace"}
	err := svc.Create(context.Background(), w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Errorf("repo.Create called %d times, want 1", repo.createCalls)
	}
	if w.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if w.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if w.Key != "my_ws" {
		t.Errorf("Key = %q, want %q", w.Key, "my_ws")
	}
}

func TestCreate_InvalidKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"too_short", "a"},
		{"special_chars_only", "!@#"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockWorkspaceRepo{}
			svc := NewService(repo)

			w := &Workspace{Key: tt.key, Name: "Valid Name"}
			err := svc.Create(context.Background(), w)

			if err == nil {
				t.Fatal("expected error for invalid key")
			}
			var apiErr *apierror.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected apierror.APIError, got %T", err)
			}
			if apiErr.MessageKey != "error.invalidWorkspaceKey" {
				t.Errorf("MessageKey = %q, want %q", apiErr.MessageKey, "error.invalidWorkspaceKey")
			}
		})
	}
}

func TestCreate_ValidKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"hyphenated", "my-ws", "my_ws"},
		{"dotted", "prod.us-east", "prod_us_east"},
		{"underscore", "v2_staging", "v2_staging"},
		{"min_valid", "ab", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockWorkspaceRepo{}
			svc := NewService(repo)

			w := &Workspace{Key: tt.key, Name: "Valid Name"}
			err := svc.Create(context.Background(), w)

			if err != nil {
				t.Errorf("unexpected error for key %q: %v", tt.key, err)
			}
			if w.Key != tt.want {
				t.Errorf("normalized key = %q, want %q", w.Key, tt.want)
			}
		})
	}
}

func TestCreate_EmptyName(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	w := &Workspace{Key: "valid_key", Name: ""}
	err := svc.Create(context.Background(), w)

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.workspaceNameRequired" {
		t.Errorf("MessageKey = %q, want %q", apiErr.MessageKey, "error.workspaceNameRequired")
	}
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	before := time.Now().UTC()
	w := &Workspace{Key: "my-ws", Name: "New Name"}
	err := svc.Update(context.Background(), w)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateCalls != 1 {
		t.Errorf("repo.Update called %d times, want 1", repo.updateCalls)
	}
	if w.UpdatedAt.Before(before) {
		t.Error("UpdatedAt should be set to current time")
	}
}

func TestUpdate_EmptyName(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	w := &Workspace{Key: "my-ws", Name: ""}
	err := svc.Update(context.Background(), w)

	if err == nil {
		t.Fatal("expected error for empty name")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.workspaceNameRequired" {
		t.Errorf("MessageKey = %q, want %q", apiErr.MessageKey, "error.workspaceNameRequired")
	}
}

func TestArchive_Success(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	err := svc.Archive(context.Background(), "my-ws", "owner@local.dev")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.archiveCalls != 1 {
		t.Errorf("repo.Archive called %d times, want 1", repo.archiveCalls)
	}
}

func TestRestore_Success(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{}
	svc := NewService(repo)

	err := svc.Restore(context.Background(), "my-ws")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.restoreCalls != 1 {
		t.Errorf("repo.Restore called %d times, want 1", repo.restoreCalls)
	}
}

func TestCountActive_Success(t *testing.T) {
	t.Parallel()

	repo := &mockWorkspaceRepo{
		countActiveFn: func(_ context.Context) (int64, error) {
			return 3, nil
		},
	}
	svc := NewService(repo)

	count, err := svc.CountActive(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("CountActive() = %d, want 3", count)
	}
}
