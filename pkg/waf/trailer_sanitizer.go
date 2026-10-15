package waf

import "strings"

// TrailerSanitizer inspects HTTP trailers for smuggling or injection attempts.
type TrailerSanitizer struct {
	anomalies int
}

func NewTrailerSanitizer() *TrailerSanitizer {
	return &TrailerSanitizer{}
}

func (s *TrailerSanitizer) SanitizeChunk(chunk []byte) bool {
	content := string(chunk)
	if strings.Contains(content, "Transfer-Encoding:") || strings.Contains(content, "Content-Length:") {
		s.anomalies++
		return false
	}
	return true
}

func (s *TrailerSanitizer) ValidateTrailer(trailer []byte) bool {
	// Basic validation: trailers should not contain control characters or specific headers
	content := string(trailer)
	if strings.Contains(content, "\r\n\r\n") {
		return false
	}
	return true
}

func (s *TrailerSanitizer) GetAnomalyCount() int {
	return s.anomalies
}
