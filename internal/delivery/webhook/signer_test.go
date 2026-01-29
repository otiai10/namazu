package webhook

import (
	"testing"
)

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		payload  []byte
		expected string
	}{
		{
			name:     "basic signature",
			secret:   "my-secret-key",
			payload:  []byte("hello world"),
			expected: "sha256=90eb182d8396f16d4341d582047f45c0a97d73388c5377d9ced478a2212295ad",
		},
		{
			name:     "empty payload",
			payload:  []byte(""),
			secret:   "my-secret-key",
			expected: "sha256=3f88a772c79764707652942745bb0a16a25f2a113acfeaea1e07ae04f8d90ac6",
		},
		{
			name:     "empty secret",
			secret:   "",
			payload:  []byte("hello world"),
			expected: "sha256=c2ea634c993f050482b4e6243224087f7c23bdd3c07ab1a45e9a21c62fad994e",
		},
		{
			name:     "json payload",
			secret:   "webhook-secret",
			payload:  []byte(`{"event":"earthquake","magnitude":5.5}`),
			expected: "sha256=",
		},
		{
			name:     "unicode characters",
			secret:   "秘密鍵",
			payload:  []byte("地震情報"),
			expected: "sha256=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sign(tt.secret, tt.payload)

			// Verify format starts with "sha256="
			if len(result) < 7 || result[:7] != "sha256=" {
				t.Errorf("Sign() signature format incorrect, expected to start with 'sha256=', got %s", result)
			}

			// For known test vectors, verify exact match
			if tt.name == "basic signature" || tt.name == "empty payload" || tt.name == "empty secret" {
				if result != tt.expected {
					t.Errorf("Sign() = %v, expected %v", result, tt.expected)
				}
			}
		})
	}
}

func TestVerify(t *testing.T) {
	secret := "my-webhook-secret"
	payload := []byte(`{"event":"earthquake","magnitude":6.5}`)

	t.Run("valid signature", func(t *testing.T) {
		signature := Sign(secret, payload)
		if !Verify(secret, payload, signature) {
			t.Error("Verify() returned false for valid signature")
		}
	})

	t.Run("invalid signature - wrong secret", func(t *testing.T) {
		signature := Sign("wrong-secret", payload)
		if Verify(secret, payload, signature) {
			t.Error("Verify() returned true for signature with wrong secret")
		}
	})

	t.Run("invalid signature - tampered payload", func(t *testing.T) {
		signature := Sign(secret, payload)
		tamperedPayload := []byte(`{"event":"earthquake","magnitude":7.5}`)
		if Verify(secret, tamperedPayload, signature) {
			t.Error("Verify() returned true for tampered payload")
		}
	})

	t.Run("invalid signature - malformed format", func(t *testing.T) {
		invalidSignature := "invalid-format-12345"
		if Verify(secret, payload, invalidSignature) {
			t.Error("Verify() returned true for malformed signature")
		}
	})

	t.Run("invalid signature - empty signature", func(t *testing.T) {
		if Verify(secret, payload, "") {
			t.Error("Verify() returned true for empty signature")
		}
	})

	t.Run("empty payload with valid signature", func(t *testing.T) {
		emptyPayload := []byte("")
		signature := Sign(secret, emptyPayload)
		if !Verify(secret, emptyPayload, signature) {
			t.Error("Verify() returned false for valid signature with empty payload")
		}
	})

	t.Run("empty secret with valid signature", func(t *testing.T) {
		emptySecret := ""
		signature := Sign(emptySecret, payload)
		if !Verify(emptySecret, payload, signature) {
			t.Error("Verify() returned false for valid signature with empty secret")
		}
	})
}

func TestSignConsistency(t *testing.T) {
	secret := "consistent-secret"
	payload := []byte("consistent payload")

	t.Run("same inputs produce same signature", func(t *testing.T) {
		sig1 := Sign(secret, payload)
		sig2 := Sign(secret, payload)

		if sig1 != sig2 {
			t.Errorf("Sign() inconsistent: sig1=%s, sig2=%s", sig1, sig2)
		}
	})
}

func BenchmarkSign(b *testing.B) {
	secret := "benchmark-secret"
	payload := []byte(`{"event":"earthquake","magnitude":5.5,"location":"Tokyo"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sign(secret, payload)
	}
}

func BenchmarkVerify(b *testing.B) {
	secret := "benchmark-secret"
	payload := []byte(`{"event":"earthquake","magnitude":5.5,"location":"Tokyo"}`)
	signature := Sign(secret, payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(secret, payload, signature)
	}
}
