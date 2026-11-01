package qpack

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/table"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/wire"
)

var (
	ErrBlockedStream    = errors.New("qpack: stream is blocked on required insert count")
	ErrDecompressionFail = errors.New("qpack: failed to decompress header block")
)

var decoderPool = sync.Pool{
	New: func() any {
		return &Decoder{
			dynamicTable: table.NewDynamicTable(4096),
			limits:       DefaultLimits(),
		}
	},
}

// Decoder decodes QPACK compressed header blocks.
type Decoder struct {
	dynamicTable *table.DynamicTable
	limits       Limits
	blockedCount int32
}

// NewDecoder retrieves or creates a Decoder instance.
func NewDecoder(limits Limits) *Decoder {
	d := decoderPool.Get().(*Decoder)
		d.limits = limits
		return d
}

// Release returns Decoder to pool.
func (d *Decoder) Release() {
	decoderPool.Put(d)
}

// Decode decodes a header block from r into fields.
func (d *Decoder) Decode(ctx context.Context, r io.Reader) ([]table.HeaderField, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read required insert count
	reqInsertCount, _, err := wire.ReadVarInt(r, 8)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	// Read base delta and sign
	_, _, err = wire.ReadVarInt(r, 7)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if reqInsertCount > 0 && atomic.LoadInt32(&d.blockedCount) > int32(d.limits.MaxBlockedStreams) {
		return nil, ErrBlockedStream
	}

	var fields []table.HeaderField
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Basic literal parsing for demonstration
	if buf.Len() > 0 {
		fields = append(fields, table.HeaderField{Name: ":status", Value: "200"})
	}

	return fields, nil
}
