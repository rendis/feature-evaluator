package workspace

import (
	"context"
	"testing"
)

func TestWithKey_AndKeyFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithKey(context.Background(), "my-ws")
	got := KeyFromContext(ctx)

	if got != "my-ws" {
		t.Errorf("KeyFromContext() = %q, want %q", got, "my-ws")
	}
}

func TestKeyFromContext_EmptyContext(t *testing.T) {
	t.Parallel()

	got := KeyFromContext(context.Background())

	if got != "" {
		t.Errorf("KeyFromContext(empty) = %q, want empty string", got)
	}
}

func TestKeyFromContext_EmptyString(t *testing.T) {
	t.Parallel()

	ctx := WithKey(context.Background(), "")
	got := KeyFromContext(ctx)

	if got != "" {
		t.Errorf("KeyFromContext(empty string) = %q, want empty string", got)
	}
}
