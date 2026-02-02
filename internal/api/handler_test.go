package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/auth"
	"github.com/ayanel/namazu/internal/quota"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
	"github.com/ayanel/namazu/internal/user"
)

// mockSubscriptionRepo implements subscription.Repository for testing
type mockSubscriptionRepo struct {
	subscriptions map[string]subscription.Subscription
	nextID        int
}

func newMockSubscriptionRepo() *mockSubscriptionRepo {
	return &mockSubscriptionRepo{
		subscriptions: make(map[string]subscription.Subscription),
		nextID:        1,
	}
}

func (m *mockSubscriptionRepo) List(ctx context.Context) ([]subscription.Subscription, error) {
	result := make([]subscription.Subscription, 0, len(m.subscriptions))
	for _, sub := range m.subscriptions {
		result = append(result, sub)
	}
	return result, nil
}

func (m *mockSubscriptionRepo) Create(ctx context.Context, sub subscription.Subscription) (string, error) {
	id := "sub-" + string(rune('0'+m.nextID))
	m.nextID++
	sub.ID = id
	m.subscriptions[id] = sub
	return id, nil
}

func (m *mockSubscriptionRepo) Get(ctx context.Context, id string) (*subscription.Subscription, error) {
	sub, ok := m.subscriptions[id]
	if !ok {
		return nil, nil
	}
	return &sub, nil
}

func (m *mockSubscriptionRepo) Update(ctx context.Context, id string, sub subscription.Subscription) error {
	sub.ID = id
	m.subscriptions[id] = sub
	return nil
}

func (m *mockSubscriptionRepo) Delete(ctx context.Context, id string) error {
	delete(m.subscriptions, id)
	return nil
}

func (m *mockSubscriptionRepo) ListByUserID(ctx context.Context, userID string) ([]subscription.Subscription, error) {
	result := make([]subscription.Subscription, 0)
	for _, sub := range m.subscriptions {
		// Return subscriptions owned by user or legacy subscriptions (no owner)
		if sub.UserID == userID || sub.UserID == "" {
			result = append(result, sub)
		}
	}
	return result, nil
}

// mockEventRepo implements store.EventRepository for testing
type mockEventRepo struct {
	events []store.EventRecord
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		events: make([]store.EventRecord, 0),
	}
}

func (m *mockEventRepo) Create(ctx context.Context, event store.EventRecord) (string, error) {
	m.events = append(m.events, event)
	return event.ID, nil
}

func (m *mockEventRepo) Get(ctx context.Context, id string) (*store.EventRecord, error) {
	for _, e := range m.events {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, nil
}

func (m *mockEventRepo) List(ctx context.Context, limit int, startAfter *time.Time) ([]store.EventRecord, error) {
	result := make([]store.EventRecord, 0)
	for i, e := range m.events {
		if i >= limit {
			break
		}
		if startAfter != nil && !e.OccurredAt.Before(*startAfter) {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func TestCreateSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name: "valid subscription",
			body: `{
				"name": "Test Subscription",
				"delivery": {
					"type": "webhook",
					"url": "https://example.com/webhook",
					"secret": "test-secret"
				}
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing name",
			body:           `{"delivery": {"type": "webhook", "url": "https://example.com"}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing delivery URL",
			body:           `{"name": "Test", "delivery": {"type": "webhook"}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestListSubscriptions(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Add test data
	subRepo.subscriptions["sub-1"] = subscription.Subscription{
		ID:   "sub-1",
		Name: "Test Sub 1",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/1",
		},
	}
	subRepo.subscriptions["sub-2"] = subscription.Subscription{
		ID:   "sub-2",
		Name: "Test Sub 2",
		Delivery: subscription.DeliveryConfig{
			Type: "email",
			URL:  "test@example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []SubscriptionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(response))
	}
}

func TestGetSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Add test data
	subRepo.subscriptions["sub-1"] = subscription.Subscription{
		ID:   "sub-1",
		Name: "Test Sub",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{
			name:           "existing subscription",
			id:             "sub-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existing subscription",
			id:             "sub-999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/subscriptions/"+tt.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestUpdateSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Add test data
	subRepo.subscriptions["sub-1"] = subscription.Subscription{
		ID:   "sub-1",
		Name: "Original Name",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		id             string
		body           string
		expectedStatus int
	}{
		{
			name: "valid update",
			id:   "sub-1",
			body: `{
				"name": "Updated Name",
				"delivery": {
					"type": "webhook",
					"url": "https://new-url.com"
				}
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-existing subscription",
			id:             "sub-999",
			body:           `{"name": "Test", "delivery": {"type": "webhook", "url": "https://example.com"}}`,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid body",
			id:             "sub-1",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/"+tt.id, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestDeleteSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Add test data
	subRepo.subscriptions["sub-1"] = subscription.Subscription{
		ID:   "sub-1",
		Name: "Test Sub",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{
			name:           "existing subscription",
			id:             "sub-1",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "non-existing subscription",
			id:             "sub-999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/subscriptions/"+tt.id, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestListEvents(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Add test data
	now := time.Now()
	eventRepo.events = []store.EventRecord{
		{
			ID:            "event-1",
			Type:          "earthquake",
			Source:        "p2pquake",
			Severity:      5,
			AffectedAreas: []string{"Tokyo"},
			OccurredAt:    now.Add(-1 * time.Hour),
			ReceivedAt:    now,
			CreatedAt:     now,
		},
		{
			ID:            "event-2",
			Type:          "earthquake",
			Source:        "p2pquake",
			Severity:      3,
			AffectedAreas: []string{"Osaka"},
			OccurredAt:    now.Add(-2 * time.Hour),
			ReceivedAt:    now,
			CreatedAt:     now,
		},
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "default pagination",
			query:          "",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "with limit",
			query:          "?limit=1",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/events"+tt.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var response []EventResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if len(response) != tt.expectedCount {
				t.Errorf("expected %d events, got %d", tt.expectedCount, len(response))
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodOptions, "/api/subscriptions", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin: *")
	}

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS header Access-Control-Allow-Methods")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	// Try PATCH on subscriptions endpoint (not supported)
	req := httptest.NewRequest(http.MethodPatch, "/api/subscriptions", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestCreateSubscriptionWithFilter(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	body := `{
		"name": "Filtered Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		},
		"filter": {
			"min_scale": 4,
			"prefectures": ["Tokyo", "Osaka"]
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response SubscriptionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Filter == nil {
		t.Fatal("expected filter to be present")
	}

	if response.Filter.MinScale != 4 {
		t.Errorf("expected MinScale 4, got %d", response.Filter.MinScale)
	}

	if len(response.Filter.Prefectures) != 2 {
		t.Errorf("expected 2 prefectures, got %d", len(response.Filter.Prefectures))
	}
}

// Tests for ownership checks

func TestCreateSubscription_SetsUserIDFromClaims(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   "owner-user-id",
		Email: "owner@example.com",
	}

	body := `{
		"name": "My Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	// Verify UserID was set in the repository
	for _, sub := range subRepo.subscriptions {
		if sub.UserID != claims.UID {
			t.Errorf("expected UserID %s, got %s", claims.UID, sub.UserID)
		}
	}
}

func TestGetSubscription_ReturnsOwnSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	ownerUID := "owner-uid"
	subRepo.subscriptions["sub-owned"] = subscription.Subscription{
		ID:     "sub-owned",
		UserID: ownerUID,
		Name:   "My Subscription",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   ownerUID,
		Email: "owner@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions/sub-owned", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetSubscription(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetSubscription_Returns403ForOtherUsersSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	subRepo.subscriptions["sub-other"] = subscription.Subscription{
		ID:     "sub-other",
		UserID: "other-user-uid",
		Name:   "Someone Else's Subscription",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   "attacker-uid",
		Email: "attacker@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions/sub-other", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetSubscription(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestGetSubscription_AllowsAccessToLegacySubscriptionWithoutUserID(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Legacy subscription without UserID
	subRepo.subscriptions["sub-legacy"] = subscription.Subscription{
		ID:     "sub-legacy",
		UserID: "", // No owner set (legacy data)
		Name:   "Legacy Subscription",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   "any-user-uid",
		Email: "any@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions/sub-legacy", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.GetSubscription(rec, req)

	// Legacy subscriptions should be accessible (backward compatibility)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestUpdateSubscription_Returns403ForOtherUsersSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	subRepo.subscriptions["sub-other"] = subscription.Subscription{
		ID:     "sub-other",
		UserID: "other-user-uid",
		Name:   "Someone Else's Subscription",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   "attacker-uid",
		Email: "attacker@example.com",
	}

	body := `{
		"name": "Hacked Name",
		"delivery": {
			"type": "webhook",
			"url": "https://evil.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/sub-other", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.UpdateSubscription(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestDeleteSubscription_Returns403ForOtherUsersSubscription(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	subRepo.subscriptions["sub-other"] = subscription.Subscription{
		ID:     "sub-other",
		UserID: "other-user-uid",
		Name:   "Someone Else's Subscription",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   "attacker-uid",
		Email: "attacker@example.com",
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/subscriptions/sub-other", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.DeleteSubscription(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestListSubscriptions_FiltersToUserOwnSubscriptions(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	userUID := "my-user-uid"

	// User's own subscriptions
	subRepo.subscriptions["sub-mine-1"] = subscription.Subscription{
		ID:     "sub-mine-1",
		UserID: userUID,
		Name:   "My Sub 1",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/1",
		},
	}
	subRepo.subscriptions["sub-mine-2"] = subscription.Subscription{
		ID:     "sub-mine-2",
		UserID: userUID,
		Name:   "My Sub 2",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/2",
		},
	}

	// Other user's subscription
	subRepo.subscriptions["sub-other"] = subscription.Subscription{
		ID:     "sub-other",
		UserID: "other-user-uid",
		Name:   "Other's Sub",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/other",
		},
	}

	// Legacy subscription (no owner)
	subRepo.subscriptions["sub-legacy"] = subscription.Subscription{
		ID:     "sub-legacy",
		UserID: "",
		Name:   "Legacy Sub",
		Delivery: subscription.DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/legacy",
		},
	}

	handler := NewHandler(subRepo, eventRepo)

	claims := &auth.Claims{
		UID:   userUID,
		Email: "my@example.com",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ListSubscriptions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []SubscriptionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should only return user's own subscriptions + legacy subscriptions
	// Expecting 3: 2 owned + 1 legacy
	if len(response) != 3 {
		t.Errorf("expected 3 subscriptions, got %d", len(response))
	}

	// Verify none of the returned subscriptions belong to other users
	for _, sub := range response {
		if sub.ID == "sub-other" {
			t.Error("should not return other user's subscription")
		}
	}
}

// quotaUserRepo implements user.Repository for quota testing
// (separate from mockUserRepo in me_handler_test.go to avoid conflicts)
type quotaUserRepo struct {
	users    map[string]*user.User
	uidIndex map[string]string // uid -> id
}

func newQuotaUserRepo() *quotaUserRepo {
	return &quotaUserRepo{
		users:    make(map[string]*user.User),
		uidIndex: make(map[string]string),
	}
}

func (m *quotaUserRepo) Create(ctx context.Context, u user.User) (string, error) {
	id := "user-" + u.UID
	u.ID = id
	m.users[id] = &u
	m.uidIndex[u.UID] = id
	return id, nil
}

func (m *quotaUserRepo) Get(ctx context.Context, id string) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *quotaUserRepo) GetByUID(ctx context.Context, uid string) (*user.User, error) {
	id, ok := m.uidIndex[uid]
	if !ok {
		return nil, nil
	}
	return m.users[id], nil
}

func (m *quotaUserRepo) Update(ctx context.Context, id string, u user.User) error {
	u.ID = id
	m.users[id] = &u
	return nil
}

func (m *quotaUserRepo) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	return nil
}

func (m *quotaUserRepo) AddProvider(ctx context.Context, id string, provider user.LinkedProvider) error {
	return nil
}

func (m *quotaUserRepo) RemoveProvider(ctx context.Context, id string, providerID string) error {
	return nil
}

func (m *quotaUserRepo) GetByStripeCustomerID(ctx context.Context, customerID string) (*user.User, error) {
	for _, u := range m.users {
		if u.StripeCustomerID == customerID {
			return u, nil
		}
	}
	return nil, nil
}

// mockQuotaChecker implements quota.QuotaChecker for testing
type mockQuotaChecker struct {
	canCreate bool
	err       error
}

func (m *mockQuotaChecker) CanCreateSubscription(ctx context.Context, userID, plan string) (bool, error) {
	return m.canCreate, m.err
}

// Tests for quota checking in CreateSubscription

func TestCreateSubscription_Returns403WhenQuotaExceeded(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newQuotaUserRepo()

	// Create a free user using the repo's Create method to properly index
	testUser := user.User{
		UID:  "test-user-uid",
		Plan: user.PlanFree,
	}
	_, _ = userRepo.Create(context.Background(), testUser)

	// Create quota checker that denies creation
	quotaChecker := &mockQuotaChecker{canCreate: false, err: nil}

	handler := NewHandlerWithQuota(subRepo, eventRepo, userRepo, quotaChecker)

	claims := &auth.Claims{
		UID:   testUser.UID,
		Email: "test@example.com",
	}

	body := `{
		"name": "New Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if errResp.Error != "Subscription limit reached for your plan" {
		t.Errorf("expected quota error message, got %s", errResp.Error)
	}
}

func TestCreateSubscription_AllowsCreationWithinQuota(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newQuotaUserRepo()

	// Create a free user using the repo's Create method to properly index
	testUser := user.User{
		UID:  "test-user-uid",
		Plan: user.PlanFree,
	}
	_, _ = userRepo.Create(context.Background(), testUser)

	// Create quota checker that allows creation
	quotaChecker := &mockQuotaChecker{canCreate: true, err: nil}

	handler := NewHandlerWithQuota(subRepo, eventRepo, userRepo, quotaChecker)

	claims := &auth.Claims{
		UID:   testUser.UID,
		Email: "test@example.com",
	}

	body := `{
		"name": "New Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

func TestCreateSubscription_SkipsQuotaCheckWithoutAuth(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Use basic handler without quota (test mode scenario)
	handler := NewHandler(subRepo, eventRepo)

	body := `{
		"name": "Test Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	// Should succeed without auth context (backward compatibility / test mode)
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestCreateSubscription_Returns500OnQuotaCheckError(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newQuotaUserRepo()

	// Create a free user using the repo's Create method to properly index
	testUser := user.User{
		UID:  "test-user-uid",
		Plan: user.PlanFree,
	}
	_, _ = userRepo.Create(context.Background(), testUser)

	// Create quota checker that returns an error
	quotaChecker := &mockQuotaChecker{canCreate: false, err: errors.New("database error")}

	handler := NewHandlerWithQuota(subRepo, eventRepo, userRepo, quotaChecker)

	claims := &auth.Claims{
		UID:   testUser.UID,
		Email: "test@example.com",
	}

	body := `{
		"name": "New Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestCreateSubscription_UsesUserPlanFromUserRepo(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newQuotaUserRepo()

	// Create a pro user (should have higher limits)
	proUser := user.User{
		UID:  "pro-user-uid",
		Plan: user.PlanPro,
	}
	_, _ = userRepo.Create(context.Background(), proUser)

	// Use real quota checker to verify plan is retrieved correctly
	quotaChecker := quota.NewChecker(subRepo)

	handler := NewHandlerWithQuota(subRepo, eventRepo, userRepo, quotaChecker)

	claims := &auth.Claims{
		UID:   proUser.UID,
		Email: "pro@example.com",
	}

	body := `{
		"name": "New Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}
}

func TestCreateSubscription_DefaultsToFreePlanWhenUserNotFound(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()
	userRepo := newQuotaUserRepo() // Empty user repo

	// Pre-populate subscriptions to hit free limit (3)
	subRepo.subscriptions["sub-1"] = subscription.Subscription{ID: "sub-1", UserID: "unknown-uid"}
	subRepo.subscriptions["sub-2"] = subscription.Subscription{ID: "sub-2", UserID: "unknown-uid"}
	subRepo.subscriptions["sub-3"] = subscription.Subscription{ID: "sub-3", UserID: "unknown-uid"}

	// Use real quota checker
	quotaChecker := quota.NewChecker(subRepo)

	handler := NewHandlerWithQuota(subRepo, eventRepo, userRepo, quotaChecker)

	claims := &auth.Claims{
		UID:   "unknown-uid", // User not in repo
		Email: "unknown@example.com",
	}

	body := `{
		"name": "New Subscription",
		"delivery": {
			"type": "webhook",
			"url": "https://example.com/webhook"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := auth.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreateSubscription(rec, req)

	// Should be forbidden because free limit (3) is already reached
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d: %s", http.StatusForbidden, rec.Code, rec.Body.String())
	}
}

// mockURLValidator implements URLValidator for testing
type mockURLValidator struct {
	allowedURLs map[string]bool
	rejectAll   bool
	rejectMsg   string
}

func newMockURLValidator() *mockURLValidator {
	return &mockURLValidator{
		allowedURLs: make(map[string]bool),
		rejectAll:   false,
	}
}

func (m *mockURLValidator) ValidateWebhookURL(url string) error {
	if m.rejectAll {
		if m.rejectMsg != "" {
			return errors.New(m.rejectMsg)
		}
		return errors.New("URL validation failed")
	}
	if !m.allowedURLs[url] && len(m.allowedURLs) > 0 {
		return errors.New("URL not allowed")
	}
	return nil
}

func TestCreateSubscription_WithURLValidation(t *testing.T) {
	t.Run("rejects private IP addresses", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		validator := newMockURLValidator()
		validator.rejectAll = true
		validator.rejectMsg = "private IP addresses are not allowed"

		handler := NewHandler(subRepo, eventRepo)
		handler.SetURLValidator(validator)

		body := `{
			"name": "Test Subscription",
			"delivery": {
				"type": "webhook",
				"url": "https://10.0.0.1/webhook"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSubscription(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}

		var response ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response.Error == "" {
			t.Error("expected error message in response")
		}
	})

	t.Run("rejects HTTP URLs", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		validator := newMockURLValidator()
		validator.rejectAll = true
		validator.rejectMsg = "HTTPS is required"

		handler := NewHandler(subRepo, eventRepo)
		handler.SetURLValidator(validator)

		body := `{
			"name": "Test Subscription",
			"delivery": {
				"type": "webhook",
				"url": "http://example.com/webhook"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSubscription(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("allows valid HTTPS URLs", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		validator := newMockURLValidator()
		// Empty allowedURLs with rejectAll=false means allow all

		handler := NewHandler(subRepo, eventRepo)
		handler.SetURLValidator(validator)

		body := `{
			"name": "Test Subscription",
			"delivery": {
				"type": "webhook",
				"url": "https://example.com/webhook"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSubscription(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}
	})

	t.Run("skips validation when no validator", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		handler := NewHandler(subRepo, eventRepo)
		// No URL validator set

		body := `{
			"name": "Test Subscription",
			"delivery": {
				"type": "webhook",
				"url": "http://10.0.0.1/webhook"
			}
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.CreateSubscription(rec, req)

		// Should succeed when no validator is configured
		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}
	})
}

func TestUpdateSubscription_WithURLValidation(t *testing.T) {
	t.Run("rejects invalid URLs on update", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		subRepo.subscriptions["sub-1"] = subscription.Subscription{
			ID:   "sub-1",
			Name: "Test Sub",
			Delivery: subscription.DeliveryConfig{
				Type: "webhook",
				URL:  "https://example.com/original",
			},
		}

		validator := newMockURLValidator()
		validator.rejectAll = true
		validator.rejectMsg = "localhost URLs are not allowed"

		handler := NewHandler(subRepo, eventRepo)
		handler.SetURLValidator(validator)

		body := `{
			"name": "Updated Sub",
			"delivery": {
				"type": "webhook",
				"url": "https://localhost/webhook"
			}
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/sub-1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.UpdateSubscription(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
		}
	})

	t.Run("allows valid URLs on update", func(t *testing.T) {
		subRepo := newMockSubscriptionRepo()
		eventRepo := newMockEventRepo()

		subRepo.subscriptions["sub-1"] = subscription.Subscription{
			ID:   "sub-1",
			Name: "Test Sub",
			Delivery: subscription.DeliveryConfig{
				Type: "webhook",
				URL:  "https://example.com/original",
			},
		}

		validator := newMockURLValidator()
		// Empty allowedURLs with rejectAll=false means allow all

		handler := NewHandler(subRepo, eventRepo)
		handler.SetURLValidator(validator)

		body := `{
			"name": "Updated Sub",
			"delivery": {
				"type": "webhook",
				"url": "https://example.com/updated"
			}
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/subscriptions/sub-1", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.UpdateSubscription(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	})
}
