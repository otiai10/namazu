package auth

import (
	"testing"
)

func TestGetStringClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string claim",
			claims:   map[string]interface{}{"email": "test@example.com"},
			key:      "email",
			expected: "test@example.com",
		},
		{
			name:     "missing claim",
			claims:   map[string]interface{}{},
			key:      "email",
			expected: "",
		},
		{
			name:     "wrong type claim",
			claims:   map[string]interface{}{"email": 123},
			key:      "email",
			expected: "",
		},
		{
			name:     "nil claims map",
			claims:   nil,
			key:      "email",
			expected: "",
		},
		{
			name:     "empty string claim",
			claims:   map[string]interface{}{"email": ""},
			key:      "email",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringClaim(tt.claims, tt.key)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetBoolClaim(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]interface{}
		key      string
		expected bool
	}{
		{
			name:     "existing true claim",
			claims:   map[string]interface{}{"email_verified": true},
			key:      "email_verified",
			expected: true,
		},
		{
			name:     "existing false claim",
			claims:   map[string]interface{}{"email_verified": false},
			key:      "email_verified",
			expected: false,
		},
		{
			name:     "missing claim",
			claims:   map[string]interface{}{},
			key:      "email_verified",
			expected: false,
		},
		{
			name:     "wrong type claim - string",
			claims:   map[string]interface{}{"email_verified": "true"},
			key:      "email_verified",
			expected: false,
		},
		{
			name:     "wrong type claim - int",
			claims:   map[string]interface{}{"email_verified": 1},
			key:      "email_verified",
			expected: false,
		},
		{
			name:     "nil claims map",
			claims:   nil,
			key:      "email_verified",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolClaim(tt.claims, tt.key)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
