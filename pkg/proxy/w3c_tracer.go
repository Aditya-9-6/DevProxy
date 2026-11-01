package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var (
	ErrInvalidTraceParent = errors.New("w3c: invalid traceparent header format")
	ErrUnsupportedVersion = errors.New("w3c: unsupported traceparent version")
)

const (
	TraceParentHeader = "traceparent"
	TraceStateHeader  = "tracestate"
	VersionSupported  = "00"
)

// SpanContext holds parsed W3C Trace Context metadata.
type SpanContext struct {
	Version   string
	TraceID   string
	ParentID  string
	TraceFlags string
	TraceState string
	IsValid   bool
}

var spanContextPool = sync.Pool{
	New: func() any {
		return &SpanContext{}
	},
}

// AcquireSpanContext retrieves a SpanContext from the pool.
func AcquireSpanContext() *SpanContext {
	return spanContextPool.Get().(*SpanContext)
}

// ReleaseSpanContext resets and returns a SpanContext to the pool.
func ReleaseSpanContext(sc *SpanContext) {
	if sc == nil {
		return
	}
	sc.Version = ""
	sc.TraceID = ""
	sc.ParentID = ""
	sc.TraceFlags = ""
	sc.TraceState = ""
	sc.IsValid = false
	spanContextPool.Put(sc)
}

// ParseTraceParent parses a W3C traceparent header string.
// Format: version-traceid-parentid-traceflags (e.g., 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01)
func ParseTraceParent(headerVal string) (SpanContext, error) {
	headerVal = strings.TrimSpace(headerVal)
	if headerVal == "" {
		return SpanContext{}, ErrInvalidTraceParent
	}

	parts := strings.Split(headerVal, "-")
	if len(parts) != 4 {
		return SpanContext{}, ErrInvalidTraceParent
	}

	version := parts[0]
	if version != VersionSupported {
		return SpanContext{}, ErrUnsupportedVersion
	}

	traceID := parts[1]
	if len(traceID) != 32 || !isHex(traceID) || traceID == "00000000000000000000000000000000" {
		return SpanContext{}, ErrInvalidTraceParent
	}

	parentID := parts[2]
	if len(parentID) != 16 || !isHex(parentID) || parentID == "0000000000000000" {
		return SpanContext{}, ErrInvalidTraceParent
	}

	traceFlags := parts[3]
	if len(traceFlags) != 2 || !isHex(traceFlags) {
		return SpanContext{}, ErrInvalidTraceParent
	}

	return SpanContext{
		Version:    version,
		TraceID:    traceID,
		ParentID:   parentID,
		TraceFlags: traceFlags,
		IsValid:    true,
	}, nil
}

// Inject adds or updates traceparent and tracestate headers on an http.Header.
func Inject(sc SpanContext, header http.Header) {
	if !sc.IsValid {
		return
	}
	parentVal := fmt.Sprintf("%s-%s-%s-%s", sc.Version, sc.TraceID, sc.ParentID, sc.TraceFlags)
	header.Set(TraceParentHeader, parentVal)
	if sc.TraceState != "" {
		header.Set(TraceStateHeader, sc.TraceState)
	}
}

// GenerateTraceID generates a random 16-byte hex-encoded trace ID (32 chars).
func GenerateTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// GenerateSpanID generates a random 8-byte hex-encoded span ID (16 chars).
func GenerateSpanID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func isHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
