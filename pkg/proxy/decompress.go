package proxy

import (
	"compress/flate"
	"compress/gzip"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const maxDecompressedSize = 10 * 1024 * 1024 // 10MB limit to prevent decompression bombs

// GetDecompressor returns an io.ReadCloser that decompresses the given reader based on the encoding.
// It enforces a size limit on the decompressed stream.
func GetDecompressor(r io.Reader, encoding string) (io.ReadCloser, error) {
	limitedReader := io.LimitReader(r, maxDecompressedSize)

	switch encoding {
	case "gzip":
		zr, err := gzip.NewReader(limitedReader)
		if err != nil {
			return nil, err
		}
		return zr, nil
	case "deflate":
		return io.NopCloser(flate.NewReader(limitedReader)), nil
	case "br":
		return io.NopCloser(brotli.NewReader(limitedReader)), nil
	case "zstd":
		zr, err := zstd.NewReader(limitedReader)
		if err != nil {
			return nil, err
		}
		return zr.IOReadCloser(), nil
	default:
		return io.NopCloser(limitedReader), nil
	}
}
