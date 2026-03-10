package authprofile

import (
	"context"
	"errors"
	"testing"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

type serviceTestRepo struct {
	getByKeyFn func(ctx context.Context, key string) (*Profile, error)
	updateFn   func(ctx context.Context, currentKey string, profile *Profile) error
}

func (m *serviceTestRepo) Create(_ context.Context, _ *Profile) error {
	return nil
}

func (m *serviceTestRepo) GetByKey(ctx context.Context, key string) (*Profile, error) {
	if m.getByKeyFn != nil {
		return m.getByKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *serviceTestRepo) Update(ctx context.Context, currentKey string, profile *Profile) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, currentKey, profile)
	}
	return nil
}

func (m *serviceTestRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *serviceTestRepo) List(_ context.Context) ([]Profile, error) {
	return nil, nil
}

func (m *serviceTestRepo) CountFeatureUsages(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

type serviceTestCipher struct{}

func (serviceTestCipher) EncryptMap(payload map[string]string, _ string) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	return "encrypted", nil
}

func (serviceTestCipher) DecryptMap(_ string, _ string) (map[string]string, error) {
	return map[string]string{"apiKey": "secret"}, nil
}

func TestServiceUpdateRejectsTypeChange(t *testing.T) {
	t.Parallel()

	svc := NewService(&serviceTestRepo{
		getByKeyFn: func(_ context.Context, key string) (*Profile, error) {
			return &Profile{
				Key:          key,
				Name:         "Existing",
				Type:         TypeAPIKey,
				Config:       map[string]any{"location": "header", "name": "X-Api-Key"},
				WorkspaceKey: "ws",
			}, nil
		},
	}, serviceTestCipher{})

	err := svc.Update(context.Background(), "existing_key", &Profile{
		Key:    "existing_key",
		Name:   "Existing",
		Type:   TypeCustom,
		Config: map[string]any{"url": "https://validator.example.com", "method": "POST"},
	}, nil, false)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.authProfileTypeImmutable" {
		t.Fatalf("MessageKey = %q, want %q", apiErr.MessageKey, "error.authProfileTypeImmutable")
	}
}

func TestServiceUpdateAllowsKeyRenameWhenTypeIsUnchanged(t *testing.T) {
	t.Parallel()

	var gotCurrentKey string
	var gotProfile *Profile
	svc := NewService(&serviceTestRepo{
		getByKeyFn: func(_ context.Context, key string) (*Profile, error) {
			return &Profile{
				Key:          key,
				Name:         "Existing",
				Type:         TypeAPIKey,
				Config:       map[string]any{"location": "header", "name": "X-Api-Key"},
				WorkspaceKey: "ws",
				Version:      1,
			}, nil
		},
		updateFn: func(_ context.Context, currentKey string, profile *Profile) error {
			gotCurrentKey = currentKey
			cp := *profile
			gotProfile = &cp
			return nil
		},
	}, serviceTestCipher{})

	err := svc.Update(context.Background(), "existing_key", &Profile{
		Key:    "renamed_key",
		Name:   "Renamed",
		Type:   TypeAPIKey,
		Config: map[string]any{"location": "header", "name": "X-Api-Key"},
	}, map[string]string{"apiKey": "secret"}, true)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotCurrentKey != "existing_key" {
		t.Fatalf("currentKey = %q, want %q", gotCurrentKey, "existing_key")
	}
	if gotProfile == nil {
		t.Fatal("expected updated profile")
	}
	if gotProfile.Key != "renamed_key" {
		t.Fatalf("profile.Key = %q, want %q", gotProfile.Key, "renamed_key")
	}
	if gotProfile.Type != TypeAPIKey {
		t.Fatalf("profile.Type = %q, want %q", gotProfile.Type, TypeAPIKey)
	}
}
