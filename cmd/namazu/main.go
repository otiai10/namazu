package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/ayanel/namazu/internal/api"
	"github.com/ayanel/namazu/internal/app"
	"github.com/ayanel/namazu/internal/config"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
)

func main() {
	// Load .env file if it exists (for local development)
	// Silently ignore if .env doesn't exist (production uses real env vars)
	_ = godotenv.Load()

	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize repositories based on configuration
	var subRepo subscription.Repository
	var eventRepo store.EventRepository
	var firestoreClient *store.FirestoreClient

	if cfg.Store != nil {
		// Phase 2 mode: Use Firestore for dynamic subscriptions and event storage
		log.Printf("Initializing Firestore client for project: %s, database: %s",
			cfg.Store.ProjectID, cfg.Store.Database)
		firestoreClient, err = store.NewFirestoreClient(ctx, store.FirestoreConfig{
			ProjectID:   cfg.Store.ProjectID,
			Database:    cfg.Store.Database,
			Credentials: cfg.Store.Credentials,
		})
		if err != nil {
			log.Fatalf("Failed to create Firestore client: %v", err)
		}
		defer firestoreClient.Close()

		subRepo = subscription.NewFirestoreRepository(firestoreClient.Client())
		eventRepo = store.NewFirestoreEventRepository(firestoreClient.Client())
		log.Println("Using Firestore for subscriptions and event storage")
	} else {
		// Phase 1 mode: Use static subscriptions from config file
		subRepo = subscription.NewStaticRepository(cfg)
		log.Println("Using static subscriptions from config file")
	}

	// Create application with options
	opts := []app.Option{}
	if eventRepo != nil {
		opts = append(opts, app.WithEventRepository(eventRepo))
	}
	application := app.NewApp(cfg, subRepo, opts...)

	// Start API server if configured
	var apiServer *api.Server
	if cfg.API != nil {
		log.Printf("Starting REST API server on %s", cfg.API.Addr)
		apiServer = api.NewServer(cfg.API.Addr, subRepo, eventRepo)
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("API server error: %v", err)
			}
		}()
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Run application (WebSocket client)
	log.Println("namazu - Earthquake Webhook Relay Server")
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down...")

	// Shutdown API server if running
	if apiServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("API server shutdown error: %v", err)
		}
		log.Println("API server stopped")
	}

	log.Println("Goodbye!")
}
