package billing

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with secret key", func(t *testing.T) {
		client := NewClient("sk_test_123")

		if client == nil {
			t.Fatal("NewClient returned nil")
		}
		if client.secretKey != "sk_test_123" {
			t.Errorf("Expected secretKey 'sk_test_123', got %s", client.secretKey)
		}
	})

	t.Run("creates client with empty secret key", func(t *testing.T) {
		client := NewClient("")

		if client == nil {
			t.Fatal("NewClient returned nil")
		}
		if client.secretKey != "" {
			t.Errorf("Expected empty secretKey, got %s", client.secretKey)
		}
	})
}

func TestClientGetSecretKey(t *testing.T) {
	t.Run("returns secret key", func(t *testing.T) {
		client := NewClient("sk_test_456")

		if client.GetSecretKey() != "sk_test_456" {
			t.Errorf("Expected 'sk_test_456', got %s", client.GetSecretKey())
		}
	})
}
