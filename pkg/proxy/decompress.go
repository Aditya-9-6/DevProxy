package proxy

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

const maxDecompressedSize = 10 * 1024 * 1024

var (
	zstdPool = sync.Pool{
		New: func() interface{} {
			dec, _ := zstd.NewReader(nil)
			return dec
		},
	}
	brPool = sync.Pool{
		New: func() interface{} {
			return brotli.NewReader(nil)
		},
	}
)

type zstdReadCloser struct {
	*zstd.Decoder
}

func (z *zstdReadCloser) Close() error {
	// Close the underlying decoder or let it be returned to pool safely
	zstdPool.Put(z.Decoder)
	return nil
}

type brotliReadCloser struct {
	*brotli.Reader
}

func (b *brotliReadCloser) Close() error {
	brPool.Put(b.Reader)
	return nil
}

type deflateReadCloser struct {
	io.ReadCloser
	fr io.ReadCloser
}

func (d *deflateReadCloser) Close() error {
	if d.fr != nil {
		return d.fr.Close()
	}
	return nil
}

func GetDecompressor(r io.Reader, encoding string) (io.ReadCloser, error) {
	limitedReader := io.LimitReader(r, maxDecompressedSize)

	switch encoding {
	case "gzip":
		return gzip.NewReader(limitedReader)
	case "deflate":
		fr := flate.NewReader(limitedReader)
		return &deflateReadCloser{
			ReadCloser: io.NopCloser(fr),
			fr:         fr,
		}, nil
	case "br":
		br := brPool.Get().(*brotli.Reader)
		br.Reset(limitedReader)
		return &brotliReadCloser{Reader: br}, nil
	case "zstd":
		dec := zstdPool.Get().(*zstd.Decoder)
		if err := dec.Reset(limitedReader); err != nil {
			zn := dec
			_ = zn
			zstdPool.Put(dec)
			return nil, err
		}
		return &zstdReadCloser{Decoder: dec}, nil
	default:
		return io.NopCloser(limitedReader), nil
	}
}
