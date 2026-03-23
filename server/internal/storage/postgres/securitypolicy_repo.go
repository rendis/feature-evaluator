package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/securitypolicy"
)

const securityPolicySingletonKey = "global"

// SecurityPolicyRepo implements securitypolicy.Repository using PostgreSQL.
type SecurityPolicyRepo struct {
	client *Client
}

// NewSecurityPolicyRepo creates a new SecurityPolicyRepo.
func NewSecurityPolicyRepo(client *Client) *SecurityPolicyRepo {
	return &SecurityPolicyRepo{client: client}
}

// Get returns the persisted managed security policy or nil when it has not been configured yet.
func (r *SecurityPolicyRepo) Get(ctx context.Context) (*securitypolicy.ManagedPolicy, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT cors_origins, updated_at, updated_by
		FROM system_security_policies
		WHERE singleton_key = $1
	`, securityPolicySingletonKey)

	var policy securitypolicy.ManagedPolicy
	var corsOriginsJSON []byte
	if err := row.Scan(
		&corsOriginsJSON,
		&policy.UpdatedAt,
		&policy.UpdatedBy,
	); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find security policy: %w", err)
	}

	if err := decodeJSON(corsOriginsJSON, &policy.CORSOrigins); err != nil {
		return nil, fmt.Errorf("decode security policy cors origins: %w", err)
	}

	return &policy, nil
}

// Upsert replaces the stored managed security policy.
func (r *SecurityPolicyRepo) Upsert(ctx context.Context, policy *securitypolicy.ManagedPolicy) error {
	corsOriginsJSON, err := jsonBytes(policy.CORSOrigins, "[]")
	if err != nil {
		return fmt.Errorf("marshal security policy cors origins: %w", err)
	}

	if _, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO system_security_policies (
			singleton_key, cors_origins, updated_at, updated_by
		) VALUES ($1, $2::jsonb, $3, $4)
		ON CONFLICT (singleton_key) DO UPDATE
		SET cors_origins = EXCLUDED.cors_origins,
		    updated_at = EXCLUDED.updated_at,
		    updated_by = EXCLUDED.updated_by
	`,
		securityPolicySingletonKey,
		corsOriginsJSON,
		policy.UpdatedAt,
		policy.UpdatedBy,
	); err != nil {
		return fmt.Errorf("upsert security policy: %w", err)
	}

	return nil
}
