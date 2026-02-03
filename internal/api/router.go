package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/otiai10/namazu/internal/auth"
	"github.com/otiai10/namazu/internal/billing"
	"github.com/otiai10/namazu/internal/config"
	"github.com/otiai10/namazu/internal/quota"
	"github.com/otiai10/namazu/internal/store"
	"github.com/otiai10/namazu/internal/subscription"
	"github.com/otiai10/namazu/internal/user"
	"github.com/otiai10/namazu/internal/version"
)

// RouterConfig holds dependencies for the router
type RouterConfig struct {
	SubscriptionRepo subscription.Repository
	EventRepo        store.EventRepository
	UserRepo         user.Repository
	TokenVerifier    auth.TokenVerifier // nil means no auth
	QuotaChecker     quota.QuotaChecker // nil means no quota checking
	BillingClient    *billing.Client    // nil means no billing
	BillingConfig    *config.BillingConfig
	SecurityConfig   *config.SecurityConfig // nil uses defaults
	URLValidator     URLValidator           // nil means no URL validation
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

	// Set URL validator if provided
	if cfg.URLValidator != nil {
		h.SetURLValidator(cfg.URLValidator)
	}

	// Public routes (no auth required)
	registerPublicRoutes(mux, h)

	// Stripe webhook route (no auth required - uses signature verification)
	if cfg.BillingClient != nil && cfg.BillingConfig != nil {
		billingHandler := NewBillingHandler(cfg.BillingClient, cfg.UserRepo, cfg.BillingConfig)
		registerStripeWebhookRoute(mux, billingHandler)
	}

	// Protected routes (auth required when TokenVerifier is provided)
	if cfg.TokenVerifier != nil {
		protectedMux := http.NewServeMux()
		meHandler := NewMeHandler(cfg.UserRepo)
		registerMeRoutes(protectedMux, meHandler)
		registerSubscriptionRoutes(protectedMux, h)

		// Register billing routes if billing is configured
		if cfg.BillingClient != nil && cfg.BillingConfig != nil {
			billingHandler := NewBillingHandler(cfg.BillingClient, cfg.UserRepo, cfg.BillingConfig)
			registerBillingRoutes(protectedMux, billingHandler)
		}

		// Apply auth middleware to protected routes
		authHandler := auth.AuthMiddleware(cfg.TokenVerifier)(protectedMux)
		mux.Handle("/api/me", authHandler)
		mux.Handle("/api/me/", authHandler)
		mux.Handle("/api/subscriptions", authHandler)
		mux.Handle("/api/subscriptions/", authHandler)
		mux.Handle("/api/billing/", authHandler)
	} else {
		// No auth mode (backward compatibility)
		registerSubscriptionRoutes(mux, h)
	}

	return applyMiddlewareChainWithConfig(mux, cfg.SecurityConfig)
}

// registerPublicRoutes registers routes that don't require authentication
func registerPublicRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ok","hash":"%s"}`, version.CommitHash)))
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

// registerBillingRoutes registers billing API routes (requires auth)
func registerBillingRoutes(mux *http.ServeMux, h *BillingHandler) {
	mux.HandleFunc("/api/billing/status", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetStatus(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/billing/create-checkout-session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateCheckoutSession(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/billing/portal-session", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetPortalSession(w, r)
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// registerStripeWebhookRoute registers the Stripe webhook route (no auth required)
func registerStripeWebhookRoute(mux *http.ServeMux, h *BillingHandler) {
	mux.HandleFunc("/api/webhooks/stripe", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.StripeWebhook(w, r)
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

// applyMiddlewareChainWithConfig wraps a handler with the middleware stack using security config
func applyMiddlewareChainWithConfig(h http.Handler, securityCfg *config.SecurityConfig) http.Handler {
	middlewares := []Middleware{
		RecoveryMiddleware,
		LoggingMiddleware,
	}

	// Add configurable CORS middleware
	if securityCfg != nil && len(securityCfg.GetCORSAllowedOrigins()) > 0 {
		corsConfig := CORSConfig{
			AllowedOrigins:   securityCfg.GetCORSAllowedOrigins(),
			AllowCredentials: true,
			AllowLocalhost:   securityCfg.AllowLocalWebhooks,
		}
		middlewares = append(middlewares, NewConfigurableCORSMiddleware(corsConfig))
	} else {
		// Default CORS for backward compatibility
		middlewares = append(middlewares, CORSMiddleware)
	}

	// Add rate limiting if enabled
	if securityCfg != nil && securityCfg.RateLimitEnabled {
		defaultRPM := 100
		if securityCfg.RateLimitRequestsPerMinute > 0 {
			defaultRPM = securityCfg.RateLimitRequestsPerMinute
		}

		subscriptionRPM := 10
		if securityCfg.RateLimitSubscriptionCreation > 0 {
			subscriptionRPM = securityCfg.RateLimitSubscriptionCreation
		}

		rateLimitConfig := EndpointRateLimitConfig{
			DefaultLimit: RateLimitConfig{
				RequestsPerMinute: defaultRPM,
				BurstSize:         defaultRPM,
			},
			EndpointLimits: map[string]RateLimitConfig{
				"/api/subscriptions": {
					RequestsPerMinute: subscriptionRPM,
					BurstSize:         subscriptionRPM,
				},
			},
		}
		middlewares = append(middlewares, NewEndpointRateLimitMiddleware(rateLimitConfig))
	}

	middlewares = append(middlewares, JSONContentTypeMiddleware)

	return Chain(middlewares...)(h)
}
