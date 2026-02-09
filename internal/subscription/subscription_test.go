package subscription

import (
	"encoding/json"
	"testing"
)

func TestDeliveryConfig_ZeroValue_BackwardCompatible(t *testing.T) {
	var dc DeliveryConfig
	if dc.Verified != false {
		t.Errorf("zero value Verified should be false, got %v", dc.Verified)
	}
	if dc.SignVersion != "" {
		t.Errorf("zero value SignVersion should be empty, got %q", dc.SignVersion)
	}
	if dc.SecretPrefix != "" {
		t.Errorf("zero value SecretPrefix should be empty, got %q", dc.SecretPrefix)
	}
}

func TestDeliveryConfig_JSONRoundTrip(t *testing.T) {
	t.Run("serializes new fields to JSON", func(t *testing.T) {
		dc := DeliveryConfig{
			Type:         "webhook",
			URL:          "https://example.com/hook",
			Secret:       "whsec_abc123",
			SecretPrefix: "whsec_",
			Verified:     true,
			SignVersion:  "v1",
		}

		data, err := json.Marshal(dc)
		if err != nil {
			t.Fatalf("failed to marshal DeliveryConfig: %v", err)
		}

		var result DeliveryConfig
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal DeliveryConfig: %v", err)
		}

		if result.SecretPrefix != "whsec_" {
			t.Errorf("SecretPrefix = %q, want %q", result.SecretPrefix, "whsec_")
		}
		if result.Verified != true {
			t.Errorf("Verified = %v, want %v", result.Verified, true)
		}
		if result.SignVersion != "v1" {
			t.Errorf("SignVersion = %q, want %q", result.SignVersion, "v1")
		}
	})

	t.Run("omits empty new fields in JSON", func(t *testing.T) {
		dc := DeliveryConfig{
			Type: "webhook",
			URL:  "https://example.com/hook",
		}

		data, err := json.Marshal(dc)
		if err != nil {
			t.Fatalf("failed to marshal DeliveryConfig: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal to map: %v", err)
		}

		if _, exists := raw["secret_prefix"]; exists {
			t.Error("expected secret_prefix to be omitted when empty")
		}
		if _, exists := raw["sign_version"]; exists {
			t.Error("expected sign_version to be omitted when empty")
		}
		// verified should always be present (no omitempty)
		if _, exists := raw["verified"]; !exists {
			t.Error("expected verified to always be present in JSON")
		}
	})

	t.Run("backward compatible with old JSON without new fields", func(t *testing.T) {
		oldJSON := `{"type":"webhook","url":"https://example.com/hook","secret":"s3cret"}`

		var dc DeliveryConfig
		if err := json.Unmarshal([]byte(oldJSON), &dc); err != nil {
			t.Fatalf("failed to unmarshal old JSON: %v", err)
		}

		if dc.Type != "webhook" {
			t.Errorf("Type = %q, want %q", dc.Type, "webhook")
		}
		if dc.URL != "https://example.com/hook" {
			t.Errorf("URL = %q, want %q", dc.URL, "https://example.com/hook")
		}
		if dc.Secret != "s3cret" {
			t.Errorf("Secret = %q, want %q", dc.Secret, "s3cret")
		}
		if dc.SecretPrefix != "" {
			t.Errorf("SecretPrefix = %q, want empty", dc.SecretPrefix)
		}
		if dc.Verified != false {
			t.Errorf("Verified = %v, want false", dc.Verified)
		}
		if dc.SignVersion != "" {
			t.Errorf("SignVersion = %q, want empty", dc.SignVersion)
		}
	})
}

func TestSubscriptionToMap_IncludesNewDeliveryFields(t *testing.T) {
	t.Run("includes new fields in delivery map", func(t *testing.T) {
		sub := Subscription{
			ID:   "test-id",
			Name: "Test Subscription",
			Delivery: DeliveryConfig{
				Type:         "webhook",
				URL:          "https://example.com/webhook",
				Secret:       "whsec_abc123",
				SecretPrefix: "whsec_",
				Verified:     true,
				SignVersion:  "v1",
			},
		}

		data := subscriptionToMap(sub)

		delivery, ok := data["delivery"].(map[string]interface{})
		if !ok {
			t.Fatal("expected delivery to be a map")
		}
		if delivery["secret_prefix"] != "whsec_" {
			t.Errorf("secret_prefix = %v, want %q", delivery["secret_prefix"], "whsec_")
		}
		if delivery["verified"] != true {
			t.Errorf("verified = %v, want true", delivery["verified"])
		}
		if delivery["sign_version"] != "v1" {
			t.Errorf("sign_version = %v, want %q", delivery["sign_version"], "v1")
		}
	})

	t.Run("includes zero values for new fields", func(t *testing.T) {
		sub := Subscription{
			ID:   "test-id",
			Name: "Test Subscription",
			Delivery: DeliveryConfig{
				Type: "webhook",
				URL:  "https://example.com/webhook",
			},
		}

		data := subscriptionToMap(sub)

		delivery, ok := data["delivery"].(map[string]interface{})
		if !ok {
			t.Fatal("expected delivery to be a map")
		}
		if delivery["secret_prefix"] != "" {
			t.Errorf("secret_prefix = %v, want empty string", delivery["secret_prefix"])
		}
		if delivery["verified"] != false {
			t.Errorf("verified = %v, want false", delivery["verified"])
		}
		if delivery["sign_version"] != "" {
			t.Errorf("sign_version = %v, want empty string", delivery["sign_version"])
		}
	})
}

func TestDocumentToSubscription_ParsesNewDeliveryFields(t *testing.T) {
	// We cannot call documentToSubscription directly without a *firestore.DocumentSnapshot,
	// but we test the round-trip via subscriptionToMap -> verify the map keys exist.
	// The actual parsing is tested in firestore_integration_test.go.
	// Here we verify the map output is correct for the new fields.

	t.Run("map contains all new delivery fields for round-trip", func(t *testing.T) {
		sub := Subscription{
			Name: "Round Trip Test",
			Delivery: DeliveryConfig{
				Type:         "webhook",
				URL:          "https://example.com/hook",
				Secret:       "secret",
				SecretPrefix: "whsec_",
				Verified:     true,
				SignVersion:  "v1",
			},
		}

		data := subscriptionToMap(sub)
		delivery := data["delivery"].(map[string]interface{})

		expectedKeys := []string{"type", "url", "secret", "secret_prefix", "verified", "sign_version"}
		for _, key := range expectedKeys {
			if _, exists := delivery[key]; !exists {
				t.Errorf("expected delivery map to contain key %q", key)
			}
		}
	})
}
