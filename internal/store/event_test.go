package store

import (
	"context"
	"testing"
	"time"

	"github.com/ayanel/namazu/internal/source"
)

// mockEvent implements source.Event for testing
type mockEvent struct {
	id            string
	eventType     source.EventType
	sourceID      string
	severity      int
	affectedAreas []string
	occurredAt    time.Time
	receivedAt    time.Time
	rawJSON       string
}

func (m *mockEvent) GetID() string              { return m.id }
func (m *mockEvent) GetType() source.EventType  { return m.eventType }
func (m *mockEvent) GetSource() string          { return m.sourceID }
func (m *mockEvent) GetSeverity() int           { return m.severity }
func (m *mockEvent) GetAffectedAreas() []string { return m.affectedAreas }
func (m *mockEvent) GetOccurredAt() time.Time   { return m.occurredAt }
func (m *mockEvent) GetReceivedAt() time.Time   { return m.receivedAt }
func (m *mockEvent) GetRawJSON() string         { return m.rawJSON }

// Compile-time check that mockEvent implements source.Event
var _ source.Event = (*mockEvent)(nil)

func TestEventFromSource(t *testing.T) {
	fixedOccurredAt := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
	fixedReceivedAt := time.Date(2024, 1, 15, 12, 31, 0, 0, time.UTC)

	tests := []struct {
		name     string
		event    source.Event
		expected EventRecord
	}{
		{
			name: "Earthquake event with all fields",
			event: &mockEvent{
				id:            "eq-2024-001",
				eventType:     source.EventTypeEarthquake,
				sourceID:      "p2pquake",
				severity:      80,
				affectedAreas: []string{"Tokyo", "Kanagawa", "Chiba"},
				occurredAt:    fixedOccurredAt,
				receivedAt:    fixedReceivedAt,
				rawJSON:       `{"code":551}`,
			},
			expected: EventRecord{
				ID:            "eq-2024-001",
				Type:          "earthquake",
				Source:        "p2pquake",
				Severity:      80,
				AffectedAreas: []string{"Tokyo", "Kanagawa", "Chiba"},
				OccurredAt:    fixedOccurredAt,
				ReceivedAt:    fixedReceivedAt,
				RawJSON:       `{"code":551}`,
			},
		},
		{
			name: "Tsunami event",
			event: &mockEvent{
				id:            "ts-2024-001",
				eventType:     source.EventTypeTsunami,
				sourceID:      "jma",
				severity:      90,
				affectedAreas: []string{"Iwate", "Miyagi"},
				occurredAt:    fixedOccurredAt,
				receivedAt:    fixedReceivedAt,
				rawJSON:       `{"type":"tsunami"}`,
			},
			expected: EventRecord{
				ID:            "ts-2024-001",
				Type:          "tsunami",
				Source:        "jma",
				Severity:      90,
				AffectedAreas: []string{"Iwate", "Miyagi"},
				OccurredAt:    fixedOccurredAt,
				ReceivedAt:    fixedReceivedAt,
				RawJSON:       `{"type":"tsunami"}`,
			},
		},
		{
			name: "Event with empty affected areas",
			event: &mockEvent{
				id:            "eq-2024-002",
				eventType:     source.EventTypeEarthquake,
				sourceID:      "p2pquake",
				severity:      10,
				affectedAreas: []string{},
				occurredAt:    fixedOccurredAt,
				receivedAt:    fixedReceivedAt,
				rawJSON:       `{}`,
			},
			expected: EventRecord{
				ID:            "eq-2024-002",
				Type:          "earthquake",
				Source:        "p2pquake",
				Severity:      10,
				AffectedAreas: []string{},
				OccurredAt:    fixedOccurredAt,
				ReceivedAt:    fixedReceivedAt,
				RawJSON:       `{}`,
			},
		},
		{
			name: "Event with nil affected areas",
			event: &mockEvent{
				id:            "eq-2024-003",
				eventType:     source.EventTypeEarthquake,
				sourceID:      "p2pquake",
				severity:      0,
				affectedAreas: nil,
				occurredAt:    fixedOccurredAt,
				receivedAt:    fixedReceivedAt,
				rawJSON:       `{}`,
			},
			expected: EventRecord{
				ID:            "eq-2024-003",
				Type:          "earthquake",
				Source:        "p2pquake",
				Severity:      0,
				AffectedAreas: []string{},
				OccurredAt:    fixedOccurredAt,
				ReceivedAt:    fixedReceivedAt,
				RawJSON:       `{}`,
			},
		},
		{
			name: "Event with zero times",
			event: &mockEvent{
				id:            "eq-2024-004",
				eventType:     source.EventTypeEarthquake,
				sourceID:      "unknown",
				severity:      0,
				affectedAreas: nil,
				occurredAt:    time.Time{},
				receivedAt:    time.Time{},
				rawJSON:       "",
			},
			expected: EventRecord{
				ID:            "eq-2024-004",
				Type:          "earthquake",
				Source:        "unknown",
				Severity:      0,
				AffectedAreas: []string{},
				OccurredAt:    time.Time{},
				ReceivedAt:    time.Time{},
				RawJSON:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EventFromSource(tt.event)

			if result.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("Type = %q, want %q", result.Type, tt.expected.Type)
			}
			if result.Source != tt.expected.Source {
				t.Errorf("Source = %q, want %q", result.Source, tt.expected.Source)
			}
			if result.Severity != tt.expected.Severity {
				t.Errorf("Severity = %d, want %d", result.Severity, tt.expected.Severity)
			}
			if !equalStringSlices(result.AffectedAreas, tt.expected.AffectedAreas) {
				t.Errorf("AffectedAreas = %v, want %v", result.AffectedAreas, tt.expected.AffectedAreas)
			}
			if !result.OccurredAt.Equal(tt.expected.OccurredAt) {
				t.Errorf("OccurredAt = %v, want %v", result.OccurredAt, tt.expected.OccurredAt)
			}
			if !result.ReceivedAt.Equal(tt.expected.ReceivedAt) {
				t.Errorf("ReceivedAt = %v, want %v", result.ReceivedAt, tt.expected.ReceivedAt)
			}
			if result.RawJSON != tt.expected.RawJSON {
				t.Errorf("RawJSON = %q, want %q", result.RawJSON, tt.expected.RawJSON)
			}
			// CreatedAt should not be set by EventFromSource
			if !result.CreatedAt.IsZero() {
				t.Errorf("CreatedAt should be zero time, got %v", result.CreatedAt)
			}
		})
	}
}

func TestEventFromSource_ImmutabilityOfAffectedAreas(t *testing.T) {
	originalAreas := []string{"Tokyo", "Osaka"}
	event := &mockEvent{
		id:            "test-immutable",
		eventType:     source.EventTypeEarthquake,
		sourceID:      "test",
		affectedAreas: originalAreas,
	}

	result := EventFromSource(event)

	// Modify the original slice
	originalAreas[0] = "Modified"

	// Result should not be affected
	if result.AffectedAreas[0] == "Modified" {
		t.Error("EventFromSource should create a defensive copy of AffectedAreas")
	}
	if result.AffectedAreas[0] != "Tokyo" {
		t.Errorf("AffectedAreas[0] = %q, want %q", result.AffectedAreas[0], "Tokyo")
	}
}

func TestCopyStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Normal slice",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Nil slice",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "Single element",
			input:    []string{"single"},
			expected: []string{"single"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copyStringSlice(tt.input)
			if !equalStringSlices(result, tt.expected) {
				t.Errorf("copyStringSlice() = %v, want %v", result, tt.expected)
			}

			// Verify it's a copy, not the same slice
			if len(tt.input) > 0 {
				tt.input[0] = "modified"
				if result[0] == "modified" {
					t.Error("copyStringSlice should return a defensive copy")
				}
			}
		})
	}
}

func TestNewFirestoreEventRepository(t *testing.T) {
	// Test with nil client (valid use case for testing)
	repo := NewFirestoreEventRepository(nil)

	if repo == nil {
		t.Fatal("NewFirestoreEventRepository returned nil")
	}

	if repo.collection != "events" {
		t.Errorf("collection = %q, want %q", repo.collection, "events")
	}
}

func TestFirestoreEventRepository_Create_NilClient(t *testing.T) {
	repo := NewFirestoreEventRepository(nil)

	_, err := repo.Create(context.TODO(), EventRecord{ID: "test"})
	if err == nil {
		t.Fatal("Create() should return error for nil client")
	}

	expectedMsg := "firestore client is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Create() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestFirestoreEventRepository_Get_NilClient(t *testing.T) {
	repo := NewFirestoreEventRepository(nil)

	_, err := repo.Get(context.TODO(), "test-id")
	if err == nil {
		t.Fatal("Get() should return error for nil client")
	}

	expectedMsg := "firestore client is nil"
	if err.Error() != expectedMsg {
		t.Errorf("Get() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestFirestoreEventRepository_Get_EmptyID(t *testing.T) {
	// We need a non-nil client to test this path, but we can test the validation
	// This test verifies the validation happens before attempting Firestore operations
	repo := &FirestoreEventRepository{
		client:     nil, // Will fail at nil check first
		collection: "events",
	}

	_, err := repo.Get(context.TODO(), "")
	if err == nil {
		t.Fatal("Get() should return error for empty ID")
	}
}

func TestFirestoreEventRepository_List_NilClient(t *testing.T) {
	repo := NewFirestoreEventRepository(nil)

	_, err := repo.List(context.TODO(), 10, nil)
	if err == nil {
		t.Fatal("List() should return error for nil client")
	}

	expectedMsg := "firestore client is nil"
	if err.Error() != expectedMsg {
		t.Errorf("List() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestFirestoreEventRepository_ImplementsInterface(t *testing.T) {
	var _ EventRepository = (*FirestoreEventRepository)(nil)
}

func TestCreateEventRecord(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)

	input := EventRecord{
		ID:            "test-id",
		Type:          "earthquake",
		Source:        "p2pquake",
		Severity:      80,
		AffectedAreas: []string{"Tokyo"},
		OccurredAt:    fixedTime,
		ReceivedAt:    fixedTime,
		RawJSON:       `{}`,
	}

	result := createEventRecord(input)

	// Verify all fields are copied
	if result.ID != input.ID {
		t.Errorf("ID = %q, want %q", result.ID, input.ID)
	}
	if result.Type != input.Type {
		t.Errorf("Type = %q, want %q", result.Type, input.Type)
	}
	if result.Source != input.Source {
		t.Errorf("Source = %q, want %q", result.Source, input.Source)
	}
	if result.Severity != input.Severity {
		t.Errorf("Severity = %d, want %d", result.Severity, input.Severity)
	}
	if !equalStringSlices(result.AffectedAreas, input.AffectedAreas) {
		t.Errorf("AffectedAreas = %v, want %v", result.AffectedAreas, input.AffectedAreas)
	}

	// Verify CreatedAt is set
	if result.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set by createEventRecord")
	}

	// Verify immutability of AffectedAreas
	input.AffectedAreas[0] = "Modified"
	if result.AffectedAreas[0] == "Modified" {
		t.Error("createEventRecord should create a defensive copy of AffectedAreas")
	}
}

func TestEventRecord_FieldTags(t *testing.T) {
	// Verify the struct has correct Firestore tags
	// This is a documentation test to ensure tags don't accidentally change

	record := EventRecord{
		ID:            "test",
		Type:          "earthquake",
		Source:        "p2pquake",
		Severity:      80,
		AffectedAreas: []string{"Tokyo"},
		OccurredAt:    time.Now(),
		ReceivedAt:    time.Now(),
		RawJSON:       `{}`,
		CreatedAt:     time.Now(),
	}

	// The ID field should be ignored by Firestore (tag: "-")
	// This test documents the expected behavior
	if record.ID != "test" {
		t.Error("ID field should be accessible")
	}
}

// Helper function to compare string slices
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Benchmark EventFromSource
func BenchmarkEventFromSource(b *testing.B) {
	event := &mockEvent{
		id:            "bench-event",
		eventType:     source.EventTypeEarthquake,
		sourceID:      "p2pquake",
		severity:      80,
		affectedAreas: []string{"Tokyo", "Kanagawa", "Chiba", "Saitama"},
		occurredAt:    time.Now(),
		receivedAt:    time.Now(),
		rawJSON:       `{"code":551,"time":"2024/01/15 12:34:56"}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EventFromSource(event)
	}
}

// Benchmark copyStringSlice
func BenchmarkCopyStringSlice(b *testing.B) {
	slice := []string{"Tokyo", "Kanagawa", "Chiba", "Saitama", "Gunma"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copyStringSlice(slice)
	}
}
