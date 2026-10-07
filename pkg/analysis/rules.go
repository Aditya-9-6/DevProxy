package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"regexp"
	"strings"
)

// Finding represents a security issue detected by a rule.
type Finding struct {
	ID          string
	RequestID   string
	Timestamp   interface{}
	Severity    string
	Category    string
	RuleName    string
	Title       string
	Description string
	Evidence    string
	Location    string
	Remediation string
	URL         string
	Method      string
}

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityMedium   = "MEDIUM"
	SeverityLow      = "LOW"
	SeverityInfo     = "INFO"
)

// Rule defines the interface for evaluating traffic events.
type Rule interface {
	Name() string
	Evaluate(event *ringbuffer.TrafficEvent) []*Finding
}

// SecurityRulesEngine coordinates and executes security inspection rules.
type SecurityRulesEngine struct {
	rules []Rule
}

// TLSFingerprintRule flags User-Agent spoofing.
type TLSFingerprintRule struct{}

func NewTLSFingerprintRule() *TLSFingerprintRule { return &TLSFingerprintRule{} }
func (r *TLSFingerprintRule) Name() string       { return "TLS Fingerprint Spoofing Detection" }

func (r *TLSFingerprintRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event.TLSFingerprint == "" {
		return nil
	}
	ua := strings.ToLower(event.ReqHeaders.Get("User-Agent"))
	if strings.Contains(ua, "chrome") && isLikelyBotFingerprint(event.TLSFingerprint) {
		return []*Finding{{}}
	}
	return nil
}

func isLikelyBotFingerprint(ja3 string) bool { return false }
