package api

import (
	"context"
	"net/http"
	"time"

	"github.com/otiai10/namazu/internal/auth"
	"github.com/otiai10/namazu/internal/user"
)

// MeHandler handles user profile endpoints
type MeHandler struct {
	userRepo user.Repository
}

// NewMeHandler creates a new MeHandler
func NewMeHandler(userRepo user.Repository) *MeHandler {
	return &MeHandler{userRepo: userRepo}
}

// GetProfile handles GET /api/me
// Returns the current user's profile, creating it if first login
func (h *MeHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustGetClaims(r.Context())

	// Try to get existing user
	u, err := h.userRepo.GetByUID(r.Context(), claims.UID)
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	// Create user if first login
	if u == nil {
		u, err = h.createNewUser(r.Context(), claims)
		if err != nil {
			writeError(w, "failed to create user", http.StatusInternalServerError)
			return
		}
	} else {
		// Update last login time
		_ = h.userRepo.UpdateLastLogin(r.Context(), u.ID, time.Now().UTC())
	}

	writeJSON(w, u, http.StatusOK)
}

// GetProviders handles GET /api/me/providers
func (h *MeHandler) GetProviders(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustGetClaims(r.Context())

	u, err := h.userRepo.GetByUID(r.Context(), claims.UID)
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}

	if u == nil {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}

	writeJSON(w, u.Providers, http.StatusOK)
}

// createNewUser creates a new user from authentication claims
func (h *MeHandler) createNewUser(ctx context.Context, claims *auth.Claims) (*user.User, error) {
	now := time.Now().UTC()
	newUser := user.User{
		UID:         claims.UID,
		Email:       claims.Email,
		DisplayName: claims.Name,
		PictureURL:  claims.Picture,
		Plan:        user.PlanFree,
		Providers: []user.LinkedProvider{
			{
				ProviderID:  claims.ProviderID,
				Subject:     claims.UID,
				Email:       claims.Email,
				DisplayName: claims.Name,
				LinkedAt:    now,
			},
		},
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: now,
	}

	id, err := h.userRepo.Create(ctx, newUser)
	if err != nil {
		return nil, err
	}

	newUser.ID = id
	return &newUser, nil
}
