package proxy

import (
	"fmt"

	"github.com/Aditya-9-6/DevProxy/pkg/security/http2guard"
	"golang.org/x/net/http2"
)

// HTTP2GuardAdapter translates HTTP/2 frame headers into security guard events.
// It inspects headers directly and never buffers frame payloads.
type HTTP2GuardAdapter struct {
	guard *http2guard.Guard
}

// NewHTTP2GuardAdapter creates an adapter around a per-connection guard.
func NewHTTP2GuardAdapter(guard *http2guard.Guard) *HTTP2GuardAdapter {
	return &HTTP2GuardAdapter{guard: guard}
}

// NewHTTP2GuardAdapterFromConfig creates an adapter and its per-connection guard.
func NewHTTP2GuardAdapterFromConfig(config http2guard.Config) (*HTTP2GuardAdapter, error) {
	guard, err := http2guard.NewGuard(config)
	if err != nil {
		return nil, fmt.Errorf("create HTTP/2 guard: %w", err)
	}
	return NewHTTP2GuardAdapter(guard), nil
}

// Guard returns the wrapped security guard.
func (a *HTTP2GuardAdapter) Guard() *http2guard.Guard {
	if a == nil {
		return nil
	}
	return a.guard
}

// AdmitFrame validates and admits an HTTP/2 frame header.
// Frames outside the protected event categories are left to the HTTP/2 stack.
func (a *HTTP2GuardAdapter) AdmitFrame(header http2.FrameHeader) http2guard.Verdict {
	if a == nil || a.guard == nil {
		return http2guard.Verdict{Action: http2guard.ActionReject, Reason: http2guard.ReasonInvalidState}
	}

	switch header.Type {
	case http2.FrameHeaders:
		if !validHeadersFrame(header) {
			return malformedFrameVerdict()
		}
		return a.guard.AdmitStreamOpen(header.StreamID)
	case http2.FrameRSTStream:
		if !validResetFrame(header) {
			return malformedFrameVerdict()
		}
		return a.guard.AdmitStreamReset(header.StreamID)
	case http2.FrameSettings:
		if !validSettingsFrame(header) {
			return malformedFrameVerdict()
		}
		if header.Flags&http2.FlagSettingsAck != 0 {
			return a.guard.AdmitSettingsACK()
		}
		return a.guard.AdmitSettings()
	default:
		return http2guard.Verdict{Action: http2guard.ActionAllow, Reason: http2guard.ReasonNone}
	}
}

func validHeadersFrame(header http2.FrameHeader) bool {
	if header.StreamID == 0 || header.StreamID > http2guard.MaxStreamID {
		return false
	}

	length := int64(header.Length)
	if header.Flags&http2.FlagHeadersPadded != 0 {
		length--
	}
	if header.Flags&http2.FlagHeadersPriority != 0 {
		length -= 5
	}
	return length >= 0
}

func validResetFrame(header http2.FrameHeader) bool {
	return header.StreamID != 0 && header.StreamID <= http2guard.MaxStreamID && header.Length == 4
}

func validSettingsFrame(header http2.FrameHeader) bool {
	if header.StreamID != 0 {
		return false
	}
	if header.Flags&http2.FlagSettingsAck != 0 {
		return header.Length == 0
	}
	return header.Length%6 == 0
}

func malformedFrameVerdict() http2guard.Verdict {
	return http2guard.Verdict{Action: http2guard.ActionReject, Reason: http2guard.ReasonMalformedFrame}
}
