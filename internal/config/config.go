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
// - NAMAZU_SOURCE_ENDPOINT overrides source.endpoint
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

	// Check at least one subscription exists
	if len(c.Subscriptions) == 0 {
		return fmt.Errorf("at least one subscription is required")
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

	return nil
}
