package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v78"
	portalsession "github.com/stripe/stripe-go/v78/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v78/checkout/session"
	"github.com/stripe/stripe-go/v78/subscription"
)

// CreateCheckoutSession creates a Stripe Checkout session for subscription
//
// Parameters:
//   - ctx: Context for cancellation control
//   - customerID: Stripe customer ID
//   - priceID: Stripe price ID for the subscription
//   - successURL: URL to redirect to after successful checkout
//   - cancelURL: URL to redirect to if checkout is canceled
//
// Returns:
//   - Stripe Checkout session
//   - Error if Stripe API call fails
func (c *Client) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*stripe.CheckoutSession, error) {
	params := buildCheckoutSessionParams(customerID, priceID, successURL, cancelURL)
	session, err := checkoutsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	}

	return session, nil
}

// CreatePortalSession creates a Stripe Customer Portal session
//
// Parameters:
//   - ctx: Context for cancellation control
//   - customerID: Stripe customer ID
//   - returnURL: URL to redirect to after portal session
//
// Returns:
//   - Stripe Billing Portal session
//   - Error if Stripe API call fails
func (c *Client) CreatePortalSession(ctx context.Context, customerID, returnURL string) (*stripe.BillingPortalSession, error) {
	params := buildPortalSessionParams(customerID, returnURL)
	session, err := portalsession.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create portal session: %w", err)
	}

	return session, nil
}

// GetSubscription retrieves a subscription by ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - subscriptionID: Stripe subscription ID
//
// Returns:
//   - Stripe subscription
//   - Error if Stripe API call fails or subscription not found
func (c *Client) GetSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

// buildCheckoutSessionParams creates Stripe Checkout session parameters
func buildCheckoutSessionParams(customerID, priceID, successURL, cancelURL string) *stripe.CheckoutSessionParams {
	return &stripe.CheckoutSessionParams{
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
	}
}

// buildPortalSessionParams creates Stripe Billing Portal session parameters
func buildPortalSessionParams(customerID, returnURL string) *stripe.BillingPortalSessionParams {
	return &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(returnURL),
	}
}
