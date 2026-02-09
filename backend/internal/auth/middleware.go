package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AuthMiddleware returns middleware that validates Firebase tokens.
// Requires Authorization header: Bearer <token>
// Returns 401 if token is missing or invalid.
// On success, adds Claims to context.
func AuthMiddleware(verifier TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			// Check for "Bearer " prefix (case-sensitive)
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSONError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			claims, err := verifier.VerifyIDToken(r.Context(), token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			// Add claims to context and continue
			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeJSONError writes a JSON error response with the given status code and message
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": message})
}
