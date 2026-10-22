package proxyproto

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"sync"
	"testing"
)

func TestDecodeValidation(t *testing.T) {
	t.Parallel()
	ipv4 := mustEncodeFrame(t, Frame{
		Command:            CommandProxy,
		Family:             AddressFamilyINET,
		Transport:          TransportStream,
		SourceAddress:      [16]byte{192, 0, 2, 10},
		DestinationAddress: [16]byte{198, 51, 100, 20},
		SourcePort:         12345,
		DestinationPort:    443,
	})
	ipv6 := mustEncodeFrame(t, Frame{
		Command:            CommandProxy,
		Family:             AddressFamilyINET6,
		Transport:          TransportDgram,
		SourceAddress:      [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		DestinationAddress: [16]byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
		SourcePort:         53,
		DestinationPort:    5353,
	})
	local := rawTestFrame(t, CommandLocal, AddressFamilyUnspec, TransportUnspec, nil, nil)
	malformedTLV := rawTestFrame(t, CommandProxy, AddressFamilyINET, TransportStream,
		[]byte{192, 0, 2, 1, 198, 51, 100, 2, 0x30, 0x39, 0x01, 0xbb},
		[]byte{0x01, 0x00, 0x02, 0xff})
	unixAddress := bytes.Repeat([]byte{'x'}, 2*UnixAddressSize)
	malformedUnix := rawTestFrame(t, CommandProxy, AddressFamilyUNIX, TransportStream, unixAddress, nil)
	invalidLocal := rawTestFrame(t, CommandLocal, AddressFamilyINET, TransportStream, nil, nil)
	unspecified := rawTestFrame(t, CommandProxy, AddressFamilyUnspec, TransportUnspec, nil, nil)

	badSignature := cloneBytes(ipv4)
	badSignature[0] ^= 0xff
	badVersion := cloneBytes(ipv4)
	badVersion[12] = 0x31
	badCommand := cloneBytes(ipv4)
	badCommand[12] = 0x27
	badFamily := cloneBytes(ipv4)
	badFamily[13] = 0x40
	badTransport := cloneBytes(ipv4)
	badTransport[13] = 0x13
	badLength := cloneBytes(ipv4)
	binary.BigEndian.PutUint16(badLength[14:16], 3)
	badChecksum := cloneBytes(ipv4)
	badChecksum[len(badChecksum)-1] ^= 0xff

	tests := []struct {
		name    string
		frame   []byte
		wantErr error
	}{
		{name: "ipv4 stream", frame: ipv4},
		{name: "ipv6 datagram", frame: ipv6},
		{name: "local", frame: local},
		{name: "unspecified proxy", frame: unspecified},
		{name: "invalid signature", frame: badSignature, wantErr: ErrInvalidSignature},
		{name: "invalid version", frame: badVersion, wantErr: ErrUnsupportedVersion},
		{name: "invalid command", frame: badCommand, wantErr: ErrInvalidCommand},
		{name: "invalid family", frame: badFamily, wantErr: ErrInvalidAddressFamily},
		{name: "invalid transport", frame: badTransport, wantErr: ErrInvalidTransportProtocol},
		{name: "invalid length", frame: badLength, wantErr: ErrInvalidLength},
		{name: "truncated", frame: ipv4[:len(ipv4)-1], wantErr: ErrTruncatedFrame},
		{name: "trailing bytes", frame: append(cloneBytes(ipv4), 0), wantErr: ErrTrailingBytes},
		{name: "checksum mismatch", frame: badChecksum, wantErr: ErrChecksumMismatch},
		{name: "malformed tlv", frame: malformedTLV, wantErr: ErrMalformedTLV},
		{name: "malformed unix", frame: malformedUnix, wantErr: ErrMalformedUnixPath},
		{name: "noncanonical local", frame: invalidLocal, wantErr: ErrInvalidLocalMetadata},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Decode(test.frame)
			if test.wantErr == nil && err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("Decode() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestDecodeEndpointMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		frame       Frame
		source      []byte
		destination []byte
		sourcePort  uint16
		destPort    uint16
	}{
		{
			name: "ipv4",
			frame: Frame{
				Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream,
				SourceAddress: [16]byte{10, 0, 0, 1}, DestinationAddress: [16]byte{10, 0, 0, 2},
				SourcePort: 1234, DestinationPort: 4321,
			},
			source: []byte{10, 0, 0, 1}, destination: []byte{10, 0, 0, 2}, sourcePort: 1234, destPort: 4321,
		},
		{
			name: "ipv6",
			frame: Frame{
				Command: CommandProxy, Family: AddressFamilyINET6, Transport: TransportDgram,
				SourceAddress:      [16]byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
				DestinationAddress: [16]byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				SourcePort:         53, DestinationPort: 5353,
			},
			source:      []byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			destination: []byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
			sourcePort:  53, destPort: 5353,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			encoded := mustEncodeFrame(t, test.frame)
			decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if !bytes.Equal(decoded.SourceIPBytes(), test.source) {
				t.Fatalf("source = %v, want %v", decoded.SourceIPBytes(), test.source)
			}
			if !bytes.Equal(decoded.DestinationIPBytes(), test.destination) {
				t.Fatalf("destination = %v, want %v", decoded.DestinationIPBytes(), test.destination)
			}
			if decoded.SourcePort != test.sourcePort || decoded.DestinationPort != test.destPort {
				t.Fatalf("ports = %d/%d, want %d/%d", decoded.SourcePort, decoded.DestinationPort, test.sourcePort, test.destPort)
			}
		})
	}
}

func TestTLVIterator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		data    []byte
		wantOK  bool
		wantErr error
	}{
		{name: "valid", data: []byte{0x01, 0x00, 0x02, 'h', '2'}, wantOK: true},
		{name: "short header", data: []byte{0x01, 0x00}, wantErr: ErrMalformedTLV},
		{name: "short value", data: []byte{0x01, 0x00, 0x02, 'h'}, wantErr: ErrMalformedTLV},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			frame := Frame{TLVData: test.data}
			iterator := frame.TLVs()
			_, ok := iterator.Next()
			if ok != test.wantOK {
				t.Fatalf("Next() ok = %v, want %v", ok, test.wantOK)
			}
			if test.wantErr != nil && !errors.Is(iterator.Err(), test.wantErr) {
				t.Fatalf("Err() = %v, want %v", iterator.Err(), test.wantErr)
			}
		})
	}
}

func TestStreamDecoderFragmentedReads(t *testing.T) {
	t.Parallel()
	frame := mustEncodeFrame(t, Frame{
		Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream,
		SourceAddress: [16]byte{127, 0, 0, 1}, DestinationAddress: [16]byte{127, 0, 0, 2},
	})
	decoder := NewStreamDecoder()
	decoded, err := decoder.Read(context.Background(), &chunkReader{data: frame, maximum: 1}, make([]byte, MaxFrameSize))
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if decoder.State() != DecodeStateComplete {
		t.Fatalf("state = %v, want %v", decoder.State(), DecodeStateComplete)
	}
	if !bytes.Equal(decoded.SourceIPBytes(), []byte{127, 0, 0, 1}) {
		t.Fatalf("source = %v", decoded.SourceIPBytes())
	}
}

func TestStreamDecoderCancellationPreservesOffset(t *testing.T) {
	t.Parallel()
	frame := mustEncodeFrame(t, Frame{
		Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream,
		SourceAddress: [16]byte{127, 0, 0, 1}, DestinationAddress: [16]byte{127, 0, 0, 2},
	})
	reader := &stagedContextReader{data: frame, blocked: make(chan struct{})}
	decoder := NewStreamDecoder()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-reader.blocked
		cancel()
	}()
	_, err := decoder.Read(ctx, reader, make([]byte, MaxFrameSize))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v, want context cancellation", err)
	}
	if decoder.State() != DecodeStateHeader {
		t.Fatalf("state = %v, want %v", decoder.State(), DecodeStateHeader)
	}
	decoded, err := decoder.Read(context.Background(), reader, make([]byte, MaxFrameSize))
	if err != nil {
		t.Fatalf("resumed Read() error = %v", err)
	}
	if decoded.Command != CommandProxy {
		t.Fatalf("command = %v", decoded.Command)
	}
}

func mustEncodeFrame(t *testing.T, frame Frame) []byte {
	t.Helper()
	buffer := make([]byte, MaxFrameSize)
	written, err := Encode(buffer, frame)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return buffer[:written]
}

func rawTestFrame(t *testing.T, command Command, family AddressFamily, transport TransportProtocol, address, tlv []byte) []byte {
	t.Helper()
	payloadLength := len(address) + len(tlv) + ChecksumSize
	frame := make([]byte, HeaderSize+payloadLength)
	copy(frame[:SignatureSize], proxyV2Signature[:])
	frame[12] = 0x20 | byte(command)
	frame[13] = byte(family)<<4 | byte(transport)
	binary.BigEndian.PutUint16(frame[14:16], uint16(payloadLength))
	copy(frame[HeaderSize:], address)
	copy(frame[HeaderSize+len(address):], tlv)
	checksum := crc32.Checksum(frame[:len(frame)-ChecksumSize], crc32cTable)
	binary.BigEndian.PutUint32(frame[len(frame)-ChecksumSize:], checksum)
	return frame
}

func cloneBytes(source []byte) []byte {
	clone := make([]byte, len(source))
	copy(clone, source)
	return clone
}

type chunkReader struct {
	data    []byte
	maximum int
}

func (r *chunkReader) Read(dst []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	count := len(r.data)
	if count > r.maximum {
		count = r.maximum
	}
	if count > len(dst) {
		count = len(dst)
	}
	copy(dst, r.data[:count])
	r.data = r.data[count:]
	return count, nil
}

type stagedContextReader struct {
	data    []byte
	calls   int
	blocked chan struct{}
	once    sync.Once
}

func (r *stagedContextReader) ReadContext(ctx context.Context, dst []byte) (int, error) {
	r.calls++
	switch r.calls {
	case 1:
		count := 2
		if count > len(r.data) {
			count = len(r.data)
		}
		copy(dst, r.data[:count])
		r.data = r.data[count:]
		return count, nil
	case 2:
		r.once.Do(func() { close(r.blocked) })
		<-ctx.Done()
		return 0, ctx.Err()
	default:
		count := copy(dst, r.data)
		r.data = r.data[count:]
		return count, nil
	}
}
