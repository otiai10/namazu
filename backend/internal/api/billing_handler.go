package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/otiai10/namazu/internal/auth"
	"github.com/otiai10/namazu/internal/billing"
	"github.com/otiai10/namazu/internal/config"
	"github.com/otiai10/namazu/internal/user"
	"github.com/stripe/stripe-go/v78"
)

// BillingHandler handles billing-related API endpoints
type BillingHandler struct {
	client   *billing.Client
	userRepo BillingUserRepository
	config   *config.BillingConfig
}

// BillingUserRepository defines the user repository interface needed by billing
type BillingUserRepository interface {
	GetByUID(ctx context.Context, uid string) (*user.User, error)
	Get(ctx context.Context, id string) (*user.User, error)
	Update(ctx context.Context, id string, u user.User) error
	GetByStripeCustomerID(ctx context.Context, customerID string) (*user.User, error)
}

// BillingStatusResponse represents the response for billing status
type BillingStatusResponse struct {
	Plan                  string     `json:"plan"`
	HasActiveSubscription bool       `json:"hasActiveSubscription"`
	SubscriptionStatus    string     `json:"subscriptionStatus,omitempty"`
	SubscriptionEndsAt    *time.Time `json:"subscriptionEndsAt,omitempty"`
	StripeCustomerID      string     `json:"stripeCustomerId,omitempty"`
}

// CheckoutSessionResponse represents the response for creating checkout session
type CheckoutSessionResponse struct {
	SessionID  string `json:"sessionId"`
	SessionURL string `json:"sessionUrl"`
}

// PortalSessionResponse represents the response for creating portal session
type PortalSessionResponse struct {
	URL string `json:"url"`
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(client *billing.Client, userRepo BillingUserRepository, cfg *config.BillingConfig) *BillingHandler {
	return &BillingHandler{
		client:   client,
		userRepo: userRepo,
		config:   cfg,
	}
}

// GetStatus handles GET /api/billing/status
// Returns the current user's billing/plan status
func (h *BillingHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustGetClaims(r.Context())

	u, err := h.userRepo.GetByUID(r.Context(), claims.UID)
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if u == nil {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}

	response := BillingStatusResponse{
		Plan:                  u.Plan,
		HasActiveSubscription: u.SubscriptionStatus == user.SubscriptionStatusActive,
		SubscriptionStatus:    u.SubscriptionStatus,
		StripeCustomerID:      u.StripeCustomerID,
	}

	if !u.SubscriptionEndsAt.IsZero() {
		response.SubscriptionEndsAt = &u.SubscriptionEndsAt
	}

	writeJSON(w, response, http.StatusOK)
}

// CreateCheckoutSession handles POST /api/billing/create-checkout-session
// Creates a Stripe Checkout session for Pro upgrade
func (h *BillingHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustGetClaims(r.Context())

	u, err := h.userRepo.GetByUID(r.Context(), claims.UID)
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if u == nil {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}

	// Check if user already has an active subscription
	if u.SubscriptionStatus == user.SubscriptionStatusActive {
		writeError(w, "user already has an active subscription", http.StatusBadRequest)
		return
	}

	// Get or create Stripe customer
	customerID, err := h.client.GetOrCreateCustomer(r.Context(), u)
	if err != nil {
		writeError(w, "failed to create Stripe customer", http.StatusInternalServerError)
		return
	}

	// Save customer ID to user if newly created
	if u.StripeCustomerID == "" {
		updatedUser := u.Copy()
		updatedUser.StripeCustomerID = customerID
		updatedUser.UpdatedAt = time.Now().UTC()
		if err := h.userRepo.Update(r.Context(), u.ID, updatedUser); err != nil {
			writeError(w, "failed to update user", http.StatusInternalServerError)
			return
		}
	}

	// Create checkout session
	session, err := h.client.CreateCheckoutSession(
		r.Context(),
		customerID,
		h.config.PriceID,
		h.config.SuccessURL,
		h.config.CancelURL,
	)
	if err != nil {
		writeError(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	response := CheckoutSessionResponse{
		SessionID:  session.ID,
		SessionURL: session.URL,
	}

	writeJSON(w, response, http.StatusOK)
}

// GetPortalSession handles GET /api/billing/portal-session
// Returns a Stripe Customer Portal URL
func (h *BillingHandler) GetPortalSession(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustGetClaims(r.Context())

	u, err := h.userRepo.GetByUID(r.Context(), claims.UID)
	if err != nil {
		writeError(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if u == nil {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}

	// Check if user has a Stripe customer ID
	if u.StripeCustomerID == "" {
		writeError(w, "user has no Stripe customer ID", http.StatusBadRequest)
		return
	}

	// Create portal session
	returnURL := r.URL.Query().Get("return_url")
	if returnURL == "" {
		returnURL = h.config.SuccessURL
	}

	session, err := h.client.CreatePortalSession(r.Context(), u.StripeCustomerID, returnURL)
	if err != nil {
		writeError(w, "failed to create portal session", http.StatusInternalServerError)
		return
	}

	response := PortalSessionResponse{
		URL: session.URL,
	}

	writeJSON(w, response, http.StatusOK)
}

// StripeWebhook handles POST /api/webhooks/stripe
// Processes Stripe webhook events
func (h *BillingHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Get signature header
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		writeError(w, "missing Stripe-Signature header", http.StatusBadRequest)
		return
	}

	// Verify signature
	event, err := billing.VerifyWebhookSignature(body, signature, h.config.WebhookSecret)
	if err != nil {
		writeError(w, "invalid webhook signature", http.StatusBadRequest)
		return
	}

	// Handle different event types
	switch event.Type {
	case billing.EventCheckoutSessionCompleted:
		h.handleCheckoutSessionCompleted(w, event)
	case billing.EventSubscriptionUpdated:
		h.handleSubscriptionUpdated(w, event)
	case billing.EventSubscriptionDeleted:
		h.handleSubscriptionDeleted(w, event)
	default:
		// Acknowledge unhandled events
		w.WriteHeader(http.StatusOK)
	}
}

// handleCheckoutSessionCompleted processes checkout.session.completed events
func (h *BillingHandler) handleCheckoutSessionCompleted(w http.ResponseWriter, event stripe.Event) {
	ctx := context.Background()

	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		writeError(w, "failed to parse checkout session", http.StatusBadRequest)
		return
	}

	customerID, subscriptionID := billing.ParseCheckoutSessionCompleted(&session)
	if customerID == "" || subscriptionID == "" {
		writeError(w, "missing customer or subscription ID", http.StatusBadRequest)
		return
	}

	// Find user by Stripe customer ID
	u, err := h.userRepo.GetByStripeCustomerID(ctx, customerID)
	if err != nil || u == nil {
		writeError(w, "user not found for customer", http.StatusNotFound)
		return
	}

	// Update user to Pro plan
	updatedUser := u.Copy()
	updatedUser.Plan = user.PlanPro
	updatedUser.SubscriptionID = subscriptionID
	updatedUser.SubscriptionStatus = user.SubscriptionStatusActive
	updatedUser.UpdatedAt = time.Now().UTC()

	if err := h.userRepo.Update(ctx, u.ID, updatedUser); err != nil {
		writeError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionUpdated processes customer.subscription.updated events
func (h *BillingHandler) handleSubscriptionUpdated(w http.ResponseWriter, event stripe.Event) {
	ctx := context.Background()

	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		writeError(w, "failed to parse subscription", http.StatusBadRequest)
		return
	}

	info := billing.ParseSubscriptionUpdate(&sub)
	if info.CustomerID == "" {
		writeError(w, "missing customer ID", http.StatusBadRequest)
		return
	}

	// Find user by Stripe customer ID
	u, err := h.userRepo.GetByStripeCustomerID(ctx, info.CustomerID)
	if err != nil || u == nil {
		writeError(w, "user not found for customer", http.StatusNotFound)
		return
	}

	// Update subscription status
	updatedUser := u.Copy()
	updatedUser.SubscriptionStatus = info.Status
	updatedUser.SubscriptionEndsAt = info.PeriodEnd
	updatedUser.UpdatedAt = time.Now().UTC()

	// Update plan based on status
	if info.Status == user.SubscriptionStatusActive {
		updatedUser.Plan = user.PlanPro
	} else if info.Status == user.SubscriptionStatusCanceled {
		updatedUser.Plan = user.PlanFree
	}

	if err := h.userRepo.Update(ctx, u.ID, updatedUser); err != nil {
		writeError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleSubscriptionDeleted processes customer.subscription.deleted events
func (h *BillingHandler) handleSubscriptionDeleted(w http.ResponseWriter, event stripe.Event) {
	ctx := context.Background()

	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		writeError(w, "failed to parse subscription", http.StatusBadRequest)
		return
	}

	info := billing.ParseSubscriptionUpdate(&sub)
	if info.CustomerID == "" {
		writeError(w, "missing customer ID", http.StatusBadRequest)
		return
	}

	// Find user by Stripe customer ID
	u, err := h.userRepo.GetByStripeCustomerID(ctx, info.CustomerID)
	if err != nil || u == nil {
		writeError(w, "user not found for customer", http.StatusNotFound)
		return
	}

	// Downgrade to free plan
	updatedUser := u.Copy()
	updatedUser.Plan = user.PlanFree
	updatedUser.SubscriptionID = ""
	updatedUser.SubscriptionStatus = user.SubscriptionStatusCanceled
	updatedUser.UpdatedAt = time.Now().UTC()

	if err := h.userRepo.Update(ctx, u.ID, updatedUser); err != nil {
		writeError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
