package engine

import (
	"fmt"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

const defaultCacheSize = 10000

// Engine compiles and evaluates expressions with caching and security checks.
type Engine struct {
	cache *CompilerCache
}

// New creates a new expression engine with an LRU cache.
func New() (*Engine, error) {
	cache, err := NewCompilerCache(defaultCacheSize)
	if err != nil {
		return nil, fmt.Errorf("creating engine: %w", err)
	}
	return &Engine{cache: cache}, nil
}

// Compile compiles an expression with security validation and caching.
//
// We use AllowUndefinedVariables instead of Env() so that functions (like
// externalApi, inSegment) are resolved from the runtime env on every call
// rather than being baked into the compiled program. This is critical because
// these functions are closures that change between evaluations.
func (e *Engine) Compile(expression string, env map[string]any) (*vm.Program, error) {
	prog, _, err := e.CompileWithStats(expression, env)
	return prog, err
}

// CompileWithStats compiles an expression and reports whether the cache was hit.
func (e *Engine) CompileWithStats(expression string, env map[string]any) (*vm.Program, bool, error) {
	// Security validation
	if err := ValidateExpression(expression); err != nil {
		return nil, false, fmt.Errorf("security check failed: %w", err)
	}

	// Check cache
	if prog, ok := e.cache.Get(expression); ok {
		return prog, true, nil
	}

	// Compile without binding env types so functions are looked up at runtime.
	prog, err := expr.Compile(expression, expr.AllowUndefinedVariables())
	if err != nil {
		return nil, false, fmt.Errorf("compiling expression: %w", err)
	}

	// Cache compiled program
	e.cache.Put(expression, prog)

	return prog, false, nil
}

// Evaluate runs a compiled program against the provided environment.
func (e *Engine) Evaluate(program *vm.Program, env map[string]any) (any, error) {
	result, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("evaluating expression: %w", err)
	}
	return result, nil
}

// CompileAndRun compiles and evaluates an expression in one step.
func (e *Engine) CompileAndRun(expression string, env map[string]any) (any, error) {
	prog, err := e.Compile(expression, env)
	if err != nil {
		return nil, err
	}
	return e.Evaluate(prog, env)
}

// CompileAndRunWithStats compiles and evaluates an expression, reporting cache hit metadata.
func (e *Engine) CompileAndRunWithStats(expression string, env map[string]any) (any, bool, error) {
	prog, hit, err := e.CompileWithStats(expression, env)
	if err != nil {
		return nil, hit, err
	}
	result, err := e.Evaluate(prog, env)
	if err != nil {
		return nil, hit, err
	}
	return result, hit, nil
}

// Validate checks if an expression is valid without evaluating it.
func (e *Engine) Validate(expression string) error {
	if err := ValidateExpression(expression); err != nil {
		return err
	}

	// Try compiling with a minimal env to check syntax
	_, err := expr.Compile(expression, expr.AllowUndefinedVariables())
	if err != nil {
		return fmt.Errorf("invalid expression: %w", err)
	}
	return nil
}

// CacheStats returns the current cache size.
func (e *Engine) CacheStats() int {
	return e.cache.Len()
}
