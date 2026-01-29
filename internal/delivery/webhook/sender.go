package webhook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DeliveryResult contains the result of a webhook delivery attempt.
// It provides detailed information about the delivery including timing,
// status codes, and any errors that occurred.
type DeliveryResult struct {
	URL          string        // The webhook URL that was targeted
	StatusCode   int           // HTTP status code (0 if request failed)
	Success      bool          // True if status code is 2xx
	ErrorMessage string        // Error description if delivery failed
	ResponseTime time.Duration // Time taken for the request
}

// Sender sends webhook notifications with configurable timeout and
// automatic signature generation using HMAC-SHA256.
//
// Sender is safe for concurrent use by multiple goroutines.
type Sender struct {
	client  *http.Client
	timeout time.Duration
}

// SenderOption configures the Sender
type SenderOption func(*Sender)

// WithTimeout sets the HTTP request timeout.
// Default timeout is 10 seconds if not specified.
//
// Example:
//
//	sender := webhook.NewSender(webhook.WithTimeout(5 * time.Second))
func WithTimeout(d time.Duration) SenderOption {
	return func(s *Sender) {
		s.timeout = d
	}
}

// NewSender creates a new webhook sender with the given options.
// The default timeout is 10 seconds.
//
// Example:
//
//	// Default configuration (10s timeout)
//	sender := webhook.NewSender()
//
//	// Custom timeout
//	sender := webhook.NewSender(webhook.WithTimeout(5 * time.Second))
func NewSender(opts ...SenderOption) *Sender {
	s := &Sender{
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.client = &http.Client{
		Timeout: s.timeout,
	}
	return s
}

// Send sends a payload to a single webhook endpoint via HTTP POST.
//
// The request includes:
//   - Content-Type: application/json
//   - X-Signature-256: HMAC-SHA256 signature for verification
//   - User-Agent: namazu/1.0
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - url: The webhook endpoint URL
//   - secret: Secret key for HMAC signature generation
//   - payload: JSON payload to send (typically in bytes)
//
// Returns:
//
//	DeliveryResult with success status, timing, and any errors
//
// The method respects context cancellation and the configured timeout.
// Success is defined as receiving a 2xx HTTP status code.
//
// Example:
//
//	sender := webhook.NewSender()
//	payload := []byte(`{"event":"user.created","user_id":123}`)
//	result := sender.Send(ctx, "https://api.example.com/webhook", "secret", payload)
//	if result.Success {
//	    log.Printf("Delivered in %v", result.ResponseTime)
//	}
func (s *Sender) Send(ctx context.Context, url, secret string, payload []byte) DeliveryResult {
	start := time.Now()
	result := DeliveryResult{
		URL: url,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-256", Sign(secret, payload))
	req.Header.Set("User-Agent", "namazu/1.0")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("request failed: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	result.ResponseTime = time.Since(start)

	if !result.Success {
		result.ErrorMessage = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}

	return result
}

// SendAll sends a payload to multiple targets concurrently using goroutines.
// Each target is sent in parallel for optimal performance.
//
// The method waits for all deliveries to complete before returning,
// ensuring all results are available when the function returns.
//
// Parameters:
//   - ctx: Context for cancellation (applies to all targets)
//   - targets: List of webhook targets with URL and secret
//   - payload: JSON payload to send to all targets
//
// Returns:
//
//	Slice of DeliveryResult in the same order as the input targets
//
// Example:
//
//	targets := []webhook.Target{
//	    {URL: "https://api1.example.com/hook", Secret: "secret1"},
//	    {URL: "https://api2.example.com/hook", Secret: "secret2"},
//	}
//	results := sender.SendAll(ctx, targets, payload)
//	for i, result := range results {
//	    if result.Success {
//	        log.Printf("Target %d delivered", i)
//	    }
//	}
func (s *Sender) SendAll(ctx context.Context, targets []Target, payload []byte) []DeliveryResult {
	if len(targets) == 0 {
		return []DeliveryResult{}
	}

	results := make([]DeliveryResult, len(targets))
	var wg sync.WaitGroup

	for i, target := range targets {
		wg.Add(1)
		go func(index int, t Target) {
			defer wg.Done()
			results[index] = s.Send(ctx, t.URL, t.Secret, payload)
		}(i, target)
	}

	wg.Wait()
	return results
}

// Target represents a webhook destination with its configuration.
type Target struct {
	URL    string // The webhook endpoint URL
	Secret string // Secret key for HMAC signature generation
	Name   string // Optional human-readable name for logging/debugging
}
