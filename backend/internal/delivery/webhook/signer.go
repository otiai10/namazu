package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const DefaultMaxAge = 5 * time.Minute

// Sign generates HMAC-SHA256 signature for the given payload.
// The signature is returned in the format "sha256=<hex-encoded-signature>".
//
// Parameters:
//   - secret: The secret key used for HMAC signing
//   - payload: The data to be signed
//
// Returns:
//   - A string in the format "sha256=<hex-encoded-signature>"
//
// Example:
//
//	secret := "my-secret-key"
//	payload := []byte("hello world")
//	signature := Sign(secret, payload)
//	// signature = "sha256=734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a"
func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := mac.Sum(nil)
	return "sha256=" + hex.EncodeToString(signature)
}

// Verify checks if the provided signature matches the expected signature
// for the given payload and secret using constant-time comparison.
//
// This function protects against timing attacks by using hmac.Equal
// which performs constant-time comparison.
//
// Parameters:
//   - secret: The secret key used for HMAC signing
//   - payload: The data that was signed
//   - signature: The signature to verify (should be in "sha256=<hex>" format)
//
// Returns:
//   - true if the signature is valid, false otherwise
//
// Example:
//
//	secret := "my-secret-key"
//	payload := []byte("hello world")
//	signature := Sign(secret, payload)
//	isValid := Verify(secret, payload, signature)
//	// isValid = true
func Verify(secret string, payload []byte, signature string) bool {
	expected := Sign(secret, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func SignV0(secret string, timestamp int64, payload []byte) string {
	baseString := fmt.Sprintf("v0:%d:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	signature := mac.Sum(nil)
	return "v0=" + hex.EncodeToString(signature)
}

func VerifyV0(secret string, timestamp int64, payload []byte, signature string, maxAge time.Duration) bool {
	now := time.Now().Unix()
	if now-timestamp > int64(maxAge.Seconds()) {
		return false
	}
	if timestamp > now+60 {
		return false
	}
	expected := SignV0(secret, timestamp, payload)
	return hmac.Equal([]byte(expected), []byte(signature))
}
