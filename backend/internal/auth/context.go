package auth

import (
	"context"
)

// contextKey type for context value keys
type contextKey string

const claimsKey contextKey = "claims"

// WithClaims adds claims to context
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// GetClaims retrieves claims from context
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

// MustGetClaims retrieves claims or panics (for use after middleware)
func MustGetClaims(ctx context.Context) *Claims {
	claims, ok := GetClaims(ctx)
	if !ok {
		panic("auth: claims not found in context")
	}
	return claims
}
