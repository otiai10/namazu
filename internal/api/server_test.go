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

func TestNewServerWithHandler(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Create a custom handler
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("custom"))
	})

	server := NewServerWithHandler(":9090", customHandler, subRepo, eventRepo)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	if server.Addr() != ":9090" {
		t.Errorf("expected addr :9090, got %s", server.Addr())
	}

	if server.httpServer == nil {
		t.Error("expected httpServer to be initialized")
	}

	if server.httpServer.Handler == nil {
		t.Error("expected handler to be set")
	}

	if server.subscriptionRepo == nil {
		t.Error("expected subscriptionRepo to be set")
	}

	if server.eventRepo == nil {
		t.Error("expected eventRepo to be set")
	}
}

func TestNewServerWithHandler_StartAndShutdown(t *testing.T) {
	subRepo := newMockSubscriptionRepo()
	eventRepo := newMockEventRepo()

	// Create a custom handler
	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServerWithHandler("127.0.0.1:0", customHandler, subRepo, eventRepo)

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
