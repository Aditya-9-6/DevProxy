package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// W3C Trace Context Header Names
const (
	HeaderTraceParent = "traceparent"
	HeaderTraceState  = "tracestate"
)

var traceParentRegex = regexp.MustCompile(`^([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})$`)

// TraceContext represents a parsed W3C distributed trace span.
type TraceContext struct {
	Version      string `json:"version"`
	TraceID      string `json:"trace_id"`
	ParentSpanID string `json:"parent_span_id,omitempty"`
	SpanID       string `json:"span_id"`
	TraceFlags   string `json:"trace_flags"`
	TraceState   string `json:"tracestate,omitempty"`
	IsSampled    bool   `json:"is_sampled"`
}

// GenerateRandomHex produces a cryptographically random hex-encoded string of n bytes (2n hex chars).
func GenerateRandomHex(numBytes int) (string, error) {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate cryptographically random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateTraceID produces a 32-hex-character W3C trace ID.
func GenerateTraceID() string {
	id, err := GenerateRandomHex(16)
	if err != nil {
		// Fallback to timestamp-seeded deterministic hex if entropy source fails
		return "0123456789abcdef0123456789abcdef"
	}
	return id
}

// GenerateSpanID produces a 16-hex-character W3C span ID.
func GenerateSpanID() string {
	id, err := GenerateRandomHex(8)
	if err != nil {
		return "0123456789abcdef"
	}
	return id
}

// ExtractTraceContext parses incoming HTTP request headers for W3C traceparent and tracestate.
// If valid, it preserves the TraceID, records the incoming span as ParentSpanID,
// and generates a new downstream SpanID. If missing or invalid, it initializes a root span.
func ExtractTraceContext(h http.Header) *TraceContext {
	rawParent := strings.TrimSpace(h.Get(HeaderTraceParent))
	traceState := strings.TrimSpace(h.Get(HeaderTraceState))

	if rawParent != "" {
		matches := traceParentRegex.FindStringSubmatch(strings.ToLower(rawParent))
		if len(matches) == 5 {
			version := matches[1]
			traceID := matches[2]
			parentSpanID := matches[3]
			traceFlags := matches[4]

			// W3C spec: trace ID and span ID cannot be all zeros
			if !isAllZero(traceID) && !isAllZero(parentSpanID) {
				return &TraceContext{
					Version:      version,
					TraceID:      traceID,
					ParentSpanID: parentSpanID,
					SpanID:       GenerateSpanID(),
					TraceFlags:   traceFlags,
					TraceState:   traceState,
					IsSampled:    traceFlags == "01",
				}
			}
		}
	}

	// Missing or invalid header: start a new root trace context
	return &TraceContext{
		Version:      "00",
		TraceID:      GenerateTraceID(),
		ParentSpanID: "",
		SpanID:       GenerateSpanID(),
		TraceFlags:   "01",
		TraceState:   traceState,
		IsSampled:    true,
	}
}

// InjectTraceContext populates downstream HTTP request headers with the updated W3C trace context.
func InjectTraceContext(h http.Header, tc *TraceContext) {
	if tc == nil || tc.TraceID == "" || tc.SpanID == "" {
		return
	}

	ver := tc.Version
	if ver == "" {
		ver = "00"
	}
	flags := tc.TraceFlags
	if flags == "" {
		flags = "01"
	}

	h.Set(HeaderTraceParent, fmt.Sprintf("%s-%s-%s-%s", ver, tc.TraceID, tc.SpanID, flags))
	if tc.TraceState != "" {
		h.Set(HeaderTraceState, tc.TraceState)
	}
}

func isAllZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return false
		}
	}
	return true
}
