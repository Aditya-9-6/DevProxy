package security

import (
	"github.com/Aditya-9-6/DevProxy/pkg/httpwire"
)

// SmugglingPolicy evaluates request framing for potential HTTP request smuggling vectors.
type SmugglingPolicy struct {
	RejectConflictingHeaders bool
	RejectInvalidContentLength bool
}

// NewSmugglingPolicy returns a secure default smuggling policy.
func NewSmugglingPolicy() *SmugglingPolicy {
	return &SmugglingPolicy{
		RejectConflictingHeaders:   true,
		RejectInvalidContentLength: true,
	}
}

// Inspect checks a Frame for CL.TE or TE.CL smuggling conflicts.
func (p *SmugglingPolicy) Inspect(frame *httpwire.Frame) error {
	hasCL := false
	hasTE := false

	for _, h := range frame.Headers {
		if h[0] == "Content-Length" {
			hasCL = true
		} else if h[0] == "Transfer-Encoding" {
			hasTE = true
		}
	}

	if p.RejectConflictingHeaders && hasCL && hasTE {
		return httpwire.ErrInvalidHeader
	}

	return nil
}
