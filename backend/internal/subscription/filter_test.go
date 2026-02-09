package subscription

import (
	"testing"
	"time"

	"github.com/otiai10/namazu/backend/internal/source"
	"github.com/otiai10/namazu/backend/internal/source/p2pquake"
)

// mockEvent implements source.Event for testing
type mockEvent struct {
	id            string
	eventType     source.EventType
	eventSource   string
	severity      int
	affectedAreas []string
	occurredAt    time.Time
	receivedAt    time.Time
	rawJSON       string
}

func (m *mockEvent) GetID() string              { return m.id }
func (m *mockEvent) GetType() source.EventType  { return m.eventType }
func (m *mockEvent) GetSource() string          { return m.eventSource }
func (m *mockEvent) GetSeverity() int           { return m.severity }
func (m *mockEvent) GetAffectedAreas() []string { return m.affectedAreas }
func (m *mockEvent) GetOccurredAt() time.Time   { return m.occurredAt }
func (m *mockEvent) GetReceivedAt() time.Time   { return m.receivedAt }
func (m *mockEvent) GetRawJSON() string         { return m.rawJSON }

// Compile-time interface check
var _ source.Event = (*mockEvent)(nil)

func newMockEvent(severity int, areas []string) *mockEvent {
	return &mockEvent{
		id:            "test-event-1",
		eventType:     source.EventTypeEarthquake,
		eventSource:   "test",
		severity:      severity,
		affectedAreas: areas,
		occurredAt:    time.Now(),
		receivedAt:    time.Now(),
		rawJSON:       "{}",
	}
}

func TestFilterConfig_Matches_NilFilter(t *testing.T) {
	// nil filter should match all events
	var filter *FilterConfig = nil

	tests := []struct {
		name     string
		event    source.Event
		expected bool
	}{
		{
			name:     "matches event with severity 10",
			event:    newMockEvent(10, []string{"東京都"}),
			expected: true,
		},
		{
			name:     "matches event with severity 100",
			event:    newMockEvent(100, []string{"大阪府"}),
			expected: true,
		},
		{
			name:     "matches event with no areas",
			event:    newMockEvent(50, []string{}),
			expected: true,
		},
		{
			name:     "matches event with severity 0",
			event:    newMockEvent(0, []string{"北海道"}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.Matches(tt.event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterConfig_Matches_MinScale(t *testing.T) {
	tests := []struct {
		name     string
		minScale int
		severity int
		expected bool
	}{
		{
			name:     "event at threshold passes (Scale4 -> severity 40)",
			minScale: p2pquake.Scale4,
			severity: 40,
			expected: true,
		},
		{
			name:     "event above threshold passes (Scale4 -> severity 50)",
			minScale: p2pquake.Scale4,
			severity: 50,
			expected: true,
		},
		{
			name:     "event below threshold fails (Scale4 -> severity 30)",
			minScale: p2pquake.Scale4,
			severity: 30,
			expected: false,
		},
		{
			name:     "Scale5Weak threshold filters severity 40",
			minScale: p2pquake.Scale5Weak,
			severity: 40,
			expected: false,
		},
		{
			name:     "Scale5Weak threshold passes severity 50",
			minScale: p2pquake.Scale5Weak,
			severity: 50,
			expected: true,
		},
		{
			name:     "Scale7 threshold passes severity 100",
			minScale: p2pquake.Scale7,
			severity: 100,
			expected: true,
		},
		{
			name:     "Scale7 threshold filters severity 80",
			minScale: p2pquake.Scale7,
			severity: 80,
			expected: false,
		},
		{
			name:     "MinScale 0 matches all (no threshold)",
			minScale: 0,
			severity: 10,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &FilterConfig{
				MinScale: tt.minScale,
			}
			event := newMockEvent(tt.severity, []string{"東京都"})

			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v (minScale=%d, severity=%d)",
					result, tt.expected, tt.minScale, tt.severity)
			}
		})
	}
}

func TestFilterConfig_Matches_Prefectures(t *testing.T) {
	tests := []struct {
		name          string
		prefectures   []string
		affectedAreas []string
		expected      bool
	}{
		{
			name:          "exact match passes",
			prefectures:   []string{"東京都"},
			affectedAreas: []string{"東京都"},
			expected:      true,
		},
		{
			name:          "multiple prefectures, one matches",
			prefectures:   []string{"東京都", "神奈川県"},
			affectedAreas: []string{"神奈川県"},
			expected:      true,
		},
		{
			name:          "no match fails",
			prefectures:   []string{"東京都"},
			affectedAreas: []string{"大阪府"},
			expected:      false,
		},
		{
			name:          "empty prefectures filter matches all",
			prefectures:   []string{},
			affectedAreas: []string{"北海道"},
			expected:      true,
		},
		{
			name:          "multiple affected areas, one matches",
			prefectures:   []string{"東京都"},
			affectedAreas: []string{"神奈川県", "東京都", "千葉県"},
			expected:      true,
		},
		{
			name:          "nil prefectures matches all",
			prefectures:   nil,
			affectedAreas: []string{"沖縄県"},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &FilterConfig{
				Prefectures: tt.prefectures,
			}
			event := newMockEvent(50, tt.affectedAreas)

			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterConfig_Matches_Combined(t *testing.T) {
	tests := []struct {
		name          string
		minScale      int
		prefectures   []string
		severity      int
		affectedAreas []string
		expected      bool
	}{
		{
			name:          "both conditions met",
			minScale:      p2pquake.Scale4,
			prefectures:   []string{"東京都"},
			severity:      50,
			affectedAreas: []string{"東京都"},
			expected:      true,
		},
		{
			name:          "severity OK but prefecture mismatch",
			minScale:      p2pquake.Scale4,
			prefectures:   []string{"東京都"},
			severity:      50,
			affectedAreas: []string{"大阪府"},
			expected:      false,
		},
		{
			name:          "prefecture OK but severity too low",
			minScale:      p2pquake.Scale4,
			prefectures:   []string{"東京都"},
			severity:      30,
			affectedAreas: []string{"東京都"},
			expected:      false,
		},
		{
			name:          "neither condition met",
			minScale:      p2pquake.Scale4,
			prefectures:   []string{"東京都"},
			severity:      30,
			affectedAreas: []string{"大阪府"},
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &FilterConfig{
				MinScale:    tt.minScale,
				Prefectures: tt.prefectures,
			}
			event := newMockEvent(tt.severity, tt.affectedAreas)

			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterConfig_Matches_EmptyAffectedAreas(t *testing.T) {
	tests := []struct {
		name          string
		prefectures   []string
		affectedAreas []string
		expected      bool
	}{
		{
			name:          "event with no areas fails prefecture filter",
			prefectures:   []string{"東京都"},
			affectedAreas: []string{},
			expected:      false,
		},
		{
			name:          "event with nil areas fails prefecture filter",
			prefectures:   []string{"東京都"},
			affectedAreas: nil,
			expected:      false,
		},
		{
			name:          "event with no areas passes empty prefecture filter",
			prefectures:   []string{},
			affectedAreas: []string{},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &FilterConfig{
				Prefectures: tt.prefectures,
			}
			event := newMockEvent(50, tt.affectedAreas)

			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFilterConfig_Matches_PrefixMatch(t *testing.T) {
	tests := []struct {
		name          string
		prefectures   []string
		affectedAreas []string
		expected      bool
	}{
		{
			name:          "東京 prefix matches 東京都",
			prefectures:   []string{"東京"},
			affectedAreas: []string{"東京都"},
			expected:      true,
		},
		{
			name:          "大阪 prefix matches 大阪府",
			prefectures:   []string{"大阪"},
			affectedAreas: []string{"大阪府"},
			expected:      true,
		},
		{
			name:          "神奈川 prefix matches 神奈川県",
			prefectures:   []string{"神奈川"},
			affectedAreas: []string{"神奈川県"},
			expected:      true,
		},
		{
			name:          "full name also works",
			prefectures:   []string{"神奈川県"},
			affectedAreas: []string{"神奈川県"},
			expected:      true,
		},
		{
			name:          "partial mismatch fails",
			prefectures:   []string{"京都"},
			affectedAreas: []string{"東京都"},
			expected:      false,
		},
		{
			name:          "北海 prefix matches 北海道",
			prefectures:   []string{"北海"},
			affectedAreas: []string{"北海道"},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &FilterConfig{
				Prefectures: tt.prefectures,
			}
			event := newMockEvent(50, tt.affectedAreas)

			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v (prefectures=%v, areas=%v)",
					result, tt.expected, tt.prefectures, tt.affectedAreas)
			}
		})
	}
}

func TestFilterConfig_Matches_EmptyFilter(t *testing.T) {
	// Empty filter (no conditions set) should match all events
	filter := &FilterConfig{}

	tests := []struct {
		name          string
		severity      int
		affectedAreas []string
		expected      bool
	}{
		{
			name:          "matches low severity event",
			severity:      10,
			affectedAreas: []string{"東京都"},
			expected:      true,
		},
		{
			name:          "matches high severity event",
			severity:      100,
			affectedAreas: []string{"大阪府"},
			expected:      true,
		},
		{
			name:          "matches event with no areas",
			severity:      50,
			affectedAreas: []string{},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := newMockEvent(tt.severity, tt.affectedAreas)
			result := filter.Matches(event)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
