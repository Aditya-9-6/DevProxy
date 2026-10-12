package proxy

import (
	"io"
	"net/http"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/compression"
)

// CompressHandler wraps an http.ResponseWriter to apply adaptive streaming compression.
type CompressHandler struct {
	w         http.ResponseWriter
	originalW http.ResponseWriter
	wc        io.WriteCloser
	started   bool
	algo      compression.CompressionType
}

// NewCompressHandler creates a new compression response wrapper.
func NewCompressHandler(w http.ResponseWriter, r *http.Request) *CompressHandler {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	_ = acceptEncoding
	mgr := compression.GetGlobalManager()
	algo := mgr.SelectAlgorithm(r.Header.Get("Content-Type"), 2048)

	return &CompressHandler{
		w:         w,
		originalW: w,
		algo:      algo,
	}
}

func (h *CompressHandler) Header() http.Header {
	return h.w.Header()
}

func (h *CompressHandler) WriteHeader(statusCode int) {
	if h.started {
		return
	}
	h.started = true
	h.Header().Set("Content-Encoding", string(h.algo))
	h.Header().Del("Content-Length") // Chunked transfer encoding
	h.originalW.WriteHeader(statusCode)

	start := time.Now()
	wc, err := compression.CompressStream(h.originalW, h.algo)
	if err == nil {
		h.wc = wc
	}
	compression.GetGlobalManager().RecordLatency(h.algo, time.Since(start))
}

func (h *CompressHandler) Write(p []byte) (int, error) {
	if !h.started {
		h.WriteHeader(http.StatusOK)
	}
	if h.wc != nil {
		return h.wc.Write(p)
	}
	return h.originalW.Write(p)
}

func (h *CompressHandler) Close() error {
	if h.wc != nil {
		return h.wc.Close()
	}
	return nil
}
