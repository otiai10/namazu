package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
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
