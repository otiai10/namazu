package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestDefaultRetryConfig verifies default values are sensible
func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if !cfg.Enabled {
		t.Error("expected Enabled=true by default")
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}

	if cfg.InitialMs != 1000 {
		t.Errorf("expected InitialMs=1000, got %d", cfg.InitialMs)
	}

	if cfg.MaxMs != 60000 {
		t.Errorf("expected MaxMs=60000, got %d", cfg.MaxMs)
	}
}

// TestNewRetryingSender verifies construction
func TestNewRetryingSender(t *testing.T) {
	sender := NewSender()
	cfg := DefaultRetryConfig()

	rs := NewRetryingSender(sender, cfg)

	if rs == nil {
		t.Fatal("NewRetryingSender returned nil")
	}

	if rs.sender != sender {
		t.Error("sender not set correctly")
	}

	if rs.config != cfg {
		t.Error("config not set correctly")
	}
}

// TestRetryingSender_SuccessOnFirstAttempt verifies no retry when first attempt succeeds
func TestRetryingSender_SuccessOnFirstAttempt(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 3,
		InitialMs:  100,
		MaxMs:      1000,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.ErrorMessage)
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt, got %d", atomic.LoadInt32(&attempts))
	}

	if result.RetryCount != 0 {
		t.Errorf("expected RetryCount=0, got %d", result.RetryCount)
	}
}

// TestRetryingSender_SuccessAfterRetry verifies retry on 5xx then success
func TestRetryingSender_SuccessAfterRetry(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 5,
		InitialMs:  10, // Short for testing
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if !result.Success {
		t.Errorf("expected success after retry, got error: %s", result.ErrorMessage)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}

	if result.RetryCount != 2 {
		t.Errorf("expected RetryCount=2, got %d", result.RetryCount)
	}
}

// TestRetryingSender_NonRetryableError verifies 4xx stops immediately
func TestRetryingSender_NonRetryableError(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"401 Unauthorized", http.StatusUnauthorized},
		{"403 Forbidden", http.StatusForbidden},
		{"404 Not Found", http.StatusNotFound},
		{"422 Unprocessable Entity", http.StatusUnprocessableEntity},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var attempts int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			sender := NewSender()
			cfg := RetryConfig{
				Enabled:    true,
				MaxRetries: 3,
				InitialMs:  10,
				MaxMs:      100,
			}
			rs := NewRetryingSender(sender, cfg)

			target := Target{URL: server.URL, Secret: "secret"}
			result := rs.Send(context.Background(), target, []byte(`{}`))

			if result.Success {
				t.Error("expected failure")
			}

			if atomic.LoadInt32(&attempts) != 1 {
				t.Errorf("expected 1 attempt (no retry for %d), got %d", tc.statusCode, atomic.LoadInt32(&attempts))
			}

			if result.RetryCount != 0 {
				t.Errorf("expected RetryCount=0, got %d", result.RetryCount)
			}
		})
	}
}

// TestRetryingSender_RetryableErrors verifies retryable status codes
func TestRetryingSender_RetryableErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"408 Request Timeout", http.StatusRequestTimeout},
		{"429 Too Many Requests", http.StatusTooManyRequests},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var attempts int32

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			sender := NewSender()
			cfg := RetryConfig{
				Enabled:    true,
				MaxRetries: 2,
				InitialMs:  10,
				MaxMs:      100,
			}
			rs := NewRetryingSender(sender, cfg)

			target := Target{URL: server.URL, Secret: "secret"}
			result := rs.Send(context.Background(), target, []byte(`{}`))

			if result.Success {
				t.Error("expected failure after all retries")
			}

			// 1 initial + 2 retries = 3 total
			expectedAttempts := int32(3)
			if atomic.LoadInt32(&attempts) != expectedAttempts {
				t.Errorf("expected %d attempts for %d, got %d", expectedAttempts, tc.statusCode, atomic.LoadInt32(&attempts))
			}

			if result.RetryCount != 2 {
				t.Errorf("expected RetryCount=2, got %d", result.RetryCount)
			}
		})
	}
}

// TestRetryingSender_MaxRetriesExceeded verifies behavior when max retries reached
func TestRetryingSender_MaxRetriesExceeded(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 2,
		InitialMs:  10,
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if result.Success {
		t.Error("expected failure when max retries exceeded")
	}

	// 1 initial + 2 retries = 3 total
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}

	if result.RetryCount != 2 {
		t.Errorf("expected RetryCount=2, got %d", result.RetryCount)
	}
}

// TestRetryingSender_BackoffTiming verifies exponential backoff timing
func TestRetryingSender_BackoffTiming(t *testing.T) {
	var attempts int32
	var timestamps []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 3,
		InitialMs:  100, // 100ms, 200ms, 400ms
		MaxMs:      1000,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	start := time.Now()
	rs.Send(context.Background(), target, []byte(`{}`))
	totalDuration := time.Since(start)

	// Verify we have 4 attempts (1 initial + 3 retries)
	if len(timestamps) != 4 {
		t.Fatalf("expected 4 timestamps, got %d", len(timestamps))
	}

	// Verify delays between attempts
	// Attempt 1: immediate
	// Attempt 2: ~100ms after attempt 1
	// Attempt 3: ~200ms after attempt 2
	// Attempt 4: ~400ms after attempt 3

	delay1 := timestamps[1].Sub(timestamps[0])
	delay2 := timestamps[2].Sub(timestamps[1])
	delay3 := timestamps[3].Sub(timestamps[2])

	// Allow 50% tolerance for timing
	if delay1 < 50*time.Millisecond || delay1 > 200*time.Millisecond {
		t.Errorf("delay1 expected ~100ms, got %v", delay1)
	}

	if delay2 < 100*time.Millisecond || delay2 > 400*time.Millisecond {
		t.Errorf("delay2 expected ~200ms, got %v", delay2)
	}

	if delay3 < 200*time.Millisecond || delay3 > 800*time.Millisecond {
		t.Errorf("delay3 expected ~400ms, got %v", delay3)
	}

	// Total should be around 100 + 200 + 400 = 700ms
	if totalDuration < 500*time.Millisecond || totalDuration > 1500*time.Millisecond {
		t.Errorf("total duration expected ~700ms, got %v", totalDuration)
	}
}

// TestRetryingSender_BackoffMaxCap verifies backoff is capped at MaxMs
func TestRetryingSender_BackoffMaxCap(t *testing.T) {
	var timestamps []time.Time

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 4,
		InitialMs:  50, // 50, 100, 200, 400, but capped at 150
		MaxMs:      150,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	rs.Send(context.Background(), target, []byte(`{}`))

	// Verify we have 5 attempts
	if len(timestamps) != 5 {
		t.Fatalf("expected 5 timestamps, got %d", len(timestamps))
	}

	// Delay 3 and 4 should be capped at 150ms
	delay3 := timestamps[3].Sub(timestamps[2])
	delay4 := timestamps[4].Sub(timestamps[3])

	// Without cap: delay3=200ms, delay4=400ms
	// With cap at 150ms: both should be ~150ms
	if delay3 > 250*time.Millisecond {
		t.Errorf("delay3 should be capped, got %v", delay3)
	}

	if delay4 > 250*time.Millisecond {
		t.Errorf("delay4 should be capped, got %v", delay4)
	}
}

// TestRetryingSender_ContextCancellation verifies context cancellation stops retries
func TestRetryingSender_ContextCancellation(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 10, // Would take a long time
		InitialMs:  100,
		MaxMs:      1000,
	}
	rs := NewRetryingSender(sender, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(ctx, target, []byte(`{}`))

	if result.Success {
		t.Error("expected failure on context cancellation")
	}

	// Should have stopped early due to cancellation
	if atomic.LoadInt32(&attempts) > 3 {
		t.Errorf("expected early stop on cancellation, got %d attempts", atomic.LoadInt32(&attempts))
	}
}

// TestRetryingSender_ConnectionError verifies retry on connection errors
func TestRetryingSender_ConnectionError(t *testing.T) {
	sender := NewSender(WithTimeout(50 * time.Millisecond))
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 2,
		InitialMs:  10,
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	// Use invalid address that will cause connection error
	target := Target{URL: "http://localhost:1", Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if result.Success {
		t.Error("expected failure for connection error")
	}

	// Connection errors should be retried
	if result.RetryCount != 2 {
		t.Errorf("expected RetryCount=2 for connection error, got %d", result.RetryCount)
	}
}

// TestRetryingSender_DisabledRetry verifies retry is disabled when Enabled=false
func TestRetryingSender_DisabledRetry(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    false, // Disabled
		MaxRetries: 5,
		InitialMs:  10,
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if result.Success {
		t.Error("expected failure")
	}

	// Should only attempt once when disabled
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt when retry disabled, got %d", atomic.LoadInt32(&attempts))
	}

	if result.RetryCount != 0 {
		t.Errorf("expected RetryCount=0, got %d", result.RetryCount)
	}
}

// TestRetryingSender_SendAll verifies SendAll with retry logic
func TestRetryingSender_SendAll(t *testing.T) {
	// Server 1: Success immediately
	var attempts1 int32
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts1, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Server 2: Success after 2 retries
	var attempts2 int32
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts2, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Server 3: Always fails with 4xx (non-retryable)
	var attempts3 int32
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts3, 1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server3.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 3,
		InitialMs:  10,
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	targets := []Target{
		{URL: server1.URL, Secret: "s1", Name: "server1"},
		{URL: server2.URL, Secret: "s2", Name: "server2"},
		{URL: server3.URL, Secret: "s3", Name: "server3"},
	}

	results := rs.SendAll(context.Background(), targets, []byte(`{}`))

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Server 1: Success on first attempt
	if !results[0].Success {
		t.Errorf("result 0: expected success")
	}
	if results[0].RetryCount != 0 {
		t.Errorf("result 0: expected RetryCount=0, got %d", results[0].RetryCount)
	}
	if atomic.LoadInt32(&attempts1) != 1 {
		t.Errorf("server1: expected 1 attempt, got %d", atomic.LoadInt32(&attempts1))
	}

	// Server 2: Success after retries
	if !results[1].Success {
		t.Errorf("result 1: expected success after retry")
	}
	if results[1].RetryCount != 2 {
		t.Errorf("result 1: expected RetryCount=2, got %d", results[1].RetryCount)
	}
	if atomic.LoadInt32(&attempts2) != 3 {
		t.Errorf("server2: expected 3 attempts, got %d", atomic.LoadInt32(&attempts2))
	}

	// Server 3: Failed with no retry (4xx)
	if results[2].Success {
		t.Errorf("result 2: expected failure")
	}
	if results[2].RetryCount != 0 {
		t.Errorf("result 2: expected RetryCount=0 for 4xx, got %d", results[2].RetryCount)
	}
	if atomic.LoadInt32(&attempts3) != 1 {
		t.Errorf("server3: expected 1 attempt for 4xx, got %d", atomic.LoadInt32(&attempts3))
	}
}

// TestRetryingSender_SendAll_EmptyTargets verifies empty targets
func TestRetryingSender_SendAll_EmptyTargets(t *testing.T) {
	sender := NewSender()
	cfg := DefaultRetryConfig()
	rs := NewRetryingSender(sender, cfg)

	results := rs.SendAll(context.Background(), []Target{}, []byte(`{}`))

	if results == nil {
		t.Error("expected non-nil results")
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestCalculateBackoff verifies backoff calculation
func TestCalculateBackoff(t *testing.T) {
	testCases := []struct {
		name      string
		attempt   int
		initialMs int
		maxMs     int
		expected  time.Duration
	}{
		{"attempt 0", 0, 1000, 60000, 1000 * time.Millisecond},
		{"attempt 1", 1, 1000, 60000, 2000 * time.Millisecond},
		{"attempt 2", 2, 1000, 60000, 4000 * time.Millisecond},
		{"attempt 3", 3, 1000, 60000, 8000 * time.Millisecond},
		{"capped at max", 10, 1000, 5000, 5000 * time.Millisecond},
		{"custom initial", 0, 500, 10000, 500 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateBackoff(tc.attempt, tc.initialMs, tc.maxMs)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestIsRetryable verifies error classification
func TestIsRetryable(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		isError    bool
		expected   bool
	}{
		// Retryable HTTP status codes
		{"408 Request Timeout", 408, false, true},
		{"429 Too Many Requests", 429, false, true},
		{"500 Internal Server Error", 500, false, true},
		{"502 Bad Gateway", 502, false, true},
		{"503 Service Unavailable", 503, false, true},
		{"504 Gateway Timeout", 504, false, true},

		// Non-retryable HTTP status codes
		{"200 OK", 200, false, false},
		{"201 Created", 201, false, false},
		{"400 Bad Request", 400, false, false},
		{"401 Unauthorized", 401, false, false},
		{"403 Forbidden", 403, false, false},
		{"404 Not Found", 404, false, false},
		{"422 Unprocessable Entity", 422, false, false},

		// Connection/network errors (status 0) are retryable
		{"Connection error", 0, true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := DeliveryResult{
				StatusCode:   tc.statusCode,
				Success:      tc.statusCode >= 200 && tc.statusCode < 300,
				ErrorMessage: "",
			}
			if tc.isError {
				result.ErrorMessage = "connection error"
			}

			retryable := isRetryable(result)
			if retryable != tc.expected {
				t.Errorf("expected isRetryable=%v for status %d, got %v", tc.expected, tc.statusCode, retryable)
			}
		})
	}
}

// TestRetryingSender_ZeroMaxRetries verifies behavior with MaxRetries=0
func TestRetryingSender_ZeroMaxRetries(t *testing.T) {
	var attempts int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	sender := NewSender()
	cfg := RetryConfig{
		Enabled:    true,
		MaxRetries: 0, // No retries
		InitialMs:  10,
		MaxMs:      100,
	}
	rs := NewRetryingSender(sender, cfg)

	target := Target{URL: server.URL, Secret: "secret"}
	result := rs.Send(context.Background(), target, []byte(`{}`))

	if result.Success {
		t.Error("expected failure")
	}

	// Only initial attempt, no retries
	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected 1 attempt with MaxRetries=0, got %d", atomic.LoadInt32(&attempts))
	}

	if result.RetryCount != 0 {
		t.Errorf("expected RetryCount=0, got %d", result.RetryCount)
	}
}
