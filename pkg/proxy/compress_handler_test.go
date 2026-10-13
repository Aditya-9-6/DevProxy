package proxy

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/compression"
)

// mockResponseWriter implements http.ResponseWriter, http.Flusher, and http.Hijacker for testing compliance.
type mockResponseWriter struct {
	http.ResponseWriter
	flushed  bool
	hijacked bool
}

func (m *mockResponseWriter) Flush() {
	m.flushed = true
}

func (m *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

func TestCompressHandlerInterfaceCompliance(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	handler := NewCompressHandler(rec, req)

	var _ http.ResponseWriter = handler
	var _ http.Flusher = handler
	var _ http.Hijacker = handler

	handler.WriteHeader(http.StatusOK)
	handler.WriteHeader(http.StatusCreated) // Should be ignored due to duplicate call protection

	_, err := handler.Write([]byte("hello compression"))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	handler.Flush()

	if err := handler.Close(); err != nil {
		t.Fatalf("failed to close handler: %v", err)
	}

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %s", rec.Header().Get("Content-Encoding"))
	}
}

func TestCompressHandlerHijackAndFlusherDelegation(t *testing.T) {
	t.Parallel()
	mock := &mockResponseWriter{ResponseWriter: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)

	handler := NewCompressHandler(mock, req)

	_, _, _ = handler.Hijack()
	if !mock.hijacked {
		iterErr := "expected underlying Hijacker to be invoked"
		t.Error(iterErr)
	}

	handler.Flush()
	if !mock.flushed {
		t.Error("expected underlying Flusher to be invoked")
	}
}

func TestCompressHandlerNoCompression(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header

	handler := NewCompressHandler(rec, req)
	if handler.algo != compression.CompressionNone {
		t.Errorf("expected CompressionNone, got %s", handler.algo)
	}

	_, _ = handler.Write([]byte("identity payload"))
	_ = handler.Close()

	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("did not expect Content-Encoding header for identity compression")
	}
}
