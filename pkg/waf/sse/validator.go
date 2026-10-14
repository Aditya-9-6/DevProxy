package sse

import (
	"bytes"
)

var (
	forbiddenMarkers = [][]byte{[]byte("<script"), []byte("javascript:"), []byte("eval(")}
)

// Validator inspects Server-Sent Events (SSE) for malicious payloads and protocol compliance.
type Validator struct{}

// NewValidator creates a new SSE validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if an SSE event block is safe and follows the basic SSE line format.
// It returns false if forbidden script markers are found or if lines violate the 'field: value' format.
func (v *Validator) Validate(event []byte) bool {
	for _, marker := range forbiddenMarkers {
		if bytes.Contains(event, marker) {
			return false
		}
	}

	lines := bytes.Split(event, []byte("\n"))
	for _, line := range lines {
		// Skip empty lines (event separators) and comments (lines starting with ':')
		if len(line) == 0 || line[0] == ':' {
			continue
		}

		// SSE lines must contain a colon to separate field from value (e.g., "data: ...")
		if !bytes.Contains(line, []byte(":")) {
			return false
		}
	}
	return true
}
