package app

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/config"
	"github.com/ayanel/namazu/internal/delivery/webhook"
	"github.com/ayanel/namazu/internal/source"
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
	t.Run("creates app with valid config", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
				{
					Name: "Webhook 2",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook2.example.com",
						Secret: "secret2",
					},
				},
			},
		}

		app := NewApp(cfg)

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
		if len(app.targets) != 2 {
			t.Errorf("Expected 2 subscription targets, got %d", len(app.targets))
		}
	})

	t.Run("converts subscription configs to targets", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)

		if len(app.targets) != 1 {
			t.Fatalf("Expected 1 target, got %d", len(app.targets))
		}
		target := app.targets[0]
		if target.URL != "https://webhook1.example.com" {
			t.Errorf("Expected URL 'https://webhook1.example.com', got '%s'", target.URL)
		}
		if target.Secret != "secret1" {
			t.Errorf("Expected secret 'secret1', got '%s'", target.Secret)
		}
		if target.Name != "Webhook 1" {
			t.Errorf("Expected name 'Webhook 1', got '%s'", target.Name)
		}
	})

	t.Run("handles empty subscription list", func(t *testing.T) {
		cfg := &config.Config{
			Source: config.SourceConfig{
				Type:     "p2pquake",
				Endpoint: "ws://example.com/ws",
			},
			Subscriptions: []config.SubscriptionConfig{},
		}

		app := NewApp(cfg)

		if len(app.targets) != 0 {
			t.Errorf("Expected 0 targets, got %d", len(app.targets))
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
				{
					Name: "Webhook 2",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook2.example.com",
						Secret: "secret2",
					},
				},
			},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)
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
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		app := NewApp(cfg)
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
