package qpack

import (
	"bytes"
	"context"
	"io"
	"sync"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/table"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/wire"
)

var encoderPool = sync.Pool{
	New: func() any {
		return &Encoder{
			dynamicTable: table.NewDynamicTable(4096),
		}
	},
}

// Encoder encodes HTTP/3 field sections into QPACK.
type Encoder struct {
	dynamicTable *table.DynamicTable
}

// NewEncoder retrieves or creates an Encoder instance.
func NewEncoder() *Encoder {
	return encoderPool.Get().(*Encoder)
}

// Release returns Encoder to pool.
func (e *Encoder) Release() {
	encoderPool.Put(e)
}

// Encode encodes header fields into destination writer.
func (e *Encoder) Encode(ctx context.Context, w io.Writer, fields []table.HeaderField) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var buf bytes.Buffer
	// Emit required insert count (0)
	var b [8]byte
	res, _ := wire.AppendVarInt(b[:0], 8, 0)
	buf.Write(res)

	// Emit delta base (0)
	res, _ = wire.AppendVarInt(b[:0], 7, 0)
	buf.Write(res)

	for _, f := range fields {
		buf.WriteString(f.Name)
		buf.WriteString(f.Value)
	}

	_, err := w.Write(buf.Bytes())
	return err
}
