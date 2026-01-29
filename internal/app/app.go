package app

import (
	"context"
	"encoding/json"
	"log"

	"github.com/ayanel/namazu/internal/config"
	"github.com/ayanel/namazu/internal/delivery/webhook"
	"github.com/ayanel/namazu/internal/source"
	"github.com/ayanel/namazu/internal/source/p2pquake"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
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

// App is the main application orchestrator.
// It coordinates the P2P地震情報 client and webhook sender,
// providing a unified interface for the earthquake notification system.
type App struct {
	config     *config.Config
	client     Client
	sender     Sender
	repository subscription.Repository
	eventRepo  store.EventRepository // optional, can be nil
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
	app := &App{
		config:     cfg,
		client:     p2pquake.NewClient(cfg.Source.Endpoint),
		sender:     webhook.NewSender(),
		repository: repo,
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

	// Convert subscriptions to webhook targets (for now, only webhook supported)
	targets := make([]webhook.Target, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if sub.Delivery.Type == "webhook" {
			targets = append(targets, webhook.Target{
				URL:    sub.Delivery.URL,
				Secret: sub.Delivery.Secret,
				Name:   sub.Name,
			})
		}
	}

	// Send to all targets
	results := a.sender.SendAll(ctx, targets, payload)

	// Log results
	for i, result := range results {
		if result.Success {
			log.Printf("Subscription [%s]: delivered in %v", targets[i].Name, result.ResponseTime)
		} else {
			log.Printf("Subscription [%s]: failed - %s", targets[i].Name, result.ErrorMessage)
		}
	}
}
