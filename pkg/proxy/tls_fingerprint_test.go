package proxy

import (
	"encoding/binary"
	"strings"
	"testing"
)

func buildMockClientHello(version uint16, ciphers []uint16, exts []uint16, sni string) []byte {
	// Construct body
	body := make([]byte, 0, 512)
	// Version
	vBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(vBuf, version)
	body = append(body, vBuf...)
	// Random 32 bytes
	body = append(body, make([]byte, 32)...)
	// Session ID len 0
	body = append(body, 0)
	// Ciphers
	cLen := make([]byte, 2)
	binary.BigEndian.PutUint16(cLen, uint16(len(ciphers)*2))
	body = append(body, cLen...)
	for _, c := range ciphers {
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf, c)
		body = append(body, buf...)
	}
	// Compression (1 byte len + 0x00)
	body = append(body, 1, 0)

	// Extensions payload
	extPayload := make([]byte, 0, 256)
	// If SNI
	if sni != "" {
		sniExt := make([]byte, 0, 64)
		// ext type 0x0000
		sniExt = append(sniExt, 0x00, 0x00)
		// sni data
		sniData := make([]byte, 0, 64)
		// list len
		listLen := uint16(len(sni) + 3)
		sniData = binary.BigEndian.AppendUint16(sniData, listLen)
		sniData = append(sniData, 0x00) // host_name type
		sniData = binary.BigEndian.AppendUint16(sniData, uint16(len(sni)))
		sniData = append(sniData, []byte(sni)...)

		sniExt = binary.BigEndian.AppendUint16(sniExt, uint16(len(sniData)))
		sniExt = append(sniExt, sniData...)
		extPayload = append(extPayload, sniExt...)
	}

	for _, e := range exts {
		if e == 0x0000 && sni != "" {
			continue // Already appended
		}
		extPayload = binary.BigEndian.AppendUint16(extPayload, e)
		extPayload = binary.BigEndian.AppendUint16(extPayload, 0) // len 0
	}

	eLen := make([]byte, 2)
	binary.BigEndian.PutUint16(eLen, uint16(len(extPayload)))
	body = append(body, eLen...)
	body = append(body, extPayload...)

	// Handshake header
	handshake := make([]byte, 4)
	handshake[0] = handshakeTypeClientHello
	handshake[1] = byte(len(body) >> 16)
	handshake[2] = byte(len(body) >> 8)
	handshake[3] = byte(len(body))
	handshake = append(handshake, body...)

	// TLS record header
	record := make([]byte, 5)
	record[0] = recordTypeHandshake
	record[1] = 0x03
	record[2] = 0x03
	binary.BigEndian.PutUint16(record[3:5], uint16(len(handshake)))
	record = append(record, handshake...)

	return record
}

func TestParseClientHello_Success(t *testing.T) {
	ciphers := []uint16{0x1301, 0x1302, 0x0a0a, 0xc02b} // 0x0a0a is GREASE
	exts := []uint16{0x0000, 0x000a, 0x1a1a, 0x0023}    // 0x1a1a is GREASE
	raw := buildMockClientHello(0x0303, ciphers, exts, "example.com")

	data, err := ParseClientHello(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing ClientHello: %v", err)
	}

	if data.Version != 0x0303 {
		t.Errorf("expected version 0x0303, got 0x%04x", data.Version)
	}
	if data.SNI != "example.com" {
		t.Errorf("expected SNI 'example.com', got %q", data.SNI)
	}
	if len(data.CipherSuites) != 3 { // GREASE 0x0a0a filtered
		t.Errorf("expected 3 ciphers (filtered GREASE), got %d", len(data.CipherSuites))
	}

	rawJA3, hashJA3 := ComputeJA3(data)
	if rawJA3 == "" || len(hashJA3) != 32 {
		t.Errorf("invalid JA3 output: raw=%q, hash=%q", rawJA3, hashJA3)
	}

	ja4 := ComputeJA4(data, true)
	if !strings.HasPrefix(ja4, "t12d") {
		t.Errorf("expected JA4 to start with 't12d', got %q", ja4)
	}
}

func TestParseClientHello_Malformed(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"nil slice", nil},
		{"empty slice", []byte{}},
		{"short header", []byte{0x16, 0x03}},
		{"wrong record type", []byte{0x17, 0x03, 0x03, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00}},
		{"truncated payload", []byte{0x16, 0x03, 0x03, 0x00, 0x50, 0x01, 0x00}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseClientHello(tc.data)
			if err == nil {
				t.Errorf("expected error for malformed case %s, got nil", tc.name)
			}
		})
	}
}

func TestClassifyClient(t *testing.T) {
	tests := []struct {
		ua       string
		ja3      string
		expected string
	}{
		{"curl/8.1.2", "abc", "bot/curl"},
		{"python-requests/2.31.0", "abc", "bot/python-requests"},
		{"Go-http-client/1.1", "abc", "bot/golang"},
		{"PostmanRuntime/7.32.3", "abc", "tool/postman"},
		{"Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36", "abc", "browser/chrome"},
		{"Mozilla/5.0 Firefox/121.0", "abc", "browser/firefox"},
		{"Mozilla/5.0 Version/17.0 Safari/605.1.15", "abc", "browser/safari"},
		{"CustomClient/1.0", "abc123md5", "client/custom-tls"},
		{"Unknown", "", "client/generic"},
	}

	for _, tc := range tests {
		result := ClassifyClient(tc.ja3, tc.ua)
		if result != tc.expected {
			t.Errorf("ClassifyClient(%q) = %q; want %q", tc.ua, result, tc.expected)
		}
	}
}
