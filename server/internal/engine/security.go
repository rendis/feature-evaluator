package engine

import (
	"fmt"
	"strings"
)

// Security limits for expression evaluation.
const (
	MaxASTDepth        = 10
	MaxNodes           = 100
	MaxInSegmentCalls  = 5
	MaxStringLitLength = 1000
)

// denyList contains patterns that are forbidden in expressions.
var denyList = []string{
	"exec", "system", "import", "require",
	"__proto__", "constructor", "prototype",
	"eval", "Function", "process",
}

// ValidateExpression checks an expression for security violations.
func ValidateExpression(expr string) error {
	// Check deny-list
	lower := strings.ToLower(expr)
	for _, pattern := range denyList {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return fmt.Errorf("expression contains forbidden pattern: %s", pattern)
		}
	}

	// Check string literal length
	if err := checkStringLiterals(expr); err != nil {
		return err
	}

	// Count inSegment calls
	count := strings.Count(expr, "inSegment(")
	if count > MaxInSegmentCalls {
		return fmt.Errorf("expression contains %d inSegment() calls, max is %d", count, MaxInSegmentCalls)
	}

	// Validate dateBefore/dateAfter argument count
	if err := validateDateFunctionArgs(expr); err != nil {
		return err
	}

	return nil
}

// validateDateFunctionArgs checks that dateBefore() and dateAfter() calls have exactly 2 arguments.
func validateDateFunctionArgs(expr string) error {
	for _, fn := range []string{"dateBefore", "dateAfter"} {
		if err := checkFunctionArgCount(expr, fn, 2); err != nil {
			return err
		}
	}
	return nil
}

// checkFunctionArgCount verifies every call to fn in expr has exactly expectedArgs arguments.
func checkFunctionArgCount(expr, fn string, expectedArgs int) error {
	search := fn + "("
	idx := 0
	for {
		pos := strings.Index(expr[idx:], search)
		if pos == -1 {
			return nil
		}
		argStart := idx + pos + len(search)
		commas := countTopLevelCommas(expr, argStart)
		if commas != expectedArgs-1 {
			return fmt.Errorf("%s() requires exactly %d arguments (dateValue, refValue)", fn, expectedArgs)
		}
		idx = argStart
	}
}

// countTopLevelCommas counts commas at parenthesis depth 1 starting from pos until the matching ')'.
func countTopLevelCommas(expr string, pos int) int {
	depth, commas := 1, 0
	for i := pos; i < len(expr) && depth > 0; i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 1 {
				commas++
			}
		}
	}
	return commas
}

func checkStringLiterals(expr string) error {
	inStr := false
	strStart := 0
	quote := byte(0)

	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if !inStr && (ch == '"' || ch == '\'') {
			inStr = true
			strStart = i
			quote = ch
		} else if inStr && ch == quote && (i == 0 || expr[i-1] != '\\') {
			length := i - strStart - 1
			if length > MaxStringLitLength {
				return fmt.Errorf("expression contains string literal exceeding %d characters", MaxStringLitLength)
			}
			inStr = false
		}
	}
	return nil
}

// ExtractInSegmentKeys extracts all segment keys from inSegment() calls in an expression.
func ExtractInSegmentKeys(expr string) []string {
	var keys []string
	search := "inSegment("
	idx := 0
	for {
		pos := strings.Index(expr[idx:], search)
		if pos == -1 {
			break
		}
		start := idx + pos + len(search)
		// Find the closing quote and paren
		if start < len(expr) && (expr[start] == '"' || expr[start] == '\'') {
			quoteChar := expr[start]
			end := strings.IndexByte(expr[start+1:], quoteChar)
			if end != -1 {
				key := expr[start+1 : start+1+end]
				keys = append(keys, key)
			}
		}
		idx = start
	}
	return keys
}
