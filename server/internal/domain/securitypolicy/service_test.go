package securitypolicy

import (
	"context"
	"testing"
)

func TestServiceLoadUsesInheritedValuesWhenRepositoryIsEmpty(t *testing.T) {
	t.Parallel()

	svc := NewService(&repositoryStub{}, ManagedPolicy{
		CORSOrigins: []string{"https://console.example.com"},
	})

	if err := svc.Load(context.Background()); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	snapshot := svc.Snapshot()
	if got := snapshot.CORSOrigins.Inherited; len(got) != 1 || got[0] != "https://console.example.com" {
		t.Fatalf("CORSOrigins.Inherited = %#v, want inherited origin", got)
	}
	if got := snapshot.CORSOrigins.Effective; len(got) != 1 || got[0] != "https://console.example.com" {
		t.Fatalf("CORSOrigins.Effective = %#v, want inherited-only origin", got)
	}
}

func TestServiceUpdateNormalizesAndUnionsManagedValues(t *testing.T) {
	t.Parallel()

	repo := &repositoryStub{}
	svc := NewService(repo, ManagedPolicy{
		CORSOrigins: []string{"https://console.example.com"},
	})

	snapshot, err := svc.Update(context.Background(), ManagedPolicy{
		CORSOrigins: []string{" https://admin.example.com ", "https://console.example.com"},
		UpdatedBy:   "owner@example.com",
	})
	if err != nil {
		t.Fatalf("Update() error = %v, want nil", err)
	}

	if repo.lastUpsert == nil {
		t.Fatal("repo.Upsert() was not called")
	}
	if got := repo.lastUpsert.CORSOrigins; len(got) != 2 || got[0] != "https://admin.example.com" || got[1] != "https://console.example.com" {
		t.Fatalf("stored CORSOrigins = %#v, want normalized managed list", got)
	}
	if got := snapshot.CORSOrigins.Effective; len(got) != 2 || got[0] != "https://console.example.com" || got[1] != "https://admin.example.com" {
		t.Fatalf("effective CORSOrigins = %#v, want inherited first then managed unique values", got)
	}
	if snapshot.UpdatedBy != "owner@example.com" {
		t.Fatalf("UpdatedBy = %q, want %q", snapshot.UpdatedBy, "owner@example.com")
	}
	if snapshot.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt = zero, want current timestamp")
	}
}

func TestServiceUpdateRejectsInvalidManagedOrigin(t *testing.T) {
	t.Parallel()

	svc := NewService(&repositoryStub{}, ManagedPolicy{})

	_, err := svc.Update(context.Background(), ManagedPolicy{
		CORSOrigins: []string{"https://console.example.com/path"},
	})
	if err == nil {
		t.Fatal("Update() error = nil, want non-nil")
	}
}

type repositoryStub struct {
	managed    *ManagedPolicy
	lastUpsert *ManagedPolicy
}

func (r *repositoryStub) Get(_ context.Context) (*ManagedPolicy, error) {
	if r.managed == nil {
		return nil, nil
	}

	copied := *r.managed
	copied.CORSOrigins = cloneStrings(r.managed.CORSOrigins)
	return &copied, nil
}

func (r *repositoryStub) Upsert(_ context.Context, policy *ManagedPolicy) error {
	copied := *policy
	copied.CORSOrigins = cloneStrings(policy.CORSOrigins)
	r.lastUpsert = &copied
	r.managed = &copied
	return nil
}
