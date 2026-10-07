package analysis

import (
	"fmt"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"strings"
)

// ... (existing code) ...

type TLSFingerprintRule struct{}

func NewTLSFingerprintRule() *TLSFingerprintRule { return &TLSFingerprintRule{} }

func (r *TLSFingerprintRule) Name() string { return "TLS Fingerprint Spoofing" }

func (r *TLSFingerprintRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	if event.TLSFingerprint == "" {
		return nil
	}
	ua := strings.ToLower(event.ReqHeaders.Get("User-Agent"))
	if strings.Contains(ua, "chrome") && !strings.Contains(event.TLSFingerprint, "known_chrome_hash") {
		return []*Finding{{
			RequestID:   event.ID,
			Severity:    SeverityMedium,
			Title:       "Potential User-Agent Spoofing",
			Description: "TLS fingerprint does not match expected browser signature for the provided User-Agent.",
		}}
	}
	return nil
}
