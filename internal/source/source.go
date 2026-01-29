package source

import (
	"context"
	"time"
)

// EventType represents different event types
type EventType string

const (
	EventTypeEarthquake EventType = "earthquake"
	EventTypeTsunami    EventType = "tsunami"
)

// Source represents a data source that provides events
type Source interface {
	Connect(ctx context.Context) error
	Events() <-chan Event
	Close() error
}

// Event represents a generic event from any source
type Event interface {
	GetID() string
	GetType() EventType
	GetSource() string
	GetSeverity() int
	GetAffectedAreas() []string
	GetOccurredAt() time.Time
	GetReceivedAt() time.Time
	GetRawJSON() string
}
