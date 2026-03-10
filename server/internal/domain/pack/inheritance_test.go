package pack

import "testing"

func TestDetectCycle_NoError(t *testing.T) {
	// A->B->C, no cycle
	allPacks := map[string][]string{
		"B": {"C"},
	}
	if err := DetectCycle("A", []string{"B"}, allPacks); err != nil {
		t.Fatalf("expected no cycle, got: %v", err)
	}
}

func TestDetectCycle_SelfReference(t *testing.T) {
	allPacks := map[string][]string{}
	if err := DetectCycle("A", []string{"A"}, allPacks); err == nil {
		t.Fatal("expected error for self-reference")
	}
}

func TestDetectCycle_DirectCycle(t *testing.T) {
	// B already inherits from A, now A tries to inherit from B
	allPacks := map[string][]string{
		"B": {"A"},
	}
	if err := DetectCycle("A", []string{"B"}, allPacks); err == nil {
		t.Fatal("expected error for direct cycle")
	}
}

func TestDetectCycle_IndirectCycle(t *testing.T) {
	// A->B->C, now C tries to inherit from A -> cycle
	allPacks := map[string][]string{
		"A": {"B"},
		"B": {"C"},
	}
	if err := DetectCycle("C", []string{"A"}, allPacks); err == nil {
		t.Fatal("expected error for indirect cycle")
	}
}

func TestDetectCycle_MultipleParentsNoCycle(t *testing.T) {
	allPacks := map[string][]string{
		"B": {},
		"C": {},
	}
	if err := DetectCycle("A", []string{"B", "C"}, allPacks); err != nil {
		t.Fatalf("expected no cycle, got: %v", err)
	}
}

func TestDetectCycle_Diamond(t *testing.T) {
	// A->[B,C], B->D, C->D (diamond, no cycle)
	allPacks := map[string][]string{
		"B": {"D"},
		"C": {"D"},
	}
	if err := DetectCycle("A", []string{"B", "C"}, allPacks); err != nil {
		t.Fatalf("expected no cycle (diamond), got: %v", err)
	}
}
