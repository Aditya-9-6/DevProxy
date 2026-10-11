package proxy

import (
	"bytes"
	"testing"
)

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		domain  string
		want    bool
	}{
		{"*.example.com", "api.example.com", true},
		{"*.example.com", "example.com", false},
		{"example.com", "example.com", true},
	}
	for _, tt := range tests {
		if got := MatchWildcard(tt.pattern, tt.domain); got != tt.want {
			t.Errorf("MatchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.domain, got, tt.want)
		}
	}
}

func TestExtractSNI(t *testing.T) {
	// Construct a minimal valid TLS ClientHello with SNI "example.com"
	validClientHello := []byte{
		0x16, 0x03, 0x01, 0x00, 0x2e, // TLS Record Header (length 46)
		0x01, 0x00, 0x00, 0x2a, // Handshake ClientHello (length 42)
		0x03, 0x03, // Version TLS 1.2
		// 32 bytes Random
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0x00,                   // Session ID Length
		0x00, 0x02, 0x00, 0x2f, // Cipher Suites (2 bytes length, cipher 0x002f)
		0x01, 0x00, // Compression Methods (1 byte length, 1 method)
		0x00, 0x0b, // Extensions Length (11 bytes)
		0x00, 0x00, 0x00, 0x07, // Extension SNI type 0, data length 7
		0x00, 0x05, 0x00, 0x00, 0x02, // SNI list len (5), type 0 (hostname), len 2 (host)
		'h', 's',
	}

	tests := []struct {
		name    string
		input   []byte
		wantSNI string
		wantErr bool
	}{
		{
			name:    "Valid ClientHello",
			input:   validClientHello,
			wantSNI: "hs",
			wantErr: false,
		},
		{
			name:    "Insufficient data",
			input:   []byte{0x16, 0x03},
			wantSNI: "",
			wantErr: true,
		},
		{
			name:    "Not TLS handshake",
			input:   []byte{0x15, 0x03, 0x01, 0x00, 0x05, 0x01, 0x02, 0x03, 0x04, 0x05},
			wantSNI: "",
			wantErr: true,
		},
		{
			name:    "Not ClientHello",
			input:   []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x02, 0x02, 0x03, 0x04, 0x05},
			wantSNI: "",
			wantErr: true,
		},
		{
			name:    "Truncated input",
			input:   validClientHello[:20],
			wantSNI: "",
			wantErr: true,
		},
	}

	router := NewSNIRouter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sni, err := router.ExtractSNI(bytes.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractSNI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if sni != tt.wantSNI {
				t.Errorf("ExtractSNI() sni = %v, wantSNI %v", sni, tt.wantSNI)
			}
		})
	}
}
