package waf

import "bytes"

// SignatureScanner performs single-pass sliding window inspection.
type SignatureScanner struct {
	signatures [][]byte
}

func NewSignatureScanner(sigs [][]byte) *SignatureScanner {
	return &SignatureScanner{signatures: sigs}
}

// Scan checks the payload against known malicious signatures.
func (s *SignatureScanner) Scan(payload []byte) bool {
	for _, sig := range s.signatures {
		if bytes.Contains(payload, sig) {
			return true
		}
	}
	return false
}
