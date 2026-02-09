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
	Auth          *AuthConfig          `yaml:"auth,omitempty"`
	Billing       *BillingConfig       `yaml:"billing,omitempty"`
	Security      *SecurityConfig      `yaml:"security,omitempty"`
}

// AuthConfig represents the authentication configuration
type AuthConfig struct {
	Enabled     bool   `yaml:"enabled"`               // Whether auth is enabled
	ProjectID   string `yaml:"project_id"`            // Firebase project ID
	TenantID    string `yaml:"tenant_id,omitempty"`   // Optional: Identity Platform tenant ID
	Credentials string `yaml:"credentials,omitempty"` // Path to service account JSON (local dev)
}

// BillingConfig represents Stripe billing configuration
type BillingConfig struct {
	SecretKey     string `yaml:"secret_key"`     // STRIPE_SECRET_KEY
	WebhookSecret string `yaml:"webhook_secret"` // STRIPE_WEBHOOK_SECRET
	PriceID       string `yaml:"price_id"`       // STRIPE_PRICE_ID (Pro plan)
	SuccessURL    string `yaml:"success_url"`    // Redirect after checkout success
	CancelURL     string `yaml:"cancel_url"`     // Redirect after checkout cancel
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

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	// AllowLocalWebhooks allows HTTP URLs for localhost webhooks (development mode)
	AllowLocalWebhooks bool `yaml:"allow_local_webhooks"`

	// CORSAllowedOrigins is a comma-separated list of allowed CORS origins
	// Use "*" to allow all origins (not recommended for production)
	CORSAllowedOrigins string `yaml:"cors_allowed_origins"`

	// RateLimitEnabled enables rate limiting (default: true in production)
	RateLimitEnabled bool `yaml:"rate_limit_enabled"`

	// RateLimitRequestsPerMinute is the default rate limit per IP (default: 100)
	RateLimitRequestsPerMinute int `yaml:"rate_limit_requests_per_minute"`

	// RateLimitSubscriptionCreation is the rate limit for subscription creation per IP (default: 10)
	RateLimitSubscriptionCreation int `yaml:"rate_limit_subscription_creation"`
}

// GetCORSAllowedOrigins returns the list of allowed CORS origins
func (s *SecurityConfig) GetCORSAllowedOrigins() []string {
	if s == nil || s.CORSAllowedOrigins == "" {
		return nil
	}
	origins := make([]string, 0)
	for _, o := range splitAndTrim(s.CORSAllowedOrigins, ",") {
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// splitAndTrim splits a string by separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		trimmed := trimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// splitString splits a string by separator (simple implementation)
func splitString(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	for len(s) > 0 {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// LoadFromEnv creates configuration entirely from environment variables.
// This is the recommended way to configure the application.
//
// Required environment variables:
//   - NAMAZU_SOURCE_TYPE: data source type (default: "p2pquake")
//   - NAMAZU_SOURCE_ENDPOINT: WebSocket endpoint URL
//
// Optional environment variables:
//   - NAMAZU_STORE_PROJECT_ID: enables Firestore with this project
//   - NAMAZU_STORE_DATABASE: Firestore database name
//   - NAMAZU_STORE_CREDENTIALS: path to service account JSON (local dev only)
//   - NAMAZU_API_ADDR: enables REST API on this address (e.g., ":8080")
//   - NAMAZU_AUTH_ENABLED: "true" to enable authentication
//   - NAMAZU_AUTH_PROJECT_ID: Firebase project ID for auth
//   - NAMAZU_AUTH_CREDENTIALS: path to service account JSON (local dev only)
//   - NAMAZU_AUTH_TENANT_ID: Identity Platform tenant ID (optional)
//   - STRIPE_SECRET_KEY: Stripe API secret key
//   - STRIPE_WEBHOOK_SECRET: Stripe webhook signing secret
//   - STRIPE_PRICE_ID: Stripe price ID for Pro plan
//   - STRIPE_SUCCESS_URL: Redirect URL after successful checkout
//   - STRIPE_CANCEL_URL: Redirect URL after canceled checkout
//   - NAMAZU_ALLOW_LOCAL_WEBHOOKS: "true" to allow HTTP localhost webhooks (dev only)
//   - NAMAZU_CORS_ALLOWED_ORIGINS: comma-separated list of allowed CORS origins
//   - NAMAZU_RATE_LIMIT_ENABLED: "true" to enable rate limiting (default: true)
//   - NAMAZU_RATE_LIMIT_RPM: requests per minute per IP (default: 100)
//   - NAMAZU_RATE_LIMIT_SUBSCRIPTION: subscription creation rate limit per IP (default: 10)
func LoadFromEnv() (*Config, error) {
	cfg := &Config{}
	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Load reads configuration from the specified YAML file.
// If path is empty, loads configuration entirely from environment variables.
//
// Environment variables override file values:
//   - NAMAZU_SOURCE_TYPE overrides source.type
//   - NAMAZU_SOURCE_ENDPOINT overrides source.endpoint
//   - NAMAZU_STORE_PROJECT_ID overrides store.project_id
//   - NAMAZU_STORE_DATABASE overrides store.database
//   - NAMAZU_STORE_CREDENTIALS overrides store.credentials (for local dev only)
//   - NAMAZU_API_ADDR overrides api.addr
//   - NAMAZU_AUTH_* overrides auth settings
func Load(path string) (*Config, error) {
	// If no path provided, load entirely from environment
	if path == "" {
		return LoadFromEnv()
	}

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
	applyEnvOverrides(&cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(cfg *Config) {
	// Apply source overrides
	if sourceType := os.Getenv("NAMAZU_SOURCE_TYPE"); sourceType != "" {
		cfg.Source.Type = sourceType
	}
	if cfg.Source.Type == "" {
		cfg.Source.Type = "p2pquake" // default
	}

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

	// Apply auth overrides
	if authEnabled := os.Getenv("NAMAZU_AUTH_ENABLED"); authEnabled == "true" {
		if cfg.Auth == nil {
			cfg.Auth = &AuthConfig{}
		}
		cfg.Auth.Enabled = true
	}
	if authProjectID := os.Getenv("NAMAZU_AUTH_PROJECT_ID"); authProjectID != "" {
		if cfg.Auth == nil {
			cfg.Auth = &AuthConfig{}
		}
		cfg.Auth.ProjectID = authProjectID
	}
	if authCredentials := os.Getenv("NAMAZU_AUTH_CREDENTIALS"); authCredentials != "" {
		if cfg.Auth == nil {
			cfg.Auth = &AuthConfig{}
		}
		cfg.Auth.Credentials = authCredentials
	}
	if authTenantID := os.Getenv("NAMAZU_AUTH_TENANT_ID"); authTenantID != "" {
		if cfg.Auth == nil {
			cfg.Auth = &AuthConfig{}
		}
		cfg.Auth.TenantID = authTenantID
	}

	// Apply billing overrides
	if secretKey := os.Getenv("STRIPE_SECRET_KEY"); secretKey != "" {
		if cfg.Billing == nil {
			cfg.Billing = &BillingConfig{}
		}
		cfg.Billing.SecretKey = secretKey
	}
	if webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET"); webhookSecret != "" {
		if cfg.Billing == nil {
			cfg.Billing = &BillingConfig{}
		}
		cfg.Billing.WebhookSecret = webhookSecret
	}
	if priceID := os.Getenv("STRIPE_PRICE_ID"); priceID != "" {
		if cfg.Billing == nil {
			cfg.Billing = &BillingConfig{}
		}
		cfg.Billing.PriceID = priceID
	}
	if successURL := os.Getenv("STRIPE_SUCCESS_URL"); successURL != "" {
		if cfg.Billing == nil {
			cfg.Billing = &BillingConfig{}
		}
		cfg.Billing.SuccessURL = successURL
	}
	if cancelURL := os.Getenv("STRIPE_CANCEL_URL"); cancelURL != "" {
		if cfg.Billing == nil {
			cfg.Billing = &BillingConfig{}
		}
		cfg.Billing.CancelURL = cancelURL
	}

	// Apply security overrides
	if allowLocal := os.Getenv("NAMAZU_ALLOW_LOCAL_WEBHOOKS"); allowLocal == "true" {
		if cfg.Security == nil {
			cfg.Security = &SecurityConfig{}
		}
		cfg.Security.AllowLocalWebhooks = true
	}
	if corsOrigins := os.Getenv("NAMAZU_CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		if cfg.Security == nil {
			cfg.Security = &SecurityConfig{}
		}
		cfg.Security.CORSAllowedOrigins = corsOrigins
	}
	if rateLimitEnabled := os.Getenv("NAMAZU_RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		if cfg.Security == nil {
			cfg.Security = &SecurityConfig{}
		}
		cfg.Security.RateLimitEnabled = rateLimitEnabled == "true"
	}
	if rpm := os.Getenv("NAMAZU_RATE_LIMIT_RPM"); rpm != "" {
		if cfg.Security == nil {
			cfg.Security = &SecurityConfig{}
		}
		if v, err := parseIntEnv(rpm); err == nil {
			cfg.Security.RateLimitRequestsPerMinute = v
		}
	}
	if subLimit := os.Getenv("NAMAZU_RATE_LIMIT_SUBSCRIPTION"); subLimit != "" {
		if cfg.Security == nil {
			cfg.Security = &SecurityConfig{}
		}
		if v, err := parseIntEnv(subLimit); err == nil {
			cfg.Security.RateLimitSubscriptionCreation = v
		}
	}
}

// parseIntEnv parses an integer from a string, returning an error if invalid
func parseIntEnv(s string) (int, error) {
	var result int
	negative := false
	i := 0

	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}

	if s[0] == '-' {
		negative = true
		i = 1
	}

	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid character: %c", c)
		}
		result = result*10 + int(c-'0')
	}

	if negative {
		result = -result
	}

	return result, nil
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

	// Validate auth configuration if present
	if c.Auth != nil {
		if err := c.Auth.Validate(); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	// Validate billing configuration if present
	if c.Billing != nil {
		if err := c.Billing.Validate(); err != nil {
			return fmt.Errorf("billing: %w", err)
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

// Validate checks if the auth configuration is valid
func (a *AuthConfig) Validate() error {
	if !a.Enabled {
		return nil
	}
	if a.ProjectID == "" {
		return fmt.Errorf("project_id is required when auth is enabled")
	}
	return nil
}

// Validate checks if the billing configuration is valid
func (b *BillingConfig) Validate() error {
	if b.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	if b.WebhookSecret == "" {
		return fmt.Errorf("webhook_secret is required")
	}
	if b.PriceID == "" {
		return fmt.Errorf("price_id is required")
	}
	if b.SuccessURL == "" {
		return fmt.Errorf("success_url is required")
	}
	if b.CancelURL == "" {
		return fmt.Errorf("cancel_url is required")
	}
	return nil
}
