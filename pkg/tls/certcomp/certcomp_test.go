package certcomp

import (
	"bytes"
	"testing"
)

func TestZlibCodecAndMessageRoundtrip(t *testing.T) {
	codec := ZlibCodec{}
	original := []byte("---BEGIN MOCK TLS CERTIFICATE CHAIN---")

	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	err := codec.Compress(buf, original)
	if err != nil {
		Fatalf(t, "Compress failed: %v", err)
	}

	msgBytes := MarshalCompressedCertificate(AlgZlib, uint32(len(original)), buf.Bytes())

	parsed, err := ParseCompressedCertificate(bytes.NewReader(msgBytes), DefaultLimits())
	if err != nil {
		Fatalf(t, "ParseCompressedCertificate failed: %v", err)
	}

	if parsed.Algorithm != AlgZlib {
		Errorf(t, "expected alg Zlib, got %v", parsed.Algorithm)
	}

	outBuf := new(bytes.Buffer)
	n, err := codec.Decompress(outBuf, parsed.CompressedCertData, DefaultLimits().MaxUncompressedSize)
	if err != nil {
		Fatalf(t, "Decompress failed: %v", err)
	}

	if n != int64(len(original)) || !bytes.Equal(outBuf.Bytes(), original) {
		Errorf(t, "decompressed output mismatch: got %s", outBuf.String())
	}
}

func TestNegotiator(t *testing.T) {
	neg := NewNegotiator([]Algorithm{AlgZstd, AlgZlib})
	neg.SetPeerSupported([]Algorithm{AlgZlib, AlgBrotli})

	alg, ok := neg.Negotiate(AlgZstd)
	if !ok || alg != AlgZlib {
		Errorf(t, "expected negotiation to fall back/agree on AlgZlib, got %v (ok=%v)", alg, ok)
	}
}

func Fatalf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}

func Errorf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Errorf(format, args...)
}