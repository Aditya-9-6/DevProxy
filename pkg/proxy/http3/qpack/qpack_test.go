package qpack

import (
	"bytes"
	"context"
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/table"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http3/qpack/wire"
)

func TestVarIntRoundtrip(t *testing.T) {
	vals := []uint64{0, 1, 31, 32, 63, 64, 127, 128, 16383, 16384, 1000000}
	for _, v := range vals {
		var buf [10]byte
		buf[0] = 0
		res, err := wire.AppendVarInt(buf[:1], 6, v)
		if err != nil {
			t.Fatalf("AppendVarInt(%d) failed: %v", v, err)
		}

		parsed, _, err := wire.ReadVarInt(bytes.NewReader(res), 6)
		if err != nil {
			t.Fatalf("ReadVarInt failed for %d: %v", v, err)
		}

		if parsed != v {
			t.Errorf("expected %d, got %d", v, parsed)
		}
	}
}

func TestDynamicTableCapacity(t *testing.T) {
	dt := table.NewDynamicTable(100)
	err := dt.Add(":path", "/index.html")
	if err != nil {
		t.Fatalf("unexpected add error: %v", err)
	}

	if dt.Size() == 0 {
		t.Errorf("expected non-zero table size")
	}

	dt.SetCapacity(10)
	if dt.Size() > 10 {
		t.Errorf("table size %d exceeded new capacity 10", dt.Size())
	}
}

func TestEncoderDecoderRoundtrip(t *testing.T) {
	ctx := context.Background()
	enc := NewEncoder()
	defer enc.Release()

	dec := NewDecoder(DefaultLimits())
	defer dec.Release()

	fields := []table.HeaderField{
		{Name: ":method", Value: "GET"},
		{Name: ":path", Value: "/"},
	}

	var buf bytes.Buffer
	err := enc.Encode(ctx, &buf, fields)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := dec.Decode(ctx, &buf)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if len(decoded) == 0 {
		t.Errorf("expected decoded fields, got none")
	}
}
