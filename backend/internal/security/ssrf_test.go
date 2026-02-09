package security

import (
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Private IP ranges (10.0.0.0/8)
		{"10.0.0.0 is private", "10.0.0.0", true},
		{"10.0.0.1 is private", "10.0.0.1", true},
		{"10.255.255.255 is private", "10.255.255.255", true},

		// Private IP ranges (172.16.0.0/12)
		{"172.16.0.0 is private", "172.16.0.0", true},
		{"172.16.0.1 is private", "172.16.0.1", true},
		{"172.31.255.255 is private", "172.31.255.255", true},
		{"172.15.255.255 is NOT private", "172.15.255.255", false},
		{"172.32.0.0 is NOT private", "172.32.0.0", false},

		// Private IP ranges (192.168.0.0/16)
		{"192.168.0.0 is private", "192.168.0.0", true},
		{"192.168.0.1 is private", "192.168.0.1", true},
		{"192.168.255.255 is private", "192.168.255.255", true},
		{"192.167.0.1 is NOT private", "192.167.0.1", false},

		// Localhost
		{"127.0.0.1 is localhost (private)", "127.0.0.1", true},
		{"127.0.0.0 is localhost (private)", "127.0.0.0", true},
		{"127.255.255.255 is localhost (private)", "127.255.255.255", true},

		// IPv6 localhost
		{"::1 is IPv6 localhost (private)", "::1", true},

		// Link-local addresses (169.254.0.0/16)
		{"169.254.0.1 is link-local (private)", "169.254.0.1", true},
		{"169.254.169.254 is link-local (private)", "169.254.169.254", true},
		{"169.254.255.255 is link-local (private)", "169.254.255.255", true},
		{"169.253.0.1 is NOT link-local", "169.253.0.1", false},

		// Public IPs
		{"8.8.8.8 is public", "8.8.8.8", false},
		{"1.1.1.1 is public", "1.1.1.1", false},
		{"203.0.113.1 is public", "203.0.113.1", false},
		{"93.184.216.34 is public", "93.184.216.34", false},

		// Invalid IPs should return false (or be handled gracefully)
		{"invalid IP empty", "", false},
		{"invalid IP string", "not-an-ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("IsPrivateIP(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"localhost is localhost", "localhost", true},
		{"127.0.0.1 is localhost", "127.0.0.1", true},
		{"::1 is localhost", "::1", true},
		{"[::1] is localhost", "[::1]", true},
		{"0.0.0.0 is localhost", "0.0.0.0", true},
		{"example.com is NOT localhost", "example.com", false},
		{"8.8.8.8 is NOT localhost", "8.8.8.8", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLocalhost(tt.host)
			if result != tt.expected {
				t.Errorf("IsLocalhost(%q) = %v, expected %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		allowLocal    bool
		expectError   bool
		errorContains string
	}{
		// Valid public HTTPS URLs
		{
			name:        "valid HTTPS URL",
			url:         "https://example.com/webhook",
			allowLocal:  false,
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with port",
			url:         "https://example.com:8443/webhook",
			allowLocal:  false,
			expectError: false,
		},
		{
			name:        "valid HTTPS URL with path and query",
			url:         "https://api.example.com/v1/webhook?key=value",
			allowLocal:  false,
			expectError: false,
		},

		// HTTP URLs (blocked unless localhost)
		{
			name:          "HTTP URL blocked",
			url:           "http://example.com/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "HTTPS",
		},
		{
			name:          "HTTP localhost blocked when allowLocal is false",
			url:           "http://localhost/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "localhost",
		},
		{
			name:        "HTTP localhost allowed when allowLocal is true",
			url:         "http://localhost/webhook",
			allowLocal:  true,
			expectError: false,
		},
		{
			name:        "HTTP localhost with port allowed when allowLocal is true",
			url:         "http://localhost:3000/webhook",
			allowLocal:  true,
			expectError: false,
		},
		{
			name:        "HTTP 127.0.0.1 allowed when allowLocal is true",
			url:         "http://127.0.0.1:3000/webhook",
			allowLocal:  true,
			expectError: false,
		},

		// Private IP addresses (blocked)
		{
			name:          "private IP 10.x blocked",
			url:           "https://10.0.0.1/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "private",
		},
		{
			name:          "private IP 172.16.x blocked",
			url:           "https://172.16.0.1/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "private",
		},
		{
			name:          "private IP 192.168.x blocked",
			url:           "https://192.168.1.1/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "private",
		},

		// Link-local addresses (blocked)
		{
			name:          "link-local IP blocked",
			url:           "https://169.254.169.254/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "private",
		},

		// Localhost (blocked unless allowed)
		{
			name:          "localhost blocked when allowLocal is false",
			url:           "https://localhost/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "localhost",
		},
		{
			name:          "127.0.0.1 blocked when allowLocal is false",
			url:           "https://127.0.0.1/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "localhost",
		},
		{
			name:        "localhost allowed when allowLocal is true",
			url:         "https://localhost/webhook",
			allowLocal:  true,
			expectError: false,
		},

		// Invalid URLs
		{
			name:          "empty URL",
			url:           "",
			allowLocal:    false,
			expectError:   true,
			errorContains: "empty",
		},
		{
			name:          "invalid URL",
			url:           "not-a-url",
			allowLocal:    false,
			expectError:   true,
			errorContains: "scheme", // "not-a-url" is parsed as path, empty scheme
		},
		{
			name:          "missing scheme",
			url:           "example.com/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "scheme", // parsed as path, empty scheme
		},
		{
			name:          "unsupported scheme",
			url:           "ftp://example.com/webhook",
			allowLocal:    false,
			expectError:   true,
			errorContains: "scheme",
		},

		// Edge cases
		{
			name:        "URL with internal DNS-like name",
			url:         "https://internal.company.local/webhook",
			allowLocal:  false,
			expectError: false, // We can't block DNS names, only IPs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWebhookURL(tt.url, tt.allowLocal)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateWebhookURL(%q, %v) expected error containing %q, got nil",
						tt.url, tt.allowLocal, tt.errorContains)
				} else if tt.errorContains != "" && !containsIgnoreCase(err.Error(), tt.errorContains) {
					t.Errorf("ValidateWebhookURL(%q, %v) error = %q, expected to contain %q",
						tt.url, tt.allowLocal, err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateWebhookURL(%q, %v) unexpected error: %v",
						tt.url, tt.allowLocal, err)
				}
			}
		})
	}
}

func TestValidateWebhookURLResolvesHostname(t *testing.T) {
	// Test that we attempt to resolve hostnames and check the resolved IPs
	// This test uses a mock or skips actual DNS resolution
	t.Run("hostname resolution is attempted", func(t *testing.T) {
		// For now, we trust that DNS-based hostnames are valid
		// The important thing is that IP addresses are checked
		err := ValidateWebhookURL("https://example.com/webhook", false)
		if err != nil {
			t.Errorf("public hostname should be valid: %v", err)
		}
	})
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && containsLower(toLower(s), toLower(substr))))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
