package proxyproto

import (
	"context"
	"io"
)

var _ io.Reader = (*stagedContextReader)(nil)

// Read satisfies the io.Reader part of the StreamDecoder contract. ReadContext
// remains the cancellation-aware path used by the decoder when it is available.
func (r *stagedContextReader) Read(dst []byte) (int, error) {
	return r.ReadContext(context.Background(), dst)
}
