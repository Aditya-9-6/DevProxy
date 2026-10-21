package proxy

import (
	"errors"
	"io"
	"net/netip"

	"github.com/Aditya-9-6/DevProxy/pkg/protocol/proxyproto"
)

var (
	ErrPROXYProtocolDisabled           = errors.New("proxy: PROXY protocol support is disabled")
	ErrPROXYProtocolUntrustedPeer      = errors.New("proxy: PROXY protocol peer is not trusted")
	ErrPROXYProtocolNoTrustedCIDRs     = errors.New("proxy: PROXY protocol requires at least one trusted CIDR")
	ErrPROXYProtocolInvalidTrustedCIDR = errors.New("proxy: invalid PROXY protocol trusted CIDR")
)

// PROXYProtocolConfig explicitly controls PROXY protocol v2 support. The
// feature remains disabled unless Enabled is true, and enabling it without a
// trusted CIDR is rejected so that it can never fail open.
type PROXYProtocolConfig struct {
	Enabled      bool
	TrustedCIDRs []netip.Prefix
}

// PROXYProtocolGate validates the real connection peer against trusted CIDRs
// before decoding a PROXY header. It is immutable after construction and is
// safe for concurrent use.
type PROXYProtocolGate struct {
	enabled bool
	trusted []netip.Prefix
	decoder proxyproto.Decoder
}

// NewPROXYProtocolGate constructs a trusted-CIDR gate for the upstream
// handshake boundary.
func NewPROXYProtocolGate(config PROXYProtocolConfig) (*PROXYProtocolGate, error) {
	trusted := make([]netip.Prefix, 0, len(config.TrustedCIDRs))
	for _, prefix := range config.TrustedCIDRs {
		normalized, ok := normalizeTrustedPrefix(prefix)
		if !ok {
			return nil, ErrPROXYProtocolInvalidTrustedCIDR
		}
		trusted = append(trusted, normalized)
	}
	if config.Enabled && len(trusted) == 0 {
		return nil, ErrPROXYProtocolNoTrustedCIDRs
	}
	return &PROXYProtocolGate{
		enabled: config.Enabled,
		trusted: trusted,
		decoder: proxyproto.NewDecoder(),
	}, nil
}

// Enabled reports whether PROXY protocol handshake processing is active.
func (gate *PROXYProtocolGate) Enabled() bool {
	return gate != nil && gate.enabled
}

// IsTrusted reports whether the actual connection peer belongs to a trusted
// CIDR. IPv4-mapped addresses and IPv6 zones are normalized before matching.
func (gate *PROXYProtocolGate) IsTrusted(peer netip.Addr) bool {
	if gate == nil || !gate.enabled || !peer.IsValid() {
		return false
	}
	peer = peer.Unmap()
	if peer.Is6() {
		peer = peer.WithZone("")
	}
	for _, prefix := range gate.trusted {
		if prefix.Contains(peer) {
			return true
		}
	}
	return false
}

// Handshake must be called before the connection is handed to the HTTP
// protocol parser. Untrusted peers are rejected before any byte is read, so a
// spoofed PROXY header cannot consume application-protocol bytes. A trusted
// malformed header fails closed with the protocol decoder's error.
func (gate *PROXYProtocolGate) Handshake(peer netip.Addr, stream io.Reader) (proxyproto.Header, error) {
	if gate == nil || !gate.enabled {
		return proxyproto.Header{}, ErrPROXYProtocolDisabled
	}
	if !gate.IsTrusted(peer) {
		return proxyproto.Header{}, ErrPROXYProtocolUntrustedPeer
	}
	return gate.decoder.Decode(stream)
}

// Decode is an alias for Handshake for callers using decoder terminology.
func (gate *PROXYProtocolGate) Decode(peer netip.Addr, stream io.Reader) (proxyproto.Header, error) {
	return gate.Handshake(peer, stream)
}

func normalizeTrustedPrefix(prefix netip.Prefix) (netip.Prefix, bool) {
	if !prefix.IsValid() {
		return netip.Prefix{}, false
	}
	address := prefix.Addr()
	bits := prefix.Bits()
	if address.Is4In6() {
		if bits < 96 {
			return netip.Prefix{}, false
		}
		address = address.Unmap()
		bits -= 96
	}
	normalized := netip.PrefixFrom(address, bits)
	if !normalized.IsValid() {
		return netip.Prefix{}, false
	}
	return normalized.Masked(), true
}
