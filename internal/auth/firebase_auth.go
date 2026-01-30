package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// idTokenVerifier is an interface for verifying ID tokens
// Both firebaseAuth.Client and firebaseAuth.TenantClient implement this
type idTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseAuth.Token, error)
}

// FirebaseTokenVerifier implements TokenVerifier using Firebase Admin SDK
type FirebaseTokenVerifier struct {
	verifier idTokenVerifier
	tenantID string
}

// FirebaseTokenVerifierConfig holds configuration for FirebaseTokenVerifier
type FirebaseTokenVerifierConfig struct {
	ProjectID       string
	CredentialsPath string
	TenantID        string // Optional: for multi-tenant Identity Platform
}

// NewFirebaseTokenVerifier creates a new Firebase token verifier
func NewFirebaseTokenVerifier(ctx context.Context, projectID string, credentialsPath string) (*FirebaseTokenVerifier, error) {
	return NewFirebaseTokenVerifierWithConfig(ctx, FirebaseTokenVerifierConfig{
		ProjectID:       projectID,
		CredentialsPath: credentialsPath,
	})
}

// NewFirebaseTokenVerifierWithConfig creates a new Firebase token verifier with full configuration
func NewFirebaseTokenVerifierWithConfig(ctx context.Context, cfg FirebaseTokenVerifierConfig) (*FirebaseTokenVerifier, error) {
	var opts []option.ClientOption
	if cfg.CredentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsPath))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.ProjectID,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth client: %w", err)
	}

	var verifier idTokenVerifier

	if cfg.TenantID != "" {
		// Multi-tenant mode: use tenant-specific auth client
		tenantClient, err := authClient.TenantManager.AuthForTenant(cfg.TenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tenant auth client for %s: %w", cfg.TenantID, err)
		}
		verifier = tenantClient
	} else {
		// Single-tenant mode
		verifier = authClient
	}

	return &FirebaseTokenVerifier{
		verifier: verifier,
		tenantID: cfg.TenantID,
	}, nil
}

// VerifyIDToken verifies a Firebase ID token and returns the decoded claims
func (v *FirebaseTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*Claims, error) {
	token, err := v.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	claims := &Claims{
		UID:           token.UID,
		Email:         getStringClaim(token.Claims, "email"),
		EmailVerified: getBoolClaim(token.Claims, "email_verified"),
		Name:          getStringClaim(token.Claims, "name"),
		Picture:       getStringClaim(token.Claims, "picture"),
	}

	// Set provider ID from Firebase token
	if token.Firebase.SignInProvider != "" {
		claims.ProviderID = token.Firebase.SignInProvider
	}

	return claims, nil
}

// getStringClaim safely extracts a string claim from the claims map
func getStringClaim(claims map[string]any, key string) string {
	val, ok := claims[key]
	if !ok {
		return ""
	}
	str, ok := val.(string)
	if !ok {
		return ""
	}
	return str
}

// getBoolClaim safely extracts a boolean claim from the claims map
func getBoolClaim(claims map[string]any, key string) bool {
	val, ok := claims[key]
	if !ok {
		return false
	}
	b, ok := val.(bool)
	if !ok {
		return false
	}
	return b
}
