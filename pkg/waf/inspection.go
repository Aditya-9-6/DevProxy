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

// WebSocket_test.go implementation
// package waf
// import "testing"
// func TestDeframeAndScan(t *testing.T) {
// 	scanner := NewSignatureScanner([][]byte{[]byte("malware")})
// 	frame := []byte{0x81, 0x07, 0x00, 0x00, 0x00, 0x00, 'm', 'a', 'l', 'w', 'a', 'r', 'e'}
// 	payload, fin := DeframeAndScan(frame, scanner)
// 	if !fin || string(payload) != "malware" {
// 		t.Errorf("Failed to de-frame correctly")
// 	}
// }
