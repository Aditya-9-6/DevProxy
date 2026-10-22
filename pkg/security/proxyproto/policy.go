package proxyproto

import (
	"errors"
	"fmt"

	wire "github.com/Aditya-9-6/DevProxy/pkg/proxy/proxyproto"
)

// AddressFamily aliases the wire-level address-family type.
type AddressFamily = wire.AddressFamily

// TransportProtocol aliases the wire-level transport type.
type TransportProtocol = wire.TransportProtocol

// Command aliases the wire-level command type.
type Command = wire.Command

// TLVType aliases the wire-level TLV type.
type TLVType = wire.TLVType

const (
	AddressFamilyUnspec = wire.AddressFamilyUnspec
	AddressFamilyINET   = wire.AddressFamilyINET
	AddressFamilyINET6  = wire.AddressFamilyINET6
	AddressFamilyUNIX   = wire.AddressFamilyUNIX
	TransportUnspec     = wire.TransportUnspec
	TransportStream     = wire.TransportStream
	TransportDgram      = wire.TransportDgram
	CommandLocal        = wire.CommandLocal
	CommandProxy        = wire.CommandProxy
	TLVTypeALPN         = wire.TLVTypeALPN
	TLVTypeAuthority    = wire.TLVTypeAuthority
	TLVTypeCRC32C       = wire.TLVTypeCRC32C
	TLVTypeNoop         = wire.TLVTypeNoop
	TLVTypeUniqueID     = wire.TLVTypeUniqueID
	TLVTypeSSL          = wire.TLVTypeSSL
	TLVTypeSSLVersion   = wire.TLVTypeSSLVersion
	TLVTypeSSLCN        = wire.TLVTypeSSLCN
	TLVTypeSSLCipher    = wire.TLVTypeSSLCipher
	TLVTypeSSLSigAlg    = wire.TLVTypeSSLSigAlg
	TLVTypeSSLKeyAlg    = wire.TLVTypeSSLKeyAlg
	TLVTypeNetNS        = wire.TLVTypeNetNS

	// TLVTypeReserved is the reserved TLV type and cannot be allowed by a policy.
	TLVTypeReserved TLVType = 0x00
)

var (
	// ErrInvalidPolicy indicates an invalid policy configuration.
	ErrInvalidPolicy = errors.New("proxyproto: invalid policy")
	// ErrCommandNotAllowed indicates that the frame command is not allowlisted.
	ErrCommandNotAllowed = errors.New("proxyproto: command not allowed")
	// ErrFamilyNotAllowed indicates that the address family is not allowlisted.
	ErrFamilyNotAllowed = errors.New("proxyproto: address family not allowed")
	// ErrTransportNotAllowed indicates that the transport is not allowlisted.
	ErrTransportNotAllowed = errors.New("proxyproto: transport not allowed")
	// ErrTLVNotAllowed indicates that a TLV type is not allowlisted.
	ErrTLVNotAllowed = errors.New("proxyproto: TLV not allowed")
	// ErrDuplicateTLV indicates an ambiguous repeated TLV type.
	ErrDuplicateTLV = errors.New("proxyproto: duplicate TLV")
	// ErrNestedTLV indicates a nested, orphaned, or checksum-ambiguous TLV.
	ErrNestedTLV = errors.New("proxyproto: nested or ambiguous TLV")
	// ErrTooManyTLVs indicates that the TLV count limit was exceeded.
	ErrTooManyTLVs = errors.New("proxyproto: too many TLVs")
	// ErrTLVBytesExceeded indicates that the TLV byte limit was exceeded.
	ErrTLVBytesExceeded = errors.New("proxyproto: TLV byte limit exceeded")
	// ErrMalformedTLV indicates malformed TLV framing.
	ErrMalformedTLV = wire.ErrMalformedTLV
)

const (
	defaultMaxTLVCount = 32
	defaultMaxTLVBytes = 4096
	maxPolicyTLVCount  = (1 << 16) - 1
)

// Config defines an immutable PROXY protocol ingress policy.
// Nil allowlists select secure defaults. Empty non-nil allowlists deny all.
type Config struct {
	AllowedCommands           []Command
	AllowedAddressFamilies    []AddressFamily
	AllowedTransportProtocols []TransportProtocol
	AllowedTLVTypes           []TLVType
	MaxTLVCount               int
	MaxTLVBytes               int
}

// Policy is an immutable, concurrency-safe PROXY protocol security policy.
type Policy struct {
	commandMask   uint8
	familyMask    uint8
	transportMask uint8
	tlvMask       [32]uint8
	maxTLVCount   uint16
	maxTLVBytes   uint32
}

// DefaultPolicy returns a strict policy that accepts PROXY IPv4 and IPv6
// STREAM or DGRAM frames and rejects every TLV.
func DefaultPolicy() *Policy {
	policy, err := NewPolicy(Config{})
	if err != nil {
		panic(err)
	}
	return policy
}

// NewPolicy validates and compiles a security policy.
func NewPolicy(config Config) (*Policy, error) {
	policy := &Policy{
		maxTLVCount: uint16(defaultMaxTLVCount),
		maxTLVBytes: uint32(defaultMaxTLVBytes),
	}

	commandMask, err := buildCommandMask(config.AllowedCommands)
	if err != nil {
		return nil, err
	}
	policy.commandMask = commandMask

	familyMask, err := buildFamilyMask(config.AllowedAddressFamilies)
	if err != nil {
		return nil, err
	}
	policy.familyMask = familyMask

	transportMask, err := buildTransportMask(config.AllowedTransportProtocols)
	if err != nil {
		return nil, err
	}
	policy.transportMask = transportMask

	tlvMask, err := buildTLVMask(config.AllowedTLVTypes)
	if err != nil {
		return nil, err
	}
	policy.tlvMask = tlvMask

	if err := applyPolicyLimits(policy, config); err != nil {
		return nil, err
	}
	return policy, nil
}

func applyPolicyLimits(policy *Policy, config Config) error {
	if config.MaxTLVCount < 0 {
		return fmt.Errorf("%w: negative TLV count", ErrInvalidPolicy)
	}
	if config.MaxTLVCount > maxPolicyTLVCount {
		return fmt.Errorf("%w: TLV count exceeds %d", ErrInvalidPolicy, maxPolicyTLVCount)
	}
	if config.MaxTLVCount > 0 {
		policy.maxTLVCount = uint16(config.MaxTLVCount)
	}

	if config.MaxTLVBytes < 0 {
		return fmt.Errorf("%w: negative TLV byte limit", ErrInvalidPolicy)
	}
	if config.MaxTLVBytes > wire.MaxPayloadSize {
		return fmt.Errorf("%w: TLV byte limit exceeds frame limit", ErrInvalidPolicy)
	}
	if config.MaxTLVBytes > 0 {
		policy.maxTLVBytes = uint32(config.MaxTLVBytes)
	}
	return nil
}

func buildCommandMask(commands []Command) (uint8, error) {
	if commands == nil {
		commands = []Command{CommandProxy}
	}
	var mask uint8
	for _, command := range commands {
		if command != CommandLocal && command != CommandProxy {
			return 0, fmt.Errorf("%w: command %d", ErrInvalidPolicy, command)
		}
		mask |= 1 << uint(command)
	}
	return mask, nil
}

func buildFamilyMask(families []AddressFamily) (uint8, error) {
	if families == nil {
		families = []AddressFamily{AddressFamilyINET, AddressFamilyINET6}
	}
	var mask uint8
	for _, family := range families {
		if family > AddressFamilyUNIX {
			return 0, fmt.Errorf("%w: address family %d", ErrInvalidPolicy, family)
		}
		mask |= 1 << uint(family)
	}
	return mask, nil
}

func buildTransportMask(transports []TransportProtocol) (uint8, error) {
	if transports == nil {
		transports = []TransportProtocol{TransportStream, TransportDgram}
	}
	var mask uint8
	for _, transport := range transports {
		if transport > TransportDgram {
			return 0, fmt.Errorf("%w: transport %d", ErrInvalidPolicy, transport)
		}
		mask |= 1 << uint(transport)
	}
	return mask, nil
}

func buildTLVMask(types []TLVType) ([32]uint8, error) {
	var mask [32]uint8
	seen := [256]bool{}
	for _, tlvType := range types {
		if seen[tlvType] {
			return mask, fmt.Errorf("%w: duplicate allowed TLV type %d", ErrInvalidPolicy, tlvType)
		}
		seen[tlvType] = true
		if isForbiddenTLV(tlvType) {
			return mask, fmt.Errorf("%w: TLV type %d", ErrNestedTLV, tlvType)
		}
		index := int(tlvType)
		mask[index>>3] |= 1 << uint(index&7)
	}
	return mask, nil
}

// Validate checks a decoded frame against the complete policy.
func (p *Policy) Validate(frame wire.Frame) error {
	if p == nil {
		return fmt.Errorf("%w: nil policy", ErrInvalidPolicy)
	}
	if p.commandMask&(1<<uint(frame.Command)) == 0 {
		return fmt.Errorf("%w: %s", ErrCommandNotAllowed, frame.Command)
	}
	if p.familyMask&(1<<uint(frame.Family)) == 0 {
		return fmt.Errorf("%w: %s", ErrFamilyNotAllowed, frame.Family)
	}
	if p.transportMask&(1<<uint(frame.Transport)) == 0 {
		return fmt.Errorf("%w: %s", ErrTransportNotAllowed, frame.Transport)
	}
	return p.ValidateTLVs(frame)
}

// ValidateTLVs applies bounded iteration, allowlisting, duplicate detection,
// and nested/ambiguous TLV rejection to a frame's TLV region.
func (p *Policy) ValidateTLVs(frame wire.Frame) error {
	if p == nil {
		return fmt.Errorf("%w: nil policy", ErrInvalidPolicy)
	}
	if uint32(len(frame.TLVData)) > p.maxTLVBytes {
		return fmt.Errorf("%w: %d > %d", ErrTLVBytesExceeded, len(frame.TLVData), p.maxTLVBytes)
	}
	iterator := frame.TLVs()
	var seen [256]bool
	var count uint16
	var encodedBytes uint32
	for {
		tlv, ok := iterator.Next()
		if !ok {
			if err := iterator.Err(); err != nil {
				return fmt.Errorf("%w: %v", ErrMalformedTLV, err)
			}
			break
		}
		count++
		if count > p.maxTLVCount {
			return fmt.Errorf("%w: %d > %d", ErrTooManyTLVs, count, p.maxTLVCount)
		}
		if isForbiddenTLV(tlv.Type) {
			return fmt.Errorf("%w: type %d", ErrNestedTLV, tlv.Type)
		}
		if seen[tlv.Type] {
			return fmt.Errorf("%w: type %d", ErrDuplicateTLV, tlv.Type)
		}
		seen[tlv.Type] = true
		if !p.tlvAllowed(tlv.Type) {
			return fmt.Errorf("%w: type %d", ErrTLVNotAllowed, tlv.Type)
		}
		encodedBytes += uint32(3 + len(tlv.Value))
		if encodedBytes > p.maxTLVBytes {
			return fmt.Errorf("%w: %d > %d", ErrTLVBytesExceeded, encodedBytes, p.maxTLVBytes)
		}
	}
	return nil
}

func (p *Policy) tlvAllowed(tlvType TLVType) bool {
	index := int(tlvType)
	return p.tlvMask[index>>3]&(1<<uint(index&7)) != 0
}

func isForbiddenTLV(tlvType TLVType) bool {
	switch tlvType {
	case TLVTypeReserved, TLVTypeCRC32C, TLVTypeSSL, TLVTypeSSLVersion,
		TLVTypeSSLCN, TLVTypeSSLCipher, TLVTypeSSLSigAlg, TLVTypeSSLKeyAlg:
		return true
	default:
		return false
	}
}
