package waf

import "testing"

func TestDeframeAndScan(t *testing.T) {
	scanner := NewSignatureScanner([][]byte{[]byte("malware")})
	// Frame: FIN=1, Opcode=1 (Text), Masked=0, Len=7, Payload="malware"
	frame := []byte{0x81, 0x07, 'm', 'a', 'l', 'w', 'a', 'r', 'e'}
	payload, fin := DeframeAndScan(frame, scanner)
	if !fin || string(payload) != "malware" {
		t.Errorf("Failed to de-frame correctly: got %s, fin=%v", string(payload), fin)
	}
}
