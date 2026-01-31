package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ayanel/namazu/internal/auth"
	"github.com/ayanel/namazu/internal/quota"
	"github.com/ayanel/namazu/internal/store"
	"github.com/ayanel/namazu/internal/subscription"
	"github.com/ayanel/namazu/internal/user"
)

// SubscriptionRequest represents the request body for creating/updating a subscription
type SubscriptionRequest struct {
	Name     string                      `json:"name"`
	Delivery subscription.DeliveryConfig `json:"delivery"`
	Filter   *subscription.FilterConfig  `json:"filter,omitempty"`
}

// SubscriptionResponse represents the response for subscription endpoints
type SubscriptionResponse struct {
	ID       string                      `json:"id"`
	Name     string                      `json:"name"`
	Delivery subscription.DeliveryConfig `json:"delivery"`
	Filter   *subscription.FilterConfig  `json:"filter,omitempty"`
}

// EventResponse represents the response for event endpoints
type EventResponse struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Source        string    `json:"source"`
	Severity      int       `json:"severity"`
	AffectedAreas []string  `json:"affectedAreas"`
	OccurredAt    time.Time `json:"occurredAt"`
	ReceivedAt    time.Time `json:"receivedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// URLValidator validates webhook URLs for security
type URLValidator interface {
	ValidateWebhookURL(url string) error
}

// Handler contains the HTTP handlers for the API
type Handler struct {
	subscriptionRepo subscription.Repository
	eventRepo        store.EventRepository
	userRepo         user.Repository
	quotaChecker     quota.QuotaChecker
	urlValidator     URLValidator
}

// NewHandler creates a new Handler instance (backward compatible, no quota checking)
func NewHandler(subRepo subscription.Repository, eventRepo store.EventRepository) *Handler {
	return &Handler{
		subscriptionRepo: subRepo,
		eventRepo:        eventRepo,
		userRepo:         nil,
		quotaChecker:     nil,
		urlValidator:     nil,
	}
}

// NewHandlerWithQuota creates a new Handler with quota checking support
func NewHandlerWithQuota(subRepo subscription.Repository, eventRepo store.EventRepository, userRepo user.Repository, quotaChecker quota.QuotaChecker) *Handler {
	return &Handler{
		subscriptionRepo: subRepo,
		eventRepo:        eventRepo,
		userRepo:         userRepo,
		quotaChecker:     quotaChecker,
		urlValidator:     nil,
	}
}

// NewHandlerWithSecurity creates a new Handler with full security support
func NewHandlerWithSecurity(subRepo subscription.Repository, eventRepo store.EventRepository, userRepo user.Repository, quotaChecker quota.QuotaChecker, urlValidator URLValidator) *Handler {
	return &Handler{
		subscriptionRepo: subRepo,
		eventRepo:        eventRepo,
		userRepo:         userRepo,
		quotaChecker:     quotaChecker,
		urlValidator:     urlValidator,
	}
}

// SetURLValidator sets the URL validator for the handler
func (h *Handler) SetURLValidator(v URLValidator) {
	h.urlValidator = v
}

// CreateSubscription handles POST /api/subscriptions
func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Delivery.Type == "" || req.Delivery.URL == "" {
		writeError(w, "delivery type and URL are required", http.StatusBadRequest)
		return
	}

	// Validate webhook URL for security (SSRF prevention, HTTPS enforcement)
	if req.Delivery.Type == "webhook" && h.urlValidator != nil {
		if err := h.urlValidator.ValidateWebhookURL(req.Delivery.URL); err != nil {
			writeError(w, "invalid webhook URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	sub := subscription.Subscription{
		Name:     req.Name,
		Delivery: copyDeliveryConfig(req.Delivery),
		Filter:   copyFilterConfig(req.Filter),
	}

	// Set UserID from claims if authenticated and check quota
	if claims, ok := auth.GetClaims(r.Context()); ok {
		sub.UserID = claims.UID

		// Check quota if quota checker is configured
		if h.quotaChecker != nil {
			plan := h.getUserPlan(r.Context(), claims.UID)
			canCreate, err := h.quotaChecker.CanCreateSubscription(r.Context(), claims.UID, plan)
			if err != nil {
				writeError(w, "failed to check quota", http.StatusInternalServerError)
				return
			}
			if !canCreate {
				writeError(w, "Subscription limit reached for your plan", http.StatusForbidden)
				return
			}
		}
	}

	id, err := h.subscriptionRepo.Create(r.Context(), sub)
	if err != nil {
		writeError(w, "failed to create subscription", http.StatusInternalServerError)
		return
	}

	response := SubscriptionResponse{
		ID:       id,
		Name:     sub.Name,
		Delivery: sub.Delivery,
		Filter:   sub.Filter,
	}

	writeJSON(w, response, http.StatusCreated)
}

// ListSubscriptions handles GET /api/subscriptions
// When authenticated, returns only user's own subscriptions + legacy (ownerless) subscriptions
func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var subs []subscription.Subscription
	var err error

	// If authenticated, filter by user ID
	if claims, ok := auth.GetClaims(r.Context()); ok {
		subs, err = h.subscriptionRepo.ListByUserID(r.Context(), claims.UID)
	} else {
		subs, err = h.subscriptionRepo.List(r.Context())
	}

	if err != nil {
		writeError(w, "failed to list subscriptions", http.StatusInternalServerError)
		return
	}

	responses := make([]SubscriptionResponse, 0, len(subs))
	for _, sub := range subs {
		responses = append(responses, subscriptionToResponse(sub))
	}

	writeJSON(w, responses, http.StatusOK)
}

// GetSubscription handles GET /api/subscriptions/{id}
func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/subscriptions/")
	if id == "" {
		writeError(w, "subscription ID is required", http.StatusBadRequest)
		return
	}

	sub, forbidden, err := h.checkOwnership(r.Context(), id)
	if err != nil {
		writeError(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}
	if sub == nil {
		writeError(w, "subscription not found", http.StatusNotFound)
		return
	}
	if forbidden {
		writeError(w, "forbidden", http.StatusForbidden)
		return
	}

	writeJSON(w, subscriptionToResponse(*sub), http.StatusOK)
}

// UpdateSubscription handles PUT /api/subscriptions/{id}
func (h *Handler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/subscriptions/")
	if id == "" {
		writeError(w, "subscription ID is required", http.StatusBadRequest)
		return
	}

	var req SubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Delivery.Type == "" || req.Delivery.URL == "" {
		writeError(w, "delivery type and URL are required", http.StatusBadRequest)
		return
	}

	// Validate webhook URL for security (SSRF prevention, HTTPS enforcement)
	if req.Delivery.Type == "webhook" && h.urlValidator != nil {
		if err := h.urlValidator.ValidateWebhookURL(req.Delivery.URL); err != nil {
			writeError(w, "invalid webhook URL: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Check if subscription exists and verify ownership
	existing, forbidden, err := h.checkOwnership(r.Context(), id)
	if err != nil {
		writeError(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		writeError(w, "subscription not found", http.StatusNotFound)
		return
	}
	if forbidden {
		writeError(w, "forbidden", http.StatusForbidden)
		return
	}

	sub := subscription.Subscription{
		ID:       id,
		UserID:   existing.UserID, // Preserve the original owner
		Name:     req.Name,
		Delivery: copyDeliveryConfig(req.Delivery),
		Filter:   copyFilterConfig(req.Filter),
	}

	if err := h.subscriptionRepo.Update(r.Context(), id, sub); err != nil {
		writeError(w, "failed to update subscription", http.StatusInternalServerError)
		return
	}

	writeJSON(w, subscriptionToResponse(sub), http.StatusOK)
}

// DeleteSubscription handles DELETE /api/subscriptions/{id}
func (h *Handler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/subscriptions/")
	if id == "" {
		writeError(w, "subscription ID is required", http.StatusBadRequest)
		return
	}

	// Check if subscription exists and verify ownership
	existing, forbidden, err := h.checkOwnership(r.Context(), id)
	if err != nil {
		writeError(w, "failed to get subscription", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		writeError(w, "subscription not found", http.StatusNotFound)
		return
	}
	if forbidden {
		writeError(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.subscriptionRepo.Delete(r.Context(), id); err != nil {
		writeError(w, "failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListEvents handles GET /api/events
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var startAfter *time.Time
	if startAfterStr := r.URL.Query().Get("start_after"); startAfterStr != "" {
		if t, err := time.Parse(time.RFC3339, startAfterStr); err == nil {
			startAfter = &t
		}
	}

	events, err := h.eventRepo.List(r.Context(), limit, startAfter)
	if err != nil {
		writeError(w, "failed to list events", http.StatusInternalServerError)
		return
	}

	responses := make([]EventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, eventToResponse(event))
	}

	writeJSON(w, responses, http.StatusOK)
}

// Helper functions

// checkOwnership verifies that the current user owns the subscription.
// Returns:
//   - subscription: The subscription if found (nil if not found)
//   - forbidden: true if user is not the owner
//   - err: database error
//
// Rules:
//   - If no auth claims in context: allow access (backward compatibility during transition)
//   - If subscription has no owner (UserID == ""): allow access (legacy data)
//   - If subscription owner matches current user: allow access
//   - Otherwise: forbidden
func (h *Handler) checkOwnership(ctx context.Context, subID string) (*subscription.Subscription, bool, error) {
	sub, err := h.subscriptionRepo.Get(ctx, subID)
	if err != nil {
		return nil, false, err
	}
	if sub == nil {
		return nil, false, nil // not found
	}

	claims, ok := auth.GetClaims(ctx)
	if !ok {
		// No auth context, allow access (backward compatibility)
		return sub, false, nil
	}

	// Legacy subscription with no owner
	if sub.UserID == "" {
		return sub, false, nil
	}

	// Check ownership
	if sub.UserID != claims.UID {
		return sub, true, nil // forbidden
	}

	return sub, false, nil
}

func extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimPrefix(path, prefix)
}

func subscriptionToResponse(sub subscription.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:       sub.ID,
		Name:     sub.Name,
		Delivery: sub.Delivery,
		Filter:   sub.Filter,
	}
}

func eventToResponse(event store.EventRecord) EventResponse {
	return EventResponse{
		ID:            event.ID,
		Type:          event.Type,
		Source:        event.Source,
		Severity:      event.Severity,
		AffectedAreas: event.AffectedAreas,
		OccurredAt:    event.OccurredAt,
		ReceivedAt:    event.ReceivedAt,
		CreatedAt:     event.CreatedAt,
	}
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Already wrote headers, can only log
		return
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// copyDeliveryConfig creates an immutable copy of DeliveryConfig
func copyDeliveryConfig(d subscription.DeliveryConfig) subscription.DeliveryConfig {
	return subscription.DeliveryConfig{
		Type:   d.Type,
		URL:    d.URL,
		Secret: d.Secret,
	}
}

// copyFilterConfig creates an immutable copy of FilterConfig
func copyFilterConfig(f *subscription.FilterConfig) *subscription.FilterConfig {
	if f == nil {
		return nil
	}
	prefectures := make([]string, len(f.Prefectures))
	copy(prefectures, f.Prefectures)
	return &subscription.FilterConfig{
		MinScale:    f.MinScale,
		Prefectures: prefectures,
	}
}

// getUserPlan retrieves the user's plan from the user repository
// Returns "free" as default if user is not found or no user repo is configured
func (h *Handler) getUserPlan(ctx context.Context, uid string) string {
	if h.userRepo == nil {
		return user.PlanFree
	}

	u, err := h.userRepo.GetByUID(ctx, uid)
	if err != nil || u == nil {
		return user.PlanFree
	}

	if u.Plan == "" {
		return user.PlanFree
	}

	return u.Plan
}
