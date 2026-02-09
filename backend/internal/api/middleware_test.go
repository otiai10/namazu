package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestChain(t *testing.T) {
	order := make([]string, 0)
	mu := sync.Mutex{}

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			order = append(order, "m1-before")
			mu.Unlock()
			next.ServeHTTP(w, r)
			mu.Lock()
			order = append(order, "m1-after")
			mu.Unlock()
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			order = append(order, "m2-before")
			mu.Unlock()
			next.ServeHTTP(w, r)
			mu.Lock()
			order = append(order, "m2-after")
			mu.Unlock()
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(middleware1, middleware2)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	chained.ServeHTTP(rec, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, expected %q", i, order[i], v)
		}
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wrapped := LoggingMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware(handler)

	t.Run("sets CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("expected Access-Control-Allow-Origin header")
		}
		if rec.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("handles preflight OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status %d for OPTIONS, got %d", http.StatusNoContent, rec.Code)
		}
	})
}

func TestConfigurableCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("allows configured origins", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
		}

		wrapped := NewConfigurableCORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "https://example.com" {
			t.Errorf("expected origin 'https://example.com', got %q", origin)
		}
	})

	t.Run("rejects non-configured origins", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins: []string{"https://example.com"},
		}

		wrapped := NewConfigurableCORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://evil.com")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin == "https://evil.com" {
			t.Error("should not allow non-configured origin")
		}
	})

	t.Run("handles wildcard origin", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins: []string{"*"},
		}

		wrapped := NewConfigurableCORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://any-origin.com")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "*" {
			t.Errorf("expected wildcard origin '*', got %q", origin)
		}
	})

	t.Run("allows localhost in development", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowLocalhost:   true,
			LocalhostPattern: "http://localhost:5173",
		}

		wrapped := NewConfigurableCORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:5173" {
			t.Errorf("expected localhost origin, got %q", origin)
		}
	})

	t.Run("handles preflight with credentials", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowCredentials: true,
		}

		wrapped := NewConfigurableCORSMiddleware(config)(handler)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("expected Access-Control-Allow-Credentials header")
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestJSONContentTypeMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := JSONContentTypeMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", contentType)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	t.Run("allows requests under limit", func(t *testing.T) {
		limiter := NewInMemoryRateLimiter(RateLimitConfig{
			RequestsPerMinute: 10,
			BurstSize:         10,
		})

		wrapped := NewRateLimitMiddleware(limiter)(handler)

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, rec.Code)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		limiter := NewInMemoryRateLimiter(RateLimitConfig{
			RequestsPerMinute: 2,
			BurstSize:         2,
		})

		wrapped := NewRateLimitMiddleware(limiter)(handler)

		// First 2 requests should succeed
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, rec.Code)
			}
		}

		// 3rd request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
		}
	})

	t.Run("rate limits are per IP", func(t *testing.T) {
		limiter := NewInMemoryRateLimiter(RateLimitConfig{
			RequestsPerMinute: 2,
			BurstSize:         2,
		})

		wrapped := NewRateLimitMiddleware(limiter)(handler)

		// Exhaust limit for IP1
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}

		// IP2 should still have quota
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("different IP should have separate limit, got status %d", rec.Code)
		}
	})

	t.Run("uses X-Forwarded-For header when present", func(t *testing.T) {
		limiter := NewInMemoryRateLimiter(RateLimitConfig{
			RequestsPerMinute: 2,
			BurstSize:         2,
		})

		wrapped := NewRateLimitMiddleware(limiter)(handler)

		// Exhaust limit using X-Forwarded-For
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Forwarded-For", "10.0.0.1")
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}

		// Same X-Forwarded-For should be blocked
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d for same X-Forwarded-For, got %d", http.StatusTooManyRequests, rec.Code)
		}
	})

	t.Run("returns Retry-After header", func(t *testing.T) {
		limiter := NewInMemoryRateLimiter(RateLimitConfig{
			RequestsPerMinute: 1,
			BurstSize:         1,
		})

		wrapped := NewRateLimitMiddleware(limiter)(handler)

		// First request exhausts limit
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		// Second request should get Retry-After
		req = httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec = httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("expected Retry-After header")
		}
	})
}

func TestInMemoryRateLimiter_Cleanup(t *testing.T) {
	limiter := NewInMemoryRateLimiter(RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         100,
		CleanupInterval:   50 * time.Millisecond,
	})

	// Make a request to create an entry
	allowed, _ := limiter.Allow("192.168.1.1")
	if !allowed {
		t.Error("first request should be allowed")
	}

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Entry should still exist (not expired yet based on last access)
	// This tests that cleanup runs but doesn't break the limiter
	allowed, _ = limiter.Allow("192.168.1.1")
	if !allowed {
		t.Error("request after cleanup should still work")
	}

	limiter.Stop()
}

func TestRateLimitConfig_Defaults(t *testing.T) {
	config := RateLimitConfig{}
	config = config.WithDefaults()

	if config.RequestsPerMinute == 0 {
		t.Error("expected non-zero RequestsPerMinute default")
	}
	if config.BurstSize == 0 {
		t.Error("expected non-zero BurstSize default")
	}
	if config.CleanupInterval == 0 {
		t.Error("expected non-zero CleanupInterval default")
	}
}

func TestEndpointRateLimitMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("applies different limits per path prefix", func(t *testing.T) {
		config := EndpointRateLimitConfig{
			DefaultLimit: RateLimitConfig{
				RequestsPerMinute: 100,
				BurstSize:         100,
			},
			EndpointLimits: map[string]RateLimitConfig{
				"/api/subscriptions": {
					RequestsPerMinute: 2,
					BurstSize:         2,
				},
			},
		}

		wrapped := NewEndpointRateLimitMiddleware(config)(handler)

		// Subscription creation should be limited to 2
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected status %d, got %d", i, http.StatusOK, rec.Code)
			}
		}

		// 3rd subscription request should be blocked
		req := httptest.NewRequest(http.MethodPost, "/api/subscriptions", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected status %d for subscription endpoint, got %d", http.StatusTooManyRequests, rec.Code)
		}

		// Other endpoints should still work with default limit
		req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec = httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d for events endpoint, got %d", http.StatusOK, rec.Code)
		}
	})
}
