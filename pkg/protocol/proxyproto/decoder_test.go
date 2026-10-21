package proxyproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"net/netip"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestDecoderValidHeaders(t *testing.T) {
	inetPayload := []byte{
		192, 0, 2, 1,
		198, 51, 100, 2,
		0x1f, 0x90,
		0x00, 0x50,
	}
	inet6Payload := make([]byte, INET6PayloadLength)
	copy(inet6Payload[0:16], netip.MustParseAddr("2001:db8::1").AsSlice())
	copy(inet6Payload[16:32], netip.MustParseAddr("2001:db8::2").AsSlice())
	binary.BigEndian.PutUint16(inet6Payload[32:34], 443)
	binary.BigEndian.PutUint16(inet6Payload[34:36], 8443)
	unixPayload := make([]byte, UNIXPayloadLength)
	copy(unixPayload, "/tmp/source")
	copy(unixPayload[UnixAddressSize:], "/tmp/destination")

	tests := []struct {
		name          string
		command       byte
		family        byte
		protocol      byte
		payload       []byte
		payloadLength int
		check         func(*testing.T, Header)
	}{
		{
			name:          "INET",
			command:       byte(CommandProxy),
			family:        byte(FamilyINET),
			protocol:      byte(ProtocolStream),
			payload:       inetPayload,
			payloadLength: INETPayloadLength,
			check: func(t *testing.T, header Header) {
				wantSource := netip.AddrPortFrom(netip.MustParseAddr("192.0.2.1"), 8080)
				wantDestination := netip.AddrPortFrom(netip.MustParseAddr("198.51.100.2"), 80)
				if header.Source != wantSource || header.Destination != wantDestination {
					t.Fatalf("unexpected endpoints: source=%v destination=%v", header.Source, header.Destination)
				}
			},
		},
		{
			name:          "INET6",
			command:       byte(CommandProxy),
			family:        byte(FamilyINET6),
			protocol:      byte(ProtocolDatagram),
			payload:       inet6Payload,
			payloadLength: INET6PayloadLength,
			check: func(t *testing.T, header Header) {
				wantSource := netip.AddrPortFrom(netip.MustParseAddr("2001:db8::1"), 443)
				wantDestination := netip.AddrPortFrom(netip.MustParseAddr("2001:db8::2"), 8443)
				if header.Source != wantSource || header.Destination != wantDestination {
					t.Fatalf("unexpected endpoints: source=%v destination=%v", header.Source, header.Destination)
				}
			},
		},
		{
			name:          "UNIX",
			command:       byte(CommandProxy),
			family:        byte(FamilyUNIX),
			protocol:      byte(ProtocolStream),
			payload:       unixPayload,
			payloadLength: UNIXPayloadLength,
			check: func(t *testing.T, header Header) {
				if header.SourceUnix != expectedUnixAddress("/tmp/source") {
					t.Fatalf("unexpected source UNIX address: %+v", header.SourceUnix)
				}
				if header.DestinationUnix != expectedUnixAddress("/tmp/destination") {
					t.Fatalf("unexpected destination UNIX address: %+v", header.DestinationUnix)
				}
			},
		},
		{
			name:          "LOCAL ignores opaque payload",
			command:       byte(CommandLocal),
			family:        byte(FamilyUnspecified),
			protocol:      byte(ProtocolUnspecified),
			payload:       []byte{1, 2, 3},
			payloadLength: 3,
		},
		{
			name:          "PROXY UNSPEC ignores opaque payload",
			command:       byte(CommandProxy),
			family:        byte(FamilyUnspecified),
			protocol:      byte(ProtocolUnspecified),
			payload:       []byte{4, 5, 6},
			payloadLength: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			frame := buildTestHeader(test.command, test.family, test.protocol, test.payload)
			header, err := (V2Decoder{}).Decode(bytes.NewReader(frame))
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if header.Version != 2 || header.Command != Command(test.command) ||
				header.Family != Family(test.family) || header.Protocol != Protocol(test.protocol) {
				t.Fatalf("unexpected fixed header: %+v", header)
			}
			if int(header.PayloadLength) != test.payloadLength {
				t.Fatalf("PayloadLength = %d, want %d", header.PayloadLength, test.payloadLength)
			}
			if test.check != nil {
				test.check(t, header)
			}
		})
	}
}

func TestDecoderRejectsInvalidHeaders(t *testing.T) {
	tests := []struct {
		name    string
		frame   []byte
		mutate  func([]byte)
		wantErr error
	}{
		{
			name:    "invalid signature",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[0] ^= 0xff },
			wantErr: ErrInvalidSignature,
		},
		{
			name:    "invalid version",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[12] = 0x10 },
			wantErr: ErrInvalidVersion,
		},
		{
			name:    "invalid command",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[12] = 0x22 },
			wantErr: ErrInvalidCommand,
		},
		{
			name:    "reserved family",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[13] = 0x40 },
			wantErr: ErrInvalidFamily,
		},
		{
			name:    "reserved protocol",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[13] = 0x03 },
			wantErr: ErrInvalidProtocol,
		},
		{
			name:    "ambiguous unspecified family with stream protocol",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyUnspecified), byte(ProtocolUnspecified), nil),
			mutate:  func(frame []byte) { frame[13] = 0x01 },
			wantErr: ErrInvalidProtocol,
		},
		{
			name:    "INET with unspecified transport",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength)),
			mutate:  func(frame []byte) { frame[13] = 0x10 },
			wantErr: ErrInvalidProtocol,
		},
		{
			name:    "short INET payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength-1)),
			wantErr: ErrInvalidLength,
		},
		{
			name:    "overlong INET payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), make([]byte, INETPayloadLength+1)),
			wantErr: ErrInvalidLength,
		},
		{
			name:    "short INET6 payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET6), byte(ProtocolStream), make([]byte, INET6PayloadLength-1)),
			wantErr: ErrInvalidLength,
		},
		{
			name:    "overlong INET6 payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyINET6), byte(ProtocolStream), make([]byte, INET6PayloadLength+1)),
			wantErr: ErrInvalidLength,
		},
		{
			name:    "short UNIX payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyUNIX), byte(ProtocolStream), make([]byte, UNIXPayloadLength-1)),
			wantErr: ErrInvalidLength,
		},
		{
			name:    "overlong UNIX payload",
			frame:   buildTestHeader(byte(CommandProxy), byte(FamilyUNIX), byte(ProtocolStream), make([]byte, UNIXPayloadLength+1)),
			wantErr: ErrInvalidLength,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mutate != nil {
				test.mutate(test.frame)
			}
			_, err := (V2Decoder{}).Decode(bytes.NewReader(test.frame))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Decode() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestDecoderRejectsTruncatedHeaders(t *testing.T) {
	full := buildTestHeader(
		byte(CommandProxy),
		byte(FamilyINET),
		byte(ProtocolStream),
		make([]byte, INETPayloadLength),
	)
	for _, length := range []int{0, 1, 11, 12, 15, 16, len(full) - 1} {
		t.Run(string(rune(length)), func(t *testing.T) {
			_, err := (V2Decoder{}).Decode(bytes.NewReader(full[:length]))
			if !errors.Is(err, ErrTruncatedHeader) {
				t.Fatalf("Decode() error = %v, want %v", err, ErrTruncatedHeader)
			}
		})
	}
}

func TestDecoderPreservesNetworkError(t *testing.T) {
	wantErr := errors.New("network read failed")
	_, err := (V2Decoder{}).Decode(failingReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Decode() error = %v, want %v", err, wantErr)
	}
}

func TestDecoderConsumesExactlyDeclaredHeader(t *testing.T) {
	payload := []byte{192, 0, 2, 1, 198, 51, 100, 2, 0x1f, 0x90, 0, 80}
	frame := buildTestHeader(byte(CommandProxy), byte(FamilyINET), byte(ProtocolStream), payload)
	application := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	stream := make([]byte, 0, len(frame)+len(application))
	stream = append(stream, frame...)
	stream = append(stream, application...)
	reader := &trackingReader{remaining: stream}

	if _, err := (V2Decoder{}).Decode(reader); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if reader.bytesRead != HeaderSize+len(payload) {
		t.Fatalf("bytes read = %d, want %d", reader.bytesRead, HeaderSize+len(payload))
	}
	if !bytes.Equal(reader.remaining, application) {
		t.Fatalf("application bytes were consumed or changed: %q", reader.remaining)
	}
}

func TestDecoderRejectsOverlongGeometryBeforePayloadRead(t *testing.T) {
	frame := buildTestHeader(
		byte(CommandProxy),
		byte(FamilyINET),
		byte(ProtocolStream),
		make([]byte, INETPayloadLength+1),
	)
	application := []byte("GET / HTTP/1.1\r\n\r\n")
	stream := make([]byte, 0, HeaderSize+len(application))
	stream = append(stream, frame[:HeaderSize]...)
	stream = append(stream, application...)
	reader := &trackingReader{remaining: stream}

	_, err := (V2Decoder{}).Decode(reader)
	if !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("Decode() error = %v, want %v", err, ErrInvalidLength)
	}
	if reader.bytesRead != HeaderSize {
		t.Fatalf("bytes read = %d, want %d", reader.bytesRead, HeaderSize)
	}
	if !bytes.Equal(reader.remaining, application) {
		t.Fatalf("application bytes were consumed: %q", reader.remaining)
	}
}

func TestProtocolPackageHasNoProxyDependency(t *testing.T) {
	const proxyPackagePath = "github.com/Aditya-9-6/DevProxy/pkg/proxy"

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	protocolRoot := filepath.Join(root, "pkg", "protocol")

	err := filepath.WalkDir(protocolRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, importSpec := range file.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				return err
			}
			if importPath == proxyPackagePath || strings.HasPrefix(importPath, proxyPackagePath+"/") {
				t.Errorf("protocol package %s must not import %s", path, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("architecture scan failed: %v", err)
	}
}

func buildTestHeader(command, family, protocol byte, payload []byte) []byte {
	frame := make([]byte, HeaderSize+len(payload))
	copy(frame[:12], Signature)
	frame[12] = 0x20 | (command & 0x0f)
	frame[13] = (family << 4) | (protocol & 0x0f)
	binary.BigEndian.PutUint16(frame[14:16], uint16(len(payload)))
	copy(frame[HeaderSize:], payload)
	return frame
}

func expectedUnixAddress(value string) UnixAddress {
	var address UnixAddress
	copy(address.Bytes[:], value)
	address.Length = uint8(len(value))
	return address
}

type failingReader struct {
	err error
}

func (reader failingReader) Read([]byte) (int, error) {
	return 0, reader.err
}

type trackingReader struct {
	remaining []byte
	bytesRead int
}

func (reader *trackingReader) Read(buffer []byte) (int, error) {
	if len(reader.remaining) == 0 {
		return 0, io.EOF
	}
	count := copy(buffer, reader.remaining)
	reader.remaining = reader.remaining[count:]
	reader.bytesRead += count
	return count, nil
}
