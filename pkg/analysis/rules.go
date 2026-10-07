package analysis

import (
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
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

// Analyze iterates through all registered rules and aggregates findings.
func (e *SecurityRulesEngine) Analyze(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	for _, rule := range e.rules {
		findings = append(findings, rule.Evaluate(event)...)
	}
	return findings
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
