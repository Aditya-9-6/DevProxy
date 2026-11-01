package certcomp

import (
	"bytes"
	"compress/zlib"
	"io"
	"sync"
)

// Algorithm identifies the TLS certificate compression algorithm per RFC 8879.
type Algorithm uint16

const (
	AlgZlib   Algorithm = 0x0001
	AlgBrotli Algorithm = 0x0002
	AlgZstd   Algorithm = 0x0003
)

// Codec defines the interface for TLS certificate compression and decompression.
type Codec interface {
	Algorithm() Algorithm
	Compress(dst *bytes.Buffer, src []byte) error
	Decompress(dst io.Writer, src []byte, limit int) (int64, error)
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// ZlibCodec implements RFC 8879 zlib certificate compression.
type ZlibCodec struct{}

func (ZlibCodec) Algorithm() Algorithm {
	return AlgZlib
}

func (ZlibCodec) Compress(dst *bytes.Buffer, src []byte) error {
	w := zlib.NewWriter(dst)
	if _, err := w.Write(src); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (ZlibCodec) Decompress(dst io.Writer, src []byte, limit int) (int64, error) {
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return 0, err
	}
	defer r.Close()

	limited := io.LimitReader(r, int64(limit+1))
	n, err := io.Copy(dst, limited)
	if err != nil {
		return n, err
	}
	if n > int64(limit) {
		return n, ErrExpansionRatioExceeded
	}
	return n, nil
}

// CodecRegistry manages available compression codecs.
type CodecRegistry struct {
	mu     sync.RWMutex
	codecs map[Algorithm]Codec
}

var defaultRegistry = &CodecRegistry{
	codecs: map[Algorithm]Codec{
		AlgZlib: ZlibCodec{},
	},
}

func Register(c Codec) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.codecs[c.Algorithm()] = c
}

func GetCodec(alg Algorithm) (Codec, bool) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	c, ok := defaultRegistry.codecs[alg]
	return c, ok
}