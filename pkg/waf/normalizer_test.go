package waf

import (
	"bytes"
	"testing"
)

func TestNormalizer_NormalizeKey(t *testing.T) {
	n := NewNormalizer()
	scratch := make([]byte, 64)

	tests := []struct {
		input    string
		expected string
	}{
		{"content-type", "Content-Type"},
		{"x-forwarded-for", "X-Forwarded-For"},
		{"host", "Host"},
		{"ETAG", "Etag"},
		{"a-b-c-d", "A-B-C-D"},
	}

	for _, tc := range tests {
		res := n.NormalizeKey([]byte(tc.input), scratch)
		if string(res) != tc.expected {
			t.Errorf("NormalizeKey(%q) = %q, expected %q", tc.input, string(res), tc.expected)
		}
	}
}

func TestNormalizer_NormalizeString(t *testing.T) {
	n := NewNormalizer()
	res := n.NormalizeString("accept-encoding")
	if res != "Accept-Encoding" {
		t.Errorf("NormalizeString failed: got %q", res)
	}
}

func TestFastContainsCRLF(t *testing.T) {
	if !FastContainsCRLF([]byte("foo\r\nbar")) {
		t.Error("Expected CRLF detection to be true")
	}
	if FastContainsCRLF([]byte("foobar")) {
		t.Error("Expected CRLF detection to be false")
	}
}

func TestSanitizer_SanitizeValue(t *testing.T) {
	s := NewSanitizer()

	_, err := s.SanitizeValue([]byte("valid-value"))
	if err != nil {
		t.Errorf("Unexpected error on valid header value: %v", err)
	}

	_, err = s.SanitizeValue([]byte("malicious\r\nInjected: header"))
	if !bytes.Equal([]byte(err.Error()), []byte(ErrHeaderInjection.Error())) {
		t.Errorf("Expected ErrHeaderInjection, got %v", err)
	}
}

func TestSanitizer_SanitizeHeaderTuple(t *testing.T) {
	s := NewSanitizer()
	scratch := make([]byte, 32)

	k, v, err := s.SanitizeHeaderTuple([]byte("user-agent"), []byte("DevProxy/1.0"), scratch)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(k) != "User-Agent" || string(v) != "DevProxy/1.0" {
		t.Errorf("Tuple sanitization failed: key=%q val=%q", string(k), string(v))
	}
}
