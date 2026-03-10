package engine

import (
	"crypto/sha256"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/expr-lang/expr/vm"
)

// CompilerCache is an LRU cache for compiled expressions.
type CompilerCache struct {
	cache *lru.Cache[string, *vm.Program]
}

// NewCompilerCache creates a new compiler cache with the given max size.
func NewCompilerCache(maxSize int) (*CompilerCache, error) {
	cache, err := lru.New[string, *vm.Program](maxSize)
	if err != nil {
		return nil, fmt.Errorf("creating compiler cache: %w", err)
	}
	return &CompilerCache{cache: cache}, nil
}

// Get retrieves a compiled program from cache by expression hash.
func (c *CompilerCache) Get(expression string) (*vm.Program, bool) {
	key := hashExpression(expression)
	return c.cache.Get(key)
}

// Put stores a compiled program in the cache.
func (c *CompilerCache) Put(expression string, program *vm.Program) {
	key := hashExpression(expression)
	c.cache.Add(key, program)
}

// Len returns the number of cached programs.
func (c *CompilerCache) Len() int {
	return c.cache.Len()
}

func hashExpression(expr string) string {
	h := sha256.Sum256([]byte(expr))
	return fmt.Sprintf("%x", h)
}
