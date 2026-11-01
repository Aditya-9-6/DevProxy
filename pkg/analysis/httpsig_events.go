package analysis

import (
	"time"
)

// Event represents an analysis event record.
type Event struct {
	Type      string
	Timestamp time.Time
	Details   map[string]any
}

// RecordHTTPSigEvent records a security/HTTP signature event for analysis.
func RecordHTTPSigEvent(ev Event) {
	// No-op or implementation for tracking proxy events
}
