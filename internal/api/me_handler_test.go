package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/otiai10/namazu/internal/auth"
	"github.com/otiai10/namazu/internal/user"
)

// mockUserRepo implements user.Repository for testing
type mockUserRepo struct {
	users       map[string]*user.User
	uidIndex    map[string]string // uid -> id
	createError error
	getError    error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:    make(map[string]*user.User),
		uidIndex: make(map[string]string),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, u user.User) (string, error) {
	if m.createError != nil {
		return "", m.createError
	}
	id := "user-" + u.UID
	u.ID = id
	m.users[id] = &u
	m.uidIndex[u.UID] = id
	return id, nil
}

func (m *mockUserRepo) Get(ctx context.Context, id string) (*user.User, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) GetByUID(ctx context.Context, uid string) (*user.User, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	id, ok := m.uidIndex[uid]
	if !ok {
		return nil, nil
	}
	return m.users[id], nil
}

func (m *mockUserRepo) Update(ctx context.Context, id string, u user.User) error {
	u.ID = id
	m.users[id] = &u
	return nil
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	if u, ok := m.users[id]; ok {
		u.LastLoginAt = t
	}
	return nil
}

func (m *mockUserRepo) AddProvider(ctx context.Context, id string, provider user.LinkedProvider) error {
	if u, ok := m.users[id]; ok {
		u.Providers = append(u.Providers, provider)
	}
	return nil
}

func (m *mockUserRepo) RemoveProvider(ctx context.Context, id string, providerID string) error {
	if u, ok := m.users[id]; ok {
		for i, p := range u.Providers {
			if p.ProviderID == providerID {
				u.Providers = append(u.Providers[:i], u.Providers[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (m *mockUserRepo) GetByStripeCustomerID(ctx context.Context, customerID string) (*user.User, error) {
	for _, u := range m.users {
		if u.StripeCustomerID == customerID {
			return u, nil
		}
	}
	return nil, nil
}

func TestMeHandler_GetProfile_CreatesUserOnFirstLogin(t *testing.T) {
	userRepo := newMockUserRepo()
	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:        "new-user-uid",
		Email:      "new@example.com",
		Name:       "New User",
		Picture:    "https://example.com/photo.jpg",
		ProviderID: "google.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.UID != claims.UID {
		t.Errorf("expected UID %s, got %s", claims.UID, response.UID)
	}

	if response.Email != claims.Email {
		t.Errorf("expected Email %s, got %s", claims.Email, response.Email)
	}

	if response.DisplayName != claims.Name {
		t.Errorf("expected DisplayName %s, got %s", claims.Name, response.DisplayName)
	}

	if response.Plan != user.PlanFree {
		t.Errorf("expected Plan %s, got %s", user.PlanFree, response.Plan)
	}

	if len(response.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(response.Providers))
	}

	if response.Providers[0].ProviderID != claims.ProviderID {
		t.Errorf("expected ProviderID %s, got %s", claims.ProviderID, response.Providers[0].ProviderID)
	}
}

func TestMeHandler_GetProfile_ReturnsExistingUser(t *testing.T) {
	userRepo := newMockUserRepo()

	existingUser := &user.User{
		ID:          "user-existing-uid",
		UID:         "existing-uid",
		Email:       "existing@example.com",
		DisplayName: "Existing User",
		Plan:        user.PlanPro,
		Providers: []user.LinkedProvider{
			{
				ProviderID:  "google.com",
				Subject:     "existing-uid",
				Email:       "existing@example.com",
				DisplayName: "Existing User",
				LinkedAt:    time.Now().Add(-24 * time.Hour),
			},
		},
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
		LastLoginAt: time.Now().Add(-1 * time.Hour),
	}
	userRepo.users[existingUser.ID] = existingUser
	userRepo.uidIndex[existingUser.UID] = existingUser.ID

	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:        "existing-uid",
		Email:      "existing@example.com",
		Name:       "Existing User",
		ProviderID: "google.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should return existing user with Pro plan (not create new with Free)
	if response.Plan != user.PlanPro {
		t.Errorf("expected Plan %s, got %s", user.PlanPro, response.Plan)
	}

	if response.ID != existingUser.ID {
		t.Errorf("expected ID %s, got %s", existingUser.ID, response.ID)
	}
}

func TestMeHandler_GetProfile_ReturnsErrorOnDBFailure(t *testing.T) {
	userRepo := newMockUserRepo()
	userRepo.getError = context.DeadlineExceeded
	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:   "any-uid",
		Email: "any@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestMeHandler_GetProfile_ReturnsErrorOnCreateFailure(t *testing.T) {
	userRepo := newMockUserRepo()
	userRepo.createError = context.DeadlineExceeded
	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:   "new-user-uid",
		Email: "new@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProfile(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestMeHandler_GetProviders_ReturnsProvidersList(t *testing.T) {
	userRepo := newMockUserRepo()

	providers := []user.LinkedProvider{
		{
			ProviderID:  "google.com",
			Subject:     "google-sub",
			Email:       "user@gmail.com",
			DisplayName: "User G",
			LinkedAt:    time.Now().Add(-48 * time.Hour),
		},
		{
			ProviderID:  "apple.com",
			Subject:     "apple-sub",
			Email:       "user@icloud.com",
			DisplayName: "User A",
			LinkedAt:    time.Now().Add(-24 * time.Hour),
		},
	}

	existingUser := &user.User{
		ID:          "user-multi-uid",
		UID:         "multi-uid",
		Email:       "user@gmail.com",
		DisplayName: "User",
		Plan:        user.PlanFree,
		Providers:   providers,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now(),
		LastLoginAt: time.Now(),
	}
	userRepo.users[existingUser.ID] = existingUser
	userRepo.uidIndex[existingUser.UID] = existingUser.ID

	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:   "multi-uid",
		Email: "user@gmail.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me/providers", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProviders(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []user.LinkedProvider
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 providers, got %d", len(response))
	}
}

func TestMeHandler_GetProviders_ReturnsNotFoundForMissingUser(t *testing.T) {
	userRepo := newMockUserRepo()
	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:   "non-existent-uid",
		Email: "nobody@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me/providers", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProviders(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestMeHandler_GetProviders_ReturnsErrorOnDBFailure(t *testing.T) {
	userRepo := newMockUserRepo()
	userRepo.getError = context.DeadlineExceeded
	handler := NewMeHandler(userRepo)

	claims := &auth.Claims{
		UID:   "any-uid",
		Email: "any@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me/providers", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetProviders(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
