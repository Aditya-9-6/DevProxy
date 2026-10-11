package proxy

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

var readerPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// GetDecompressor returns an io.ReadCloser that decompresses the given reader based on the encoding.
func GetDecompressor(r io.Reader, encoding string) (io.ReadCloser, error) {
	switch encoding {
	case "gzip":
		return gzip.NewReader(r)
	case "deflate":
		return flate.NewReader(r), nil
	case "br":
		return io.NopCloser(brotli.NewReader(r)), nil
	case "zstd":
		zr, err := zstd.NewReader(r)
		if err != nil {
			return nil, err
		}
		return zr.IOReadCloser(), nil
	default:
		return io.NopCloser(r), nil
	}
}
