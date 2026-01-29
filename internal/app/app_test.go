package app

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/config"
	"github.com/ayanel/namazu/internal/delivery/webhook"
	"github.com/ayanel/namazu/internal/source"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
)

// mockClient is a mock implementation of p2pquake.Client for testing
type mockClient struct {
	events     chan source.Event
	connected  bool
	connectErr error
	closed     bool
	mu         sync.Mutex
}

func newMockClient() *mockClient {
	return &mockClient{
		events: make(chan source.Event, 10),
	}
}

func (m *mockClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockClient) Events() <-chan source.Event {
	return m.events
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	close(m.events)
	return nil
}

// mockSender is a mock implementation of webhook.Sender for testing
type mockSender struct {
	sendAllCalls []sendAllCall
	results      []webhook.DeliveryResult
	mu           sync.Mutex
}

type sendAllCall struct {
	targets []webhook.Target
	payload []byte
}

func newMockSender() *mockSender {
	return &mockSender{
		sendAllCalls: make([]sendAllCall, 0),
	}
}

func (m *mockSender) SendAll(ctx context.Context, targets []webhook.Target, payload []byte) []webhook.DeliveryResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	payloadCopy := make([]byte, len(payload))
	copy(payloadCopy, payload)
	m.sendAllCalls = append(m.sendAllCalls, sendAllCall{
		targets: targets,
		payload: payloadCopy,
	})

	// Return mock results
	if m.results != nil {
		return m.results
	}

	// Default: success for all targets
	results := make([]webhook.DeliveryResult, len(targets))
	for i, target := range targets {
		results[i] = webhook.DeliveryResult{
			URL:          target.URL,
			StatusCode:   200,
			Success:      true,
			ResponseTime: 100 * time.Millisecond,
		}
	}
	return results
}

func (m *mockSender) GetSendAllCalls() []sendAllCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendAllCalls
}

// mockRepository is a mock implementation of subscription.Repository for testing
type mockRepository struct {
	subscriptions []subscription.Subscription
	listErr       error
	mu            sync.Mutex
}

func newMockRepository(subs []subscription.Subscription) *mockRepository {
	return &mockRepository{
		subscriptions: subs,
	}
}

func (m *mockRepository) List(ctx context.Context) ([]subscription.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	// Return a copy
	result := make([]subscription.Subscription, len(m.subscriptions))
	copy(result, m.subscriptions)
	return result, nil
}

func (m *mockRepository) Create(ctx context.Context, sub subscription.Subscription) (string, error) {
	return "", errors.New("mock repository: create not implemented")
}

func (m *mockRepository) Get(ctx context.Context, id string) (*subscription.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sub := range m.subscriptions {
		if sub.ID == id {
			result := sub
			return &result, nil
		}
	}
	return nil, nil
}

func (m *mockRepository) Update(ctx context.Context, id string, sub subscription.Subscription) error {
	return errors.New("mock repository: update not implemented")
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	return errors.New("mock repository: delete not implemented")
}

// mockEventRepository is a mock implementation of store.EventRepository for testing
type mockEventRepository struct {
	events    []store.EventRecord
	createErr error
	mu        sync.Mutex
}

func newMockEventRepository() *mockEventRepository {
	return &mockEventRepository{
		events: make([]store.EventRecord, 0),
	}
}

func (m *mockEventRepository) Create(ctx context.Context, event store.EventRecord) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	if event.ID != "" {
		return event.ID, nil
	}
	return "generated-id", nil
}

func (m *mockEventRepository) Get(ctx context.Context, id string) (*store.EventRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, event := range m.events {
		if event.ID == id {
			result := event
			return &result, nil
		}
	}
	return nil, nil
}

func (m *mockEventRepository) List(ctx context.Context, limit int, startAfter *time.Time) ([]store.EventRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.events) {
		limit = len(m.events)
	}
	result := make([]store.EventRecord, limit)
	copy(result, m.events[:limit])
	return result, nil
}

func (m *mockEventRepository) GetEvents() []store.EventRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]store.EventRecord, len(m.events))
	copy(result, m.events)
	return result
}

// mockEvent implements source.Event for testing
type mockEvent struct {
	id            string
	eventType     source.EventType
	source        string
	severity      int
	affectedAreas []string
	occurredAt    time.Time
	receivedAt    time.Time
	rawJSON       string
}

func (m *mockEvent) GetID() string              { return m.id }
func (m *mockEvent) GetType() source.EventType  { return m.eventType }
func (m *mockEvent) GetSource() string          { return m.source }
func (m *mockEvent) GetSeverity() int           { return m.severity }
func (m *mockEvent) GetAffectedAreas() []string { return m.affectedAreas }
func (m *mockEvent) GetOccurredAt() time.Time   { return m.occurredAt }
func (m *mockEvent) GetReceivedAt() time.Time   { return m.receivedAt }
func (m *mockEvent) GetRawJSON() string         { return m.rawJSON }

// TestNewApp tests the App constructor
func TestNewApp(t *testing.T) {
	t.Run("creates app with valid config and repository", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
			{
				Name: "Webhook 2",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook2.example.com",
					Secret: "secret2",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)

		if app == nil {
			t.Fatal("NewApp returned nil")
		}
		if app.config != cfg {
			t.Error("App config not set correctly")
		}
		if app.client == nil {
			t.Error("App client not initialized")
		}
		if app.sender == nil {
			t.Error("App sender not initialized")
		}
		if app.repository == nil {
			t.Error("App repository not initialized")
		}
	})

	t.Run("works with empty repository", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		repo := newMockRepository([]subscription.Subscription{})

		app := NewApp(cfg, repo)

		if app == nil {
			t.Fatal("NewApp returned nil")
		}
		if app.repository == nil {
			t.Error("App repository not initialized")
		}
	})
}

// TestApp_handleEvent tests the event handling logic
func TestApp_handleEvent(t *testing.T) {
	t.Run("sends event to all subscriptions", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
			{
				Name: "Webhook 2",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook2.example.com",
					Secret: "secret2",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockSender := newMockSender()
		app.sender = mockSender

		event := &mockEvent{
			id:       "test-event-123",
			severity: 50,
			source:   "p2pquake",
			rawJSON:  `{"_id":"test-event-123","code":551}`,
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Fatalf("Expected 1 SendAll call, got %d", len(calls))
		}

		call := calls[0]
		if len(call.targets) != 2 {
			t.Errorf("Expected 2 targets, got %d", len(call.targets))
		}
		if string(call.payload) != `{"_id":"test-event-123","code":551}` {
			t.Errorf("Unexpected payload: %s", string(call.payload))
		}
	})

	t.Run("uses JSON encoding if RawJSON is empty", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockSender := newMockSender()
		app.sender = mockSender

		now := time.Now()
		event := &mockEvent{
			id:            "test-event-456",
			eventType:     source.EventTypeEarthquake,
			severity:      50,
			source:        "p2pquake",
			affectedAreas: []string{"Tokyo", "Osaka"},
			occurredAt:    now,
			receivedAt:    now,
			rawJSON:       "", // Empty RawJSON
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Fatalf("Expected 1 SendAll call, got %d", len(calls))
		}

		// Should have marshaled the event - verify it's valid JSON
		if len(calls[0].payload) == 0 {
			t.Error("Expected non-empty payload")
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(calls[0].payload, &decoded); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}
		// Verify the payload is valid JSON (structure will vary based on mockEvent implementation)
	})

	t.Run("handles no subscriptions configured", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		repo := newMockRepository([]subscription.Subscription{})

		app := NewApp(cfg, repo)
		mockSender := newMockSender()
		app.sender = mockSender

		event := &mockEvent{
			id:       "test-event-789",
			severity: 50,
			source:   "p2pquake",
			rawJSON:  `{"_id":"test-event-789"}`,
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Fatalf("Expected 1 SendAll call, got %d", len(calls))
		}
		if len(calls[0].targets) != 0 {
			t.Errorf("Expected 0 targets, got %d", len(calls[0].targets))
		}
	})
}

// TestApp_Run tests the main Run loop
func TestApp_Run(t *testing.T) {
	t.Run("connects to client and processes events", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockClient := newMockClient()
		mockSender := newMockSender()
		app.client = mockClient
		app.sender = mockSender

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start Run in goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run(ctx)
		}()

		// Wait for connection
		time.Sleep(50 * time.Millisecond)

		// Send test event
		event := &mockEvent{
			id:       "test-123",
			severity: 60,
			source:   "p2pquake",
			rawJSON:  `{"_id":"test-123"}`,
		}
		mockClient.events <- event

		// Wait for processing
		time.Sleep(50 * time.Millisecond)

		// Cancel context to stop Run
		cancel()

		// Wait for Run to finish
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Run returned error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Run did not finish in time")
		}

		// Verify client was connected
		mockClient.mu.Lock()
		connected := mockClient.connected
		closed := mockClient.closed
		mockClient.mu.Unlock()

		if !connected {
			t.Error("Client was not connected")
		}
		if !closed {
			t.Error("Client was not closed")
		}

		// Verify event was sent to subscriptions
		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Errorf("Expected 1 SendAll call, got %d", len(calls))
		}
	})

	t.Run("stops gracefully on context cancellation", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockClient := newMockClient()
		app.client = mockClient

		ctx, cancel := context.WithCancel(context.Background())

		// Start Run in goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Run(ctx)
		}()

		// Wait for connection
		time.Sleep(50 * time.Millisecond)

		// Cancel immediately
		cancel()

		// Wait for Run to finish
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("Run returned error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Run did not finish in time")
		}

		// Verify client was closed
		mockClient.mu.Lock()
		closed := mockClient.closed
		mockClient.mu.Unlock()

		if !closed {
			t.Error("Client was not closed after context cancellation")
		}
	})

	t.Run("returns error on connection failure", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockClient := newMockClient()
		mockClient.connectErr = context.DeadlineExceeded
		app.client = mockClient

		ctx := context.Background()
		err := app.Run(ctx)

		if err == nil {
			t.Error("Expected error on connection failure, got nil")
		}
		if err != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded error, got %v", err)
		}
	})
}

// TestApp_Integration tests integration scenarios
func TestApp_Integration(t *testing.T) {
	t.Run("processes multiple events in sequence", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		app := NewApp(cfg, repo)
		mockClient := newMockClient()
		mockSender := newMockSender()
		app.client = mockClient
		app.sender = mockSender

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start Run in goroutine
		go app.Run(ctx)

		// Wait for connection
		time.Sleep(50 * time.Millisecond)

		// Send multiple events
		for i := 1; i <= 3; i++ {
			event := &mockEvent{
				id:       string(rune('A' + i - 1)),
				severity: i * 10,
				source:   "p2pquake",
				rawJSON:  `{"_id":"` + string(rune('A'+i-1)) + `"}`,
			}
			mockClient.events <- event
			time.Sleep(20 * time.Millisecond)
		}

		// Wait for all events to be processed
		time.Sleep(100 * time.Millisecond)

		// Cancel and wait
		cancel()
		time.Sleep(100 * time.Millisecond)

		// Verify all events were sent
		calls := mockSender.GetSendAllCalls()
		if len(calls) != 3 {
			t.Errorf("Expected 3 SendAll calls, got %d", len(calls))
		}
	})
}

// TestApp_EventRepository tests the event repository integration
func TestApp_EventRepository(t *testing.T) {
	t.Run("saves event when eventRepo is configured", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)
		eventRepo := newMockEventRepository()

		app := NewApp(cfg, repo, WithEventRepository(eventRepo))
		mockSender := newMockSender()
		app.sender = mockSender

		now := time.Now()
		event := &mockEvent{
			id:            "test-event-save-123",
			eventType:     source.EventTypeEarthquake,
			severity:      50,
			source:        "p2pquake",
			affectedAreas: []string{"Tokyo", "Osaka"},
			occurredAt:    now,
			receivedAt:    now,
			rawJSON:       `{"_id":"test-event-save-123","code":551}`,
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		// Verify event was saved
		savedEvents := eventRepo.GetEvents()
		if len(savedEvents) != 1 {
			t.Fatalf("Expected 1 saved event, got %d", len(savedEvents))
		}

		saved := savedEvents[0]
		if saved.ID != "test-event-save-123" {
			t.Errorf("Expected ID 'test-event-save-123', got '%s'", saved.ID)
		}
		if saved.Severity != 50 {
			t.Errorf("Expected severity 50, got %d", saved.Severity)
		}
		if saved.Source != "p2pquake" {
			t.Errorf("Expected source 'p2pquake', got '%s'", saved.Source)
		}

		// Verify webhook was also called
		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Errorf("Expected 1 SendAll call, got %d", len(calls))
		}
	})

	t.Run("continues processing when event save fails", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)
		eventRepo := newMockEventRepository()
		eventRepo.createErr = errors.New("database connection failed")

		app := NewApp(cfg, repo, WithEventRepository(eventRepo))
		mockSender := newMockSender()
		app.sender = mockSender

		event := &mockEvent{
			id:       "test-event-fail-456",
			severity: 60,
			source:   "p2pquake",
			rawJSON:  `{"_id":"test-event-fail-456","code":551}`,
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		// Verify no events were saved (due to error)
		savedEvents := eventRepo.GetEvents()
		if len(savedEvents) != 0 {
			t.Errorf("Expected 0 saved events, got %d", len(savedEvents))
		}

		// Verify webhook was still called despite save failure
		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Fatalf("Expected 1 SendAll call, got %d", len(calls))
		}

		// Verify correct payload was sent
		if string(calls[0].payload) != `{"_id":"test-event-fail-456","code":551}` {
			t.Errorf("Unexpected payload: %s", string(calls[0].payload))
		}
	})

	t.Run("backward compatibility - works without eventRepo", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}

		subs := []subscription.Subscription{
			{
				Name: "Webhook 1",
				Delivery: subscription.DeliveryConfig{
					Type:   "webhook",
					URL:    "https://webhook1.example.com",
					Secret: "secret1",
				},
			},
		}
		repo := newMockRepository(subs)

		// Create app without event repository (backward compatible)
		app := NewApp(cfg, repo)
		mockSender := newMockSender()
		app.sender = mockSender

		event := &mockEvent{
			id:       "test-event-no-repo-789",
			severity: 70,
			source:   "p2pquake",
			rawJSON:  `{"_id":"test-event-no-repo-789","code":551}`,
		}

		ctx := context.Background()
		app.handleEvent(ctx, event)

		// Verify webhook was called
		calls := mockSender.GetSendAllCalls()
		if len(calls) != 1 {
			t.Fatalf("Expected 1 SendAll call, got %d", len(calls))
		}

		// Verify correct payload was sent
		if string(calls[0].payload) != `{"_id":"test-event-no-repo-789","code":551}` {
			t.Errorf("Unexpected payload: %s", string(calls[0].payload))
		}
	})

	t.Run("app eventRepo is nil by default", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}
		repo := newMockRepository([]subscription.Subscription{})

		app := NewApp(cfg, repo)

		if app.eventRepo != nil {
			t.Error("Expected eventRepo to be nil by default")
		}
	})

	t.Run("WithEventRepository option sets eventRepo", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
		}
		repo := newMockRepository([]subscription.Subscription{})
		eventRepo := newMockEventRepository()

		app := NewApp(cfg, repo, WithEventRepository(eventRepo))

		if app.eventRepo == nil {
			t.Error("Expected eventRepo to be set")
		}
		if app.eventRepo != eventRepo {
			t.Error("Expected eventRepo to match provided repository")
		}
	})
}
