package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/otiai10/namazu/backend/internal/auth"
	"github.com/otiai10/namazu/backend/internal/billing"
	"github.com/otiai10/namazu/backend/internal/config"
	"github.com/otiai10/namazu/backend/internal/user"
)

// billingMockUserRepo extends mockUserRepo with billing-specific methods
type billingMockUserRepo struct {
	users    map[string]*user.User
	uidIndex map[string]string // uid -> id
}

func newBillingMockUserRepo() *billingMockUserRepo {
	return &billingMockUserRepo{
		users:    make(map[string]*user.User),
		uidIndex: make(map[string]string),
	}
}

func (m *billingMockUserRepo) Create(ctx context.Context, u user.User) (string, error) {
	id := "mock-" + u.UID
	u.ID = id
	m.users[id] = &u
	m.uidIndex[u.UID] = id
	return id, nil
}

func (m *billingMockUserRepo) Get(ctx context.Context, id string) (*user.User, error) {
	return m.users[id], nil
}

func (m *billingMockUserRepo) GetByUID(ctx context.Context, uid string) (*user.User, error) {
	id, ok := m.uidIndex[uid]
	if !ok {
		return nil, nil
	}
	return m.users[id], nil
}

func (m *billingMockUserRepo) Update(ctx context.Context, id string, u user.User) error {
	u.ID = id
	m.users[id] = &u
	return nil
}

func (m *billingMockUserRepo) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	if u, ok := m.users[id]; ok {
		u.LastLoginAt = t
	}
	return nil
}

func (m *billingMockUserRepo) AddProvider(ctx context.Context, id string, provider user.LinkedProvider) error {
	return nil
}

func (m *billingMockUserRepo) RemoveProvider(ctx context.Context, id string, providerID string) error {
	return nil
}

// GetByStripeCustomerID gets a user by their Stripe customer ID
func (m *billingMockUserRepo) GetByStripeCustomerID(ctx context.Context, customerID string) (*user.User, error) {
	for _, u := range m.users {
		if u.StripeCustomerID == customerID {
			return u, nil
		}
	}
	return nil, nil
}

func TestNewBillingHandler(t *testing.T) {
	t.Run("creates handler with dependencies", func(t *testing.T) {
		client := billing.NewClient("sk_test_123")
		repo := newBillingMockUserRepo()
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}

		handler := NewBillingHandler(client, repo, cfg)

		if handler == nil {
			t.Fatal("NewBillingHandler returned nil")
		}
	})
}

func TestBillingHandler_GetStatus(t *testing.T) {
	t.Run("returns billing status for authenticated user", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		testUser := &user.User{
			ID:                 "user-123",
			UID:                "uid-456",
			Email:              "test@example.com",
			Plan:               user.PlanFree,
			StripeCustomerID:   "",
			SubscriptionID:     "",
			SubscriptionStatus: "",
		}
		repo.users["user-123"] = testUser
		repo.uidIndex["uid-456"] = "user-123"

		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)

		// Add auth claims to context
		claims := &auth.Claims{
			UID:   "uid-456",
			Email: "test@example.com",
		}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response BillingStatusResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Plan != user.PlanFree {
			t.Errorf("Expected plan 'free', got %s", response.Plan)
		}
		if response.HasActiveSubscription {
			t.Error("Expected HasActiveSubscription to be false")
		}
	})

	t.Run("returns active subscription for pro user", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		testUser := &user.User{
			ID:                 "user-123",
			UID:                "uid-456",
			Email:              "test@example.com",
			Plan:               user.PlanPro,
			StripeCustomerID:   "cus_123",
			SubscriptionID:     "sub_456",
			SubscriptionStatus: user.SubscriptionStatusActive,
			SubscriptionEndsAt: time.Now().Add(30 * 24 * time.Hour),
		}
		repo.users["user-123"] = testUser
		repo.uidIndex["uid-456"] = "user-123"

		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
		claims := &auth.Claims{UID: "uid-456", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response BillingStatusResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Plan != user.PlanPro {
			t.Errorf("Expected plan 'pro', got %s", response.Plan)
		}
		if !response.HasActiveSubscription {
			t.Error("Expected HasActiveSubscription to be true")
		}
		if response.SubscriptionStatus != user.SubscriptionStatusActive {
			t.Errorf("Expected subscription status 'active', got %s", response.SubscriptionStatus)
		}
	})

	t.Run("returns 404 for user not found", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/billing/status", nil)
		claims := &auth.Claims{UID: "unknown-uid", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetStatus(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestBillingHandler_CreateCheckoutSession(t *testing.T) {
	t.Run("returns error for user not found", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodPost, "/api/billing/create-checkout-session", nil)
		claims := &auth.Claims{UID: "unknown-uid", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateCheckoutSession(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("returns error for already subscribed user", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		testUser := &user.User{
			ID:                 "user-123",
			UID:                "uid-456",
			Email:              "test@example.com",
			Plan:               user.PlanPro,
			SubscriptionStatus: user.SubscriptionStatusActive,
		}
		repo.users["user-123"] = testUser
		repo.uidIndex["uid-456"] = "user-123"

		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodPost, "/api/billing/create-checkout-session", nil)
		claims := &auth.Claims{UID: "uid-456", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.CreateCheckoutSession(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestBillingHandler_StripeWebhook(t *testing.T) {
	t.Run("returns error for missing signature", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		body := []byte(`{"type":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(body))

		w := httptest.NewRecorder()
		handler.StripeWebhook(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("returns error for invalid signature", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		body := []byte(`{"type":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/stripe", bytes.NewReader(body))
		req.Header.Set("Stripe-Signature", "invalid_signature")

		w := httptest.NewRecorder()
		handler.StripeWebhook(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestBillingHandler_GetPortalSession(t *testing.T) {
	t.Run("returns error for user not found", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/billing/portal-session", nil)
		claims := &auth.Claims{UID: "unknown-uid", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetPortalSession(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("returns error for user without Stripe customer ID", func(t *testing.T) {
		repo := newBillingMockUserRepo()
		testUser := &user.User{
			ID:               "user-123",
			UID:              "uid-456",
			Email:            "test@example.com",
			Plan:             user.PlanFree,
			StripeCustomerID: "",
		}
		repo.users["user-123"] = testUser
		repo.uidIndex["uid-456"] = "user-123"

		client := billing.NewClient("sk_test_123")
		cfg := &config.BillingConfig{
			SecretKey:     "sk_test_123",
			WebhookSecret: "whsec_123",
			PriceID:       "price_123",
			SuccessURL:    "https://example.com/success",
			CancelURL:     "https://example.com/cancel",
		}
		handler := NewBillingHandler(client, repo, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/billing/portal-session", nil)
		claims := &auth.Claims{UID: "uid-456", Email: "test@example.com"}
		ctx := auth.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.GetPortalSession(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestBillingStatusResponse(t *testing.T) {
	t.Run("response fields are correct", func(t *testing.T) {
		now := time.Now()
		response := BillingStatusResponse{
			Plan:                  user.PlanPro,
			HasActiveSubscription: true,
			SubscriptionStatus:    user.SubscriptionStatusActive,
			SubscriptionEndsAt:    &now,
			StripeCustomerID:      "cus_123",
		}

		if response.Plan != user.PlanPro {
			t.Errorf("Expected plan 'pro', got %s", response.Plan)
		}
		if !response.HasActiveSubscription {
			t.Error("Expected HasActiveSubscription to be true")
		}
		if response.SubscriptionStatus != user.SubscriptionStatusActive {
			t.Errorf("Expected status 'active', got %s", response.SubscriptionStatus)
		}
		if response.StripeCustomerID != "cus_123" {
			t.Errorf("Expected customer ID 'cus_123', got %s", response.StripeCustomerID)
		}
	})
}

func TestCheckoutSessionResponse(t *testing.T) {
	t.Run("response fields are correct", func(t *testing.T) {
		response := CheckoutSessionResponse{
			SessionID:  "cs_123",
			SessionURL: "https://checkout.stripe.com/xxx",
		}

		if response.SessionID != "cs_123" {
			t.Errorf("Expected session ID 'cs_123', got %s", response.SessionID)
		}
		if response.SessionURL != "https://checkout.stripe.com/xxx" {
			t.Errorf("Expected URL, got %s", response.SessionURL)
		}
	})
}

func TestPortalSessionResponse(t *testing.T) {
	t.Run("response fields are correct", func(t *testing.T) {
		response := PortalSessionResponse{
			URL: "https://billing.stripe.com/xxx",
		}

		if response.URL != "https://billing.stripe.com/xxx" {
			t.Errorf("Expected URL, got %s", response.URL)
		}
	})
}
