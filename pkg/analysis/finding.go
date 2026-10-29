package analysis

import (
	"time"
)

// Severity levels for security findings.
const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// Finding represents a detected vulnerability or security misconfiguration.
type Finding struct {
	ID          string    `json:"id"`
	RequestID   string    `json:"request_id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	RuleName    string    `json:"rule_name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Evidence    string    `json:"evidence"`
	Location    string    `json:"location"` // e.g. "Response Header: Set-Cookie", "Request Body", etc.
	Remediation string    `json:"remediation"`
	URL         string    `json:"url"`
	Method      string    `json:"method"`
}
