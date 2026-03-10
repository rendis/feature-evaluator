package tier

import (
	"context"
	"errors"
	"testing"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// --- mock Repository ---

type mockRepo struct {
	tiers     map[string]*Tier
	packCount int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{tiers: make(map[string]*Tier)}
}

func (m *mockRepo) Create(_ context.Context, t *Tier) error {
	if _, ok := m.tiers[t.Key]; ok {
		return errors.New("already exists")
	}
	m.tiers[t.Key] = t
	return nil
}

func (m *mockRepo) Update(_ context.Context, t *Tier) error {
	m.tiers[t.Key] = t
	return nil
}

func (m *mockRepo) Delete(_ context.Context, key string) error {
	delete(m.tiers, key)
	return nil
}

func (m *mockRepo) FindByKey(_ context.Context, key string) (*Tier, error) {
	t, ok := m.tiers[key]
	if !ok {
		return nil, apierror.NewNotFound("tier not found", "error.tierNotFound")
	}
	return t, nil
}

func (m *mockRepo) FindByKeys(_ context.Context, keys []string) ([]Tier, error) {
	var result []Tier
	for _, k := range keys {
		if t, ok := m.tiers[k]; ok {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (m *mockRepo) List(_ context.Context, _ string) ([]Tier, error) {
	var result []Tier
	for _, t := range m.tiers {
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockRepo) CountPacksByTier(_ context.Context, _ string) (int64, error) {
	return m.packCount, nil
}

// --- mock IconRepository ---

type mockIconRepo struct {
	icons map[string]*TierIcon
}

func newMockIconRepo() *mockIconRepo {
	return &mockIconRepo{icons: make(map[string]*TierIcon)}
}

func (m *mockIconRepo) CreateIcon(_ context.Context, icon *TierIcon) error {
	icon.ID = "icon-1"
	m.icons[icon.ID] = icon
	return nil
}

func (m *mockIconRepo) DeleteIcon(_ context.Context, id string) error {
	delete(m.icons, id)
	return nil
}

func (m *mockIconRepo) FindIconByID(_ context.Context, id string) (*TierIcon, error) {
	icon, ok := m.icons[id]
	if !ok {
		return nil, apierror.NewNotFound("icon not found", "error.iconNotFound")
	}
	return icon, nil
}

func (m *mockIconRepo) ListIcons(_ context.Context) ([]TierIcon, error) {
	var result []TierIcon
	for _, ic := range m.icons {
		result = append(result, *ic)
	}
	return result, nil
}

// --- tests ---

func TestTierCreate_ValidData(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	tier, err := svc.Create(context.Background(), "Premium Plan", 1, "#EF4444", "builtin:crown", "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tier.Name != "Premium Plan" {
		t.Errorf("expected name %q, got %q", "Premium Plan", tier.Name)
	}
	if tier.Key == "" {
		t.Error("expected non-empty key")
	}
	if tier.Level != 1 {
		t.Errorf("expected level 1, got %d", tier.Level)
	}
	if tier.Icon != "builtin:crown" {
		t.Errorf("expected icon %q, got %q", "builtin:crown", tier.Icon)
	}
}

func TestTierCreate_EmptyName(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	_, err := svc.Create(context.Background(), "  ", 1, "#EF4444", "builtin:crown", "user-1")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.tierNameRequired" {
		t.Errorf("expected messageKey %q, got %q", "error.tierNameRequired", apiErr.MessageKey)
	}
}

func TestTierCreate_ZeroLevel(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	_, err := svc.Create(context.Background(), "Basic", 0, "#EF4444", "builtin:star", "user-1")
	if err == nil {
		t.Fatal("expected error for level 0")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.tierLevelInvalid" {
		t.Errorf("expected messageKey %q, got %q", "error.tierLevelInvalid", apiErr.MessageKey)
	}
}

func TestTierCreate_InvalidIconFormat(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	_, err := svc.Create(context.Background(), "Gold", 2, "#F59E0B", "invalid:crown", "user-1")
	if err == nil {
		t.Fatal("expected error for invalid icon format")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.tierIconInvalid" {
		t.Errorf("expected messageKey %q, got %q", "error.tierIconInvalid", apiErr.MessageKey)
	}
}

func TestTierDelete_InUse(t *testing.T) {
	repo := newMockRepo()
	repo.tiers["premium"] = &Tier{Key: "premium", Name: "Premium"}
	repo.packCount = 3

	svc := NewService(repo, newMockIconRepo())
	err := svc.Delete(context.Background(), "premium")
	if err == nil {
		t.Fatal("expected conflict error when tier is in use")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.tierInUse" {
		t.Errorf("expected messageKey %q, got %q", "error.tierInUse", apiErr.MessageKey)
	}
}

func TestTierUploadIcon_InvalidContentType(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	_, err := svc.UploadIcon(context.Background(), "logo", "image/jpeg", []byte("data"), "user-1")
	if err == nil {
		t.Fatal("expected error for invalid content type")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.iconContentTypeInvalid" {
		t.Errorf("expected messageKey %q, got %q", "error.iconContentTypeInvalid", apiErr.MessageKey)
	}
}

func TestTierUploadIcon_ExceedsMaxSize(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	bigData := make([]byte, MaxIconSize+1)
	_, err := svc.UploadIcon(context.Background(), "logo", "image/png", bigData, "user-1")
	if err == nil {
		t.Fatal("expected error for oversized icon")
	}
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.iconSizeTooLarge" {
		t.Errorf("expected messageKey %q, got %q", "error.iconSizeTooLarge", apiErr.MessageKey)
	}
}

func TestTierFindByKeys_EmptySlice(t *testing.T) {
	svc := NewService(newMockRepo(), newMockIconRepo())
	result, err := svc.FindByKeys(context.Background(), []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}
