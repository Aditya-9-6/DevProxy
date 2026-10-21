package proxyproto

import (
	"encoding/binary"
	"errors"
	"io"
	"net/netip"
	"sync"
)

const (
	// HeaderSize is the size of the fixed PROXY protocol v2 header.
	HeaderSize = 16
	// UnixAddressSize is the size of one UNIX socket address in a v2 header.
	UnixAddressSize = 108
	// INETPayloadLength is the exact payload size for an IPv4 endpoint pair.
	INETPayloadLength = 12
	// INET6PayloadLength is the exact payload size for an IPv6 endpoint pair.
	INET6PayloadLength = 36
	// UNIXPayloadLength is the exact payload size for a UNIX endpoint pair.
	UNIXPayloadLength = 2 * UnixAddressSize
	// MaxPayloadLength is the largest payload representable by the v2 length field.
	MaxPayloadLength = 65535
	// MaxHeaderSize bounds the complete header and payload held by the decoder.
	MaxHeaderSize = HeaderSize + MaxPayloadLength
	// Signature is the fixed 12-byte PROXY protocol v2 signature.
	Signature = "\r\n\r\n\x00\r\nQUIT\n"
)

var (
	ErrInvalidSignature = errors.New("proxyproto: invalid signature")
	ErrInvalidVersion   = errors.New("proxyproto: invalid version")
	ErrInvalidCommand   = errors.New("proxyproto: invalid command")
	ErrInvalidFamily    = errors.New("proxyproto: invalid address family")
	ErrInvalidProtocol  = errors.New("proxyproto: invalid transport protocol")
	ErrInvalidLength    = errors.New("proxyproto: invalid payload length")
	ErrTruncatedHeader  = errors.New("proxyproto: truncated header")
)

// Command identifies the purpose of a PROXY protocol v2 header.
type Command uint8

const (
	CommandLocal Command = 0x00
	CommandProxy Command = 0x01
)

// Family identifies the address family carried by a PROXY protocol v2 header.
type Family uint8

const (
	FamilyUnspecified Family = 0x00
	FamilyINET        Family = 0x01
	FamilyINET6       Family = 0x02
	FamilyUNIX        Family = 0x03
)

// Protocol identifies the transport protocol carried by a PROXY protocol v2 header.
type Protocol uint8

const (
	ProtocolUnspecified Protocol = 0x00
	ProtocolStream      Protocol = 0x01
	ProtocolDatagram    Protocol = 0x02
)

// UnixAddress stores a UNIX socket address without converting it to a string.
// Length excludes the terminating NUL for pathname addresses. An abstract
// address, identified by a leading zero byte, retains all 108 bytes.
type UnixAddress struct {
	Bytes  [UnixAddressSize]byte
	Length uint8
}

// Header is the value-type metadata exposed after a complete, valid header
// has been consumed. Source and Destination are populated for INET and INET6;
// SourceUnix and DestinationUnix are populated for UNIX.
type Header struct {
	Version         uint8
	Command         Command
	Family          Family
	Protocol        Protocol
	PayloadLength   uint16
	Source          netip.AddrPort
	Destination     netip.AddrPort
	SourceUnix      UnixAddress
	DestinationUnix UnixAddress
}

// Metadata is an alias retained for callers that prefer metadata terminology.
type Metadata = Header

// Decoder is the narrow interface consumed by the proxy handshake boundary.
type Decoder interface {
	Decode(io.Reader) (Header, error)
}

// V2Decoder is a stateless PROXY protocol v2 decoder.
type V2Decoder struct{}

var _ Decoder = V2Decoder{}

// NewDecoder returns a concurrency-safe PROXY protocol v2 decoder.
func NewDecoder() Decoder {
	return V2Decoder{}
}

type decodeBuffer struct {
	data [MaxHeaderSize]byte
}

// decodeBufferPool bounds retained memory to one maximum-sized v2 header per
// concurrently active decode. A pointer is pooled to avoid slice boxing.
var decodeBufferPool = sync.Pool{
	New: func() any {
		return &decodeBuffer{}
	},
}

// Decode consumes exactly one complete PROXY protocol v2 header. It never
// reads beyond the declared payload length, leaving application bytes unread.
func (V2Decoder) Decode(reader io.Reader) (Header, error) {
	if reader == nil {
		return Header{}, ErrTruncatedHeader
	}

	buffer := decodeBufferPool.Get().(*decodeBuffer)
	defer decodeBufferPool.Put(buffer)

	if _, err := io.ReadFull(reader, buffer.data[:HeaderSize]); err != nil {
		return Header{}, normalizeReadError(err)
	}
	if !hasSignature(buffer.data[:HeaderSize]) {
		return Header{}, ErrInvalidSignature
	}

	header, err := parseFixedHeader(
		buffer.data[12],
		buffer.data[13],
		binary.BigEndian.Uint16(buffer.data[14:16]),
	)
	if err != nil {
		return Header{}, err
	}

	payloadLength := int(header.PayloadLength)
	payload := buffer.data[HeaderSize : HeaderSize+payloadLength]
	if payloadLength > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
			return Header{}, normalizeReadError(err)
		}
	}
	if err := parseAddresses(&header, payload); err != nil {
		return Header{}, err
	}
	return header, nil
}

// Decode consumes one PROXY protocol v2 header using the default decoder.
func Decode(reader io.Reader) (Header, error) {
	return (V2Decoder{}).Decode(reader)
}

func parseFixedHeader(versionCommand, familyProtocol byte, payloadLength uint16) (Header, error) {
	command, err := parseVersionCommand(versionCommand)
	if err != nil {
		return Header{}, err
	}
	family, protocol, err := parseFamilyProtocol(familyProtocol)
	if err != nil {
		return Header{}, err
	}
	if err := validateProtocolCombination(command, family, protocol); err != nil {
		return Header{}, err
	}
	if err := validatePayloadLength(command, family, payloadLength); err != nil {
		return Header{}, err
	}
	return Header{
		Version:       2,
		Command:       command,
		Family:        family,
		Protocol:      protocol,
		PayloadLength: payloadLength,
	}, nil
}

func parseVersionCommand(value byte) (Command, error) {
	if value>>4 != 2 {
		return 0, ErrInvalidVersion
	}
	command := Command(value & 0x0f)
	if command != CommandLocal && command != CommandProxy {
		return 0, ErrInvalidCommand
	}
	return command, nil
}

func parseFamilyProtocol(value byte) (Family, Protocol, error) {
	family := Family(value >> 4)
	switch family {
	case FamilyUnspecified, FamilyINET, FamilyINET6, FamilyUNIX:
	default:
		return 0, 0, ErrInvalidFamily
	}

	protocol := Protocol(value & 0x0f)
	switch protocol {
	case ProtocolUnspecified, ProtocolStream, ProtocolDatagram:
	default:
		return 0, 0, ErrInvalidProtocol
	}
	return family, protocol, nil
}

func validateProtocolCombination(command Command, family Family, protocol Protocol) error {
	if command == CommandLocal {
		return nil
	}
	if family == FamilyUnspecified {
		if protocol != ProtocolUnspecified {
			return ErrInvalidProtocol
		}
		return nil
	}
	if (family == FamilyINET || family == FamilyINET6) && protocol == ProtocolUnspecified {
		return ErrInvalidProtocol
	}
	return nil
}

func validatePayloadLength(command Command, family Family, payloadLength uint16) error {
	if command == CommandLocal {
		return nil
	}
	switch family {
	case FamilyUnspecified:
		return nil
	case FamilyINET:
		if payloadLength != INETPayloadLength {
			return ErrInvalidLength
		}
	case FamilyINET6:
		if payloadLength != INET6PayloadLength {
			return ErrInvalidLength
		}
	case FamilyUNIX:
		if payloadLength != UNIXPayloadLength {
			return ErrInvalidLength
		}
	}
	return nil
}

func parseAddresses(header *Header, payload []byte) error {
	if header.Command == CommandLocal || header.Family == FamilyUnspecified {
		return nil
	}
	switch header.Family {
	case FamilyINET:
		if len(payload) != INETPayloadLength {
			return ErrInvalidLength
		}
		parseINET(header, payload)
	case FamilyINET6:
		if len(payload) != INET6PayloadLength {
			return ErrInvalidLength
		}
		parseINET6(header, payload)
	case FamilyUNIX:
		if len(payload) != UNIXPayloadLength {
			return ErrInvalidLength
		}
		parseUNIX(header, payload)
	}
	return nil
}

func parseINET(header *Header, payload []byte) {
	var sourceIP, destinationIP [4]byte
	copy(sourceIP[:], payload[0:4])
	copy(destinationIP[:], payload[4:8])
	header.Source = netip.AddrPortFrom(
		netip.AddrFrom4(sourceIP),
		binary.BigEndian.Uint16(payload[8:10]),
	)
	header.Destination = netip.AddrPortFrom(
		netip.AddrFrom4(destinationIP),
		binary.BigEndian.Uint16(payload[10:12]),
	)
}

func parseINET6(header *Header, payload []byte) {
	var sourceIP, destinationIP [16]byte
	copy(sourceIP[:], payload[0:16])
	copy(destinationIP[:], payload[16:32])
	header.Source = netip.AddrPortFrom(
		netip.AddrFrom16(sourceIP),
		binary.BigEndian.Uint16(payload[32:34]),
	)
	header.Destination = netip.AddrPortFrom(
		netip.AddrFrom16(destinationIP),
		binary.BigEndian.Uint16(payload[34:36]),
	)
}

func parseUNIX(header *Header, payload []byte) {
	copy(header.SourceUnix.Bytes[:], payload[:UnixAddressSize])
	copy(header.DestinationUnix.Bytes[:], payload[UnixAddressSize:UNIXPayloadLength])
	header.SourceUnix.Length = unixAddressLength(payload[:UnixAddressSize])
	header.DestinationUnix.Length = unixAddressLength(payload[UnixAddressSize:UNIXPayloadLength])
}

func unixAddressLength(path []byte) uint8 {
	if len(path) == 0 {
		return 0
	}

	abstract := path[0] == 0
	var length uint8
	for _, value := range path {
		if !abstract && value == 0 {
			return length
		}
		// Count directly in uint8 to avoid an unchecked int-to-uint8
		// conversion. Saturation also prevents oversized inputs wrapping.
		if length == 0xff {
			return length
		}
		length++
	}
	return length
}

func hasSignature(data []byte) bool {
	if len(data) < len(Signature) {
		return false
	}
	for index := 0; index < len(Signature); index++ {
		if data[index] != Signature[index] {
			return false
		}
	}
	return true
}

func normalizeReadError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrTruncatedHeader
	}
	return err
}
