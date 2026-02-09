package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/otiai10/namazu/internal/auth"
	"github.com/otiai10/namazu/internal/delivery/webhook"
	"github.com/otiai10/namazu/internal/user"
)

// mockTokenVerifier implements auth.TokenVerifier for testing
type mockTokenVerifier struct {
	claims *auth.Claims
	err    error
}

func (m *mockTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*auth.Claims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

func TestNewRouterWithConfig_AuthEnabled(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newMockUserRepo()

	// Add a test user
	testUser := &user.User{
		ID:          "user-test-uid",
		UID:         "test-uid",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Plan:        user.PlanFree,
		Providers: []user.LinkedProvider{
			{
				ProviderID:  "google.com",
				Subject:     "test-uid",
				Email:       "test@example.com",
				DisplayName: "Test User",
				LinkedAt:    time.Now(),
			},
		},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastLoginAt: time.Now(),
	}
	userRepo.users[testUser.ID] = testUser
	userRepo.uidIndex[testUser.UID] = testUser.ID

	verifier := &mockTokenVerifier{
		claims: &auth.Claims{
			UID:        "test-uid",
			Email:      "test@example.com",
			Name:       "Test User",
			ProviderID: "google.com",
		},
	}

	cfg := RouterConfig{
		SubscriptionRepo: subRepo,
		EventRepo:        eventRepo,
		UserRepo:         userRepo,
		TokenVerifier:    verifier,
	}

	router := NewRouterWithConfig(cfg)

	t.Run("GET /api/me returns user profile with valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var response user.User
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response.UID != "test-uid" {
			t.Errorf("expected UID 'test-uid', got '%s'", response.UID)
		}
	})

	t.Run("GET /api/me returns 401 without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("GET /api/me/providers returns providers with valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/me/providers", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var response []user.LinkedProvider
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(response) != 1 {
			t.Errorf("expected 1 provider, got %d", len(response))
		}
	})

	t.Run("GET /health returns 200 without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("GET /api/events returns 200 without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestNewRouterWithConfig_AuthDisabled(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newMockUserRepo()

	cfg := RouterConfig{
		SubscriptionRepo: subRepo,
		EventRepo:        eventRepo,
		UserRepo:         userRepo,
		TokenVerifier:    nil, // No auth
	}

	router := NewRouterWithConfig(cfg)

	t.Run("GET /api/subscriptions works without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("GET /health returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestNewRouterWithConfig_ProtectedSubscriptionRoutes(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newMockUserRepo()

	verifier := &mockTokenVerifier{
		claims: &auth.Claims{
			UID:   "test-uid",
			Email: "test@example.com",
		},
	}

	cfg := RouterConfig{
		SubscriptionRepo: subRepo,
		EventRepo:        eventRepo,
		UserRepo:         userRepo,
		TokenVerifier:    verifier,
	}

	router := NewRouterWithConfig(cfg)

	t.Run("GET /api/subscriptions requires auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("GET /api/subscriptions works with valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

// routerMockChallenger implements Challenger for router tests
type routerMockChallenger struct {
	result webhook.ChallengeResult
	called bool
}

func (m *routerMockChallenger) VerifyURL(ctx context.Context, url, secret string) webhook.ChallengeResult {
	m.called = true
	return m.result
}

func TestNewRouterWithConfig_ChallengerWired(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	challenger := &routerMockChallenger{
		result: webhook.ChallengeResult{Success: false, ErrorMessage: "challenge failed"},
	}

	cfg := RouterConfig{
		SubscriptionRepo: subRepo,
		EventRepo:        eventRepo,
		Challenger:       challenger,
	}

	router := NewRouterWithConfig(cfg)

	body := `{"name":"Test","delivery":{"type":"webhook","url":"https://example.com/hook"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if !challenger.called {
		t.Error("expected challenger to be called when set via RouterConfig")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for failed challenge, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestNewRouterWithConfig_NilChallengerSkipsVerification(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	cfg := RouterConfig{
		SubscriptionRepo: subRepo,
		EventRepo:        eventRepo,
		Challenger:       nil,
	}

	router := NewRouterWithConfig(cfg)

	body := `{"name":"Test","delivery":{"type":"webhook","url":"https://example.com/hook"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d without challenger, got %d", http.StatusCreated, rec.Code)
	}
}
