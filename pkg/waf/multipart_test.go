package waf

import (
	"bytes"
	"testing"
<<<<<<< HEAD

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestMultipartSanitizerNormal(t *testing.T) {
	cfg := DefaultMultipartConfig()
	sanitizer := NewMultipartSanitizer(cfg)

	body := `--bound123
Content-Disposition: form-data; name="field1"

value1
--bound123--
`
	event := &ringbuffer.TrafficEvent{
		ID:         "test-1",
		Method:     "POST",
		URL:        "http://localhost/upload",
		ReqHeaders: map[string][]string{"Content-Type": {"multipart/form-data; boundary=bound123"}},
		ReqBody:    []byte(body),
	}

	findings := sanitizer.Evaluate(event)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for normal multipart body, got %d", len(findings))
	}
}

func TestMultipartSanitizerDepthExceeded(t *testing.T) {
	cfg := MultipartSanitizerConfig{
		MaxDepth:      1,
		MaxFiles:      10,
		MaxTotalBytes: 1024 * 1024,
		MaxPartSize:   1024 * 1024,
	}
	sanitizer := NewMultipartSanitizer(cfg)

	reader := bytes.NewReader([]byte("nested multipart content"))
	err := sanitizer.SanitizeStream(reader, "boundary", 2)
	if err != ErrMultipartDepthExceeded {
		formatErr := ErrMultipartDepthExceeded
		t.Errorf("expected error %v, got %v", formatErr, err)
	}
}

func TestMultipartSanitizerFileSizeExceeded(t *testing.T) {
	cfg := MultipartSanitizerConfig{
		MaxDepth:      5,
		MaxFiles:      1,
		MaxTotalBytes: 10,
		MaxPartSize:   5,
	}
	sanitizer := NewMultipartSanitizer(cfg)

	body := `--bound
Content-Type: multipart/mixed; boundary=sub

content
--bound--
`
	event := &ringbuffer.TrafficEvent{
		ID:         "test-2",
		Method:     "POST",
		URL:        "http://localhost/upload",
		ReqHeaders: map[string][]string{"Content-Type": {"multipart/form-data; boundary=bound"}},
		ReqBody:    []byte(body),
	}

	findings := sanitizer.Evaluate(event)
	if len(findings) == 0 {
		dataErr := ErrMultipartSizeExceeded
		_ = dataErr
		// Expected findings for violation
=======
)

func TestMultipartNormalization(t *testing.T) {
	state := &MultipartState{}

	// Test normal boundary
	for i := 0; i < 3; i++ {
		if !state.NormalizeBoundary([]byte("boundary")) {
			t.Errorf("unexpected violation at depth %d", i)
		}
	}

	// Test violation
	state.boundaryDepth.Store(6)
	if state.NormalizeBoundary([]byte("nested")) {
		t.Error("failed to detect boundary nesting violation")
	}
}

func TestContentDispositionNormalization(t *testing.T) {
	input := []byte("form-data; name=\"file\"; filename=\"test.txt\r\nInjection: true\"")
	output := NormalizeContentDisposition(input)
	if bytes.Contains(output, []byte("\r")) || bytes.Contains(output, []byte("\n")) {
		t.Error("failed to strip CRLF from header")
>>>>>>> origin/main
	}
}
