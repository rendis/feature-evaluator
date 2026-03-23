package securitypolicy

import "context"

// Repository persists the single global managed security policy.
type Repository interface {
	Get(ctx context.Context) (*ManagedPolicy, error)
	Upsert(ctx context.Context, policy *ManagedPolicy) error
}
