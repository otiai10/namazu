package app

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/otiai10/namazu/internal/config"
	"github.com/otiai10/namazu/internal/delivery/webhook"
	"github.com/otiai10/namazu/internal/source"
	"github.com/otiai10/namazu/internal/source/p2pquake"
	"github.com/otiai10/namazu/internal/store"
	"github.com/otiai10/namazu/internal/subscription"
)

// Client interface abstracts the p2pquake.Client for testing
type Client interface {
	Connect(ctx context.Context) error
	Events() <-chan source.Event
	Close() error
}

// Sender interface abstracts the webhook.Sender for testing
type Sender interface {
	SendAll(ctx context.Context, targets []webhook.Target, payload []byte) []webhook.DeliveryResult
}

// SingleSender interface abstracts sending to a single target
type SingleSender interface {
	Send(ctx context.Context, url, secret string, payload []byte) webhook.DeliveryResult
}

// App is the main application orchestrator.
// It coordinates the P2P地震情報 client and webhook sender,
// providing a unified interface for the earthquake notification system.
type App struct {
	config       *config.Config
	client       Client
	sender       Sender
	singleSender SingleSender
	repository   subscription.Repository
	eventRepo    store.EventRepository // optional, can be nil
}

// Option is a functional option for configuring the App.
type Option func(*App)

// WithEventRepository sets the event repository for storing events.
// If not provided, events will not be persisted.
func WithEventRepository(repo store.EventRepository) Option {
	return func(a *App) {
		a.eventRepo = repo
	}
}

// NewApp creates a new application instance with the provided configuration and repository.
// It initializes the P2P地震情報 WebSocket client and webhook sender.
//
// Parameters:
//   - cfg: Application configuration containing source settings
//   - repo: Subscription repository for dynamic subscription loading
//
// Returns:
//
//	Configured App instance ready to run
//
// Example:
//
//	cfg, err := config.Load("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	repo := subscription.NewStaticRepository(cfg)
//	app := NewApp(cfg, repo)
//	if err := app.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// With optional event repository:
//
//	eventRepo := store.NewFirestoreEventRepository(firestoreClient)
//	app := NewApp(cfg, repo, WithEventRepository(eventRepo))
func NewApp(cfg *config.Config, repo subscription.Repository, opts ...Option) *App {
	baseSender := webhook.NewSender()
	app := &App{
		config:       cfg,
		client:       p2pquake.NewClient(cfg.Source.Endpoint),
		sender:       baseSender,
		singleSender: baseSender,
		repository:   repo,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Run starts the application and blocks until the context is cancelled.
// It connects to the P2P地震情報 WebSocket API, processes incoming events,
// and fans them out to all configured webhook targets.
//
// The method supports graceful shutdown through context cancellation,
// ensuring all resources are properly cleaned up.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//
//	Error if connection fails, nil on graceful shutdown
//
// The method will:
//  1. Connect to the P2P地震情報 WebSocket endpoint
//  2. Start receiving earthquake events
//  3. Fan out each event to all configured webhooks
//  4. Log all events and delivery results
//  5. Close the connection on context cancellation
//
// Example:
//
//	app := NewApp(cfg)
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// Handle shutdown signal
//	go func() {
//	    <-sigterm
//	    cancel()
//	}()
//
//	if err := app.Run(ctx); err != nil {
//	    log.Fatal(err)
//	}
func (a *App) Run(ctx context.Context) error {
	log.Printf("Starting namazu - connecting to %s", a.config.Source.Endpoint)

	// Connect to P2P地震情報 API
	if err := a.client.Connect(ctx); err != nil {
		return err
	}
	defer a.client.Close()

	// Process events
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			return nil
		case event := <-a.client.Events():
			a.handleEvent(ctx, event)
		}
	}
}

// handleEvent processes a single earthquake event and sends it to all webhooks.
// It queries the repository for current subscriptions, extracts the raw JSON payload
// from the event, and uses the webhook sender to deliver it to all targets in parallel.
//
// The method logs:
//   - Incoming event details (ID, severity, source)
//   - Number of active subscriptions
//   - Delivery results for each webhook (success/failure, response time)
//
// Parameters:
//   - ctx: Context for cancellation control
//   - event: Earthquake event to process and forward
//
// If the event's RawJSON is empty, the method falls back to JSON encoding
// the event structure itself.
func (a *App) handleEvent(ctx context.Context, event source.Event) {
	log.Printf("Received earthquake: ID=%s, Severity=%d, Source=%s",
		event.GetID(), event.GetSeverity(), event.GetSource())

	// Save event to repository (if configured)
	if a.eventRepo != nil {
		record := store.EventFromSource(event)
		if _, err := a.eventRepo.Create(ctx, record); err != nil {
			log.Printf("Failed to save event: %v", err)
			// Continue processing even if save fails
		}
	}

	// Get current subscriptions (dynamic)
	subscriptions, err := a.repository.List(ctx)
	if err != nil {
		log.Printf("Failed to get subscriptions: %v", err)
		return
	}

	log.Printf("Delivering to %d subscription(s)", len(subscriptions))

	// Get raw JSON for webhook payload
	payload := []byte(event.GetRawJSON())
	if len(payload) == 0 {
		// Fallback to encoding if RawJSON is empty
		var err error
		payload, err = json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			return
		}
	}

	// Filter and collect webhook subscriptions
	webhookSubs := filterWebhookSubscriptions(subscriptions, event)

	// Deliver to all filtered subscriptions concurrently
	a.deliverToSubscriptions(ctx, webhookSubs, payload)
}

// deliveryTarget holds subscription info for delivery
type deliveryTarget struct {
	sub    subscription.Subscription
	target webhook.Target
}

// filterWebhookSubscriptions filters subscriptions to only include webhook
// subscriptions that match the event filter.
func filterWebhookSubscriptions(subs []subscription.Subscription, event source.Event) []deliveryTarget {
	targets := make([]deliveryTarget, 0, len(subs))
	for _, sub := range subs {
		if sub.Delivery.Type != "webhook" {
			continue
		}
		// Check filter - skip if event doesn't match
		if sub.Filter != nil && !sub.Filter.Matches(event) {
			log.Printf("Subscription [%s]: filtered out (MinScale=%d, Prefectures=%v)",
				sub.Name, sub.Filter.MinScale, sub.Filter.Prefectures)
			continue
		}
		targets = append(targets, deliveryTarget{
			sub: sub,
			target: webhook.Target{
				URL:    sub.Delivery.URL,
				Secret: sub.Delivery.Secret,
				Name:   sub.Name,
			},
		})
	}
	return targets
}

// deliverToSubscriptions sends the payload to all targets concurrently,
// using per-subscription retry configuration if available.
func (a *App) deliverToSubscriptions(ctx context.Context, targets []deliveryTarget, payload []byte) {
	// Check if any subscription has retry config
	hasRetryConfig := false
	for _, dt := range targets {
		if dt.sub.Delivery.Retry != nil && dt.sub.Delivery.Retry.Enabled {
			hasRetryConfig = true
			break
		}
	}

	// If no retry config, use standard SendAll for backward compatibility
	if !hasRetryConfig {
		webhookTargets := make([]webhook.Target, len(targets))
		for i, dt := range targets {
			webhookTargets[i] = dt.target
		}
		results := a.sender.SendAll(ctx, webhookTargets, payload)
		for i, result := range results {
			logDeliveryResult(targets[i].target.Name, result)
		}
		return
	}

	// Use per-subscription delivery with retry
	var wg sync.WaitGroup
	results := make([]webhook.DeliveryResult, len(targets))

	for i, dt := range targets {
		wg.Add(1)
		go func(index int, target deliveryTarget) {
			defer wg.Done()
			results[index] = a.deliverWithRetry(ctx, target, payload)
		}(i, dt)
	}

	wg.Wait()

	// Log results
	for i, result := range results {
		logDeliveryResult(targets[i].target.Name, result)
	}
}

// deliverWithRetry sends the payload to a single target with retry logic
// based on the subscription's retry configuration.
func (a *App) deliverWithRetry(ctx context.Context, dt deliveryTarget, payload []byte) webhook.DeliveryResult {
	// If no retry config or retry disabled, use direct send
	if dt.sub.Delivery.Retry == nil || !dt.sub.Delivery.Retry.Enabled {
		return a.singleSender.Send(ctx, dt.target.URL, dt.target.Secret, payload)
	}

	// Convert subscription RetryConfig to webhook RetryConfig
	retryConfig := webhook.RetryConfig{
		Enabled:    dt.sub.Delivery.Retry.Enabled,
		MaxRetries: dt.sub.Delivery.Retry.MaxRetries,
		InitialMs:  dt.sub.Delivery.Retry.InitialMs,
		MaxMs:      dt.sub.Delivery.Retry.MaxMs,
	}

	// Create retrying sender with per-subscription config
	// Note: We create a new RetryingSender per delivery to use the subscription's config.
	// This is lightweight as it just wraps the existing sender.
	baseSender, ok := a.singleSender.(*webhook.Sender)
	if !ok {
		// Fallback: if not a Sender (e.g., mock), use direct send
		return a.singleSender.Send(ctx, dt.target.URL, dt.target.Secret, payload)
	}

	retryingSender := webhook.NewRetryingSender(baseSender, retryConfig)
	return retryingSender.Send(ctx, dt.target, payload)
}

// logDeliveryResult logs the result of a delivery attempt.
func logDeliveryResult(name string, result webhook.DeliveryResult) {
	if result.Success {
		if result.RetryCount > 0 {
			log.Printf("Subscription [%s]: delivered in %v (after %d retries)",
				name, result.ResponseTime, result.RetryCount)
		} else {
			log.Printf("Subscription [%s]: delivered in %v", name, result.ResponseTime)
		}
	} else {
		if result.RetryCount > 0 {
			log.Printf("Subscription [%s]: failed after %d retries - %s",
				name, result.RetryCount, result.ErrorMessage)
		} else {
			log.Printf("Subscription [%s]: failed - %s", name, result.ErrorMessage)
		}
	}
}
