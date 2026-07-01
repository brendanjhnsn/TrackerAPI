package auth

import "context"

// NewContext returns ctx with role and userID injected.
// Used by handler tests to simulate authenticated requests.
func NewContext(ctx context.Context, role Role, userID string) context.Context {
	ctx = context.WithValue(ctx, contextKey{}, role)
	return context.WithValue(ctx, userIDKey{}, userID)
}
