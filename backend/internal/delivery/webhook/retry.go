package webhook

import (
	"context"
	"sync"
	"time"
)

// RetryConfig holds retry settings for webhook delivery
type RetryConfig struct {
	Enabled    bool `json:"enabled"`     // Whether retry is enabled
	MaxRetries int  `json:"max_retries"` // Maximum number of retry attempts (default: 3)
	InitialMs  int  `json:"initial_ms"`  // Initial backoff delay in milliseconds (default: 1000)
	MaxMs      int  `json:"max_ms"`      // Maximum backoff delay in milliseconds (default: 60000)
}

// DefaultRetryConfig returns sensible default retry configuration.
// Enabled with 3 retries, starting at 1 second and capped at 60 seconds.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:    true,
		MaxRetries: 3,
		InitialMs:  1000,
		MaxMs:      60000,
	}
}

// RetryingSender wraps a Sender with retry logic using exponential backoff.
// It is safe for concurrent use by multiple goroutines.
type RetryingSender struct {
	sender *Sender
	config RetryConfig
}

// NewRetryingSender creates a new retrying sender that wraps the given sender
// with the specified retry configuration.
func NewRetryingSender(sender *Sender, config RetryConfig) *RetryingSender {
	return &RetryingSender{
		sender: sender,
		config: config,
	}
}

// Send attempts delivery with retries using exponential backoff.
// It will retry on retryable errors (5xx, 408, 429, connection errors)
// and stop immediately on non-retryable errors (4xx except 408, 429).
//
// The backoff schedule is:
//   - Attempt 1: immediate
//   - Attempt 2: wait InitialMs
//   - Attempt 3: wait InitialMs * 2
//   - Attempt 4: wait InitialMs * 4
//   - (capped at MaxMs)
func (r *RetryingSender) Send(ctx context.Context, target Target, payload []byte) DeliveryResult {
	// If retry is disabled, just send once
	if !r.config.Enabled {
		return r.sender.sendTarget(ctx, target, payload)
	}

	var result DeliveryResult
	retryCount := 0

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Check context before each attempt
		if err := ctx.Err(); err != nil {
			result = DeliveryResult{
				URL:          target.URL,
				Success:      false,
				ErrorMessage: "context cancelled",
				RetryCount:   retryCount,
			}
			return result
		}

		// Wait before retry (not on first attempt)
		if attempt > 0 {
			backoff := calculateBackoff(attempt-1, r.config.InitialMs, r.config.MaxMs)
			select {
			case <-ctx.Done():
				result = DeliveryResult{
					URL:          target.URL,
					Success:      false,
					ErrorMessage: "context cancelled during backoff",
					RetryCount:   retryCount,
				}
				return result
			case <-time.After(backoff):
				// Continue with retry
			}
			retryCount++
		}

		// Attempt delivery
		result = r.sender.sendTarget(ctx, target, payload)
		result.RetryCount = retryCount

		// Success - no need to retry
		if result.Success {
			return result
		}

		// Check if error is retryable
		if !isRetryable(result) {
			return result
		}

		// Continue to next retry attempt
	}

	return result
}

// SendAll sends to all targets with individual retry logic.
// Each target is processed concurrently with its own retry handling.
func (r *RetryingSender) SendAll(ctx context.Context, targets []Target, payload []byte) []DeliveryResult {
	if len(targets) == 0 {
		return []DeliveryResult{}
	}

	results := make([]DeliveryResult, len(targets))
	var wg sync.WaitGroup

	for i, target := range targets {
		wg.Add(1)
		go func(index int, t Target) {
			defer wg.Done()
			results[index] = r.Send(ctx, t, payload)
		}(i, target)
	}

	wg.Wait()
	return results
}

// calculateBackoff returns the backoff duration for a given retry attempt.
// Uses exponential backoff: initialMs * 2^attempt, capped at maxMs.
func calculateBackoff(attempt, initialMs, maxMs int) time.Duration {
	backoffMs := initialMs
	for range attempt {
		backoffMs *= 2
		if backoffMs >= maxMs {
			return time.Duration(maxMs) * time.Millisecond
		}
	}

	return time.Duration(backoffMs) * time.Millisecond
}

// isRetryable determines if a delivery result indicates a retryable error.
// Retryable errors include:
//   - HTTP 5xx (server errors)
//   - HTTP 408 (Request Timeout)
//   - HTTP 429 (Too Many Requests)
//   - Connection errors (status code 0 with error message)
func isRetryable(result DeliveryResult) bool {
	if result.Success {
		return false
	}

	statusCode := result.StatusCode

	// Connection errors (no status code) are retryable
	if statusCode == 0 && result.ErrorMessage != "" {
		return true
	}

	// Retryable status codes
	if statusCode == 408 || statusCode == 429 {
		return true
	}

	// All 5xx errors are retryable
	return statusCode >= 500 && statusCode < 600
}
