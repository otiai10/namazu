package api

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	server := NewServer(":8080", subRepo, eventRepo)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	if server.Addr() != ":8080" {
		t.Errorf("expected addr :8080, got %s", server.Addr())
	}

	if server.httpServer == nil {
		t.Error("expected httpServer to be initialized")
	}

	if server.subscriptionRepo == nil {
		t.Error("expected subscriptionRepo to be set")
	}

	if server.eventRepo == nil {
		t.Error("expected eventRepo to be set")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Use a random available port
	server := NewServer("127.0.0.1:0", subRepo, eventRepo)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
		close(errChan)
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("shutdown failed: %v", err)
	}

	// Check for start errors
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("server start error: %v", err)
		}
	case <-time.After(1 * time.Second):
		// Server started and shutdown successfully
	}
}

func TestServerShutdownNilServer(t *testing.T) {
	server := &Server{}

	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("expected nil error for nil httpServer, got: %v", err)
	}
}
