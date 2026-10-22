package proxyproto

import (
	"errors"
	"runtime"
	"sync"
	"testing"

	wire "github.com/Aditya-9-6/DevProxy/pkg/proxy/proxyproto"
)

type policyValidationCase struct {
	name    string
	policy  *Policy
	frame   wire.Frame
	wantErr error
}

func TestPolicyValidation(t *testing.T) {
	t.Parallel()
	tests := append(policyValidationCases(t), policyDenyValidationCases(t)...)
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.policy.Validate(test.frame)
			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func policyValidationCases(tb testing.TB) []policyValidationCase {
	base := policyFrame(tb, nil)
	allowedTLV := policyFrame(tb, []byte{byte(TLVTypeALPN), 0, 2, 'h', '2'})
	nestedTLV := policyFrame(tb, []byte{byte(TLVTypeSSL), 0, 1, 0})
	duplicateTLV := policyFrame(tb, []byte{
		byte(TLVTypeALPN), 0, 1, 'a', byte(TLVTypeALPN), 0, 1, 'b',
	})
	unknownTLV := policyFrame(tb, []byte{0x7f, 0, 1, 'x'})
	malformed := base
	malformed.TLVData = []byte{byte(TLVTypeALPN), 0}

	custom := mustPolicy(tb, Config{
		AllowedAddressFamilies:    []AddressFamily{AddressFamilyINET},
		AllowedTransportProtocols: []TransportProtocol{TransportStream},
		AllowedTLVTypes:           []TLVType{TLVTypeALPN},
	})
	oneTLV := mustPolicy(tb, Config{
		AllowedAddressFamilies: []AddressFamily{AddressFamilyINET},
		AllowedTLVTypes:        []TLVType{TLVTypeALPN},
		MaxTLVCount:            1,
	})
	fourBytes := mustPolicy(tb, Config{
		AllowedAddressFamilies: []AddressFamily{AddressFamilyINET},
		AllowedTLVTypes:        []TLVType{TLVTypeALPN},
		MaxTLVBytes:            4,
	})
	local := mustPolicy(tb, Config{
		AllowedCommands:           []Command{CommandLocal},
		AllowedAddressFamilies:    []AddressFamily{AddressFamilyUnspec},
		AllowedTransportProtocols: []TransportProtocol{TransportUnspec},
	})

	return []policyValidationCase{
		{name: "default ipv4", policy: DefaultPolicy(), frame: base},
		{name: "default ipv6 dgram", policy: DefaultPolicy(), frame: policyIPv6Frame()},
		{name: "custom tlv", policy: custom, frame: allowedTLV},
		{name: "nested tlv", policy: custom, frame: nestedTLV, wantErr: ErrNestedTLV},
		{name: "duplicate tlv", policy: custom, frame: duplicateTLV, wantErr: ErrDuplicateTLV},
		{name: "unknown tlv", policy: custom, frame: unknownTLV, wantErr: ErrTLVNotAllowed},
		{name: "malformed tlv", policy: custom, frame: malformed, wantErr: ErrMalformedTLV},
		{name: "too many tlvs", policy: oneTLV, frame: duplicateTLV, wantErr: ErrTooManyTLVs},
		{name: "too many bytes", policy: fourBytes, frame: allowedTLV, wantErr: ErrTLVBytesExceeded},
		{name: "local explicitly allowed", policy: local, frame: policyLocalFrame()},
		{name: "local denied by default", policy: DefaultPolicy(), frame: policyLocalFrame(), wantErr: ErrCommandNotAllowed},
	}
}

func policyDenyValidationCases(tb testing.TB) []policyValidationCase {
	base := policyFrame(tb, nil)
	allowedTLV := policyFrame(tb, []byte{byte(TLVTypeALPN), 0, 2, 'h', '2'})
	return []policyValidationCase{
		{name: "nil policy", policy: nil, frame: base, wantErr: ErrInvalidPolicy},
		{name: "empty command allowlist", policy: mustPolicy(tb, Config{AllowedCommands: []Command{}}), frame: base, wantErr: ErrCommandNotAllowed},
		{name: "empty family allowlist", policy: mustPolicy(tb, Config{AllowedAddressFamilies: []AddressFamily{}}), frame: base, wantErr: ErrFamilyNotAllowed},
		{name: "empty transport allowlist", policy: mustPolicy(tb, Config{AllowedTransportProtocols: []TransportProtocol{}}), frame: base, wantErr: ErrTransportNotAllowed},
		{name: "empty tlv allowlist", policy: mustPolicy(tb, Config{AllowedTLVTypes: []TLVType{}}), frame: allowedTLV, wantErr: ErrTLVNotAllowed},
	}
}

func TestNewPolicyRejectsAmbiguousTLV(t *testing.T) {
	t.Parallel()
	_, err := NewPolicy(Config{AllowedTLVTypes: []TLVType{TLVTypeSSL}})
	if !errors.Is(err, ErrNestedTLV) {
		t.Fatalf("NewPolicy() error = %v, want %v", err, ErrNestedTLV)
	}
}

func TestNewPolicyRejectsForbiddenTLVTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tlvType TLVType
	}{
		{name: "reserved", tlvType: TLVTypeReserved},
		{name: "crc32c", tlvType: TLVTypeCRC32C},
		{name: "ssl", tlvType: TLVTypeSSL},
		{name: "ssl version", tlvType: TLVTypeSSLVersion},
		{name: "ssl common name", tlvType: TLVTypeSSLCN},
		{name: "ssl cipher", tlvType: TLVTypeSSLCipher},
		{name: "ssl signature algorithm", tlvType: TLVTypeSSLSigAlg},
		{name: "ssl key algorithm", tlvType: TLVTypeSSLKeyAlg},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPolicy(Config{AllowedTLVTypes: []TLVType{test.tlvType}})
			if !errors.Is(err, ErrNestedTLV) {
				t.Fatalf("NewPolicy() error = %v, want %v", err, ErrNestedTLV)
			}
		})
	}
}

func TestNewPolicyRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config Config
	}{
		{name: "invalid command", config: Config{AllowedCommands: []Command{Command(0xff)}}},
		{name: "invalid family", config: Config{AllowedAddressFamilies: []AddressFamily{AddressFamily(0xff)}}},
		{name: "invalid transport", config: Config{AllowedTransportProtocols: []TransportProtocol{TransportProtocol(0xff)}}},
		{name: "duplicate tlv", config: Config{AllowedTLVTypes: []TLVType{TLVTypeALPN, TLVTypeALPN}}},
		{name: "negative tlv count", config: Config{MaxTLVCount: -1}},
		{name: "tlv count overflow", config: Config{MaxTLVCount: maxPolicyTLVCount + 1}},
		{name: "negative tlv bytes", config: Config{MaxTLVBytes: -1}},
		{name: "tlv bytes exceed frame", config: Config{MaxTLVBytes: wire.MaxPayloadSize + 1}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPolicy(test.config)
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("NewPolicy() error = %v, want %v", err, ErrInvalidPolicy)
			}
		})
	}
}

func TestNewPolicyAcceptsBoundaryConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config Config
	}{
		{name: "defaults", config: Config{}},
		{name: "zero limits use defaults", config: Config{MaxTLVCount: 0, MaxTLVBytes: 0}},
		{name: "maximum tlv count", config: Config{MaxTLVCount: maxPolicyTLVCount}},
		{name: "maximum tlv bytes", config: Config{MaxTLVBytes: wire.MaxPayloadSize}},
		{
			name: "explicitly empty allowlists",
			config: Config{
				AllowedCommands:           []Command{},
				AllowedAddressFamilies:    []AddressFamily{},
				AllowedTransportProtocols: []TransportProtocol{},
				AllowedTLVTypes:           []TLVType{},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			policy, err := NewPolicy(test.config)
			if err != nil {
				t.Fatalf("NewPolicy() error = %v", err)
			}
			if policy == nil {
				t.Fatal("NewPolicy() returned a nil policy")
			}
		})
	}
}

func TestPolicyConfigurationIsImmutable(t *testing.T) {
	t.Parallel()
	families := []AddressFamily{AddressFamilyINET}
	policy, err := NewPolicy(Config{AllowedAddressFamilies: families})
	if err != nil {
		t.Fatalf("NewPolicy() error = %v", err)
	}
	families[0] = AddressFamilyINET6
	frame := policyFrame(t, nil)
	if err := policy.Validate(frame); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPolicyConcurrentValidation(t *testing.T) {
	t.Parallel()
	policy := DefaultPolicy()
	frame := policyFrame(t, nil)
	const workerCount = 32

	var wait sync.WaitGroup
	errorsChannel := make(chan error, workerCount)
	for worker := 0; worker < workerCount; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := 0; iteration < 1000; iteration++ {
				if err := policy.Validate(frame); err != nil {
					errorsChannel <- err
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPolicyAllocations(t *testing.T) {
	// AllocsPerRun requires a stable process-wide allocation baseline, so this
	// measurement intentionally remains sequential.
	tests := []struct {
		name   string
		policy *Policy
		frame  wire.Frame
	}{
		{name: "without tlv", policy: DefaultPolicy(), frame: policyFrame(t, nil)},
		{
			name:   "with tlv",
			policy: mustPolicy(t, Config{AllowedTLVTypes: []TLVType{TLVTypeALPN}}),
			frame:  policyFrame(t, []byte{byte(TLVTypeALPN), 0, 2, 'h', '2'}),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			allocations := testing.AllocsPerRun(1000, func() {
				if err := test.policy.Validate(test.frame); err != nil {
					panic(err)
				}
				runtime.KeepAlive(test.policy)
				runtime.KeepAlive(test.frame)
			})
			if allocations != 0 {
				t.Fatalf("Validate() allocations = %v, want 0", allocations)
			}
		})
	}
}

func BenchmarkPolicyValidate(b *testing.B) {
	policy := DefaultPolicy()
	frame := policyFrame(b, nil)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if err := policy.Validate(frame); err != nil {
			b.Fatal(err)
		}
	}
}

func mustPolicy(tb testing.TB, config Config) *Policy {
	tb.Helper()
	policy, err := NewPolicy(config)
	if err != nil {
		tb.Fatal(err)
	}
	return policy
}

func policyLocalFrame() wire.Frame {
	return wire.Frame{
		Command:   wire.CommandLocal,
		Family:    wire.AddressFamilyUnspec,
		Transport: wire.TransportUnspec,
	}
}

func policyIPv6Frame() wire.Frame {
	return wire.Frame{
		Command:   wire.CommandProxy,
		Family:    wire.AddressFamilyINET6,
		Transport: wire.TransportDgram,
		SourceAddress: [16]byte{
			0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		},
		DestinationAddress: [16]byte{
			0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2,
		},
		SourcePort:      53,
		DestinationPort: 5353,
	}
}

func policyFrame(tb testing.TB, tlv []byte) wire.Frame {
	tb.Helper()
	frame := wire.Frame{
		Command: wire.CommandProxy, Family: wire.AddressFamilyINET, Transport: wire.TransportStream,
		SourceAddress: [16]byte{192, 0, 2, 1}, DestinationAddress: [16]byte{198, 51, 100, 2},
		SourcePort: 12345, DestinationPort: 443, TLVData: tlv,
	}
	buffer := make([]byte, wire.MaxFrameSize)
	written, err := wire.Encode(buffer, frame)
	if err != nil {
		tb.Fatal(err)
	}
	decoded, err := wire.Decode(buffer[:written])
	if err != nil {
		tb.Fatal(err)
	}
	return decoded
}
