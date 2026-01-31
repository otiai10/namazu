package billing

import (
	"context"
	"fmt"

	"github.com/ayanel/namazu/internal/user"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/customer"
)

// GetOrCreateCustomer retrieves an existing Stripe customer ID or creates a new one
//
// Parameters:
//   - ctx: Context for cancellation control
//   - u: User to get or create Stripe customer for
//
// Returns:
//   - Stripe customer ID
//   - Error if Stripe API call fails
func (c *Client) GetOrCreateCustomer(ctx context.Context, u *user.User) (string, error) {
	// If user already has a Stripe customer ID, return it
	if u.StripeCustomerID != "" {
		return u.StripeCustomerID, nil
	}

	// Create new Stripe customer
	params := buildCustomerParams(u)
	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	return cust.ID, nil
}

// buildCustomerParams creates Stripe customer parameters from a user
func buildCustomerParams(u *user.User) *stripe.CustomerParams {
	return &stripe.CustomerParams{
		Email: stripe.String(u.Email),
		Name:  stripe.String(u.DisplayName),
		Metadata: map[string]string{
			"user_id": u.ID,
			"uid":     u.UID,
		},
	}
}
