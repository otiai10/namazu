package quota

import (
	"testing"
)

func TestGetLimits_FreePlan(t *testing.T) {
	limits := GetLimits("free")

	if limits.MaxSubscriptions != 1 {
		t.Errorf("expected MaxSubscriptions = 1, got %d", limits.MaxSubscriptions)
	}
}

func TestGetLimits_ProPlan(t *testing.T) {
	limits := GetLimits("pro")

	if limits.MaxSubscriptions != 12 {
		t.Errorf("expected MaxSubscriptions = 12, got %d", limits.MaxSubscriptions)
	}
}

func TestGetLimits_UnknownPlan_DefaultsToFree(t *testing.T) {
	limits := GetLimits("unknown")

	if limits.MaxSubscriptions != 1 {
		t.Errorf("expected MaxSubscriptions = 1 for unknown plan, got %d", limits.MaxSubscriptions)
	}
}

func TestGetLimits_EmptyPlan_DefaultsToFree(t *testing.T) {
	limits := GetLimits("")

	if limits.MaxSubscriptions != 1 {
		t.Errorf("expected MaxSubscriptions = 1 for empty plan, got %d", limits.MaxSubscriptions)
	}
}

func TestPlanLimitsConstants(t *testing.T) {
	if FreePlanLimits.MaxSubscriptions != 1 {
		t.Errorf("FreePlanLimits.MaxSubscriptions should be 1, got %d", FreePlanLimits.MaxSubscriptions)
	}

	if ProPlanLimits.MaxSubscriptions != 12 {
		t.Errorf("ProPlanLimits.MaxSubscriptions should be 12, got %d", ProPlanLimits.MaxSubscriptions)
	}
}
