package changelog

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockChangelogRepo is an inline mock of Repository.
type mockChangelogRepo struct {
	createFn       func(ctx context.Context, entry *ChangeEntry) error
	listFn         func(ctx context.Context, params ListParams) (*ListResult, error)
	listByEntityFn func(ctx context.Context, entityType, entityKey string, params ListParams) (*ListResult, error)
	createCalls    int
}

func (m *mockChangelogRepo) Create(ctx context.Context, entry *ChangeEntry) error {
	m.createCalls++
	if m.createFn != nil {
		return m.createFn(ctx, entry)
	}
	return nil
}

func (m *mockChangelogRepo) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return &ListResult{}, nil
}

func (m *mockChangelogRepo) ListByEntity(ctx context.Context, entityType, entityKey string, params ListParams) (*ListResult, error) {
	if m.listByEntityFn != nil {
		return m.listByEntityFn(ctx, entityType, entityKey, params)
	}
	return &ListResult{}, nil
}

func TestRecord_Success(t *testing.T) {
	t.Parallel()

	var savedEntry *ChangeEntry
	repo := &mockChangelogRepo{
		createFn: func(_ context.Context, entry *ChangeEntry) error {
			savedEntry = entry
			return nil
		},
	}
	svc := NewService(repo)

	entry := &ChangeEntry{
		EntityType: EntityFeature,
		EntityKey:  "dark-mode",
		Action:     ActionCreate,
		Actor:      "user-1",
	}
	before := time.Now().UTC()
	err := svc.Record(context.Background(), entry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Errorf("repo.Create called %d times, want 1", repo.createCalls)
	}
	if savedEntry.CreatedAt.Before(before) {
		t.Error("CreatedAt should be set to current time")
	}
}

func TestRecord_RepoError(t *testing.T) {
	t.Parallel()

	repo := &mockChangelogRepo{
		createFn: func(_ context.Context, _ *ChangeEntry) error {
			return errors.New("db connection lost")
		},
	}
	svc := NewService(repo)

	entry := &ChangeEntry{
		EntityType: EntityFeature,
		EntityKey:  "dark-mode",
		Action:     ActionCreate,
	}
	err := svc.Record(context.Background(), entry)

	if err == nil {
		t.Fatal("expected error to be propagated")
	}
}

func TestList_DefaultPagination(t *testing.T) {
	t.Parallel()

	var receivedParams ListParams
	repo := &mockChangelogRepo{
		listFn: func(_ context.Context, params ListParams) (*ListResult, error) {
			receivedParams = params
			return &ListResult{Page: params.Page, PageSize: params.PageSize}, nil
		},
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListParams{Page: 0, PageSize: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedParams.Page != 1 {
		t.Errorf("Page = %d, want 1", receivedParams.Page)
	}
	if receivedParams.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20", receivedParams.PageSize)
	}
}

func TestList_MaxPageSize(t *testing.T) {
	t.Parallel()

	var receivedParams ListParams
	repo := &mockChangelogRepo{
		listFn: func(_ context.Context, params ListParams) (*ListResult, error) {
			receivedParams = params
			return &ListResult{}, nil
		},
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListParams{Page: 1, PageSize: 200})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedParams.PageSize != 20 {
		t.Errorf("PageSize = %d, want 20 (clamped from 200)", receivedParams.PageSize)
	}
}

func TestList_ValidParams(t *testing.T) {
	t.Parallel()

	var receivedParams ListParams
	repo := &mockChangelogRepo{
		listFn: func(_ context.Context, params ListParams) (*ListResult, error) {
			receivedParams = params
			return &ListResult{}, nil
		},
	}
	svc := NewService(repo)

	_, err := svc.List(context.Background(), ListParams{Page: 2, PageSize: 50})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedParams.Page != 2 {
		t.Errorf("Page = %d, want 2", receivedParams.Page)
	}
	if receivedParams.PageSize != 50 {
		t.Errorf("PageSize = %d, want 50", receivedParams.PageSize)
	}
}

func TestListByEntity_Normalization(t *testing.T) {
	t.Parallel()

	var receivedParams ListParams
	repo := &mockChangelogRepo{
		listByEntityFn: func(_ context.Context, _, _ string, params ListParams) (*ListResult, error) {
			receivedParams = params
			return &ListResult{}, nil
		},
	}
	svc := NewService(repo)

	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"zero_values", 0, 0, 1, 20},
		{"negative_page", -1, 10, 1, 10},
		{"exceeds_max", 1, 200, 1, 20},
		{"valid", 3, 50, 3, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ListByEntity(context.Background(), "feature", "dark-mode", ListParams{
				Page:     tt.page,
				PageSize: tt.pageSize,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if receivedParams.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", receivedParams.Page, tt.wantPage)
			}
			if receivedParams.PageSize != tt.wantPageSize {
				t.Errorf("PageSize = %d, want %d", receivedParams.PageSize, tt.wantPageSize)
			}
		})
	}
}
