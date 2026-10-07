package analysis

import (
	"fmt"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"strings"
)

// ... existing rules ...

// TLSFingerprintRule flags User-Agent spoofing.
type TLSFingerprintRule struct{}

func NewTLSFingerprintRule() *TLSFingerprintRule { return &TLSFingerprintRule{} }
func (r *TLSFingerprintRule) Name() string       { return "TLS Fingerprint Spoofing Detection" }

func (r *TLSFingerprintRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event.TLSFingerprint == "" {
		return nil
	}
	ua := strings.ToLower(event.ReqHeaders.Get("User-Agent"))
	// Simple heuristic: if UA claims Chrome but fingerprint is known as non-browser
	if strings.Contains(ua, "chrome") && isLikelyBotFingerprint(event.TLSFingerprint) {
		return []*Finding{{
			Severity:    SeverityHigh,
			Title:       "Potential User-Agent Spoofing",
			Description: "The TLS fingerprint does not match a standard browser, but the User-Agent claims to be Chrome.",
		}}
	}
	return nil
}

func isLikelyBotFingerprint(ja3 string) bool {
	// Placeholder for known bot JA3 hashes
	return false
}
