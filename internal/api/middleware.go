package api

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Middleware represents an HTTP middleware function
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order, with the first middleware being the outermost
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// LoggingMiddleware logs request method, path, status, and duration
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.status, duration)
	})
}

// statusResponseWriter wraps http.ResponseWriter to capture status code
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before writing
func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware adds CORS headers for development
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// JSONContentTypeMiddleware sets Content-Type to application/json for responses
func JSONContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSConfig holds configuration for the configurable CORS middleware
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to make requests.
	// Use "*" to allow all origins.
	AllowedOrigins []string

	// AllowCredentials indicates whether credentials (cookies, authorization headers)
	// are allowed in cross-origin requests.
	AllowCredentials bool

	// AllowLocalhost enables localhost origins in development mode.
	AllowLocalhost bool

	// LocalhostPattern is the localhost origin pattern to match (e.g., "http://localhost:5173").
	LocalhostPattern string
}

// NewConfigurableCORSMiddleware creates a CORS middleware with configurable origins.
func NewConfigurableCORSMiddleware(config CORSConfig) Middleware {
	allowedOrigins := make(map[string]bool)
	hasWildcard := false

	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			hasWildcard = true
		}
		allowedOrigins[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine if origin is allowed
			var allowedOrigin string
			if hasWildcard {
				allowedOrigin = "*"
			} else if origin != "" {
				if allowedOrigins[origin] {
					allowedOrigin = origin
				} else if config.AllowLocalhost && config.LocalhostPattern != "" && origin == config.LocalhostPattern {
					allowedOrigin = origin
				} else if config.AllowLocalhost && isLocalhostOrigin(origin) {
					allowedOrigin = origin
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if config.AllowCredentials && allowedOrigin != "" && allowedOrigin != "*" {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isLocalhostOrigin checks if the origin is a localhost URL
func isLocalhostOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "https://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1") ||
		strings.HasPrefix(origin, "https://127.0.0.1") ||
		strings.HasPrefix(origin, "http://[::1]") ||
		strings.HasPrefix(origin, "https://[::1]")
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// RequestsPerMinute is the number of requests allowed per minute per IP.
	RequestsPerMinute int

	// BurstSize is the maximum burst size for token bucket algorithm.
	BurstSize int

	// CleanupInterval is how often to clean up expired entries.
	CleanupInterval time.Duration
}

// WithDefaults returns a copy of the config with default values for unset fields.
func (c RateLimitConfig) WithDefaults() RateLimitConfig {
	result := c
	if result.RequestsPerMinute == 0 {
		result.RequestsPerMinute = 100
	}
	if result.BurstSize == 0 {
		result.BurstSize = result.RequestsPerMinute
	}
	if result.CleanupInterval == 0 {
		result.CleanupInterval = 5 * time.Minute
	}
	return result
}

// RateLimiter is an interface for rate limiting implementations
type RateLimiter interface {
	// Allow checks if a request from the given key (IP) is allowed.
	// Returns (allowed, retryAfter) where retryAfter is seconds until tokens are available.
	Allow(key string) (bool, int)
}

// tokenBucket represents a token bucket for rate limiting
type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// InMemoryRateLimiter implements rate limiting with an in-memory store
type InMemoryRateLimiter struct {
	config  RateLimitConfig
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(config RateLimitConfig) *InMemoryRateLimiter {
	config = config.WithDefaults()

	limiter := &InMemoryRateLimiter{
		config:  config,
		buckets: make(map[string]*tokenBucket),
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go limiter.cleanupLoop()

	return limiter
}

// Allow implements RateLimiter interface using token bucket algorithm
func (l *InMemoryRateLimiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	rate := float64(l.config.RequestsPerMinute) / 60.0 // tokens per second

	bucket, exists := l.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     float64(l.config.BurstSize),
			lastUpdate: now,
		}
		l.buckets[key] = bucket
	}

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens += elapsed * rate
	if bucket.tokens > float64(l.config.BurstSize) {
		bucket.tokens = float64(l.config.BurstSize)
	}
	bucket.lastUpdate = now

	// Check if we can consume a token
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}

	// Calculate retry after
	tokensNeeded := 1 - bucket.tokens
	retryAfter := int(tokensNeeded / rate)
	if retryAfter < 1 {
		retryAfter = 1
	}

	return false, retryAfter
}

// Stop stops the cleanup goroutine
func (l *InMemoryRateLimiter) Stop() {
	close(l.stopCh)
}

// cleanupLoop periodically removes old entries
func (l *InMemoryRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(l.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCh:
			return
		}
	}
}

// cleanup removes entries that haven't been used recently
func (l *InMemoryRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-2 * l.config.CleanupInterval)
	for key, bucket := range l.buckets {
		if bucket.lastUpdate.Before(cutoff) {
			delete(l.buckets, key)
		}
	}
}

// NewRateLimitMiddleware creates a rate limiting middleware
func NewRateLimitMiddleware(limiter RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			allowed, retryAfter := limiter.Allow(ip)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts the client IP from the request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port from RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx > 0 {
		// Check if this looks like a port (not IPv6)
		possiblePort := ip[colonIdx+1:]
		if _, err := strconv.Atoi(possiblePort); err == nil {
			ip = ip[:colonIdx]
		}
	}

	return ip
}

// EndpointRateLimitConfig holds per-endpoint rate limit configuration
type EndpointRateLimitConfig struct {
	// DefaultLimit is the default rate limit for endpoints not in EndpointLimits
	DefaultLimit RateLimitConfig

	// EndpointLimits maps path prefixes to their rate limit configs
	EndpointLimits map[string]RateLimitConfig
}

// EndpointRateLimiter holds rate limiters per endpoint
type EndpointRateLimiter struct {
	defaultLimiter   *InMemoryRateLimiter
	endpointLimiters map[string]*InMemoryRateLimiter
}

// NewEndpointRateLimiter creates a new endpoint-aware rate limiter
func NewEndpointRateLimiter(config EndpointRateLimitConfig) *EndpointRateLimiter {
	erl := &EndpointRateLimiter{
		defaultLimiter:   NewInMemoryRateLimiter(config.DefaultLimit),
		endpointLimiters: make(map[string]*InMemoryRateLimiter),
	}

	for path, limitConfig := range config.EndpointLimits {
		erl.endpointLimiters[path] = NewInMemoryRateLimiter(limitConfig)
	}

	return erl
}

// GetLimiter returns the appropriate limiter for the given path
func (erl *EndpointRateLimiter) GetLimiter(path string) *InMemoryRateLimiter {
	// Check for path prefix matches
	for prefix, limiter := range erl.endpointLimiters {
		if strings.HasPrefix(path, prefix) {
			return limiter
		}
	}
	return erl.defaultLimiter
}

// NewEndpointRateLimitMiddleware creates a rate limiting middleware with per-endpoint limits
func NewEndpointRateLimitMiddleware(config EndpointRateLimitConfig) Middleware {
	erl := NewEndpointRateLimiter(config)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)
			limiter := erl.GetLimiter(r.URL.Path)

			allowed, retryAfter := limiter.Allow(ip)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
