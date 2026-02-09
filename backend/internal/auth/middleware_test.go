package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockTokenVerifier implements TokenVerifier for testing
type mockTokenVerifier struct {
	claims *Claims
	err    error
}

func (m *mockTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*Claims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

func TestAuthMiddleware_MissingAuthorizationHeader(t *testing.T) {
	verifier := &mockTokenVerifier{}
	middleware := AuthMiddleware(verifier)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when authorization header is missing")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["error"] != "Authorization header required" {
		t.Errorf("expected error 'Authorization header required', got '%s'", response["error"])
	}
}

func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	verifier := &mockTokenVerifier{}
	middleware := AuthMiddleware(verifier)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when authorization format is invalid")
	}))

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing bearer prefix",
			header: "some-token",
		},
		{
			name:   "lowercase bearer",
			header: "bearer some-token",
		},
		{
			name:   "only bearer prefix",
			header: "Bearer ",
		},
		{
			name:   "empty after bearer",
			header: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if response["error"] != "Invalid authorization header format" {
				t.Errorf("expected error 'Invalid authorization header format', got '%s'", response["error"])
			}
		})
	}
}

func TestAuthMiddleware_TokenVerificationFailed(t *testing.T) {
	verifier := &mockTokenVerifier{
		err: errors.New("token expired"),
	}
	middleware := AuthMiddleware(verifier)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token verification fails")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid-looking-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["error"] != "Invalid token" {
		t.Errorf("expected error 'Invalid token', got '%s'", response["error"])
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	expectedClaims := &Claims{
		UID:           "user-123",
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Picture:       "https://example.com/photo.jpg",
		ProviderID:    "google.com",
	}

	verifier := &mockTokenVerifier{
		claims: expectedClaims,
	}
	middleware := AuthMiddleware(verifier)

	var receivedClaims *Claims
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := GetClaims(r.Context())
		if !ok {
			t.Error("claims should be present in context")
			return
		}
		receivedClaims = claims
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if receivedClaims == nil {
		t.Fatal("expected claims to be set")
	}

	if receivedClaims.UID != expectedClaims.UID {
		t.Errorf("expected UID %s, got %s", expectedClaims.UID, receivedClaims.UID)
	}

	if receivedClaims.Email != expectedClaims.Email {
		t.Errorf("expected Email %s, got %s", expectedClaims.Email, receivedClaims.Email)
	}

	if receivedClaims.EmailVerified != expectedClaims.EmailVerified {
		t.Errorf("expected EmailVerified %v, got %v", expectedClaims.EmailVerified, receivedClaims.EmailVerified)
	}

	if receivedClaims.Name != expectedClaims.Name {
		t.Errorf("expected Name %s, got %s", expectedClaims.Name, receivedClaims.Name)
	}
}

func TestWithClaims(t *testing.T) {
	ctx := context.Background()
	claims := &Claims{
		UID:   "user-456",
		Email: "another@example.com",
	}

	ctxWithClaims := WithClaims(ctx, claims)

	// Original context should not have claims
	_, ok := GetClaims(ctx)
	if ok {
		t.Error("original context should not have claims")
	}

	// New context should have claims
	retrievedClaims, ok := GetClaims(ctxWithClaims)
	if !ok {
		t.Fatal("context with claims should have claims")
	}

	if retrievedClaims.UID != claims.UID {
		t.Errorf("expected UID %s, got %s", claims.UID, retrievedClaims.UID)
	}
}

func TestGetClaims_NoClaimsInContext(t *testing.T) {
	ctx := context.Background()

	claims, ok := GetClaims(ctx)

	if ok {
		t.Error("expected ok to be false when no claims in context")
	}

	if claims != nil {
		t.Error("expected claims to be nil when no claims in context")
	}
}

func TestGetClaims_WrongTypeInContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), claimsKey, "not-a-claims-struct")

	claims, ok := GetClaims(ctx)

	if ok {
		t.Error("expected ok to be false when wrong type in context")
	}

	if claims != nil {
		t.Error("expected claims to be nil when wrong type in context")
	}
}

func TestMustGetClaims_Success(t *testing.T) {
	expectedClaims := &Claims{
		UID:   "user-789",
		Email: "must@example.com",
	}

	ctx := WithClaims(context.Background(), expectedClaims)

	claims := MustGetClaims(ctx)

	if claims.UID != expectedClaims.UID {
		t.Errorf("expected UID %s, got %s", expectedClaims.UID, claims.UID)
	}
}

func TestMustGetClaims_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustGetClaims to panic when no claims in context")
		}
	}()

	ctx := context.Background()
	MustGetClaims(ctx)
}

func TestClaims_Fields(t *testing.T) {
	claims := &Claims{
		UID:           "uid-test",
		Email:         "email@test.com",
		EmailVerified: true,
		Name:          "Test Name",
		Picture:       "https://picture.url",
		ProviderID:    "firebase",
	}

	if claims.UID != "uid-test" {
		t.Errorf("expected UID 'uid-test', got '%s'", claims.UID)
	}

	if claims.Email != "email@test.com" {
		t.Errorf("expected Email 'email@test.com', got '%s'", claims.Email)
	}

	if !claims.EmailVerified {
		t.Error("expected EmailVerified to be true")
	}

	if claims.Name != "Test Name" {
		t.Errorf("expected Name 'Test Name', got '%s'", claims.Name)
	}

	if claims.Picture != "https://picture.url" {
		t.Errorf("expected Picture 'https://picture.url', got '%s'", claims.Picture)
	}

	if claims.ProviderID != "firebase" {
		t.Errorf("expected ProviderID 'firebase', got '%s'", claims.ProviderID)
	}
}

func TestAuthMiddleware_ContentTypeJSON(t *testing.T) {
	verifier := &mockTokenVerifier{
		err: errors.New("any error"),
	}
	middleware := AuthMiddleware(verifier)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not reach here
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}
