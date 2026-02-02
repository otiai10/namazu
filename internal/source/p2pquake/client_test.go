package p2pquake

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/source"
	"github.com/gorilla/websocket"
)

// Test NewClient creates client correctly
func TestNewClient(t *testing.T) {
	endpoint := "wss://api.example.com/ws"
	client := NewClient(endpoint)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.endpoint != endpoint {
		t.Errorf("endpoint = %q, want %q", client.endpoint, endpoint)
	}

	if client.events == nil {
		t.Error("events channel is nil")
	}

	if client.done == nil {
		t.Error("done channel is nil")
	}

	if client.seenIDs == nil {
		t.Error("seenIDs map is nil")
	}

	if client.maxSeenIDs != 1000 {
		t.Errorf("maxSeenIDs = %d, want 1000", client.maxSeenIDs)
	}

	if len(client.seenIDsList) != 0 {
		t.Errorf("seenIDsList should be empty initially, got length %d", len(client.seenIDsList))
	}
}

// Test isDuplicate deduplication logic
func TestClient_IsDuplicate(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")

	tests := []struct {
		name       string
		id         string
		wantResult bool
	}{
		{
			name:       "First occurrence - not duplicate",
			id:         "id-001",
			wantResult: false,
		},
		{
			name:       "Second occurrence - is duplicate",
			id:         "id-001",
			wantResult: true,
		},
		{
			name:       "Different ID - not duplicate",
			id:         "id-002",
			wantResult: false,
		},
		{
			name:       "Third occurrence - still duplicate",
			id:         "id-001",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isDuplicate(tt.id)
			if result != tt.wantResult {
				t.Errorf("isDuplicate(%q) = %v, want %v", tt.id, result, tt.wantResult)
			}
		})
	}
}

// Test LRU eviction when exceeding maxSeenIDs
func TestClient_IsDuplicate_LRUEviction(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")
	client.maxSeenIDs = 3 // Set small limit for testing

	// Add 3 IDs (within limit)
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("id-%03d", i)
		if client.isDuplicate(id) {
			t.Errorf("First occurrence of %q should not be duplicate", id)
		}
	}

	// Verify all 3 are in the map
	if len(client.seenIDs) != 3 {
		t.Errorf("seenIDs map length = %d, want 3", len(client.seenIDs))
	}

	// Add 4th ID - should evict oldest (id-001)
	if client.isDuplicate("id-004") {
		t.Error("First occurrence of id-004 should not be duplicate")
	}

	// Verify map still has 3 items
	if len(client.seenIDs) != 3 {
		t.Errorf("seenIDs map length = %d, want 3 (after eviction)", len(client.seenIDs))
	}

	// Verify oldest (id-001) was evicted
	if _, exists := client.seenIDs["id-001"]; exists {
		t.Error("id-001 should have been evicted")
	}

	// Verify newest 3 are present
	for _, id := range []string{"id-002", "id-003", "id-004"} {
		if _, exists := client.seenIDs[id]; !exists {
			t.Errorf("%q should still be in map", id)
		}
	}

	// id-001 should now be treated as new (since it was evicted)
	if client.isDuplicate("id-001") {
		t.Error("id-001 should not be duplicate after eviction")
	}
}

// Test LRU list maintains correct order
func TestClient_IsDuplicate_LRUOrder(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")
	client.maxSeenIDs = 5

	// Add IDs in order
	ids := []string{"id-A", "id-B", "id-C", "id-D", "id-E"}
	for _, id := range ids {
		client.isDuplicate(id)
	}

	// Verify list order
	for i, id := range ids {
		if client.seenIDsList[i] != id {
			t.Errorf("seenIDsList[%d] = %q, want %q", i, client.seenIDsList[i], id)
		}
	}

	// Add one more - oldest should be evicted
	client.isDuplicate("id-F")

	// Verify id-A was removed from both map and list
	if _, exists := client.seenIDs["id-A"]; exists {
		t.Error("id-A should have been evicted from map")
	}

	// Verify list doesn't contain id-A
	for _, id := range client.seenIDsList {
		if id == "id-A" {
			t.Error("id-A should have been evicted from list")
		}
	}

	// Verify list has correct items in order
	expectedList := []string{"id-B", "id-C", "id-D", "id-E", "id-F"}
	if len(client.seenIDsList) != len(expectedList) {
		t.Fatalf("seenIDsList length = %d, want %d", len(client.seenIDsList), len(expectedList))
	}

	for i, expected := range expectedList {
		if client.seenIDsList[i] != expected {
			t.Errorf("seenIDsList[%d] = %q, want %q", i, client.seenIDsList[i], expected)
		}
	}
}

// Mock WebSocket server for testing
func newMockWSServer(t *testing.T, handler func(conn *websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		handler(conn)
	}))

	return server
}

// Test message parsing with code 551 (JMAQuake)
func TestClient_MessageParsing_Code551(t *testing.T) {
	// Create mock server that sends a code 551 message
	server := newMockWSServer(t, func(conn *websocket.Conn) {
		quake := JMAQuake{
			ID:   "test-quake-001",
			Code: 551,
			Time: "2024/01/15 12:34:56",
			Issue: Issue{
				Source: "気象庁",
				Time:   "2024/01/15 12:35:00",
				Type:   "ScalePrompt",
			},
			Earthquake: &Earthquake{
				Time: "2024/01/15 12:34:45",
				Hypocenter: Hypocenter{
					Name:      "石川県能登地方",
					Latitude:  37.5,
					Longitude: 137.2,
					Depth:     10,
					Magnitude: 7.6,
				},
				MaxScale:        Scale6Strong,
				DomesticTsunami: "Warning",
			},
		}

		data, _ := json.Marshal(quake)
		_ = conn.WriteMessage(websocket.TextMessage, data)

		// Keep connection open
		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start client
	go func() { _ = client.Connect(ctx) }()

	// Wait for event
	select {
	case event := <-client.Events():
		if event.GetID() != "test-quake-001" {
			t.Errorf("GetID() = %q, want %q", event.GetID(), "test-quake-001")
		}
		if event.GetType() != source.EventTypeEarthquake {
			t.Errorf("GetType() = %q, want %q", event.GetType(), source.EventTypeEarthquake)
		}
		if event.GetSeverity() != 80 { // Scale6Strong = 80
			t.Errorf("GetSeverity() = %d, want 80", event.GetSeverity())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

// Test filtering non-551 codes
func TestClient_MessageParsing_FilterNon551(t *testing.T) {
	server := newMockWSServer(t, func(conn *websocket.Conn) {
		// Send non-551 message (should be filtered out)
		message := map[string]interface{}{
			"_id":  "test-other-001",
			"code": 555, // Different code
			"time": "2024/01/15 12:34:56",
		}
		data, _ := json.Marshal(message)
		_ = conn.WriteMessage(websocket.TextMessage, data)

		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() { _ = client.Connect(ctx) }()

	// Should NOT receive event (filtered out)
	select {
	case event := <-client.Events():
		t.Fatalf("Should not receive non-551 event, got: %v", event)
	case <-time.After(500 * time.Millisecond):
		// Expected - no event received
	}
}

// Test deduplication in message stream
func TestClient_MessageParsing_Deduplication(t *testing.T) {
	server := newMockWSServer(t, func(conn *websocket.Conn) {
		quake := JMAQuake{
			ID:   "duplicate-test-001",
			Code: 551,
			Time: "2024/01/15 12:34:56",
		}
		data, _ := json.Marshal(quake)

		// Send same message twice
		_ = conn.WriteMessage(websocket.TextMessage, data)
		time.Sleep(50 * time.Millisecond)
		_ = conn.WriteMessage(websocket.TextMessage, data)

		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() { _ = client.Connect(ctx) }()

	// Should only receive ONE event (second is duplicate)
	receivedCount := 0
	timeout := time.After(500 * time.Millisecond)

	for {
		select {
		case event := <-client.Events():
			receivedCount++
			if event.GetID() != "duplicate-test-001" {
				t.Errorf("Unexpected event ID: %q", event.GetID())
			}
		case <-timeout:
			if receivedCount != 1 {
				t.Errorf("Received %d events, want 1 (duplicates should be filtered)", receivedCount)
			}
			return
		}
	}
}

// Test Close method
func TestClient_Close(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")

	// Close should not panic on nil connection
	err := client.Close()
	if err != nil {
		t.Errorf("Close() with nil connection should not error, got: %v", err)
	}

	// Verify done channel is closed
	select {
	case <-client.done:
		// Expected - channel closed
	case <-time.After(100 * time.Millisecond):
		t.Error("done channel should be closed")
	}
}

// Test Events channel returns correct channel
func TestClient_Events(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")

	ch := client.Events()
	if ch == nil {
		t.Error("Events() returned nil channel")
	}

	// Verify it's the same channel as internal events
	if ch != client.events {
		t.Error("Events() should return internal events channel")
	}
}

// Test concurrent access to isDuplicate (race condition check)
func TestClient_IsDuplicate_ConcurrentAccess(t *testing.T) {
	client := NewClient("wss://test.example.com/ws")

	done := make(chan bool)
	iterations := 100

	// Run multiple goroutines concurrently
	for i := 0; i < 10; i++ {
		go func(routineID int) {
			for j := 0; j < iterations; j++ {
				id := fmt.Sprintf("routine-%d-id-%d", routineID, j)
				client.isDuplicate(id)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify no race conditions caused incorrect state
	if len(client.seenIDs) > client.maxSeenIDs {
		t.Errorf("seenIDs map size %d exceeds maxSeenIDs %d", len(client.seenIDs), client.maxSeenIDs)
	}
}

// Test context cancellation
func TestClient_ContextCancellation(t *testing.T) {
	server := newMockWSServer(t, func(conn *websocket.Conn) {
		// Keep connection alive
		for {
			time.Sleep(100 * time.Millisecond)
		}
	})
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	client := NewClient(wsURL)
	ctx, cancel := context.WithCancel(context.Background())

	// Start client
	go func() { _ = client.Connect(ctx) }()

	// Give it time to connect
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Client should handle cancellation gracefully
	time.Sleep(200 * time.Millisecond)

	// No panic means test passes
}

// Benchmark isDuplicate operation
func BenchmarkClient_IsDuplicate(b *testing.B) {
	client := NewClient("wss://test.example.com/ws")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id-%d", i%1000) // Cycle through 1000 IDs
		client.isDuplicate(id)
	}
}

// Benchmark isDuplicate with high eviction rate
func BenchmarkClient_IsDuplicate_HighEviction(b *testing.B) {
	client := NewClient("wss://test.example.com/ws")
	client.maxSeenIDs = 100 // Small cache for high eviction

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("id-%d", i) // Always new IDs
		client.isDuplicate(id)
	}
}
