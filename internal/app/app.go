package app

import (
	"context"
	"encoding/json"
	"log"

	"github.com/ayanel/namazu/internal/config"
	"github.com/ayanel/namazu/internal/delivery/webhook"
	"github.com/ayanel/namazu/internal/source"
	"github.com/ayanel/namazu/internal/source/p2pquake"
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
	config  *config.Config
	client  Client
	sender  Sender
	targets []webhook.Target
}

// NewApp creates a new application instance with the provided configuration.
// It initializes the P2P地震情報 WebSocket client and webhook sender,
// and converts webhook configurations into webhook targets.
//
// Parameters:
//   - cfg: Application configuration containing source and webhook settings
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
//	app := NewApp(cfg)
//	if err := app.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
func NewApp(cfg *config.Config) *App {
	// Convert config.SubscriptionConfig to webhook.Target
	targets := make([]webhook.Target, len(cfg.Subscriptions))
	for i, sub := range cfg.Subscriptions {
		targets[i] = webhook.Target{
			URL:    sub.Delivery.URL,
			Secret: sub.Delivery.Secret,
			Name:   sub.Name,
		}
	}

	return &App{
		config:  cfg,
		client:  p2pquake.NewClient(cfg.Source.Endpoint),
		sender:  webhook.NewSender(),
		targets: targets,
	}
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
	log.Printf("Configured %d subscription(s)", len(a.targets))

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
// It extracts the raw JSON payload from the event and uses the webhook sender
// to deliver it to all configured targets in parallel.
//
// The method logs:
//   - Incoming event details (ID, severity, source)
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

	// Send to all subscriptions
	results := a.sender.SendAll(ctx, a.targets, payload)

	// Log results
	for i, result := range results {
		name := a.targets[i].Name
		if name == "" {
			name = a.targets[i].URL
		}
		if result.Success {
			log.Printf("Subscription [%s]: delivered in %v", name, result.ResponseTime)
		} else {
			log.Printf("Subscription [%s]: failed - %s", name, result.ErrorMessage)
		}
	}
}
