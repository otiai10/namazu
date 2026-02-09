package subscription

import (
	"strings"

	"github.com/otiai10/namazu/backend/internal/source"
	"github.com/otiai10/namazu/backend/internal/source/p2pquake"
)

// Matches checks if an event matches the filter criteria.
// Returns true if filter is nil (no filter = match all).
// Both MinScale AND Prefectures conditions must be satisfied (AND logic).
func (f *FilterConfig) Matches(event source.Event) bool {
	if f == nil {
		return true
	}

	// Check MinScale (convert JMA scale to severity for comparison)
	if f.MinScale > 0 {
		minSeverity := p2pquake.ScaleToSeverity(f.MinScale)
		if event.GetSeverity() < minSeverity {
			return false
		}
	}

	// Check Prefectures (if specified)
	if len(f.Prefectures) > 0 {
		if !matchesPrefectures(f.Prefectures, event.GetAffectedAreas()) {
			return false
		}
	}

	return true
}

// matchesPrefectures checks if any affected area matches any filter prefecture.
// Supports both exact match and prefix match (e.g., "東京" matches "東京都").
func matchesPrefectures(filterPrefectures, affectedAreas []string) bool {
	for _, area := range affectedAreas {
		for _, pref := range filterPrefectures {
			if area == pref || strings.HasPrefix(area, pref) {
				return true
			}
		}
	}
	return false
}
