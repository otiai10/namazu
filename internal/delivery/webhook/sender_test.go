package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestNewSender_DefaultTimeout verifies that NewSender creates a sender with default 10s timeout
func TestNewSender_DefaultTimeout(t *testing.T) {
	sender := NewSender()

	if sender == nil {
		t.Fatal("NewSender returned nil")
	}

	if sender.timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", sender.timeout)
	}

	if sender.client == nil {
		t.Error("client should be initialized")
	}

	if sender.client.Timeout != 10*time.Second {
		t.Errorf("expected client timeout 10s, got %v", sender.client.Timeout)
	}
}

// TestWithTimeout_Option verifies that WithTimeout option works correctly
func TestWithTimeout_Option(t *testing.T) {
	customTimeout := 5 * time.Second
	sender := NewSender(WithTimeout(customTimeout))

	if sender.timeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, sender.timeout)
	}

	if sender.client.Timeout != customTimeout {
		t.Errorf("expected client timeout %v, got %v", customTimeout, sender.client.Timeout)
	}
}

// TestSend_Success verifies successful webhook delivery
func TestSend_Success(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"test","data":"value"}`)

	// Create mock server
	var receivedSignature string
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Signature-256")
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender()
	ctx := context.Background()
	result := sender.Send(ctx, server.URL, secret, payload)

	// Verify result
	if !result.Success {
		t.Errorf("expected success=true, got false: %s", result.ErrorMessage)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}

	if result.URL != server.URL {
		t.Errorf("expected URL %s, got %s", server.URL, result.URL)
	}

	if result.ErrorMessage != "" {
		t.Errorf("expected no error message, got %s", result.ErrorMessage)
	}

	if result.ResponseTime <= 0 {
		t.Error("expected positive response time")
	}

	// Verify headers
	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", receivedContentType)
	}

	// Verify signature
	expectedSignature := Sign(secret, payload)
	if receivedSignature != expectedSignature {
		t.Errorf("expected signature %s, got %s", expectedSignature, receivedSignature)
	}

	// Verify body
	if string(receivedBody) != string(payload) {
		t.Errorf("expected body %s, got %s", payload, receivedBody)
	}
}

// TestSend_SignatureHeader verifies X-Signature-256 header is set correctly
func TestSend_SignatureHeader(t *testing.T) {
	testCases := []struct {
		name    string
		secret  string
		payload []byte
	}{
		{
			name:    "simple payload",
			secret:  "secret123",
			payload: []byte(`{"test":"data"}`),
		},
		{
			name:    "empty payload",
			secret:  "secret456",
			payload: []byte{},
		},
		{
			name:    "complex payload",
			secret:  "complex-secret",
			payload: []byte(`{"event":"user.created","data":{"id":123,"name":"John"}}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var receivedSignature string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedSignature = r.Header.Get("X-Signature-256")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewSender()
			ctx := context.Background()
			result := sender.Send(ctx, server.URL, tc.secret, tc.payload)

			if !result.Success {
				t.Fatalf("send failed: %s", result.ErrorMessage)
			}

			expectedSignature := Sign(tc.secret, tc.payload)
			if receivedSignature != expectedSignature {
				t.Errorf("signature mismatch: expected %s, got %s", expectedSignature, receivedSignature)
			}

			// Verify signature is verifiable
			if !Verify(tc.secret, tc.payload, receivedSignature) {
				t.Error("signature verification failed")
			}
		})
	}
}

// TestSend_Non2xxStatus verifies handling of non-2xx status codes
func TestSend_Non2xxStatus(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		wantSuccess bool
	}{
		{"200 OK", http.StatusOK, true},
		{"201 Created", http.StatusCreated, true},
		{"204 No Content", http.StatusNoContent, true},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"401 Unauthorized", http.StatusUnauthorized, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal Server Error", http.StatusInternalServerError, false},
		{"503 Service Unavailable", http.StatusServiceUnavailable, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			sender := NewSender()
			ctx := context.Background()
			result := sender.Send(ctx, server.URL, "secret", []byte(`{}`))

			if result.Success != tc.wantSuccess {
				t.Errorf("expected success=%v, got %v", tc.wantSuccess, result.Success)
			}

			if result.StatusCode != tc.statusCode {
				t.Errorf("expected status %d, got %d", tc.statusCode, result.StatusCode)
			}

			if !tc.wantSuccess && result.ErrorMessage == "" {
				t.Error("expected error message for non-success status")
			}
		})
	}
}

// TestSend_ConnectionRefused verifies handling of connection errors
func TestSend_ConnectionRefused(t *testing.T) {
	sender := NewSender()
	ctx := context.Background()

	// Use invalid URL that will refuse connection
	result := sender.Send(ctx, "http://localhost:1", "secret", []byte(`{}`))

	if result.Success {
		t.Error("expected failure for connection refused")
	}

	if result.ErrorMessage == "" {
		t.Error("expected error message")
	}

	if result.ResponseTime <= 0 {
		t.Error("expected positive response time even on error")
	}
}

// TestSend_Timeout verifies timeout handling
func TestSend_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create sender with very short timeout
	sender := NewSender(WithTimeout(50 * time.Millisecond))
	ctx := context.Background()
	result := sender.Send(ctx, server.URL, "secret", []byte(`{}`))

	if result.Success {
		t.Error("expected failure for timeout")
	}

	if result.ErrorMessage == "" {
		t.Error("expected error message")
	}

	if result.StatusCode != 0 {
		t.Errorf("expected status 0 for timeout, got %d", result.StatusCode)
	}
}

// TestSend_ContextCancellation verifies context cancellation handling
func TestSend_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	result := sender.Send(ctx, server.URL, "secret", []byte(`{}`))

	if result.Success {
		t.Error("expected failure for cancelled context")
	}

	if result.ErrorMessage == "" {
		t.Error("expected error message")
	}
}

// TestSendAll_Success verifies concurrent sending to multiple webhooks
func TestSendAll_Success(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"test"}`)

	// Track received requests
	mu := &sync.Mutex{}
	receivedURLs := make(map[string]bool)

	// Create multiple mock servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedURLs["server1"] = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedURLs["server2"] = true
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedURLs["server3"] = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server3.Close()

	targets := []Target{
		{URL: server1.URL, Secret: secret, Name: "server1"},
		{URL: server2.URL, Secret: secret, Name: "server2"},
		{URL: server3.URL, Secret: secret, Name: "server3"},
	}

	sender := NewSender()
	ctx := context.Background()
	results := sender.SendAll(ctx, targets, payload)

	// Verify results
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("result %d: expected success, got error: %s", i, result.ErrorMessage)
		}

		if result.URL != targets[i].URL {
			t.Errorf("result %d: URL mismatch", i)
		}

		if result.ResponseTime <= 0 {
			t.Errorf("result %d: expected positive response time", i)
		}
	}

	// Verify all servers received requests (concurrent execution)
	mu.Lock()
	defer mu.Unlock()
	if len(receivedURLs) != 3 {
		t.Errorf("expected 3 servers to receive requests, got %d", len(receivedURLs))
	}
}

// TestSendAll_MixedResults verifies handling of mixed success/failure
func TestSendAll_MixedResults(t *testing.T) {
	payload := []byte(`{"test":"data"}`)

	// Server 1: Success
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Server 2: Failure
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	targets := []Target{
		{URL: server1.URL, Secret: "secret1", Name: "success-server"},
		{URL: server2.URL, Secret: "secret2", Name: "error-server"},
		{URL: "http://localhost:1", Secret: "secret3", Name: "connection-refused"},
	}

	sender := NewSender()
	ctx := context.Background()
	results := sender.SendAll(ctx, targets, payload)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Result 1: Success
	if !results[0].Success {
		t.Error("result 0: expected success")
	}

	// Result 2: Server error
	if results[1].Success {
		t.Error("result 1: expected failure")
	}
	if results[1].StatusCode != http.StatusInternalServerError {
		t.Errorf("result 1: expected status 500, got %d", results[1].StatusCode)
	}

	// Result 3: Connection refused
	if results[2].Success {
		t.Error("result 2: expected failure")
	}
	if results[2].ErrorMessage == "" {
		t.Error("result 2: expected error message")
	}
}

// TestSendAll_EmptyWebhooks verifies handling of empty webhook list
func TestSendAll_EmptyTargets(t *testing.T) {
	sender := NewSender()
	ctx := context.Background()
	results := sender.SendAll(ctx, []Target{}, []byte(`{}`))

	if results == nil {
		t.Error("expected non-nil results")
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestSendAll_Concurrency verifies that requests are sent concurrently
func TestSendAll_Concurrency(t *testing.T) {
	mu := &sync.Mutex{}
	activeRequests := 0
	maxConcurrent := 0

	// Create servers that simulate slow endpoints
	createServer := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			activeRequests++
			if activeRequests > maxConcurrent {
				maxConcurrent = activeRequests
			}
			mu.Unlock()

			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			activeRequests--
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
	}

	server1 := createServer()
	defer server1.Close()
	server2 := createServer()
	defer server2.Close()
	server3 := createServer()
	defer server3.Close()

	targets := []Target{
		{URL: server1.URL, Secret: "s1"},
		{URL: server2.URL, Secret: "s2"},
		{URL: server3.URL, Secret: "s3"},
	}

	sender := NewSender()
	ctx := context.Background()
	start := time.Now()
	results := sender.SendAll(ctx, targets, []byte(`{}`))
	elapsed := time.Since(start)

	// Verify all succeeded
	for i, result := range results {
		if !result.Success {
			t.Errorf("result %d failed: %s", i, result.ErrorMessage)
		}
	}

	// Verify concurrent execution
	// If sequential, would take 150ms+. If concurrent, ~50-100ms
	if elapsed > 120*time.Millisecond {
		t.Errorf("expected concurrent execution (~50-100ms), took %v", elapsed)
	}

	// Verify max concurrent requests
	mu.Lock()
	defer mu.Unlock()
	if maxConcurrent < 2 {
		t.Errorf("expected at least 2 concurrent requests, got %d", maxConcurrent)
	}
}

// TestSend_InvalidURL verifies handling of invalid URLs
func TestSend_InvalidURL(t *testing.T) {
	sender := NewSender()
	ctx := context.Background()

	result := sender.Send(ctx, "not-a-valid-url", "secret", []byte(`{}`))

	if result.Success {
		t.Error("expected failure for invalid URL")
	}

	if result.ErrorMessage == "" {
		t.Error("expected error message")
	}
}

// TestDeliveryResult_Structure verifies DeliveryResult contains all expected fields
func TestDeliveryResult_Structure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sender := NewSender()
	ctx := context.Background()
	result := sender.Send(ctx, server.URL, "secret", []byte(`{}`))

	// Verify all fields are populated
	if result.URL == "" {
		t.Error("URL should be populated")
	}

	if result.StatusCode == 0 {
		t.Error("StatusCode should be populated")
	}

	// Success should be determinable
	if result.StatusCode >= 200 && result.StatusCode < 300 && !result.Success {
		t.Error("Success should be true for 2xx status")
	}

	if result.ResponseTime == 0 {
		t.Error("ResponseTime should be populated")
	}

	// ErrorMessage may or may not be populated depending on success
}

// Benchmark tests
func BenchmarkSend_Success(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender()
	ctx := context.Background()
	payload := []byte(`{"event":"test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.Send(ctx, server.URL, "secret", payload)
	}
}

func BenchmarkSendAll_10Targets(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targets := make([]Target, 10)
	for i := 0; i < 10; i++ {
		targets[i] = Target{
			URL:    server.URL,
			Secret: "secret",
		}
	}

	sender := NewSender()
	ctx := context.Background()
	payload := []byte(`{"event":"test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sender.SendAll(ctx, targets, payload)
	}
}
