package webhook

import (
	"strings"
	"testing"
)

func TestGenerateSecret_Format(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(secret, SecretPrefix) {
		t.Errorf("expected secret to start with %q, got %q", SecretPrefix, secret)
	}

	// SecretPrefix (4 chars "nmz_") + hex-encoded 32 bytes (64 chars) = 68
	expectedLength := len(SecretPrefix) + SecretLength*2
	if len(secret) != expectedLength {
		t.Errorf("expected secret length %d, got %d", expectedLength, len(secret))
	}
}

func TestGenerateSecret_Uniqueness(t *testing.T) {
	secret1, err := GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error generating first secret: %v", err)
	}

	secret2, err := GenerateSecret()
	if err != nil {
		t.Fatalf("unexpected error generating second secret: %v", err)
	}

	if secret1 == secret2 {
		t.Error("expected two generated secrets to be different")
	}
}

func TestMaskSecret(t *testing.T) {
	// A typical generated secret: "nmz_" + 64 hex chars = 68 chars total
	secret := "nmz_abcdef1234567890abcdef1234567890abcdef1234567890abcdef12345678"

	masked := MaskSecret(secret)

	// Should show first 8 chars and last 4 chars with "..." in the middle
	expectedPrefix := secret[:8]
	expectedSuffix := secret[len(secret)-4:]

	if !strings.HasPrefix(masked, expectedPrefix) {
		t.Errorf("expected masked secret to start with %q, got %q", expectedPrefix, masked)
	}
	if !strings.HasSuffix(masked, expectedSuffix) {
		t.Errorf("expected masked secret to end with %q, got %q", expectedSuffix, masked)
	}
	if !strings.Contains(masked, "...") {
		t.Error("expected masked secret to contain '...'")
	}

	expectedMasked := expectedPrefix + "..." + expectedSuffix
	if masked != expectedMasked {
		t.Errorf("expected %q, got %q", expectedMasked, masked)
	}
}

func TestMaskSecret_ShortSecret(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expected string
	}{
		{name: "empty", secret: "", expected: "****"},
		{name: "very short", secret: "abc", expected: "****"},
		{name: "exactly 12", secret: "123456789012", expected: "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := MaskSecret(tt.secret)
			if masked != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, masked)
			}
		})
	}
}

func TestSecretPrefixFromSecret(t *testing.T) {
	secret := "nmz_abcdef1234567890"

	prefix := SecretPrefixFromSecret(secret)

	expected := "nmz_abcd"
	if prefix != expected {
		t.Errorf("expected %q, got %q", expected, prefix)
	}
}

func TestSecretPrefixFromSecret_ShortInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "3 chars", input: "abc", expected: "abc"},
		{name: "7 chars", input: "abcdefg", expected: "abcdefg"},
		{name: "exactly 8 chars", input: "abcdefgh", expected: "abcdefgh"},
		{name: "9 chars", input: "abcdefghi", expected: "abcdefgh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SecretPrefixFromSecret(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
