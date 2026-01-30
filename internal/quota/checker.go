package quota

import (
	"context"
	"fmt"

	"github.com/ayanel/namazu/internal/subscription"
)

// QuotaChecker checks if operations are allowed within quota
type QuotaChecker interface {
	// CanCreateSubscription checks if a user can create a new subscription
	// based on their plan limits and current subscription count
	CanCreateSubscription(ctx context.Context, userID string, plan string) (bool, error)
}

// Checker implements QuotaChecker using subscription repository
type Checker struct {
	subRepo subscription.Repository
}

// NewChecker creates a new Checker instance
func NewChecker(subRepo subscription.Repository) *Checker {
	return &Checker{
		subRepo: subRepo,
	}
}

// CanCreateSubscription checks if the user can create a new subscription
// Returns true if the user is under their plan's subscription limit
func (c *Checker) CanCreateSubscription(ctx context.Context, userID, plan string) (bool, error) {
	// Get current subscription count for user
	subs, err := c.subRepo.ListByUserID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user subscriptions: %w", err)
	}

	// Get limits for the plan
	limits := GetLimits(plan)

	// Check if under limit
	currentCount := len(subs)
	return currentCount < limits.MaxSubscriptions, nil
}
