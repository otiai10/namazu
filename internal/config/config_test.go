package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `source:
  type: p2pquake
  endpoint: wss://api-realtime-sandbox.p2pquake.net/v2/ws

subscriptions:
  - name: webhook1
    delivery:
      type: webhook
      url: https://example.com/webhook1
      secret: secret1
  - name: webhook2
    delivery:
      type: webhook
      url: https://example.com/webhook2
      secret: secret2
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load the configuration
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Verify source configuration
	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Source.Type = %q, want %q", cfg.Source.Type, "p2pquake")
	}

	if cfg.Source.Endpoint != "wss://api-realtime-sandbox.p2pquake.net/v2/ws" {
		t.Errorf("Source.Endpoint = %q, want %q", cfg.Source.Endpoint, "wss://api-realtime-sandbox.p2pquake.net/v2/ws")
	}

	// Verify subscriptions
	if len(cfg.Subscriptions) != 2 {
		t.Fatalf("len(Subscriptions) = %d, want 2", len(cfg.Subscriptions))
	}

	if cfg.Subscriptions[0].Name != "webhook1" {
		t.Errorf("Subscriptions[0].Name = %q, want %q", cfg.Subscriptions[0].Name, "webhook1")
	}

	if cfg.Subscriptions[0].Delivery.Type != "webhook" {
		t.Errorf("Subscriptions[0].Delivery.Type = %q, want %q", cfg.Subscriptions[0].Delivery.Type, "webhook")
	}

	if cfg.Subscriptions[0].Delivery.URL != "https://example.com/webhook1" {
		t.Errorf("Subscriptions[0].Delivery.URL = %q, want %q", cfg.Subscriptions[0].Delivery.URL, "https://example.com/webhook1")
	}

	if cfg.Subscriptions[0].Delivery.Secret != "secret1" {
		t.Errorf("Subscriptions[0].Delivery.Secret = %q, want %q", cfg.Subscriptions[0].Delivery.Secret, "secret1")
	}

	if cfg.Subscriptions[1].Name != "webhook2" {
		t.Errorf("Subscriptions[1].Name = %q, want %q", cfg.Subscriptions[1].Name, "webhook2")
	}

	if cfg.Subscriptions[1].Delivery.URL != "https://example.com/webhook2" {
		t.Errorf("Subscriptions[1].Delivery.URL = %q, want %q", cfg.Subscriptions[1].Delivery.URL, "https://example.com/webhook2")
	}

	if cfg.Subscriptions[1].Delivery.Secret != "secret2" {
		t.Errorf("Subscriptions[1].Delivery.Secret = %q, want %q", cfg.Subscriptions[1].Delivery.Secret, "secret2")
	}
}

func TestLoad_EnvironmentVariableOverride(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `source:
  type: p2pquake
  endpoint: wss://api-realtime-sandbox.p2pquake.net/v2/ws

subscriptions:
  - name: test-webhook
    delivery:
      type: webhook
      url: https://example.com/webhook
      secret: secret1
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Set environment variable
	overrideEndpoint := "wss://custom-endpoint.example.com/ws"
	os.Setenv("NAMAZU_SOURCE_ENDPOINT", overrideEndpoint)
	defer os.Unsetenv("NAMAZU_SOURCE_ENDPOINT")

	// Load the configuration
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	// Verify environment variable override
	if cfg.Source.Endpoint != overrideEndpoint {
		t.Errorf("Source.Endpoint = %q, want %q (from env var)", cfg.Source.Endpoint, overrideEndpoint)
	}

	// Verify other fields are unchanged
	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Source.Type = %q, want %q", cfg.Source.Type, "p2pquake")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() error = nil, want error for non-existent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `source:
  type: p2pquake
  endpoint: wss://example.com
  invalid yaml content [[[
`

	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want error for invalid YAML")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "secret123",
				},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_EmptySourceType(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty source type")
	}
}

func TestValidate_EmptySourceEndpoint(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for empty endpoint")
	}
}

func TestValidate_UnsupportedSourceType(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "unsupported",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for unsupported source type")
	}
}

func TestValidate_NoSubscriptions(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for no subscriptions")
	}
}

func TestValidate_SubscriptionMissingName(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for subscription missing name")
	}
}

func TestValidate_SubscriptionMissingDeliveryType(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for subscription missing delivery type")
	}
}

func TestValidate_SubscriptionUnsupportedDeliveryType(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "unsupported",
					URL:    "https://example.com/webhook",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for unsupported delivery type")
	}
}

func TestValidate_SubscriptionMissingURL(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "",
					Secret: "secret",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for subscription missing URL")
	}
}

func TestValidate_SubscriptionMissingSecret(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "test-webhook",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example.com/webhook",
					Secret: "",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for subscription missing secret")
	}
}

func TestValidate_MultipleSubscriptions(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{
			{
				Name: "webhook1",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example1.com/webhook",
					Secret: "secret1",
				},
			},
			{
				Name: "webhook2",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example2.com/webhook",
					Secret: "secret2",
				},
			},
			{
				Name: "webhook3",
				Delivery: DeliveryConfig{
					Type:   "webhook",
					URL:    "https://example3.com/webhook",
					Secret: "secret3",
				},
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for multiple valid subscriptions", err)
	}
}

func TestLoad_ExampleConfigFile(t *testing.T) {
	// Test loading the actual example config file from the project root
	exampleConfigPath := "../../config.example.yaml"

	cfg, err := Load(exampleConfigPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for example config", err)
	}

	// Verify the example config structure
	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Example config Source.Type = %q, want %q", cfg.Source.Type, "p2pquake")
	}

	if cfg.Source.Endpoint != "wss://api-realtime-sandbox.p2pquake.net/v2/ws" {
		t.Errorf("Example config Source.Endpoint = %q, want %q",
			cfg.Source.Endpoint,
			"wss://api-realtime-sandbox.p2pquake.net/v2/ws")
	}

	if len(cfg.Subscriptions) != 1 {
		t.Errorf("Example config has %d subscriptions, want 1", len(cfg.Subscriptions))
	}

	if len(cfg.Subscriptions) > 0 {
		if cfg.Subscriptions[0].Name != "my-webhook" {
			t.Errorf("Example config subscription Name = %q, want %q",
				cfg.Subscriptions[0].Name,
				"my-webhook")
		}

		if cfg.Subscriptions[0].Delivery.Type != "webhook" {
			t.Errorf("Example config subscription Delivery.Type = %q, want %q",
				cfg.Subscriptions[0].Delivery.Type,
				"webhook")
		}

		if cfg.Subscriptions[0].Delivery.URL != "https://example.com/earthquake" {
			t.Errorf("Example config subscription Delivery.URL = %q, want %q",
				cfg.Subscriptions[0].Delivery.URL,
				"https://example.com/earthquake")
		}

		if cfg.Subscriptions[0].Delivery.Secret != "your-secret-key-here" {
			t.Errorf("Example config subscription Delivery.Secret = %q, want %q",
				cfg.Subscriptions[0].Delivery.Secret,
				"your-secret-key-here")
		}
	}
}
