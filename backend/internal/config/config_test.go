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

func TestLoad_WithStoreConfig(t *testing.T) {
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

store:
  type: firestore
  project_id: test-project-id
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Store == nil {
		t.Fatal("Store config should not be nil")
	}

	if cfg.Store.Type != "firestore" {
		t.Errorf("Store.Type = %q, want %q", cfg.Store.Type, "firestore")
	}

	if cfg.Store.ProjectID != "test-project-id" {
		t.Errorf("Store.ProjectID = %q, want %q", cfg.Store.ProjectID, "test-project-id")
	}
}

func TestLoad_WithoutStoreConfig(t *testing.T) {
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

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Store != nil {
		t.Errorf("Store config should be nil when not specified, got %+v", cfg.Store)
	}
}

func TestLoad_StoreProjectIDEnvironmentOverride(t *testing.T) {
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

store:
  type: firestore
  project_id: original-project
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	overrideProjectID := "env-override-project"
	os.Setenv("NAMAZU_STORE_PROJECT_ID", overrideProjectID)
	defer os.Unsetenv("NAMAZU_STORE_PROJECT_ID")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Store == nil {
		t.Fatal("Store config should not be nil")
	}

	if cfg.Store.ProjectID != overrideProjectID {
		t.Errorf("Store.ProjectID = %q, want %q (from env var)", cfg.Store.ProjectID, overrideProjectID)
	}
}

func TestLoad_StoreProjectIDEnvironmentOverrideCreatesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without store section
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

	overrideProjectID := "env-only-project"
	os.Setenv("NAMAZU_STORE_PROJECT_ID", overrideProjectID)
	defer os.Unsetenv("NAMAZU_STORE_PROJECT_ID")

	// Environment variable creates a Store config with type "firestore" automatically
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Store == nil {
		t.Fatal("Store config should be created by env var")
	}
	if cfg.Store.Type != "firestore" {
		t.Errorf("Store.Type = %q, want %q", cfg.Store.Type, "firestore")
	}
	if cfg.Store.ProjectID != overrideProjectID {
		t.Errorf("Store.ProjectID = %q, want %q", cfg.Store.ProjectID, overrideProjectID)
	}
}

func TestValidate_StoreConfigValid(t *testing.T) {
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
		Store: &StoreConfig{
			Type:      "firestore",
			ProjectID: "test-project",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_StoreConfigMissingType(t *testing.T) {
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
		Store: &StoreConfig{
			Type:      "",
			ProjectID: "test-project",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for store missing type")
	}
}

func TestValidate_StoreConfigUnsupportedType(t *testing.T) {
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
		Store: &StoreConfig{
			Type:      "mongodb",
			ProjectID: "test-project",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for unsupported store type")
	}
}

func TestValidate_StoreConfigMissingProjectID(t *testing.T) {
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
		Store: &StoreConfig{
			Type:      "firestore",
			ProjectID: "",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for store missing project_id")
	}
}

func TestStoreConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  StoreConfig
		wantErr bool
	}{
		{
			name: "valid firestore config",
			config: StoreConfig{
				Type:      "firestore",
				ProjectID: "my-project",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			config: StoreConfig{
				Type:      "",
				ProjectID: "my-project",
			},
			wantErr: true,
		},
		{
			name: "unsupported type",
			config: StoreConfig{
				Type:      "redis",
				ProjectID: "my-project",
			},
			wantErr: true,
		},
		{
			name: "missing project_id",
			config: StoreConfig{
				Type:      "firestore",
				ProjectID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Test loading configuration entirely from environment variables
	// This is the primary way to configure the application

	// Set required environment variables
	os.Setenv("NAMAZU_SOURCE_TYPE", "p2pquake")
	os.Setenv("NAMAZU_SOURCE_ENDPOINT", "wss://api-realtime-sandbox.p2pquake.net/v2/ws")
	os.Setenv("NAMAZU_STORE_PROJECT_ID", "test-project")
	os.Setenv("NAMAZU_API_ADDR", ":9898")
	defer os.Unsetenv("NAMAZU_SOURCE_TYPE")
	defer os.Unsetenv("NAMAZU_SOURCE_ENDPOINT")
	defer os.Unsetenv("NAMAZU_STORE_PROJECT_ID")
	defer os.Unsetenv("NAMAZU_API_ADDR")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v, want nil", err)
	}

	// Verify the config structure
	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Source.Type = %q, want %q", cfg.Source.Type, "p2pquake")
	}

	if cfg.Source.Endpoint != "wss://api-realtime-sandbox.p2pquake.net/v2/ws" {
		t.Errorf("Source.Endpoint = %q, want %q",
			cfg.Source.Endpoint,
			"wss://api-realtime-sandbox.p2pquake.net/v2/ws")
	}

	// subscriptions should be empty
	if len(cfg.Subscriptions) != 0 {
		t.Errorf("Subscriptions has %d items, want 0", len(cfg.Subscriptions))
	}

	// Store should be configured from env var
	if cfg.Store == nil {
		t.Fatal("Store should not be nil")
	}

	if cfg.Store.ProjectID != "test-project" {
		t.Errorf("Store.ProjectID = %q, want %q",
			cfg.Store.ProjectID,
			"test-project")
	}

	// API should be configured from env var
	if cfg.API == nil {
		t.Fatal("API should not be nil")
	}

	if cfg.API.Addr != ":9898" {
		t.Errorf("API.Addr = %q, want %q",
			cfg.API.Addr,
			":9898")
	}
}

func TestLoad_EmptyPathUsesEnv(t *testing.T) {
	// Test that Load("") delegates to LoadFromEnv

	os.Setenv("NAMAZU_SOURCE_ENDPOINT", "wss://test.example.com/ws")
	os.Setenv("NAMAZU_API_ADDR", ":9898") // Required: either subscriptions or API
	defer os.Unsetenv("NAMAZU_SOURCE_ENDPOINT")
	defer os.Unsetenv("NAMAZU_API_ADDR")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v, want nil", err)
	}

	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Source.Type = %q, want %q (default)", cfg.Source.Type, "p2pquake")
	}

	if cfg.Source.Endpoint != "wss://test.example.com/ws" {
		t.Errorf("Source.Endpoint = %q, want %q",
			cfg.Source.Endpoint,
			"wss://test.example.com/ws")
	}
}

func TestLoadFromEnv_DefaultSourceType(t *testing.T) {
	// Test that source.type defaults to "p2pquake" when not specified

	// Only set endpoint, not type
	os.Setenv("NAMAZU_SOURCE_ENDPOINT", "wss://test.example.com/ws")
	os.Setenv("NAMAZU_API_ADDR", ":9898") // Required: either subscriptions or API
	defer os.Unsetenv("NAMAZU_SOURCE_ENDPOINT")
	defer os.Unsetenv("NAMAZU_API_ADDR")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v, want nil", err)
	}

	if cfg.Source.Type != "p2pquake" {
		t.Errorf("Source.Type = %q, want %q (default)", cfg.Source.Type, "p2pquake")
	}
}

func TestLoad_WithAPIConfig(t *testing.T) {
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

api:
  addr: ":9898"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.API == nil {
		t.Fatal("API config should not be nil")
	}

	if cfg.API.Addr != ":9898" {
		t.Errorf("API.Addr = %q, want %q", cfg.API.Addr, ":9898")
	}
}

func TestLoad_APIAddrEnvironmentOverride(t *testing.T) {
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

api:
  addr: ":9898"
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	overrideAddr := ":9090"
	os.Setenv("NAMAZU_API_ADDR", overrideAddr)
	defer os.Unsetenv("NAMAZU_API_ADDR")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.API == nil {
		t.Fatal("API config should not be nil")
	}

	if cfg.API.Addr != overrideAddr {
		t.Errorf("API.Addr = %q, want %q (from env var)", cfg.API.Addr, overrideAddr)
	}
}

func TestValidate_NoSubscriptionsWithAPI(t *testing.T) {
	// When API is enabled, empty subscriptions should be allowed
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{},
		API: &APIConfig{
			Addr: ":9898",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for empty subscriptions with API enabled", err)
	}
}

func TestValidate_APIConfigMissingAddr(t *testing.T) {
	cfg := &Config{
		Source: SourceConfig{
			Type:     "p2pquake",
			Endpoint: "wss://example.com/ws",
		},
		Subscriptions: []SubscriptionConfig{},
		API: &APIConfig{
			Addr: "",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for API missing addr")
	}
}

func TestAPIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  APIConfig
		wantErr bool
	}{
		{
			name: "valid api config",
			config: APIConfig{
				Addr: ":9898",
			},
			wantErr: false,
		},
		{
			name: "valid api config with host",
			config: APIConfig{
				Addr: "0.0.0.0:9898",
			},
			wantErr: false,
		},
		{
			name: "missing addr",
			config: APIConfig{
				Addr: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("APIConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_WithAuthConfig(t *testing.T) {
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

auth:
  enabled: true
  project_id: test-firebase-project
  credentials: /path/to/credentials.json
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Auth == nil {
		t.Fatal("Auth config should not be nil")
	}

	if !cfg.Auth.Enabled {
		t.Error("Auth.Enabled = false, want true")
	}

	if cfg.Auth.ProjectID != "test-firebase-project" {
		t.Errorf("Auth.ProjectID = %q, want %q", cfg.Auth.ProjectID, "test-firebase-project")
	}

	if cfg.Auth.Credentials != "/path/to/credentials.json" {
		t.Errorf("Auth.Credentials = %q, want %q", cfg.Auth.Credentials, "/path/to/credentials.json")
	}
}

func TestLoad_WithAuthDisabled(t *testing.T) {
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

auth:
  enabled: false
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Auth == nil {
		t.Fatal("Auth config should not be nil")
	}

	if cfg.Auth.Enabled {
		t.Error("Auth.Enabled = true, want false")
	}
}

func TestLoad_AuthEnvironmentOverrides(t *testing.T) {
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

	// Set environment variables
	os.Setenv("NAMAZU_AUTH_ENABLED", "true")
	os.Setenv("NAMAZU_AUTH_PROJECT_ID", "env-firebase-project")
	os.Setenv("NAMAZU_AUTH_CREDENTIALS", "/env/path/credentials.json")
	os.Setenv("NAMAZU_AUTH_TENANT_ID", "test-tenant-123")
	defer os.Unsetenv("NAMAZU_AUTH_ENABLED")
	defer os.Unsetenv("NAMAZU_AUTH_PROJECT_ID")
	defer os.Unsetenv("NAMAZU_AUTH_CREDENTIALS")
	defer os.Unsetenv("NAMAZU_AUTH_TENANT_ID")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.Auth == nil {
		t.Fatal("Auth config should be created by env var")
	}

	if !cfg.Auth.Enabled {
		t.Error("Auth.Enabled = false, want true (from env var)")
	}

	if cfg.Auth.ProjectID != "env-firebase-project" {
		t.Errorf("Auth.ProjectID = %q, want %q (from env var)", cfg.Auth.ProjectID, "env-firebase-project")
	}

	if cfg.Auth.Credentials != "/env/path/credentials.json" {
		t.Errorf("Auth.Credentials = %q, want %q (from env var)", cfg.Auth.Credentials, "/env/path/credentials.json")
	}

	if cfg.Auth.TenantID != "test-tenant-123" {
		t.Errorf("Auth.TenantID = %q, want %q (from env var)", cfg.Auth.TenantID, "test-tenant-123")
	}
}

func TestValidate_AuthConfigValid(t *testing.T) {
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
		Auth: &AuthConfig{
			Enabled:   true,
			ProjectID: "test-project",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_AuthConfigDisabledNoProjectID(t *testing.T) {
	// When auth is disabled, project_id is not required
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
		Auth: &AuthConfig{
			Enabled: false,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil for disabled auth without project_id", err)
	}
}

func TestValidate_AuthConfigMissingProjectID(t *testing.T) {
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
		Auth: &AuthConfig{
			Enabled:   true,
			ProjectID: "",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for auth missing project_id when enabled")
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AuthConfig
		wantErr bool
	}{
		{
			name: "valid auth config with credentials",
			config: AuthConfig{
				Enabled:     true,
				ProjectID:   "my-project",
				Credentials: "/path/to/creds.json",
			},
			wantErr: false,
		},
		{
			name: "valid auth config without credentials",
			config: AuthConfig{
				Enabled:   true,
				ProjectID: "my-project",
			},
			wantErr: false,
		},
		{
			name: "disabled auth without project_id",
			config: AuthConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "enabled auth missing project_id",
			config: AuthConfig{
				Enabled:   true,
				ProjectID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityConfig_GetCORSAllowedOrigins(t *testing.T) {
	tests := []struct {
		name     string
		config   *SecurityConfig
		expected []string
	}{
		{
			name:     "nil config returns nil",
			config:   nil,
			expected: nil,
		},
		{
			name:     "empty string returns nil",
			config:   &SecurityConfig{CORSAllowedOrigins: ""},
			expected: nil,
		},
		{
			name:     "single origin",
			config:   &SecurityConfig{CORSAllowedOrigins: "https://example.com"},
			expected: []string{"https://example.com"},
		},
		{
			name:     "multiple origins",
			config:   &SecurityConfig{CORSAllowedOrigins: "https://example.com,https://app.example.com"},
			expected: []string{"https://example.com", "https://app.example.com"},
		},
		{
			name:     "origins with spaces",
			config:   &SecurityConfig{CORSAllowedOrigins: "https://example.com, https://app.example.com"},
			expected: []string{"https://example.com", "https://app.example.com"},
		},
		{
			name:     "wildcard origin",
			config:   &SecurityConfig{CORSAllowedOrigins: "*"},
			expected: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetCORSAllowedOrigins()
			if len(result) != len(tt.expected) {
				t.Fatalf("GetCORSAllowedOrigins() returned %d items, expected %d: %v", len(result), len(tt.expected), result)
			}
			for i, origin := range result {
				if origin != tt.expected[i] {
					t.Errorf("GetCORSAllowedOrigins()[%d] = %q, expected %q", i, origin, tt.expected[i])
				}
			}
		})
	}
}

func TestSecurityConfig_EnvironmentOverrides(t *testing.T) {
	// Save original environment
	origAllowLocal := os.Getenv("NAMAZU_ALLOW_LOCAL_WEBHOOKS")
	origCORSOrigins := os.Getenv("NAMAZU_CORS_ALLOWED_ORIGINS")
	origRateLimitEnabled := os.Getenv("NAMAZU_RATE_LIMIT_ENABLED")
	origRateLimitRPM := os.Getenv("NAMAZU_RATE_LIMIT_RPM")
	origRateLimitSub := os.Getenv("NAMAZU_RATE_LIMIT_SUBSCRIPTION")

	defer func() {
		os.Setenv("NAMAZU_ALLOW_LOCAL_WEBHOOKS", origAllowLocal)
		os.Setenv("NAMAZU_CORS_ALLOWED_ORIGINS", origCORSOrigins)
		os.Setenv("NAMAZU_RATE_LIMIT_ENABLED", origRateLimitEnabled)
		os.Setenv("NAMAZU_RATE_LIMIT_RPM", origRateLimitRPM)
		os.Setenv("NAMAZU_RATE_LIMIT_SUBSCRIPTION", origRateLimitSub)
	}()

	t.Run("applies security environment variables", func(t *testing.T) {
		os.Setenv("NAMAZU_SOURCE_TYPE", "p2pquake")
		os.Setenv("NAMAZU_SOURCE_ENDPOINT", "wss://test.example.com/ws")
		os.Setenv("NAMAZU_API_ADDR", ":9898")
		os.Setenv("NAMAZU_ALLOW_LOCAL_WEBHOOKS", "true")
		os.Setenv("NAMAZU_CORS_ALLOWED_ORIGINS", "https://example.com,https://app.example.com")
		os.Setenv("NAMAZU_RATE_LIMIT_ENABLED", "true")
		os.Setenv("NAMAZU_RATE_LIMIT_RPM", "200")
		os.Setenv("NAMAZU_RATE_LIMIT_SUBSCRIPTION", "20")

		cfg, err := LoadFromEnv()
		if err != nil {
			t.Fatalf("LoadFromEnv() error = %v", err)
		}

		if cfg.Security == nil {
			t.Fatal("Security config should not be nil")
		}

		if !cfg.Security.AllowLocalWebhooks {
			t.Error("AllowLocalWebhooks should be true")
		}

		if cfg.Security.CORSAllowedOrigins != "https://example.com,https://app.example.com" {
			t.Errorf("CORSAllowedOrigins = %q, expected %q", cfg.Security.CORSAllowedOrigins, "https://example.com,https://app.example.com")
		}

		if !cfg.Security.RateLimitEnabled {
			t.Error("RateLimitEnabled should be true")
		}

		if cfg.Security.RateLimitRequestsPerMinute != 200 {
			t.Errorf("RateLimitRequestsPerMinute = %d, expected %d", cfg.Security.RateLimitRequestsPerMinute, 200)
		}

		if cfg.Security.RateLimitSubscriptionCreation != 20 {
			t.Errorf("RateLimitSubscriptionCreation = %d, expected %d", cfg.Security.RateLimitSubscriptionCreation, 20)
		}
	})
}
