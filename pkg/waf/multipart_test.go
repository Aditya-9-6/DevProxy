package waf

import (
	"bytes"
	"net/http"
	"sync"
	"testing"

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
	hdrs := make(http.Header)
	hdrs.Set("Content-Type", "multipart/form-data; boundary=bound123")
	event := &ringbuffer.TrafficEvent{
		ID:         "test-1",
		Method:     "POST",
		URL:        "http://localhost/upload",
		ReqHeaders: hdrs,
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
		t.Errorf("expected error %v, got %v", ErrMultipartDepthExceeded, err)
	}
}

func TestMultipartSanitizerConcurrent(t *testing.T) {
	t.Parallel()
	cfg := DefaultMultipartConfig()
	sanitizer := NewMultipartSanitizer(cfg)

	body := `--bound123
Content-Disposition: form-data; name="field1"

value1
--bound123--
`
	var wg sync.WaitGroup
	concurrency := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			hdrs := make(http.Header)
			hdrs.Set("Content-Type", "multipart/form-data; boundary=bound123")
			event := &ringbuffer.TrafficEvent{
				ID:         "test-conc",
				Method:     "POST",
				URL:        "http://localhost/upload",
				ReqHeaders: hdrs,
				ReqBody:    []byte(body),
			}
			findings := sanitizer.Evaluate(event)
			if len(findings) != 0 {
				t.Errorf("expected 0 findings in concurrent request, got %d", len(findings))
			}
		}(i)
	}
	wg.Wait()
}

func TestMultipartNormalization(t *testing.T) {
	state := &MultipartState{}

	for i := 0; i < 3; i++ {
		if !state.NormalizeBoundary([]byte("boundary")) {
			t.Errorf("unexpected violation at depth %d", i)
		}
	}

	state.boundaryDepth.Store(6)
	if state.NormalizeBoundary([]byte("nested")) {
		t.Error("failed to detect boundary nesting violation")
	}
}
