package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const (
	SecretLength = 32
	SecretPrefix = "nmz_"
)

func GenerateSecret() (string, error) {
	b := make([]byte, SecretLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return SecretPrefix + hex.EncodeToString(b), nil
}

func MaskSecret(secret string) string {
	if len(secret) <= 12 {
		return "****"
	}
	return secret[:8] + "..." + secret[len(secret)-4:]
}

func SecretPrefixFromSecret(secret string) string {
	if len(secret) < 8 {
		return secret
	}
	return secret[:8]
}
