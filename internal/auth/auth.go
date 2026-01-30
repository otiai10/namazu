// Package auth provides authentication functionality using Firebase Admin SDK
package auth

import (
	"context"
)

// Claims represents the decoded JWT claims from Firebase Auth
type Claims struct {
	UID           string `json:"uid"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	ProviderID    string `json:"provider_id,omitempty"`
}

// TokenVerifier verifies Firebase ID tokens
type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*Claims, error)
}
