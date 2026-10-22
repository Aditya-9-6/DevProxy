package proxyproto

import (
	"encoding/binary"
	"errors"
)

const (
	// SignatureSize is the size of the PROXY protocol v2 signature.
	SignatureSize = 12
	// HeaderSize is the size of the fixed PROXY protocol v2 header.
	HeaderSize = 16
	// ChecksumSize is the size of the trailing CRC32c checksum.
	ChecksumSize = 4
	// MaxPayloadSize is the largest payload declared by the 16-bit length field.
	MaxPayloadSize = 65535
	// MaxFrameSize is the largest possible complete PROXY protocol v2 frame.
	MaxFrameSize = HeaderSize + MaxPayloadSize
	// UnixAddressSize is the size of one address in a UNIX PROXY frame.
	UnixAddressSize = 108
)

// Signature is the fixed twelve-byte PROXY protocol v2 signature.
const Signature = "\r\n\r\n\x00\r\nQUIT\n"

// Command identifies the purpose of a PROXY protocol v2 frame.
type Command uint8

const (
	// CommandLocal identifies a health-check LOCAL frame.
	CommandLocal Command = 0x00
	// CommandProxy identifies a connection-carrying PROXY frame.
	CommandProxy Command = 0x01

	// CommandLOCAL is an uppercase compatibility alias for CommandLocal.
	CommandLOCAL = CommandLocal
	// CommandPROXY is an uppercase compatibility alias for CommandProxy.
	CommandPROXY = CommandProxy
)

// AddressFamily identifies the endpoint address format.
type AddressFamily uint8

const (
	// AddressFamilyUnspec indicates that no endpoint address is present.
	AddressFamilyUnspec AddressFamily = 0x00
	// AddressFamilyINET indicates IPv4 endpoints.
	AddressFamilyINET AddressFamily = 0x01
	// AddressFamilyINET6 indicates IPv6 endpoints.
	AddressFamilyINET6 AddressFamily = 0x02
	// AddressFamilyUNIX indicates UNIX-domain socket endpoints.
	AddressFamilyUNIX AddressFamily = 0x03

	// FamilyUnspec is a short alias for AddressFamilyUnspec.
	FamilyUnspec = AddressFamilyUnspec
	// FamilyINET is a short alias for AddressFamilyINET.
	FamilyINET = AddressFamilyINET
	// FamilyINET6 is a short alias for AddressFamilyINET6.
	FamilyINET6 = AddressFamilyINET6
	// FamilyUNIX is a short alias for AddressFamilyUNIX.
	FamilyUNIX = AddressFamilyUNIX
	// AddressFamilyIPv4 is an alias for AddressFamilyINET.
	AddressFamilyIPv4 = AddressFamilyINET
	// AddressFamilyIPv6 is an alias for AddressFamilyINET6.
	AddressFamilyIPv6 = AddressFamilyINET6
)

// TransportProtocol identifies the transport carried by a frame.
type TransportProtocol uint8

const (
	// TransportUnspec indicates that the transport is unspecified.
	TransportUnspec TransportProtocol = 0x00
	// TransportStream identifies a stream transport.
	TransportStream TransportProtocol = 0x01
	// TransportDgram identifies a datagram transport.
	TransportDgram TransportProtocol = 0x02

	// TransportSTREAM is an uppercase compatibility alias for TransportStream.
	TransportSTREAM = TransportStream
	// TransportDGRAM is an uppercase compatibility alias for TransportDgram.
	TransportDGRAM = TransportDgram
	// ProtocolUnspec is a short alias for TransportUnspec.
	ProtocolUnspec = TransportUnspec
	// ProtocolStream is a short alias for TransportStream.
	ProtocolStream = TransportStream
	// ProtocolDgram is a short alias for TransportDgram.
	ProtocolDgram = TransportDgram
)

// TLVType identifies a PROXY protocol v2 type-length-value record.
type TLVType uint8

const (
	// TLVTypeALPN carries an ALPN protocol list.
	TLVTypeALPN TLVType = 0x01
	// TLVTypeAuthority carries an authority value.
	TLVTypeAuthority TLVType = 0x02
	// TLVTypeCRC32C carries an additional CRC32c checksum.
	TLVTypeCRC32C TLVType = 0x03
	// TLVTypeNoop carries no semantic data.
	TLVTypeNoop TLVType = 0x04
	// TLVTypeUniqueID carries a unique connection identifier.
	TLVTypeUniqueID TLVType = 0x05
	// TLVTypeSSL is a nested SSL TLV container.
	TLVTypeSSL TLVType = 0x20
	// TLVTypeSSLVersion is an SSL sub-TLV.
	TLVTypeSSLVersion TLVType = 0x21
	// TLVTypeSSLCN is an SSL sub-TLV.
	TLVTypeSSLCN TLVType = 0x22
	// TLVTypeSSLCipher is an SSL sub-TLV.
	TLVTypeSSLCipher TLVType = 0x23
	// TLVTypeSSLSigAlg is an SSL sub-TLV.
	TLVTypeSSLSigAlg TLVType = 0x24
	// TLVTypeSSLKeyAlg is an SSL sub-TLV.
	TLVTypeSSLKeyAlg TLVType = 0x25
	// TLVTypeNetNS carries a network namespace identifier.
	TLVTypeNetNS TLVType = 0x30
	// TLVTypeReserved is the reserved zero type.
	TLVTypeReserved TLVType = 0x00
)

var (
	// ErrInvalidSignature indicates that the v2 signature is invalid.
	ErrInvalidSignature = errors.New("proxyproto: invalid signature")
	// ErrUnsupportedVersion indicates a protocol version other than two.
	ErrUnsupportedVersion = errors.New("proxyproto: unsupported version")
	// ErrInvalidCommand indicates an unknown command nibble.
	ErrInvalidCommand = errors.New("proxyproto: invalid command")
	// ErrInvalidAddressFamily indicates an unknown address-family nibble.
	ErrInvalidAddressFamily = errors.New("proxyproto: invalid address family")
	// ErrInvalidTransportProtocol indicates an unknown transport nibble.
	ErrInvalidTransportProtocol = errors.New("proxyproto: invalid transport protocol")
	// ErrInvalidLocalMetadata indicates non-canonical LOCAL frame metadata.
	ErrInvalidLocalMetadata = errors.New("proxyproto: invalid local frame metadata")
	// ErrInvalidLength indicates an invalid declared or calculated length.
	ErrInvalidLength = errors.New("proxyproto: invalid length")
	// ErrTruncatedFrame indicates that the frame is shorter than declared.
	ErrTruncatedFrame = errors.New("proxyproto: truncated frame")
	// ErrTrailingBytes indicates bytes beyond the declared frame boundary.
	ErrTrailingBytes = errors.New("proxyproto: trailing bytes")
	// ErrChecksumMismatch indicates a CRC32c validation failure.
	ErrChecksumMismatch = errors.New("proxyproto: checksum mismatch")
	// ErrMalformedTLV indicates a truncated or otherwise invalid TLV record.
	ErrMalformedTLV = errors.New("proxyproto: malformed TLV")
	// ErrMalformedUnixPath indicates a non-canonical UNIX endpoint field.
	ErrMalformedUnixPath = errors.New("proxyproto: malformed unix path")
	// ErrBufferTooSmall indicates that a caller-owned buffer cannot hold a frame.
	ErrBufferTooSmall = errors.New("proxyproto: buffer too small")
	// ErrLengthOverflow indicates that an encoded frame exceeds protocol limits.
	ErrLengthOverflow = errors.New("proxyproto: frame length overflow")
	// ErrInvalidUnixPath indicates an invalid encoder UNIX path.
	ErrInvalidUnixPath = errors.New("proxyproto: invalid unix path")
	// ErrNilContext indicates a nil streaming decode context.
	ErrNilContext = errors.New("proxyproto: nil context")
)

// Header contains the fixed-size metadata from a PROXY protocol v2 header.
// Length includes endpoint data, TLVs, and the trailing checksum.
type Header struct {
	Command   Command
	Family    AddressFamily
	Transport TransportProtocol
	Length    uint16
}

// Frame contains validated, fixed-size endpoint metadata and borrowed TLV storage.
// The TLVData slice aliases the caller-provided frame buffer and must not be
// retained after that buffer is reused.
type Frame struct {
	Command   Command
	Family    AddressFamily
	Transport TransportProtocol

	SourceAddress            [16]byte
	DestinationAddress       [16]byte
	SourceAddressLength      uint8
	DestinationAddressLength uint8
	SourcePort               uint16
	DestinationPort          uint16

	SourceUnixPath            [UnixAddressSize]byte
	DestinationUnixPath       [UnixAddressSize]byte
	SourceUnixPathLength      uint8
	DestinationUnixPathLength uint8

	TLVData  []byte
	Checksum uint32
	Length   int
}

// SourceIPBytes returns the source IP bytes for an IPv4 or IPv6 frame.
func (f *Frame) SourceIPBytes() []byte {
	if f == nil {
		return nil
	}
	switch f.Family {
	case AddressFamilyINET:
		if f.SourceAddressLength != netIPv4Length {
			return nil
		}
		return f.SourceAddress[:netIPv4Length]
	case AddressFamilyINET6:
		if f.SourceAddressLength != netIPv6Length {
			return nil
		}
		return f.SourceAddress[:]
	default:
		return nil
	}
}

// DestinationIPBytes returns the destination IP bytes for an IPv4 or IPv6 frame.
func (f *Frame) DestinationIPBytes() []byte {
	if f == nil {
		return nil
	}
	switch f.Family {
	case AddressFamilyINET:
		if f.DestinationAddressLength != netIPv4Length {
			return nil
		}
		return f.DestinationAddress[:netIPv4Length]
	case AddressFamilyINET6:
		if f.DestinationAddressLength != netIPv6Length {
			return nil
		}
		return f.DestinationAddress[:]
	default:
		return nil
	}
}

// HasTLVs reports whether the frame contains a non-empty TLV region.
func (f *Frame) HasTLVs() bool {
	return f != nil && len(f.TLVData) != 0
}

// TLV is a borrowed view over one type-length-value record.
type TLV struct {
	Type  TLVType
	Value []byte
}

// TLVIterator iterates over a frame's TLV region without allocating.
type TLVIterator struct {
	remaining []byte
	err       error
}

// Next returns the next TLV. It returns false at the end of the region or on
// malformed input; callers can distinguish those cases with Err.
func (it *TLVIterator) Next() (TLV, bool) {
	if it == nil || it.err != nil || len(it.remaining) == 0 {
		return TLV{}, false
	}
	if len(it.remaining) < 3 {
		it.err = ErrMalformedTLV
		return TLV{}, false
	}
	valueLength := int(binary.BigEndian.Uint16(it.remaining[1:3]))
	if valueLength > len(it.remaining)-3 {
		it.err = ErrMalformedTLV
		return TLV{}, false
	}
	tlv := TLV{
		Type:  TLVType(it.remaining[0]),
		Value: it.remaining[3 : 3+valueLength],
	}
	it.remaining = it.remaining[3+valueLength:]
	return tlv, true
}

// Err returns the terminal iterator error, if any.
func (it *TLVIterator) Err() error {
	if it == nil {
		return ErrMalformedTLV
	}
	return it.err
}

// TLVs returns a zero-allocation iterator over the frame's TLV records.
func (f *Frame) TLVs() TLVIterator {
	if f == nil {
		return TLVIterator{err: ErrMalformedTLV}
	}
	return TLVIterator{remaining: f.TLVData}
}

// String returns a stable command name.
func (c Command) String() string {
	switch c {
	case CommandLocal:
		return "LOCAL"
	case CommandProxy:
		return "PROXY"
	default:
		return "UNKNOWN"
	}
}

// String returns a stable address-family name.
func (f AddressFamily) String() string {
	switch f {
	case AddressFamilyUnspec:
		return "UNSPEC"
	case AddressFamilyINET:
		return "INET"
	case AddressFamilyINET6:
		return "INET6"
	case AddressFamilyUNIX:
		return "UNIX"
	default:
		return "UNKNOWN"
	}
}

// String returns a stable transport-protocol name.
func (t TransportProtocol) String() string {
	switch t {
	case TransportUnspec:
		return "UNSPEC"
	case TransportStream:
		return "STREAM"
	case TransportDgram:
		return "DGRAM"
	default:
		return "UNKNOWN"
	}
}

const (
	netIPv4Length = 4
	netIPv6Length = 16
)

func (c Command) valid() bool {
	return c == CommandLocal || c == CommandProxy
}

func (f AddressFamily) valid() bool {
	return f <= AddressFamilyUNIX
}

func (t TransportProtocol) valid() bool {
	return t <= TransportDgram
}
