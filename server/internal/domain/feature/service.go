package feature

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles feature business logic.
type Service struct {
	repo Repository
}

// NewService creates a new feature service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new feature after validation.
func (s *Service) Create(ctx context.Context, f *Feature) error {
	f.Key = resourcekey.Normalize(f.Key)
	if !resourcekey.IsValid(f.Key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid feature key format: %s", f.Key),
			"error.invalidFeatureKey",
		)
	}
	if !f.ValueType.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid value type: %s", f.ValueType),
			"error.invalidValueType",
		)
	}
	if f.AccessPolicy == "" {
		f.AccessPolicy = AccessPolicyRequired
	}
	if !f.AccessPolicy.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid access policy: %s", f.AccessPolicy),
			"error.invalidAccessPolicy",
		)
	}
	if err := validateAuthBinding(f); err != nil {
		return err
	}
	normalizedContract, err := normalizeInputContract(f.InputContract)
	if err != nil {
		return err
	}
	f.InputContract = normalizedContract
	if err := validateScheduling(f); err != nil {
		return err
	}

	if f.RolloutSalt == "" {
		salt, err := generateRolloutSalt()
		if err != nil {
			return fmt.Errorf("generating rollout salt: %w", err)
		}
		f.RolloutSalt = salt
	}

	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now

	if f.Rules == nil {
		f.Rules = []Rule{}
	}
	if f.Tags == nil {
		f.Tags = []string{}
	}
	if f.Environments == nil {
		f.Environments = []string{}
	}

	return s.repo.Create(ctx, f)
}

// GetByKey retrieves a feature by its unique key.
func (s *Service) GetByKey(ctx context.Context, key string) (*Feature, error) {
	f, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("getting feature %s: %w", key, err)
	}
	return f, nil
}

// Update updates an existing feature.
func (s *Service) Update(ctx context.Context, f *Feature) error {
	if !f.ValueType.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid value type: %s", f.ValueType),
			"error.invalidValueType",
		)
	}
	if f.AccessPolicy == "" {
		f.AccessPolicy = AccessPolicyRequired
	}
	if !f.AccessPolicy.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid access policy: %s", f.AccessPolicy),
			"error.invalidAccessPolicy",
		)
	}
	if err := validateAuthBinding(f); err != nil {
		return err
	}
	normalizedContract, err := normalizeInputContract(f.InputContract)
	if err != nil {
		return err
	}
	f.InputContract = normalizedContract
	if err := validateScheduling(f); err != nil {
		return err
	}
	if f.Environments == nil {
		f.Environments = []string{}
	}
	f.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, f)
}

// Delete removes a feature by key.
func (s *Service) Delete(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}

// List returns a paginated list of features.
func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, params)
}

// Toggle enables or disables a feature.
func (s *Service) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	return s.repo.Toggle(ctx, key, enabled, updatedBy)
}

// AddRule adds a new rule to a feature.
func (s *Service) AddRule(ctx context.Context, featureKey string, rule *Rule) error {
	if err := validateRuleSourceBindings(rule.SourceBindings); err != nil {
		return err
	}
	if rule.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generating rule id: %w", err)
		}
		rule.ID = id.String()
	}
	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return s.repo.AddRule(ctx, featureKey, rule)
}

// UpdateRule updates an existing rule within a feature.
func (s *Service) UpdateRule(ctx context.Context, featureKey string, rule *Rule) error {
	if err := validateRuleSourceBindings(rule.SourceBindings); err != nil {
		return err
	}
	rule.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateRule(ctx, featureKey, rule)
}

// DeleteRule removes a rule from a feature.
func (s *Service) DeleteRule(ctx context.Context, featureKey string, ruleID string) error {
	return s.repo.DeleteRule(ctx, featureKey, ruleID)
}

// ReorderRules reorders rules within a feature.
func (s *Service) ReorderRules(ctx context.Context, featureKey string, ruleIDs []string) error {
	return s.repo.ReorderRules(ctx, featureKey, ruleIDs)
}

// generateRolloutSalt creates a 16-byte random hex string for rollout hashing.
func generateRolloutSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// validateScheduling validates activeFrom/activeUntil and environments.
func validateScheduling(f *Feature) error {
	if f.ActiveFrom != nil && f.ActiveUntil != nil {
		if !f.ActiveFrom.Before(*f.ActiveUntil) {
			return apierror.NewBadRequest(
				"activeFrom must be before activeUntil",
				"error.invalidSchedule",
			)
		}
	}
	for _, env := range f.Environments {
		if !ValidEnvironment(env) {
			return apierror.NewBadRequest(
				fmt.Sprintf("invalid environment: %s", env),
				"error.invalidEnvironment",
			)
		}
	}
	return nil
}

func validateAuthBinding(f *Feature) error {
	switch f.AccessPolicy {
	case AccessPolicyPublic:
		f.AuthProfileKey = ""
	case AccessPolicyRequired:
		if strings.TrimSpace(f.AuthProfileKey) == "" {
			return apierror.NewBadRequest(
				"required access policy requires authProfileKey",
				"error.authProfileRequired",
			)
		}
	case AccessPolicyOptional:
		f.AuthProfileKey = strings.TrimSpace(f.AuthProfileKey)
	}
	return nil
}
