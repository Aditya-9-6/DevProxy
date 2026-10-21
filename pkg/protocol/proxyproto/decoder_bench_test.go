package proxyproto

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecoderZeroAllocations(t *testing.T) {
	frame := benchmarkHeader()
	reader := bytes.NewReader(frame)
	decoder := V2Decoder{}

	allocations := testing.AllocsPerRun(1000, func() {
		reader.Reset(frame)
		header, err := decoder.Decode(reader)
		if err != nil {
			panic(err)
		}
		if header.Source.Port() == 0 {
			panic("decoded source port is zero")
		}
	})
	if allocations != 0 {
		t.Fatalf("Decode() allocations = %v, want 0", allocations)
	}
}

func BenchmarkDecode(b *testing.B) {
	frame := benchmarkHeader()
	reader := bytes.NewReader(frame)
	decoder := V2Decoder{}

	b.ReportAllocs()
	b.SetBytes(int64(len(frame)))
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		reader.Reset(frame)
		header, err := decoder.Decode(reader)
		if err != nil {
			b.Fatal(err)
		}
		if header.Source.Port() == 0 {
			b.Fatal("decoded source port is zero")
		}
	}
}

func BenchmarkDecodeMaximumPayload(b *testing.B) {
	frame := make([]byte, HeaderSize+MaxPayloadLength)
	copy(frame[:12], Signature)
	frame[12] = 0x20
	binary.BigEndian.PutUint16(frame[14:16], MaxPayloadLength)
	reader := bytes.NewReader(frame)
	decoder := V2Decoder{}

	b.ReportAllocs()
	b.SetBytes(int64(len(frame)))
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		reader.Reset(frame)
		if _, err := decoder.Decode(reader); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkHeader() []byte {
	payload := []byte{
		192, 0, 2, 1,
		198, 51, 100, 2,
		0x1f, 0x90,
		0x00, 0x50,
	}
	frame := make([]byte, HeaderSize+len(payload))
	copy(frame[:12], Signature)
	frame[12] = 0x20 | byte(CommandProxy)
	frame[13] = byte(FamilyINET)<<4 | byte(ProtocolStream)
	binary.BigEndian.PutUint16(frame[14:16], uint16(len(payload)))
	copy(frame[HeaderSize:], payload)
	return frame
}
