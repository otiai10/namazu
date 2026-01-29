package webhook_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ayanel/namazu/internal/delivery/webhook"
)

// Example_singleWebhook demonstrates sending to a single webhook
func Example_singleWebhook() {
	// Create a sender with default 10 second timeout
	sender := webhook.NewSender()

	// Prepare webhook details
	webhookURL := "https://api.example.com/webhooks"
	secret := "your-secret-key"
	payload := []byte(`{"event":"user.created","user_id":123}`)

	// Send the webhook
	ctx := context.Background()
	result := sender.Send(ctx, webhookURL, secret, payload)

	// Check result
	if result.Success {
		fmt.Printf("Webhook delivered successfully in %v\n", result.ResponseTime)
	} else {
		fmt.Printf("Webhook failed: %s\n", result.ErrorMessage)
	}
}

// Example_multipleWebhooks demonstrates concurrent sending to multiple webhooks
func Example_multipleWebhooks() {
	// Create a sender with custom timeout
	sender := webhook.NewSender(webhook.WithTimeout(5 * time.Second))

	// Define multiple webhook targets
	targets := []webhook.Target{
		{
			URL:    "https://api.example.com/webhooks/1",
			Secret: "secret-1",
			Name:   "Production Webhook",
		},
		{
			URL:    "https://staging.example.com/webhooks",
			Secret: "secret-2",
			Name:   "Staging Webhook",
		},
		{
			URL:    "https://logs.example.com/webhooks",
			Secret: "secret-3",
			Name:   "Logging Service",
		},
	}

	payload := []byte(`{"event":"order.completed","order_id":456}`)

	// Send to all targets concurrently
	ctx := context.Background()
	results := sender.SendAll(ctx, targets, payload)

	// Process results
	successCount := 0
	for i, result := range results {
		if result.Success {
			successCount++
			fmt.Printf("✓ %s delivered in %v\n", targets[i].Name, result.ResponseTime)
		} else {
			fmt.Printf("✗ %s failed: %s\n", targets[i].Name, result.ErrorMessage)
		}
	}

	fmt.Printf("\nTotal: %d/%d webhooks delivered successfully\n", successCount, len(targets))
}

// Example_withSignatureVerification demonstrates signature verification
func Example_withSignatureVerification() {
	secret := "shared-secret"
	payload := []byte(`{"event":"test"}`)

	// Sender side: Sign the payload
	signature := webhook.Sign(secret, payload)
	fmt.Printf("Signature: %s\n", signature)

	// Receiver side: Verify the signature
	isValid := webhook.Verify(secret, payload, signature)
	if isValid {
		fmt.Println("Signature is valid - payload authentic")
	} else {
		fmt.Println("Signature is invalid - payload may be tampered")
	}

	// Output:
	// Signature: sha256=4228b7b06efd8560de08bbf19437e95aafa158c17aec35bce0850dc76956f83e
	// Signature is valid - payload authentic
}

// Example_errorHandling demonstrates proper error handling
func Example_errorHandling() {
	sender := webhook.NewSender(webhook.WithTimeout(2 * time.Second))

	webhookURL := "https://invalid-domain-that-does-not-exist.com/webhook"
	secret := "secret"
	payload := []byte(`{"event":"test"}`)

	ctx := context.Background()
	result := sender.Send(ctx, webhookURL, secret, payload)

	// Handle different failure scenarios
	if result.Success {
		log.Printf("Delivered successfully (status: %d)", result.StatusCode)
	} else {
		// Log the failure
		log.Printf("Delivery failed to %s: %s", result.URL, result.ErrorMessage)

		// Determine if retry is appropriate
		if result.StatusCode >= 500 && result.StatusCode < 600 {
			log.Println("Server error - consider retry")
		} else if result.StatusCode >= 400 && result.StatusCode < 500 {
			log.Println("Client error - do not retry")
		} else if result.StatusCode == 0 {
			log.Println("Connection error - network issue or timeout")
		}
	}
}

// Example_contextCancellation demonstrates using context for cancellation
func Example_contextCancellation() {
	sender := webhook.NewSender()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	webhookURL := "https://api.example.com/slow-endpoint"
	secret := "secret"
	payload := []byte(`{"event":"test"}`)

	result := sender.Send(ctx, webhookURL, secret, payload)

	if result.Success {
		fmt.Println("Delivered successfully")
	} else {
		fmt.Printf("Failed: %s\n", result.ErrorMessage)
	}
}

// Example_customTimeout demonstrates different timeout configurations
func Example_customTimeout() {
	// Fast timeout for critical path
	fastSender := webhook.NewSender(webhook.WithTimeout(1 * time.Second))

	// Normal timeout for standard webhooks
	normalSender := webhook.NewSender(webhook.WithTimeout(10 * time.Second))

	// Long timeout for batch processing
	slowSender := webhook.NewSender(webhook.WithTimeout(30 * time.Second))

	_ = fastSender
	_ = normalSender
	_ = slowSender

	fmt.Println("Senders created with different timeouts")
	// Output: Senders created with different timeouts
}
