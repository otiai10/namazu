package p2pquake

import (
	"testing"
	"time"

	"github.com/otiai10/namazu/internal/source"
)

// Test ScaleToSeverity function
func TestScaleToSeverity(t *testing.T) {
	tests := []struct {
		name     string
		scale    int
		expected int
	}{
		{"震度1", Scale1, 10},
		{"震度2", Scale2, 20},
		{"震度3", Scale3, 30},
		{"震度4", Scale4, 40},
		{"震度5弱", Scale5Weak, 50},
		{"震度5強", Scale5Strong, 60},
		{"震度6弱", Scale6Weak, 70},
		{"震度6強", Scale6Strong, 80},
		{"震度7", Scale7, 100},
		{"Unknown scale", 99, 0},
		{"Zero scale", 0, 0},
		{"Negative scale", -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleToSeverity(tt.scale)
			if result != tt.expected {
				t.Errorf("ScaleToSeverity(%d) = %d, want %d", tt.scale, result, tt.expected)
			}
		})
	}
}

// Test ScaleToString function
func TestScaleToString(t *testing.T) {
	tests := []struct {
		name     string
		scale    int
		expected string
	}{
		{"震度1", Scale1, "震度1"},
		{"震度2", Scale2, "震度2"},
		{"震度3", Scale3, "震度3"},
		{"震度4", Scale4, "震度4"},
		{"震度5弱", Scale5Weak, "震度5弱"},
		{"震度5強", Scale5Strong, "震度5強"},
		{"震度6弱", Scale6Weak, "震度6弱"},
		{"震度6強", Scale6Strong, "震度6強"},
		{"震度7", Scale7, "震度7"},
		{"Unknown scale", 99, "不明"},
		{"Zero scale", 0, "不明"},
		{"Negative scale", -1, "不明"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ScaleToString(tt.scale)
			if result != tt.expected {
				t.Errorf("ScaleToString(%d) = %q, want %q", tt.scale, result, tt.expected)
			}
		})
	}
}

// Test ParseP2PTime function
func TestParseP2PTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkTime   func(*testing.T, time.Time)
	}{
		{
			name:        "Valid time string",
			input:       "2024/01/15 12:34:56",
			expectError: false,
			checkTime: func(t *testing.T, tm time.Time) {
				if tm.Year() != 2024 {
					t.Errorf("Year = %d, want 2024", tm.Year())
				}
				if tm.Month() != 1 {
					t.Errorf("Month = %d, want 1", tm.Month())
				}
				if tm.Day() != 15 {
					t.Errorf("Day = %d, want 15", tm.Day())
				}
				if tm.Hour() != 12 {
					t.Errorf("Hour = %d, want 12", tm.Hour())
				}
				if tm.Minute() != 34 {
					t.Errorf("Minute = %d, want 34", tm.Minute())
				}
				if tm.Second() != 56 {
					t.Errorf("Second = %d, want 56", tm.Second())
				}
				// Check timezone is JST (+9)
				_, offset := tm.Zone()
				if offset != 9*60*60 {
					t.Errorf("Timezone offset = %d seconds, want %d (JST)", offset, 9*60*60)
				}
			},
		},
		{
			name:        "New Year time",
			input:       "2026/01/01 00:00:00",
			expectError: false,
			checkTime: func(t *testing.T, tm time.Time) {
				if tm.Year() != 2026 || tm.Month() != 1 || tm.Day() != 1 {
					t.Errorf("Date = %v, want 2026/01/01", tm)
				}
			},
		},
		{
			name:        "Invalid format - missing slashes",
			input:       "2024-01-15 12:34:56",
			expectError: true,
			checkTime:   nil,
		},
		{
			name:        "Invalid format - wrong order",
			input:       "15/01/2024 12:34:56",
			expectError: true,
			checkTime:   nil,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
			checkTime:   nil,
		},
		{
			name:        "Invalid month",
			input:       "2024/13/15 12:34:56",
			expectError: true,
			checkTime:   nil,
		},
		{
			name:        "Invalid day",
			input:       "2024/01/32 12:34:56",
			expectError: true,
			checkTime:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseP2PTime(tt.input)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkTime != nil {
					tt.checkTime(t, result)
				}
			}
		})
	}
}

// Test JMAQuake GetID
func TestJMAQuake_GetID(t *testing.T) {
	quake := &JMAQuake{
		ID: "test-id-12345",
	}
	if got := quake.GetID(); got != "test-id-12345" {
		t.Errorf("GetID() = %q, want %q", got, "test-id-12345")
	}
}

// Test JMAQuake GetType
func TestJMAQuake_GetType(t *testing.T) {
	quake := &JMAQuake{}
	if got := quake.GetType(); got != source.EventTypeEarthquake {
		t.Errorf("GetType() = %q, want %q", got, source.EventTypeEarthquake)
	}
}

// Test JMAQuake GetSource
func TestJMAQuake_GetSource(t *testing.T) {
	quake := &JMAQuake{}
	if got := quake.GetSource(); got != "p2pquake" {
		t.Errorf("GetSource() = %q, want %q", got, "p2pquake")
	}
}

// Test JMAQuake GetSeverity
func TestJMAQuake_GetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		quake    *JMAQuake
		expected int
	}{
		{
			name: "Earthquake with Scale7",
			quake: &JMAQuake{
				Earthquake: &Earthquake{
					MaxScale: Scale7,
				},
			},
			expected: 100,
		},
		{
			name: "Earthquake with Scale5Weak",
			quake: &JMAQuake{
				Earthquake: &Earthquake{
					MaxScale: Scale5Weak,
				},
			},
			expected: 50,
		},
		{
			name: "Earthquake with Scale1",
			quake: &JMAQuake{
				Earthquake: &Earthquake{
					MaxScale: Scale1,
				},
			},
			expected: 10,
		},
		{
			name: "Nil Earthquake",
			quake: &JMAQuake{
				Earthquake: nil,
			},
			expected: 0,
		},
		{
			name: "Unknown scale",
			quake: &JMAQuake{
				Earthquake: &Earthquake{
					MaxScale: 999,
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quake.GetSeverity()
			if result != tt.expected {
				t.Errorf("GetSeverity() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// Test JMAQuake GetAffectedAreas
func TestJMAQuake_GetAffectedAreas(t *testing.T) {
	tests := []struct {
		name     string
		quake    *JMAQuake
		expected []string
	}{
		{
			name: "Multiple points with different prefectures",
			quake: &JMAQuake{
				Points: []Point{
					{Prefecture: "東京都", Name: "千代田区"},
					{Prefecture: "神奈川県", Name: "横浜市"},
					{Prefecture: "東京都", Name: "新宿区"},
					{Prefecture: "千葉県", Name: "千葉市"},
				},
			},
			expected: []string{"東京都", "神奈川県", "千葉県"},
		},
		{
			name: "Single point",
			quake: &JMAQuake{
				Points: []Point{
					{Prefecture: "大阪府", Name: "大阪市"},
				},
			},
			expected: []string{"大阪府"},
		},
		{
			name: "No points",
			quake: &JMAQuake{
				Points: []Point{},
			},
			expected: []string{},
		},
		{
			name: "Nil points",
			quake: &JMAQuake{
				Points: nil,
			},
			expected: []string{},
		},
		{
			name: "All same prefecture",
			quake: &JMAQuake{
				Points: []Point{
					{Prefecture: "北海道", Name: "札幌市"},
					{Prefecture: "北海道", Name: "函館市"},
					{Prefecture: "北海道", Name: "旭川市"},
				},
			},
			expected: []string{"北海道"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quake.GetAffectedAreas()

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("GetAffectedAreas() returned %d areas, want %d", len(result), len(tt.expected))
			}

			// Check all expected areas are present (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, area := range result {
				resultMap[area] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("GetAffectedAreas() missing expected area: %q", expected)
				}
			}
		})
	}
}

// Test JMAQuake GetOccurredAt
func TestJMAQuake_GetOccurredAt(t *testing.T) {
	tests := []struct {
		name      string
		quake     *JMAQuake
		checkTime func(*testing.T, time.Time)
	}{
		{
			name: "Uses Earthquake.Time when available",
			quake: &JMAQuake{
				Time: "2024/01/15 10:00:00",
				Earthquake: &Earthquake{
					Time: "2024/01/15 12:34:56",
				},
			},
			checkTime: func(t *testing.T, tm time.Time) {
				if tm.Hour() != 12 || tm.Minute() != 34 {
					t.Errorf("Expected time from Earthquake.Time (12:34), got %02d:%02d", tm.Hour(), tm.Minute())
				}
			},
		},
		{
			name: "Falls back to Time when Earthquake.Time is empty",
			quake: &JMAQuake{
				Time: "2024/01/15 10:00:00",
				Earthquake: &Earthquake{
					Time: "",
				},
			},
			checkTime: func(t *testing.T, tm time.Time) {
				if tm.Hour() != 10 || tm.Minute() != 0 {
					t.Errorf("Expected time from Time (10:00), got %02d:%02d", tm.Hour(), tm.Minute())
				}
			},
		},
		{
			name: "Falls back to Time when Earthquake is nil",
			quake: &JMAQuake{
				Time:       "2024/01/15 08:30:45",
				Earthquake: nil,
			},
			checkTime: func(t *testing.T, tm time.Time) {
				if tm.Hour() != 8 || tm.Minute() != 30 {
					t.Errorf("Expected time from Time (08:30), got %02d:%02d", tm.Hour(), tm.Minute())
				}
			},
		},
		{
			name: "Returns zero time on parse error",
			quake: &JMAQuake{
				Time: "invalid",
			},
			checkTime: func(t *testing.T, tm time.Time) {
				if !tm.IsZero() {
					t.Errorf("Expected zero time on parse error, got %v", tm)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.quake.GetOccurredAt()
			tt.checkTime(t, result)
		})
	}
}

// Test JMAQuake GetReceivedAt
func TestJMAQuake_GetReceivedAt(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	quake := &JMAQuake{
		ReceivedAt: fixedTime,
	}

	result := quake.GetReceivedAt()
	if !result.Equal(fixedTime) {
		t.Errorf("GetReceivedAt() = %v, want %v", result, fixedTime)
	}
}

// Test JMAQuake GetRawJSON
func TestJMAQuake_GetRawJSON(t *testing.T) {
	rawJSON := `{"code":551,"time":"2024/01/15 12:34:56"}`
	quake := &JMAQuake{
		RawJSON: rawJSON,
	}

	result := quake.GetRawJSON()
	if result != rawJSON {
		t.Errorf("GetRawJSON() = %q, want %q", result, rawJSON)
	}
}

// Test that JMAQuake implements Event interface
func TestJMAQuake_ImplementsEventInterface(t *testing.T) {
	var _ source.Event = (*JMAQuake)(nil)
}

// Test complete JMAQuake with realistic data
func TestJMAQuake_Integration(t *testing.T) {
	receivedAt := time.Now()
	quake := &JMAQuake{
		ID:   "20240115123456",
		Code: 551,
		Time: "2024/01/15 12:30:00",
		Issue: Issue{
			Source: "気象庁",
			Time:   "2024/01/15 12:34:56",
			Type:   "ScalePrompt",
		},
		Earthquake: &Earthquake{
			Time: "2024/01/15 12:30:45",
			Hypocenter: Hypocenter{
				Name:      "能登半島沖",
				Latitude:  37.5,
				Longitude: 137.2,
				Depth:     10,
				Magnitude: 7.6,
			},
			MaxScale:        Scale6Strong,
			DomesticTsunami: "Warning",
		},
		Points: []Point{
			{Prefecture: "石川県", Name: "珠洲市", Scale: Scale6Strong, IsArea: false},
			{Prefecture: "石川県", Name: "輪島市", Scale: Scale6Weak, IsArea: false},
			{Prefecture: "富山県", Name: "富山市", Scale: Scale5Strong, IsArea: false},
			{Prefecture: "新潟県", Name: "上越市", Scale: Scale5Weak, IsArea: false},
		},
		ReceivedAt: receivedAt,
		RawJSON:    `{"_id":"20240115123456"}`,
	}

	// Test all Event interface methods
	if quake.GetID() != "20240115123456" {
		t.Errorf("GetID() failed")
	}

	if quake.GetType() != source.EventTypeEarthquake {
		t.Errorf("GetType() failed")
	}

	if quake.GetSource() != "p2pquake" {
		t.Errorf("GetSource() failed")
	}

	if quake.GetSeverity() != 80 {
		t.Errorf("GetSeverity() = %d, want 80", quake.GetSeverity())
	}

	areas := quake.GetAffectedAreas()
	if len(areas) != 3 {
		t.Errorf("GetAffectedAreas() returned %d areas, want 3", len(areas))
	}

	occurredAt := quake.GetOccurredAt()
	if occurredAt.Year() != 2024 || occurredAt.Month() != 1 || occurredAt.Day() != 15 {
		t.Errorf("GetOccurredAt() = %v, want 2024/01/15", occurredAt)
	}

	if !quake.GetReceivedAt().Equal(receivedAt) {
		t.Errorf("GetReceivedAt() failed")
	}

	if quake.GetRawJSON() != `{"_id":"20240115123456"}` {
		t.Errorf("GetRawJSON() failed")
	}
}

// Benchmark ScaleToSeverity
func BenchmarkScaleToSeverity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ScaleToSeverity(Scale6Strong)
	}
}

// Benchmark ParseP2PTime
func BenchmarkParseP2PTime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseP2PTime("2024/01/15 12:34:56")
	}
}

// Benchmark GetAffectedAreas
func BenchmarkGetAffectedAreas(b *testing.B) {
	quake := &JMAQuake{
		Points: []Point{
			{Prefecture: "東京都", Name: "千代田区"},
			{Prefecture: "神奈川県", Name: "横浜市"},
			{Prefecture: "東京都", Name: "新宿区"},
			{Prefecture: "千葉県", Name: "千葉市"},
			{Prefecture: "埼玉県", Name: "さいたま市"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		quake.GetAffectedAreas()
	}
}
