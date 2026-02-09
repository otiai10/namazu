package webhook

import (
	"testing"
	"time"
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

func TestSignV0_Format(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"event":"earthquake"}`)

	signature := SignV0(secret, timestamp, payload)

	if len(signature) < 3 || signature[:3] != "v0=" {
		t.Errorf("SignV0() signature should start with 'v0=', got %s", signature)
	}
}

func TestSignV0_IncludesTimestamp(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"earthquake"}`)
	ts1 := int64(1700000000)
	ts2 := int64(1700000001)

	sig1 := SignV0(secret, ts1, payload)
	sig2 := SignV0(secret, ts2, payload)

	if sig1 == sig2 {
		t.Errorf("SignV0() should produce different signatures for different timestamps, both got %s", sig1)
	}
}

func TestSignV0_Deterministic(t *testing.T) {
	secret := "test-secret"
	timestamp := int64(1700000000)
	payload := []byte(`{"event":"earthquake"}`)

	sig1 := SignV0(secret, timestamp, payload)
	sig2 := SignV0(secret, timestamp, payload)

	if sig1 != sig2 {
		t.Errorf("SignV0() should be deterministic: sig1=%s, sig2=%s", sig1, sig2)
	}
}

func TestVerifyV0_ValidSignature(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"event":"earthquake","magnitude":5.5}`)

	signature := SignV0(secret, timestamp, payload)
	if !VerifyV0(secret, timestamp, payload, signature, DefaultMaxAge) {
		t.Error("VerifyV0() returned false for valid signature with fresh timestamp")
	}
}

func TestVerifyV0_ExpiredTimestamp(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix() - 360 // 6 minutes ago
	payload := []byte(`{"event":"earthquake"}`)

	signature := SignV0(secret, timestamp, payload)
	if VerifyV0(secret, timestamp, payload, signature, DefaultMaxAge) {
		t.Error("VerifyV0() should return false for expired timestamp (6 min old)")
	}
}

func TestVerifyV0_FutureTimestamp(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix() + 120 // 2 minutes in the future
	payload := []byte(`{"event":"earthquake"}`)

	signature := SignV0(secret, timestamp, payload)
	if VerifyV0(secret, timestamp, payload, signature, DefaultMaxAge) {
		t.Error("VerifyV0() should return false for future timestamp (>60s)")
	}
}

func TestVerifyV0_WrongSecret(t *testing.T) {
	secret := "correct-secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"event":"earthquake"}`)

	signature := SignV0(secret, timestamp, payload)
	if VerifyV0("wrong-secret", timestamp, payload, signature, DefaultMaxAge) {
		t.Error("VerifyV0() should return false for wrong secret")
	}
}

func TestVerifyV0_TamperedPayload(t *testing.T) {
	secret := "test-secret"
	timestamp := time.Now().Unix()
	payload := []byte(`{"event":"earthquake","magnitude":5.5}`)

	signature := SignV0(secret, timestamp, payload)
	tampered := []byte(`{"event":"earthquake","magnitude":9.9}`)
	if VerifyV0(secret, timestamp, tampered, signature, DefaultMaxAge) {
		t.Error("VerifyV0() should return false for tampered payload")
	}
}

func TestVerifyV0_CustomMaxAge(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"event":"earthquake"}`)

	// 7 minutes ago - should fail with default 5min but pass with 10min
	timestamp := time.Now().Unix() - 420

	signature := SignV0(secret, timestamp, payload)

	if VerifyV0(secret, timestamp, payload, signature, DefaultMaxAge) {
		t.Error("VerifyV0() should return false with default maxAge for 7-min-old timestamp")
	}

	if !VerifyV0(secret, timestamp, payload, signature, 10*time.Minute) {
		t.Error("VerifyV0() should return true with 10-minute maxAge for 7-min-old timestamp")
	}
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
