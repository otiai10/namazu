package delivery

import (
	"context"
	"time"
)

// Result contains the result of a delivery attempt
type Result struct {
	URL          string
	StatusCode   int
	Success      bool
	ErrorMessage string
	ResponseTime time.Duration
}

// Target represents a delivery destination
type Target struct {
	URL    string
	Secret string
	Name   string
}

// Sender defines the interface for sending notifications
type Sender interface {
	Send(ctx context.Context, url, secret string, payload []byte) Result
	SendAll(ctx context.Context, targets []Target, payload []byte) []Result
}
