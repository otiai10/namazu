package security

import (
	"testing"
)

func TestRequireHTTPS(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		allowLocal  bool
		expectError bool
	}{
		// HTTPS URLs (always valid)
		{
			name:        "HTTPS URL is valid",
			url:         "https://example.com/webhook",
			allowLocal:  false,
			expectError: false,
		},
		{
			name:        "HTTPS URL with port is valid",
			url:         "https://example.com:8443/webhook",
			allowLocal:  false,
			expectError: false,
		},
		{
			name:        "HTTPS localhost is valid",
			url:         "https://localhost/webhook",
			allowLocal:  false,
			expectError: false,
		},

		// HTTP URLs (blocked unless localhost allowed)
		{
			name:        "HTTP URL is blocked",
			url:         "http://example.com/webhook",
			allowLocal:  false,
			expectError: true,
		},
		{
			name:        "HTTP URL is blocked even with allowLocal",
			url:         "http://example.com/webhook",
			allowLocal:  true,
			expectError: true,
		},

		// HTTP localhost (allowed when allowLocal is true)
		{
			name:        "HTTP localhost blocked when allowLocal is false",
			url:         "http://localhost/webhook",
			allowLocal:  false,
			expectError: true,
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
			url:         "http://127.0.0.1/webhook",
			allowLocal:  true,
			expectError: false,
		},
		{
			name:        "HTTP 127.0.0.1 with port allowed when allowLocal is true",
			url:         "http://127.0.0.1:8080/webhook",
			allowLocal:  true,
			expectError: false,
		},
		{
			name:        "HTTP [::1] allowed when allowLocal is true",
			url:         "http://[::1]:8080/webhook",
			allowLocal:  true,
			expectError: false,
		},

		// Invalid URLs
		{
			name:        "empty URL",
			url:         "",
			allowLocal:  false,
			expectError: true,
		},
		{
			name:        "invalid URL",
			url:         "not-a-url",
			allowLocal:  false,
			expectError: true,
		},
		{
			name:        "unsupported scheme",
			url:         "ftp://example.com/file",
			allowLocal:  false,
			expectError: true,
		},
		{
			name:        "websocket scheme",
			url:         "wss://example.com/ws",
			allowLocal:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireHTTPS(tt.url, tt.allowLocal)

			if tt.expectError {
				if err == nil {
					t.Errorf("RequireHTTPS(%q, %v) expected error, got nil", tt.url, tt.allowLocal)
				}
			} else {
				if err != nil {
					t.Errorf("RequireHTTPS(%q, %v) unexpected error: %v", tt.url, tt.allowLocal, err)
				}
			}
		})
	}
}

func TestParseAndValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		expectHost  string
	}{
		{
			name:        "valid URL with host",
			url:         "https://example.com/path",
			expectError: false,
			expectHost:  "example.com",
		},
		{
			name:        "valid URL with port",
			url:         "https://example.com:8443/path",
			expectError: false,
			expectHost:  "example.com:8443",
		},
		{
			name:        "empty URL",
			url:         "",
			expectError: true,
		},
		{
			name:        "URL without scheme",
			url:         "example.com/path",
			expectError: true,
		},
		{
			name:        "URL without host",
			url:         "https:///path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseAndValidateURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseAndValidateURL(%q) expected error, got nil", tt.url)
				}
			} else {
				if err != nil {
					t.Errorf("ParseAndValidateURL(%q) unexpected error: %v", tt.url, err)
				}
				if parsed.Host != tt.expectHost {
					t.Errorf("ParseAndValidateURL(%q) host = %q, expected %q", tt.url, parsed.Host, tt.expectHost)
				}
			}
		})
	}
}

func TestExtractHostWithoutPort(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "host without port",
			host:     "example.com",
			expected: "example.com",
		},
		{
			name:     "host with port",
			host:     "example.com:8443",
			expected: "example.com",
		},
		{
			name:     "IPv4 without port",
			host:     "192.168.1.1",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv4 with port",
			host:     "192.168.1.1:8080",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 with brackets without port",
			host:     "[::1]",
			expected: "::1",
		},
		{
			name:     "IPv6 with brackets and port",
			host:     "[::1]:8080",
			expected: "::1",
		},
		{
			name:     "localhost",
			host:     "localhost",
			expected: "localhost",
		},
		{
			name:     "localhost with port",
			host:     "localhost:3000",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHostWithoutPort(tt.host)
			if result != tt.expected {
				t.Errorf("ExtractHostWithoutPort(%q) = %q, expected %q", tt.host, result, tt.expected)
			}
		})
	}
}
