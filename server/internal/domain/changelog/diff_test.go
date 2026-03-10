package changelog

import (
	"testing"
)

type testStruct struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
	Count       int      `json:"count"`
	Tags        []string `json:"tags"`
	ID          string   `json:"id"`
	CreatedAt   string   `json:"createdAt"`
}

func TestComputeDiff_NoChanges(t *testing.T) {
	old := testStruct{Name: "a", Enabled: true}
	new := testStruct{Name: "a", Enabled: true}

	changes := ComputeDiff(old, new)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestComputeDiff_SimpleChanges(t *testing.T) {
	old := testStruct{Name: "old-name", Description: "old-desc", Enabled: false}
	new := testStruct{Name: "new-name", Description: "new-desc", Enabled: true}

	changes := ComputeDiff(old, new)

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d: %+v", len(changes), changes)
	}

	byField := make(map[string]FieldChange)
	for _, c := range changes {
		byField[c.Field] = c
	}

	if c, ok := byField["name"]; !ok {
		t.Error("expected 'name' change")
	} else if c.OldValue != "old-name" || c.NewValue != "new-name" {
		t.Errorf("name change: old=%v new=%v", c.OldValue, c.NewValue)
	}

	if c, ok := byField["enabled"]; !ok {
		t.Error("expected 'enabled' change")
	} else if c.OldValue != "false" || c.NewValue != "true" {
		t.Errorf("enabled change: old=%v new=%v", c.OldValue, c.NewValue)
	}
}

func TestComputeDiff_SkipsIDAndTimestamps(t *testing.T) {
	old := testStruct{ID: "old-id", CreatedAt: "2024-01-01", Name: "same"}
	new := testStruct{ID: "new-id", CreatedAt: "2025-01-01", Name: "same"}

	changes := ComputeDiff(old, new)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes (id/timestamps skipped), got %d: %+v", len(changes), changes)
	}
}

func TestComputeDiff_SliceChanges(t *testing.T) {
	old := testStruct{Name: "same", Tags: []string{"a", "b"}}
	new := testStruct{Name: "same", Tags: []string{"a", "c"}}

	changes := ComputeDiff(old, new)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Field != "tags" {
		t.Errorf("expected field 'tags', got %q", changes[0].Field)
	}
}

func TestComputeDiff_NilInputs(t *testing.T) {
	changes := ComputeDiff(nil, nil)
	if changes != nil {
		t.Errorf("expected nil for nil inputs, got %+v", changes)
	}
}

func TestComputeDiff_Pointers(t *testing.T) {
	old := &testStruct{Name: "a"}
	new := &testStruct{Name: "b"}

	changes := ComputeDiff(old, new)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change with pointer inputs, got %d", len(changes))
	}
	if changes[0].Field != "name" {
		t.Errorf("expected field 'name', got %q", changes[0].Field)
	}
}
