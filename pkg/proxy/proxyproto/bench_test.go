package proxyproto

import (
	"encoding/binary"
	"runtime"
	"testing"
)

func BenchmarkDecode(b *testing.B) {
	for _, size := range []int{64, 1024, 16384} {
		frame := benchmarkFrame(b, size)
		b.Run(frameSizeName(size), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(frame)))
			for index := 0; index < b.N; index++ {
				decoded, err := Decode(frame)
				if err != nil {
					b.Fatal(err)
				}
				runtime.KeepAlive(decoded)
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	for _, size := range []int{64, 1024, 16384} {
		frame := benchmarkFrameValue(b, size)
		encoded := encodeBenchmarkFrame(b, frame)
		b.Run(frameSizeName(size), func(b *testing.B) {
			buffer := make([]byte, MaxFrameSize)
			b.ReportAllocs()
			b.SetBytes(int64(len(encoded)))
			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				written, err := Encode(buffer, frame)
				if err != nil {
					b.Fatal(err)
				}
				runtime.KeepAlive(written)
			}
		})
	}
}

func TestDecodeAllocations(t *testing.T) {
	frame := benchmarkFrame(t, 1024)
	allocations := testing.AllocsPerRun(1000, func() {
		decoded, err := Decode(frame)
		if err != nil {
			panic(err)
		}
		runtime.KeepAlive(decoded)
	})
	if allocations != 0 {
		t.Fatalf("Decode() allocations = %v, want 0", allocations)
	}
}

func TestEncodeAllocations(t *testing.T) {
	frame := benchmarkFrameValue(t, 1024)
	buffer := make([]byte, MaxFrameSize)
	allocations := testing.AllocsPerRun(1000, func() {
		written, err := Encode(buffer, frame)
		if err != nil {
			panic(err)
		}
		runtime.KeepAlive(written)
	})
	if allocations != 0 {
		t.Fatalf("Encode() allocations = %v, want 0", allocations)
	}
}

func benchmarkFrame(tb testing.TB, size int) []byte {
	tb.Helper()
	return encodeBenchmarkFrame(tb, benchmarkFrameValue(tb, size))
}

func benchmarkFrameValue(tb testing.TB, size int) Frame {
	tb.Helper()
	return Frame{
		Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream,
		SourceAddress: [16]byte{192, 0, 2, 1}, DestinationAddress: [16]byte{198, 51, 100, 2},
		SourcePort: 12345, DestinationPort: 443, TLVData: benchmarkTLV(size),
	}
}

func encodeBenchmarkFrame(tb testing.TB, frame Frame) []byte {
	tb.Helper()
	buffer := make([]byte, MaxFrameSize)
	written, err := Encode(buffer, frame)
	if err != nil {
		tb.Fatal(err)
	}
	return buffer[:written]
}

func benchmarkTLV(size int) []byte {
	data := make([]byte, 3+size)
	data[0] = byte(TLVTypeALPN)
	binary.BigEndian.PutUint16(data[1:3], uint16(size))
	return data
}

func frameSizeName(size int) string {
	switch size {
	case 64:
		return "64B"
	case 1024:
		return "1KiB"
	case 16384:
		return "16KiB"
	default:
		return "custom"
	}
}
