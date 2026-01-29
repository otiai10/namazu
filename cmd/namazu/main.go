package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayanel/namazu/internal/app"
	"github.com/ayanel/namazu/internal/config"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create application
	application := app.NewApp(cfg)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Run application
	log.Println("namazu - Earthquake Webhook Relay Server")
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	log.Println("Goodbye!")
}
