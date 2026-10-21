package proxyproto

import "testing"

func TestUnixAddressLengthAvoidsIntegerOverflow(t *testing.T) {
	pathname := make([]byte, UnixAddressSize)
	copy(pathname, "/tmp/source")
	pathname[len("/tmp/source")] = 0

	unterminated := make([]byte, UnixAddressSize)
	for index := range unterminated {
		unterminated[index] = 'x'
	}

	abstract := make([]byte, UnixAddressSize)
	abstract[1] = 0x42

	oversized := make([]byte, 256)
	for index := range oversized {
		oversized[index] = 'x'
	}

	tests := []struct {
		name    string
		address []byte
		want    uint8
	}{
		{name: "nil", address: nil, want: 0},
		{name: "empty", address: []byte{}, want: 0},
		{name: "terminated pathname", address: pathname, want: 11},
		{name: "unterminated pathname", address: unterminated, want: UnixAddressSize},
		{name: "abstract address", address: abstract, want: UnixAddressSize},
		{name: "oversized address", address: oversized, want: 255},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := unixAddressLength(test.address); got != test.want {
				t.Fatalf("unixAddressLength() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestDecoderNilReaderReturnsTruncatedHeader(t *testing.T) {
	if _, err := (V2Decoder{}).Decode(nil); err != ErrTruncatedHeader {
		t.Fatalf("Decode() error = %v, want %v", err, ErrTruncatedHeader)
	}
}
