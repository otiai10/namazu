package quota

import (
	"github.com/ayanel/namazu/internal/user"
)

// PlanLimits defines limits per plan
type PlanLimits struct {
	MaxSubscriptions int
}

var (
	// FreePlanLimits defines limits for free plan users
	FreePlanLimits = PlanLimits{MaxSubscriptions: 3}

	// ProPlanLimits defines limits for pro plan users
	ProPlanLimits = PlanLimits{MaxSubscriptions: 50}
)

// GetLimits returns limits for a plan
// Unknown or empty plans default to free plan limits
func GetLimits(plan string) PlanLimits {
	switch plan {
	case user.PlanPro:
		return ProPlanLimits
	case user.PlanFree:
		return FreePlanLimits
	default:
		// Unknown or empty plan defaults to free
		return FreePlanLimits
	}
}
