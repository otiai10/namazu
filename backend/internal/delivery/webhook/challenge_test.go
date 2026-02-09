package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVerifyURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChallengeRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("failed to decode challenge request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Type != "url_verification" {
			t.Errorf("expected type 'url_verification', got %q", req.Type)
		}

		resp := ChallengeResponse{Challenge: req.Challenge}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	challenger := NewChallenger(5 * time.Second)
	result := challenger.VerifyURL(context.Background(), server.URL, "test-secret")

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}
	if result.ResponseTime <= 0 {
		t.Error("expected positive response time")
	}
}

func TestVerifyURL_WrongChallenge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChallengeResponse{Challenge: "wrong-token"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	challenger := NewChallenger(5 * time.Second)
	result := challenger.VerifyURL(context.Background(), server.URL, "test-secret")

	if result.Success {
		t.Error("expected failure for wrong challenge")
	}
	if result.ErrorMessage != "challenge response does not match" {
		t.Errorf("unexpected error message: %s", result.ErrorMessage)
	}
}

func TestVerifyURL_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	challenger := NewChallenger(5 * time.Second)
	result := challenger.VerifyURL(context.Background(), server.URL, "test-secret")

	if result.Success {
		t.Error("expected failure for non-200 status")
	}
	if result.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}
}

func TestVerifyURL_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	challenger := NewChallenger(50 * time.Millisecond)
	result := challenger.VerifyURL(context.Background(), server.URL, "test-secret")

	if result.Success {
		t.Error("expected failure for timeout")
	}
	if result.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}
}

func TestVerifyURL_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	}))
	defer server.Close()

	challenger := NewChallenger(5 * time.Second)
	result := challenger.VerifyURL(context.Background(), server.URL, "test-secret")

	if result.Success {
		t.Error("expected failure for invalid JSON")
	}
	if result.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}
}

func TestVerifyURL_ConnectionRefused(t *testing.T) {
	challenger := NewChallenger(2 * time.Second)
	result := challenger.VerifyURL(context.Background(), "http://127.0.0.1:1", "test-secret")

	if result.Success {
		t.Error("expected failure for connection refused")
	}
	if result.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}
}

func TestVerifyURL_IncludesSignature(t *testing.T) {
	secret := "my-test-secret"
	var receivedSignature string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-Signature-256")

		var req ChallengeRequest
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		resp := ChallengeResponse{Challenge: req.Challenge}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	challenger := NewChallenger(5 * time.Second)
	result := challenger.VerifyURL(context.Background(), server.URL, secret)

	if !result.Success {
		t.Fatalf("expected success, got failure: %s", result.ErrorMessage)
	}
	if receivedSignature == "" {
		t.Error("expected X-Signature-256 header to be present")
	}
	if len(receivedSignature) < 10 {
		t.Errorf("expected valid signature, got %q", receivedSignature)
	}
}

func TestGenerateChallengeToken_Format(t *testing.T) {
	token, err := generateChallengeToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(token) != 32 {
		t.Errorf("expected 32 hex chars (16 bytes), got %d chars", len(token))
	}

	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("expected hex character, got %c", c)
		}
	}
}

func TestGenerateChallengeToken_Uniqueness(t *testing.T) {
	token1, err := generateChallengeToken()
	if err != nil {
		t.Fatalf("unexpected error generating first token: %v", err)
	}

	token2, err := generateChallengeToken()
	if err != nil {
		t.Fatalf("unexpected error generating second token: %v", err)
	}

	if token1 == token2 {
		t.Error("expected two generated tokens to be different")
	}
}
