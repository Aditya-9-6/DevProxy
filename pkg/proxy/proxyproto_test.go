package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/netip"
	"sync"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/protocol/proxyproto"
)

func TestNewPROXYProtocolGate(t *testing.T) {
	validPrefix := netip.PrefixFrom(netip.MustParseAddr("10.0.0.0"), 8)
	tests := []struct {
		name    string
		config  PROXYProtocolConfig
		wantErr error
	}{
		{
			name: "disabled by default",
		},
		{
			name: "enabled with trusted CIDR",
			config: PROXYProtocolConfig{
				Enabled:      true,
				TrustedCIDRs: []netip.Prefix{validPrefix},
			},
		},
		{
			name: "enabled without trusted CIDR",
			config: PROXYProtocolConfig{
				Enabled: true,
			},
			wantErr: ErrPROXYProtocolNoTrustedCIDRs,
		},
		{
			name: "invalid trusted CIDR",
			config: PROXYProtocolConfig{
				Enabled:      true,
				TrustedCIDRs: []netip.Prefix{{}},
			},
			wantErr: ErrPROXYProtocolInvalidTrustedCIDR,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gate, err := NewPROXYProtocolGate(test.config)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("NewPROXYProtocolGate() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPROXYProtocolGate() error = %v", err)
			}
			if gate == nil {
				t.Fatal("NewPROXYProtocolGate() returned a nil gate")
			}
		})
	}
}

func TestPROXYProtocolGateAcceptsTrustedPeerAndPreservesApplicationBytes(t *testing.T) {
	gate := newTestPROXYGate(t)
	payload := []byte{192, 0, 2, 1, 198, 51, 100, 2, 0x1f, 0x90, 0, 80}
	frame := proxyTestFrame(byte(proxyproto.CommandProxy), byte(proxyproto.FamilyINET), byte(proxyproto.ProtocolStream), payload)
	application := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	stream := make([]byte, 0, len(frame)+len(application))
	stream = append(stream, frame...)
	stream = append(stream, application...)
	reader := bytes.NewReader(stream)

	header, err := gate.Handshake(netip.MustParseAddr("10.1.2.3"), reader)
	if err != nil {
		t.Fatalf("Handshake() error = %v", err)
	}
	wantSource := netip.AddrPortFrom(netip.MustParseAddr("192.0.2.1"), 8080)
	if header.Source != wantSource {
		t.Fatalf("Source = %v, want %v", header.Source, wantSource)
	}
	if reader.Len() != len(application) {
		t.Fatalf("remaining bytes = %d, want %d", reader.Len(), len(application))
	}
	remaining := make([]byte, len(application))
	if _, err := reader.Read(remaining); err != nil {
		t.Fatalf("reading application bytes: %v", err)
	}
	if !bytes.Equal(remaining, application) {
		t.Fatalf("application bytes changed: %q", remaining)
	}
}

func TestPROXYProtocolGateRejectsUntrustedPeerBeforeReading(t *testing.T) {
	gate := newTestPROXYGate(t)
	frame := proxyTestFrame(
		byte(proxyproto.CommandProxy),
		byte(proxyproto.FamilyINET),
		byte(proxyproto.ProtocolStream),
		make([]byte, proxyproto.INETPayloadLength),
	)
	application := []byte("GET / HTTP/1.1\r\n\r\n")
	stream := make([]byte, 0, len(frame)+len(application))
	stream = append(stream, frame...)
	stream = append(stream, application...)
	reader := bytes.NewReader(stream)

	_, err := gate.Handshake(netip.MustParseAddr("192.0.2.10"), reader)
	if !errors.Is(err, ErrPROXYProtocolUntrustedPeer) {
		t.Fatalf("Handshake() error = %v, want %v", err, ErrPROXYProtocolUntrustedPeer)
	}
	if reader.Len() != len(stream) {
		t.Fatalf("untrusted peer consumed %d bytes", len(stream)-reader.Len())
	}
}

func TestPROXYProtocolGateDisabledDoesNotRead(t *testing.T) {
	gate, err := NewPROXYProtocolGate(PROXYProtocolConfig{})
	if err != nil {
		t.Fatalf("NewPROXYProtocolGate() error = %v", err)
	}
	reader := bytes.NewReader([]byte("GET / HTTP/1.1\r\n\r\n"))

	_, err = gate.Handshake(netip.MustParseAddr("10.1.2.3"), reader)
	if !errors.Is(err, ErrPROXYProtocolDisabled) {
		t.Fatalf("Handshake() error = %v, want %v", err, ErrPROXYProtocolDisabled)
	}
	if reader.Len() == 0 {
		t.Fatal("disabled gate consumed application bytes")
	}
}

func TestPROXYProtocolGateRejectsMalformedHeaderWithoutConsumingApplication(t *testing.T) {
	gate := newTestPROXYGate(t)
	frame := proxyTestFrame(
		byte(proxyproto.CommandProxy),
		byte(proxyproto.FamilyINET),
		byte(proxyproto.ProtocolStream),
		make([]byte, proxyproto.INETPayloadLength+1),
	)
	application := []byte("GET / HTTP/1.1\r\n\r\n")
	stream := make([]byte, 0, proxyproto.HeaderSize+len(application))
	stream = append(stream, frame[:proxyproto.HeaderSize]...)
	stream = append(stream, application...)
	reader := bytes.NewReader(stream)

	_, err := gate.Handshake(netip.MustParseAddr("10.1.2.3"), reader)
	if !errors.Is(err, proxyproto.ErrInvalidLength) {
		t.Fatalf("Handshake() error = %v, want %v", err, proxyproto.ErrInvalidLength)
	}
	if reader.Len() != len(application) {
		t.Fatalf("remaining bytes = %d, want %d", reader.Len(), len(application))
	}
}

func TestPROXYProtocolGatePreservesNetworkError(t *testing.T) {
	gate := newTestPROXYGate(t)
	wantErr := errors.New("network read failed")

	_, err := gate.Handshake(netip.MustParseAddr("10.1.2.3"), proxyErrorReader{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Handshake() error = %v, want %v", err, wantErr)
	}
}

func TestPROXYProtocolGateConcurrentUse(t *testing.T) {
	gate := newTestPROXYGate(t)
	frame := proxyTestFrame(
		byte(proxyproto.CommandProxy),
		byte(proxyproto.FamilyINET),
		byte(proxyproto.ProtocolStream),
		make([]byte, proxyproto.INETPayloadLength),
	)

	var waitGroup sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			reader := bytes.NewReader(frame)
			if _, err := gate.Handshake(netip.MustParseAddr("10.1.2.3"), reader); err != nil {
				t.Errorf("Handshake() error = %v", err)
			}
		}()
	}
	waitGroup.Wait()
}

func newTestPROXYGate(t *testing.T) *PROXYProtocolGate {
	t.Helper()
	gate, err := NewPROXYProtocolGate(PROXYProtocolConfig{
		Enabled: true,
		TrustedCIDRs: []netip.Prefix{
			netip.PrefixFrom(netip.MustParseAddr("10.0.0.0"), 8),
			netip.PrefixFrom(netip.MustParseAddr("2001:db8::"), 32),
		},
	})
	if err != nil {
		t.Fatalf("NewPROXYProtocolGate() error = %v", err)
	}
	return gate
}

func proxyTestFrame(command, family, protocol byte, payload []byte) []byte {
	frame := make([]byte, proxyproto.HeaderSize+len(payload))
	copy(frame[:12], proxyproto.Signature)
	frame[12] = 0x20 | (command & 0x0f)
	frame[13] = (family << 4) | (protocol & 0x0f)
	binary.BigEndian.PutUint16(frame[14:16], uint16(len(payload)))
	copy(frame[proxyproto.HeaderSize:], payload)
	return frame
}

type proxyErrorReader struct {
	err error
}

func (reader proxyErrorReader) Read([]byte) (int, error) {
	return 0, reader.err
}
