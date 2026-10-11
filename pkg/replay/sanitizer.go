package replay

import (
	"net/http"
	"regexp"
)

var sensitiveHeaders = []string{"Authorization", "Cookie", "Set-Cookie", "Proxy-Authorization"}
var sensitiveRegex = regexp.MustCompile(`(?i)(password|secret|token|key)=[^&\s]+`)

// Sanitizer strips sensitive information from HTTP headers and bodies.
type Sanitizer struct{}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

func (s *Sanitizer) SanitizeHeaders(headers http.Header) {
	for _, h := range sensitiveHeaders {
		if headers.Get(h) != "" {
			headers.Set(h, "[REDACTED]")
		}
	}
}

func (s *Sanitizer) SanitizeBody(body []byte) []byte {
	return sensitiveRegex.ReplaceAll(body, []byte("$1=[REDACTED]"))
}
