package api

import (
	"net/http"
	"strings"
)

// NewRouter creates a new router with all API routes configured
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// Subscription routes
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
		// Extract ID from path
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

	// Event routes
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

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Apply middleware chain
	middleware := Chain(
		RecoveryMiddleware,
		LoggingMiddleware,
		CORSMiddleware,
		JSONContentTypeMiddleware,
	)

	return middleware(mux)
}
