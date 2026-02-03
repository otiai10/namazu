package p2pquake

import (
	"time"

	"github.com/otiai10/namazu/internal/source"
)

// Scale constants for Japanese seismic intensity scale
const (
	Scale1       = 10 // 震度1
	Scale2       = 20 // 震度2
	Scale3       = 30 // 震度3
	Scale4       = 40 // 震度4
	Scale5Weak   = 45 // 震度5弱
	Scale5Strong = 50 // 震度5強
	Scale6Weak   = 55 // 震度6弱
	Scale6Strong = 60 // 震度6強
	Scale7       = 70 // 震度7
)

// JMAQuake represents earthquake information from JMA (code 551)
type JMAQuake struct {
	ID         string      `json:"_id"`
	Code       int         `json:"code"` // Should be 551
	Time       string      `json:"time"`
	Issue      Issue       `json:"issue"`
	Earthquake *Earthquake `json:"earthquake,omitempty"`
	Points     []Point     `json:"points,omitempty"`
	// Added fields for Event interface
	ReceivedAt time.Time `json:"-"`
	RawJSON    string    `json:"-"`
}

// Issue contains information about when/who issued the report
type Issue struct {
	Source  string `json:"source"`
	Time    string `json:"time"`
	Type    string `json:"type"`
	Correct string `json:"correct,omitempty"`
}

// Earthquake contains hypocenter and magnitude info
type Earthquake struct {
	Time            string     `json:"time"`
	Hypocenter      Hypocenter `json:"hypocenter"`
	MaxScale        int        `json:"maxScale"`
	DomesticTsunami string     `json:"domesticTsunami"`
	ForeignTsunami  string     `json:"foreignTsunami,omitempty"`
}

// Hypocenter contains epicenter location info
type Hypocenter struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Depth     int     `json:"depth"`
	Magnitude float64 `json:"magnitude"`
}

// Point contains observed intensity at a location
type Point struct {
	Prefecture string `json:"pref"`
	Name       string `json:"addr"`
	Scale      int    `json:"scale"`
	IsArea     bool   `json:"isArea"`
}

// Compile-time interface check
var _ source.Event = (*JMAQuake)(nil)

// GetID returns the unique identifier
func (q *JMAQuake) GetID() string {
	return q.ID
}

// GetType returns the event type
func (q *JMAQuake) GetType() source.EventType {
	return source.EventTypeEarthquake
}

// GetSource returns the data source identifier
func (q *JMAQuake) GetSource() string {
	return "p2pquake"
}

// GetSeverity returns normalized severity (0-100)
func (q *JMAQuake) GetSeverity() int {
	if q.Earthquake == nil {
		return 0
	}
	return ScaleToSeverity(q.Earthquake.MaxScale)
}

// GetAffectedAreas returns list of affected prefectures
func (q *JMAQuake) GetAffectedAreas() []string {
	if len(q.Points) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	areas := []string{}

	for _, p := range q.Points {
		if !seen[p.Prefecture] {
			seen[p.Prefecture] = true
			areas = append(areas, p.Prefecture)
		}
	}

	return areas
}

// GetOccurredAt returns when the earthquake occurred
func (q *JMAQuake) GetOccurredAt() time.Time {
	if q.Earthquake != nil && q.Earthquake.Time != "" {
		t, err := ParseP2PTime(q.Earthquake.Time)
		if err == nil {
			return t
		}
	}
	t, _ := ParseP2PTime(q.Time)
	return t
}

// GetReceivedAt returns when the event was received
func (q *JMAQuake) GetReceivedAt() time.Time {
	return q.ReceivedAt
}

// GetRawJSON returns the original JSON
func (q *JMAQuake) GetRawJSON() string {
	return q.RawJSON
}

// ParseP2PTime parses time string from P2P地震情報 API
// Format: "2024/01/15 12:34:56" in JST
func ParseP2PTime(s string) (time.Time, error) {
	jst := time.FixedZone("JST", 9*60*60)
	return time.ParseInLocation("2006/01/02 15:04:05", s, jst)
}

// ScaleToSeverity converts JMA scale (10-70) to normalized severity (0-100)
func ScaleToSeverity(scale int) int {
	switch scale {
	case Scale1:
		return 10
	case Scale2:
		return 20
	case Scale3:
		return 30
	case Scale4:
		return 40
	case Scale5Weak:
		return 50
	case Scale5Strong:
		return 60
	case Scale6Weak:
		return 70
	case Scale6Strong:
		return 80
	case Scale7:
		return 100
	default:
		return 0
	}
}

// ScaleToString returns human-readable scale name
func ScaleToString(scale int) string {
	switch scale {
	case Scale1:
		return "震度1"
	case Scale2:
		return "震度2"
	case Scale3:
		return "震度3"
	case Scale4:
		return "震度4"
	case Scale5Weak:
		return "震度5弱"
	case Scale5Strong:
		return "震度5強"
	case Scale6Weak:
		return "震度6弱"
	case Scale6Strong:
		return "震度6強"
	case Scale7:
		return "震度7"
	default:
		return "不明"
	}
}
