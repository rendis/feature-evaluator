package workspace

import "context"

type ctxKey struct{}

// WithKey returns a context with the workspace key set.
func WithKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey{}, key)
}

// KeyFromContext returns the workspace key from context.
func KeyFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok && v != "" {
		return v
	}
	return ""
}
