package waf

import (
	"testing"
)

func TestDeframeAndScan(t *testing.T) {
	scanner := NewSignatureScanner([][]byte{[]byte("malware")})
	state := &FrameState{}

	tests := []struct {
		name  string
		frame []byte
		fin   bool
		want  string
	}{
		{"Simple", []byte{0x81, 0x07, 'm', 'a', 'l', 'w', 'a', 'r', 'e'}, true, "malware"},
		{"Masked", []byte{0x81, 0x87, 0x01, 0x02, 0x03, 0x04, 'm' ^ 0x01, 'a' ^ 0x02, 'l' ^ 0x03, 'w' ^ 0x04, 'a' ^ 0x01, 'r' ^ 0x02, 'e' ^ 0x03}, true, "malware"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, fin := DeframeAndScan(tt.frame, scanner, state)
			if fin != tt.fin || string(payload) != tt.want {
				t.Errorf("got %s, fin=%v; want %s, fin=%v", string(payload), fin, tt.want, tt.fin)
			}
		})
	}
}
