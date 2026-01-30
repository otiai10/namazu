package api

import (
	"net/http"
	"strings"

	"github.com/ayanel/namazu/internal/auth"
	"github.com/ayanel/namazu/internal/quota"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
	"github.com/ayanel/namazu/internal/user"
)

// RouterConfig holds dependencies for the router
type RouterConfig struct {
	SubscriptionRepo subscription.Repository
	EventRepo        store.EventRepository
	UserRepo         user.Repository
	TokenVerifier    auth.TokenVerifier // nil means no auth
	QuotaChecker     quota.QuotaChecker // nil means no quota checking
}

// NewRouter creates a new router with all API routes configured
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()
	registerPublicRoutes(mux, h)
	registerSubscriptionRoutes(mux, h)
	return applyMiddlewareChain(mux)
}

// NewRouterWithConfig creates a new HTTP router with authentication support
func NewRouterWithConfig(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// Create handler with or without quota checking
	var h *Handler
	if cfg.QuotaChecker != nil {
		h = NewHandlerWithQuota(cfg.SubscriptionRepo, cfg.EventRepo, cfg.UserRepo, cfg.QuotaChecker)
	} else {
		h = NewHandler(cfg.SubscriptionRepo, cfg.EventRepo)
	}

	// Public routes (no auth required)
	registerPublicRoutes(mux, h)

	// Protected routes (auth required when TokenVerifier is provided)
	if cfg.TokenVerifier != nil {
		protectedMux := http.NewServeMux()
		meHandler := NewMeHandler(cfg.UserRepo)
		registerMeRoutes(protectedMux, meHandler)
		registerSubscriptionRoutes(protectedMux, h)

		// Apply auth middleware to protected routes
		authHandler := auth.AuthMiddleware(cfg.TokenVerifier)(protectedMux)
		mux.Handle("/api/me", authHandler)
		mux.Handle("/api/me/", authHandler)
		mux.Handle("/api/subscriptions", authHandler)
		mux.Handle("/api/subscriptions/", authHandler)
	} else {
		// No auth mode (backward compatibility)
		registerSubscriptionRoutes(mux, h)
	}

	return applyMiddlewareChain(mux)
}

// registerPublicRoutes registers routes that don't require authentication
func registerPublicRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.ListEvents(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// registerMeRoutes registers user profile routes
func registerMeRoutes(mux *http.ServeMux, h *MeHandler) {
	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetProfile(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/me/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetProviders(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// registerSubscriptionRoutes registers subscription resource routes
func registerSubscriptionRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateSubscription(w, r)
		case http.MethodGet:
			h.ListSubscriptions(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
		if path == "" || strings.Contains(path, "/") {
			writeError(w, "invalid path", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetSubscription(w, r)
		case http.MethodPut:
			h.UpdateSubscription(w, r)
		case http.MethodDelete:
			h.DeleteSubscription(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// applyMiddlewareChain wraps a handler with the standard middleware stack
func applyMiddlewareChain(h http.Handler) http.Handler {
	return Chain(
		RecoveryMiddleware,
		LoggingMiddleware,
		CORSMiddleware,
		JSONContentTypeMiddleware,
	)(h)
}
