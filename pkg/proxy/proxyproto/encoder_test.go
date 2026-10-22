package proxyproto

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"
)

func TestEncoderFrameTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		frame     Frame
		wantBytes int
	}{
		{
			name:      "local",
			frame:     Frame{Command: CommandLocal, Family: AddressFamilyUnspec, Transport: TransportUnspec},
			wantBytes: HeaderSize + ChecksumSize,
		},
		{
			name: "ipv4 stream",
			frame: Frame{
				Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream,
				SourceAddress: [16]byte{192, 0, 2, 1}, DestinationAddress: [16]byte{198, 51, 100, 2},
				SourcePort: 12345, DestinationPort: 443,
			},
			wantBytes: HeaderSize + 12 + ChecksumSize,
		},
		{
			name: "ipv6 datagram",
			frame: Frame{
				Command: CommandProxy, Family: AddressFamilyINET6, Transport: TransportDgram,
				SourceAddress:      [16]byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
				DestinationAddress: [16]byte{0x20, 1, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2},
				SourcePort:         53, DestinationPort: 5353,
			},
			wantBytes: HeaderSize + 36 + ChecksumSize,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			buffer := make([]byte, MaxFrameSize)
			written, err := Encode(buffer, test.frame)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			if written != test.wantBytes {
				t.Fatalf("written = %d, want %d", written, test.wantBytes)
			}
			decoded, err := Decode(buffer[:written])
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if decoded.Command != test.frame.Command || decoded.Family != test.frame.Family || decoded.Transport != test.frame.Transport {
				t.Fatalf("decoded metadata = %v/%v/%v", decoded.Command, decoded.Family, decoded.Transport)
			}
		})
	}
}

func TestEncoderWritesChecksum(t *testing.T) {
	t.Parallel()
	frame := Frame{Command: CommandLocal, Family: AddressFamilyUnspec, Transport: TransportUnspec}
	buffer := make([]byte, MaxFrameSize)
	written, err := Encode(buffer, frame)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	want := crc32.Checksum(buffer[:written-ChecksumSize], crc32cTable)
	got := binary.BigEndian.Uint32(buffer[written-ChecksumSize : written])
	if got != want {
		t.Fatalf("checksum = %08x, want %08x", got, want)
	}
}

func TestEncoderRejectsInvalidFrames(t *testing.T) {
	t.Parallel()
	oversizedTLV := make([]byte, 3+65520)
	oversizedTLV[0] = byte(TLVTypeALPN)
	binary.BigEndian.PutUint16(oversizedTLV[1:3], 65520)

	tests := []struct {
		name    string
		frame   Frame
		dstSize int
		wantErr error
	}{
		{
			name: "invalid command", frame: Frame{Command: Command(9), Family: AddressFamilyINET, Transport: TransportStream},
			dstSize: MaxFrameSize, wantErr: ErrInvalidCommand,
		},
		{
			name: "invalid family", frame: Frame{Command: CommandProxy, Family: AddressFamily(9), Transport: TransportStream},
			dstSize: MaxFrameSize, wantErr: ErrInvalidAddressFamily,
		},
		{
			name: "invalid transport", frame: Frame{Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportProtocol(9)},
			dstSize: MaxFrameSize, wantErr: ErrInvalidTransportProtocol,
		},
		{
			name: "noncanonical local", frame: Frame{Command: CommandLocal, Family: AddressFamilyINET, Transport: TransportStream},
			dstSize: MaxFrameSize, wantErr: ErrInvalidLocalMetadata,
		},
		{
			name: "small buffer", frame: Frame{Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream},
			dstSize: HeaderSize - 1, wantErr: ErrBufferTooSmall,
		},
		{
			name: "malformed tlv", frame: Frame{Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream, TLVData: []byte{0x01, 0x00}},
			dstSize: MaxFrameSize, wantErr: ErrMalformedTLV,
		},
		{
			name: "length overflow", frame: Frame{Command: CommandProxy, Family: AddressFamilyINET, Transport: TransportStream, TLVData: oversizedTLV},
			dstSize: MaxFrameSize, wantErr: ErrLengthOverflow,
		},
		{
			name: "invalid unix path", frame: Frame{
				Command: CommandProxy, Family: AddressFamilyUNIX, Transport: TransportStream,
				SourceUnixPathLength: UnixAddressSize,
			},
			dstSize: MaxFrameSize, wantErr: ErrInvalidUnixPath,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			buffer := make([]byte, test.dstSize)
			_, err := Encode(buffer, test.frame)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Encode() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestEncoderUnixEndpoints(t *testing.T) {
	t.Parallel()
	frame := Frame{
		Command: CommandProxy, Family: AddressFamilyUNIX, Transport: TransportStream,
		SourceUnixPath: [UnixAddressSize]byte{'s', 'o', 'c', 'k'}, SourceUnixPathLength: 4,
		DestinationUnixPath: [UnixAddressSize]byte{'c', 'l', 'i', 'e', 'n', 't'}, DestinationUnixPathLength: 6,
	}
	buffer := make([]byte, MaxFrameSize)
	written, err := Encode(buffer, frame)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(buffer[:written])
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if string(decoded.SourceUnixPath[:decoded.SourceUnixPathLength]) != "sock" {
		t.Fatalf("source path = %q", decoded.SourceUnixPath[:decoded.SourceUnixPathLength])
	}
	if string(decoded.DestinationUnixPath[:decoded.DestinationUnixPathLength]) != "client" {
		t.Fatalf("destination path = %q", decoded.DestinationUnixPath[:decoded.DestinationUnixPathLength])
	}
}
