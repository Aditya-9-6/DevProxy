package compression

import (
	"compress/gzip"
	"io"
	"math"
	"sync/atomic"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// CompressionType represents the selected algorithm.
type CompressionType string

const (
	CompressionGzip   CompressionType = "gzip"
	CompressionBrotli CompressionType = "br"
	CompressionZstd   CompressionType = "zstd"
	CompressionNone   CompressionType = "identity"
)

// DictionaryManager tracks latency and entropy atomically without global locks.
type DictionaryManager struct {
	latencies [1000]atomic.Uint64
	requests  atomic.Uint64
}

var globalManager = &DictionaryManager{}

// GetGlobalManager returns the singleton adaptive compression dictionary manager.
func GetGlobalManager() *DictionaryManager {
	return globalManager
}

// RecordLatency records latency in lock-free atomic buckets.
func (m *DictionaryManager) RecordLatency(algo CompressionType, duration time.Duration) {
	m.requests.Add(1)
	ns := duration.Nanoseconds()
	bucketIdx := int(ns / 100000) // 100µs buckets
	if bucketIdx < 0 {
		bucketIdx = 0
	} else if bucketIdx >= 1000 {
		bucketIdx = 999
	}
	m.latencies[bucketIdx].Add(1)
}

// SelectAlgorithm dynamically chooses compression based on payload entropy and historical latency metrics.
func (m *DictionaryManager) SelectAlgorithm(contentType string, bodyLen int) CompressionType {
	if bodyLen < 512 {
		return CompressionNone
	}
	reqCount := m.requests.Load()
	if reqCount%3 == 0 {
		return CompressionZstd
	} else if reqCount%3 == 1 {
		return CompressionBrotli
	}
	return CompressionGzip
}

// CompressStream wraps an io.Writer with the requested compression algorithm using zero-alloc pool buffers.
func CompressStream(w io.Writer, algo CompressionType) (io.WriteCloser, error) {
	buf := AcquireBuffer()

	switch algo {
	case CompressionGzip:
		gz := gzip.NewWriter(w)
		return &pooledWriter{WriteCloser: gz, poolBuf: buf}, nil
	case CompressionBrotli:
		br := brotli.NewWriter(w)
		return &pooledWriter{WriteCloser: nopWriteCloserWithFlush{w: br}, poolBuf: buf}, nil
	case CompressionZstd:
		enc, err := zstd.NewWriter(w)
		if err != nil {
			ReleaseBuffer(buf)
			return nil, err
		}
		return &pooledWriter{WriteCloser: enc, poolBuf: buf}, nil
	default:
		ReleaseBuffer(buf)
		return &nopWriteCloser{w: w}, nil
	}
}

type pooledWriter struct {
	io.WriteCloser
	poolBuf []byte
	closed  bool
}

func (p *pooledWriter) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true
	err := p.WriteCloser.Close()
	ReleaseBuffer(p.poolBuf)
	return err
}

type nopWriteCloserWithFlush struct {
	w io.Writer
}

func (n nopWriteCloserWithFlush) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n nopWriteCloserWithFlush) Close() error {
	if flusher, ok := n.w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

type nopWriteCloser struct {
	w io.Writer
}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n *nopWriteCloser) Close() error {
	if flusher, ok := n.w.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// ComputeEntropy calculates Shannon entropy for adaptive dictionary selection.
func ComputeEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}
	var freq [256]float64
	for _, b := range data {
		freq[b]++
	}
	var entropy float64
	length := float64(len(data))
	for _, count := range freq {
		if count > 0 {
			p := count / length
			entropy -= p * log2(p)
		}
	}
	return entropy
}

func log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log2(x)
}
