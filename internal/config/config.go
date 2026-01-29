package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Source        SourceConfig         `yaml:"source"`
	Subscriptions []SubscriptionConfig `yaml:"subscriptions"`
	Store         *StoreConfig         `yaml:"store,omitempty"`
	API           *APIConfig           `yaml:"api,omitempty"`
}

// APIConfig represents the REST API server configuration
type APIConfig struct {
	Addr string `yaml:"addr"` // e.g., ":8080"
}

// StoreConfig represents the data store configuration
type StoreConfig struct {
	Type        string `yaml:"type"`                  // "firestore"
	ProjectID   string `yaml:"project_id"`            // GCP Project ID
	Database    string `yaml:"database,omitempty"`    // Firestore database name (default: "(default)")
	Credentials string `yaml:"credentials,omitempty"` // Path to service account JSON file
}

// SourceConfig represents the data source configuration
type SourceConfig struct {
	Type     string `yaml:"type"`     // "p2pquake"
	Endpoint string `yaml:"endpoint"` // WebSocket URL
}

// SubscriptionConfig represents a subscription with delivery and filter settings
type SubscriptionConfig struct {
	Name     string         `yaml:"name"`
	Delivery DeliveryConfig `yaml:"delivery"`
	Filter   *FilterConfig  `yaml:"filter,omitempty"`
}

// DeliveryConfig represents how to deliver notifications
type DeliveryConfig struct {
	Type   string `yaml:"type"`             // "webhook" | "email" | "slack"
	URL    string `yaml:"url,omitempty"`    // for webhook
	Secret string `yaml:"secret,omitempty"` // for webhook
}

// FilterConfig represents event filtering conditions (Phase 4)
type FilterConfig struct {
	MinScale    int      `yaml:"min_scale,omitempty"`
	Prefectures []string `yaml:"prefectures,omitempty"`
}

// Load reads configuration from the specified YAML file
// Environment variables override file values:
//   - NAMAZU_SOURCE_ENDPOINT overrides source.endpoint
//   - NAMAZU_STORE_PROJECT_ID overrides store.project_id
//   - NAMAZU_STORE_DATABASE overrides store.database
//   - NAMAZU_STORE_CREDENTIALS overrides store.credentials (for local dev only)
//   - NAMAZU_API_ADDR overrides api.addr
func Load(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply environment variable overrides
	if endpoint := os.Getenv("NAMAZU_SOURCE_ENDPOINT"); endpoint != "" {
		cfg.Source.Endpoint = endpoint
	}

	// Apply store overrides
	if projectID := os.Getenv("NAMAZU_STORE_PROJECT_ID"); projectID != "" {
		if cfg.Store == nil {
			cfg.Store = &StoreConfig{Type: "firestore"}
		}
		cfg.Store.ProjectID = projectID
	}
	if database := os.Getenv("NAMAZU_STORE_DATABASE"); database != "" {
		if cfg.Store == nil {
			cfg.Store = &StoreConfig{Type: "firestore"}
		}
		cfg.Store.Database = database
	}
	if credentials := os.Getenv("NAMAZU_STORE_CREDENTIALS"); credentials != "" {
		if cfg.Store == nil {
			cfg.Store = &StoreConfig{Type: "firestore"}
		}
		cfg.Store.Credentials = credentials
	}

	// Apply API address override
	if apiAddr := os.Getenv("NAMAZU_API_ADDR"); apiAddr != "" {
		if cfg.API == nil {
			cfg.API = &APIConfig{}
		}
		cfg.API.Addr = apiAddr
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check source type is supported
	if c.Source.Type == "" {
		return fmt.Errorf("source.type is required")
	}

	if c.Source.Type != "p2pquake" {
		return fmt.Errorf("unsupported source type: %q (supported: p2pquake)", c.Source.Type)
	}

	// Check endpoint is not empty
	if c.Source.Endpoint == "" {
		return fmt.Errorf("source.endpoint is required")
	}

	// Check at least one subscription exists (unless API is enabled for dynamic management)
	if len(c.Subscriptions) == 0 && c.API == nil {
		return fmt.Errorf("at least one subscription is required (or enable API for dynamic management)")
	}

	// Check each subscription configuration
	for i, sub := range c.Subscriptions {
		if sub.Name == "" {
			return fmt.Errorf("subscription[%d].name is required", i)
		}
		if sub.Delivery.Type == "" {
			return fmt.Errorf("subscription[%d].delivery.type is required", i)
		}
		// Validate based on delivery type
		switch sub.Delivery.Type {
		case "webhook":
			if sub.Delivery.URL == "" {
				return fmt.Errorf("subscription[%d].delivery.url is required for webhook", i)
			}
			if sub.Delivery.Secret == "" {
				return fmt.Errorf("subscription[%d].delivery.secret is required for webhook", i)
			}
		default:
			return fmt.Errorf("subscription[%d].delivery.type %q is not supported (supported: webhook)", i, sub.Delivery.Type)
		}
	}

	// Validate store configuration if present
	if c.Store != nil {
		if err := c.Store.Validate(); err != nil {
			return fmt.Errorf("store: %w", err)
		}
	}

	// Validate API configuration if present
	if c.API != nil {
		if err := c.API.Validate(); err != nil {
			return fmt.Errorf("api: %w", err)
		}
	}

	return nil
}

// Validate checks if the API configuration is valid
func (a *APIConfig) Validate() error {
	if a.Addr == "" {
		return fmt.Errorf("addr is required")
	}

	return nil
}

// Validate checks if the store configuration is valid
func (s *StoreConfig) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("type is required")
	}

	if s.Type != "firestore" {
		return fmt.Errorf("unsupported store type: %q (supported: firestore)", s.Type)
	}

	if s.ProjectID == "" {
		return fmt.Errorf("project_id is required for firestore")
	}

	return nil
}
