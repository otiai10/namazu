package billing

import (
	"github.com/stripe/stripe-go/v78"
)

// Client wraps Stripe API operations
type Client struct {
	secretKey string
}

// NewClient creates a new Stripe billing client
//
// Parameters:
//   - secretKey: Stripe API secret key (sk_test_xxx or sk_live_xxx)
//
// Returns:
//   - Client instance configured with the secret key
func NewClient(secretKey string) *Client {
	// Set the global API key for the stripe-go library
	stripe.Key = secretKey

	return &Client{
		secretKey: secretKey,
	}
}

// GetSecretKey returns the configured secret key
// Useful for testing and debugging
func (c *Client) GetSecretKey() string {
	return c.secretKey
}
