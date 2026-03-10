package pack

import "fmt"

// DetectCycle checks if adding the given parent keys to packKey would create a cycle.
// allPacks maps pack key -> its current parent keys.
func DetectCycle(packKey string, newParents []string, allPacks map[string][]string) error {
	for _, parent := range newParents {
		if parent == packKey {
			return fmt.Errorf("pack %q cannot inherit from itself", packKey)
		}
		visited := make(map[string]bool)
		if hasCycle(parent, packKey, allPacks, visited) {
			return fmt.Errorf("inheriting from %q would create a cycle", parent)
		}
	}
	return nil
}

// hasCycle does a DFS from current, checking if we can reach target.
func hasCycle(current, target string, allPacks map[string][]string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true
	for _, parent := range allPacks[current] {
		if hasCycle(parent, target, allPacks, visited) {
			return true
		}
	}
	return false
}
