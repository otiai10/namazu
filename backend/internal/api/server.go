package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/otiai10/namazu/backend/internal/store"
	"github.com/otiai10/namazu/backend/internal/subscription"
)

// Server represents the REST API server
type Server struct {
	addr             string
	subscriptionRepo subscription.Repository
	eventRepo        store.EventRepository
	httpServer       *http.Server
}

// NewServer creates a new API server instance
func NewServer(addr string, subRepo subscription.Repository, eventRepo store.EventRepository) *Server {
	s := &Server{
		addr:             addr,
		subscriptionRepo: subRepo,
		eventRepo:        eventRepo,
	}

	handler := NewHandler(subRepo, eventRepo)
	router := NewRouter(handler)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

// NewServerWithHandler creates a new Server with a custom handler
func NewServerWithHandler(addr string, handler http.Handler, subRepo subscription.Repository, eventRepo store.EventRepository) *Server {
	return &Server{
		addr:             addr,
		subscriptionRepo: subRepo,
		eventRepo:        eventRepo,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Start begins listening for HTTP requests
func (s *Server) Start() error {
	if s.httpServer == nil {
		return fmt.Errorf("server not initialized")
	}

	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.addr
}
