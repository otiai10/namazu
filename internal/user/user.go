package user

import (
	"time"
)

// User represents an authenticated user stored in Firestore
type User struct {
	ID          string           `firestore:"-" json:"id,omitempty"`
	UID         string           `firestore:"uid" json:"uid"` // Identity Platform UID
	Email       string           `firestore:"email" json:"email"`
	DisplayName string           `firestore:"displayName" json:"displayName"`
	PictureURL  string           `firestore:"pictureUrl,omitempty" json:"pictureUrl,omitempty"`
	Plan        string           `firestore:"plan" json:"plan"`           // "free" | "pro"
	Providers   []LinkedProvider `firestore:"providers" json:"providers"` // Account Linking
	CreatedAt   time.Time        `firestore:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time        `firestore:"updatedAt" json:"updatedAt"`
	LastLoginAt time.Time        `firestore:"lastLoginAt" json:"lastLoginAt"`
}

// LinkedProvider represents a linked authentication provider
type LinkedProvider struct {
	ProviderID  string    `firestore:"providerId" json:"providerId"` // "google.com", "apple.com", "password"
	Subject     string    `firestore:"subject" json:"subject"`       // OIDC sub claim
	Email       string    `firestore:"email,omitempty" json:"email,omitempty"`
	DisplayName string    `firestore:"displayName,omitempty" json:"displayName,omitempty"`
	LinkedAt    time.Time `firestore:"linkedAt" json:"linkedAt"`
}

// PlanType constants for user subscription plans
const (
	PlanFree = "free"
	PlanPro  = "pro"
)

// ProviderID constants for authentication providers
const (
	ProviderGoogle   = "google.com"
	ProviderApple    = "apple.com"
	ProviderPassword = "password"
)

// Copy creates a deep copy of the User to prevent mutation
func (u User) Copy() User {
	copied := User{
		ID:          u.ID,
		UID:         u.UID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		PictureURL:  u.PictureURL,
		Plan:        u.Plan,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
		LastLoginAt: u.LastLoginAt,
	}

	// Deep copy providers slice
	if u.Providers != nil {
		copied.Providers = make([]LinkedProvider, len(u.Providers))
		copy(copied.Providers, u.Providers)
	}

	return copied
}

// Copy creates a deep copy of the LinkedProvider to prevent mutation
func (p LinkedProvider) Copy() LinkedProvider {
	return LinkedProvider{
		ProviderID:  p.ProviderID,
		Subject:     p.Subject,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		LinkedAt:    p.LinkedAt,
	}
}
