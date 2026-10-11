package replay

import (
	"net/http"
	"regexp"
)

var sensitiveHeaders = []string{"Authorization", "Cookie", "Set-Cookie", "Proxy-Authorization"}

// Sanitizer strips sensitive information from HTTP headers and bodies.
type Sanitizer struct {
	sensitiveRegex *regexp.Regexp
}

// NewSanitizer initializes a new Sanitizer with default sensitive patterns.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		sensitiveRegex: regexp.MustCompile(`(?i)(password|secret|token|key)=[^&\s]+`),
	}
}

// SanitizeHeaders redacts sensitive HTTP headers in place.
func (s *Sanitizer) SanitizeHeaders(headers http.Header) {
	if headers == nil {
		return
	}
	for _, h := range sensitiveHeaders {
		if headers.Get(h) != "" {
			headers.Set(h, "[REDACTED]")
		}
	}
}

// SanitizeBody returns a copy of the body with sensitive parameters redacted.
func (s *Sanitizer) SanitizeBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	return s.sensitiveRegex.ReplaceAll(body, []byte("$1=[REDACTED]"))
}
