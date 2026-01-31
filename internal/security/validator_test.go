package security

import (
	"testing"
)

func TestWebhookURLValidator_ValidateWebhookURL(t *testing.T) {
	t.Run("production mode (allowLocalhost=false)", func(t *testing.T) {
		validator := NewWebhookURLValidator(false)

		tests := []struct {
			name        string
			url         string
			expectError bool
		}{
			{"valid HTTPS URL", "https://example.com/webhook", false},
			{"HTTP URL blocked", "http://example.com/webhook", true},
			{"localhost blocked", "https://localhost/webhook", true},
			{"127.0.0.1 blocked", "https://127.0.0.1/webhook", true},
			{"private IP 10.x blocked", "https://10.0.0.1/webhook", true},
			{"private IP 192.168.x blocked", "https://192.168.1.1/webhook", true},
			{"link-local blocked", "https://169.254.169.254/webhook", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validator.ValidateWebhookURL(tt.url)
				if tt.expectError && err == nil {
					t.Errorf("ValidateWebhookURL(%q) expected error, got nil", tt.url)
				}
				if !tt.expectError && err != nil {
					t.Errorf("ValidateWebhookURL(%q) unexpected error: %v", tt.url, err)
				}
			})
		}
	})

	t.Run("development mode (allowLocalhost=true)", func(t *testing.T) {
		validator := NewWebhookURLValidator(true)

		tests := []struct {
			name        string
			url         string
			expectError bool
		}{
			{"valid HTTPS URL", "https://example.com/webhook", false},
			{"HTTP localhost allowed", "http://localhost:3000/webhook", false},
			{"HTTPS localhost allowed", "https://localhost/webhook", false},
			{"HTTP 127.0.0.1 allowed", "http://127.0.0.1:8080/webhook", false},
			{"HTTP public URL still blocked", "http://example.com/webhook", true},
			{"private IP 10.x still blocked", "https://10.0.0.1/webhook", true},
			{"private IP 192.168.x still blocked", "https://192.168.1.1/webhook", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := validator.ValidateWebhookURL(tt.url)
				if tt.expectError && err == nil {
					t.Errorf("ValidateWebhookURL(%q) expected error, got nil", tt.url)
				}
				if !tt.expectError && err != nil {
					t.Errorf("ValidateWebhookURL(%q) unexpected error: %v", tt.url, err)
				}
			})
		}
	})
}
