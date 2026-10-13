package proxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/compression"
)

// CompressHandler wraps an http.ResponseWriter to apply adaptive streaming compression.
type CompressHandler struct {
	w         http.ResponseWriter
	originalW http.ResponseWriter
	wc        io.WriteCloser
	started   bool
	closed    bool
	algo      compression.CompressionType
}

// NewCompressHandler creates a new compression response wrapper respecting client Accept-Encoding.
func NewCompressHandler(w http.ResponseWriter, r *http.Request) *CompressHandler {
	acceptEncoding := r.Header.Get("Accept-Encoding")
	mgr := compression.GetGlobalManager()
	rawAlgo := mgr.SelectAlgorithm(r.Header.Get("Content-Type"), 2048)

	algo := compression.CompressionNone
	if strings.Contains(acceptEncoding, "zstd") && (rawAlgo == compression.CompressionZstd || rawAlgo == "") {
		algo = compression.CompressionZstd
	} else if strings.Contains(acceptEncoding, "br") && (rawAlgo == compression.CompressionBrotli || rawAlgo == "") {
		algo = compression.CompressionBrotli
	} else if strings.Contains(acceptEncoding, "gzip") {
		algo = compression.CompressionGzip
	} else if strings.Contains(acceptEncoding, "*") {
		algo = rawAlgo
	}

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
	if h.algo != compression.CompressionNone {
		h.Header().Set("Content-Encoding", string(h.algo))
		h.Header().Del("Content-Length")
	}
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

func (h *CompressHandler) Flush() {
	if !h.started {
		h.WriteHeader(http.StatusOK)
	}
	if flusher, ok := h.wc.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	} else if flusher, ok := h.originalW.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (h *CompressHandler) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := h.originalW.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (h *CompressHandler) Close() error {
	if h.closed {
		return nil
	}
	h.closed = true
	if h.wc != nil {
		err := h.wc.Close()
		h.wc = nil
		return err
	}
	return nil
}
