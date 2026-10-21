package proxyproto

import (
	"bytes"
	"testing"
)

func TestUnixAddressLengthNoWrapRegression(t *testing.T) {
	tests := []struct {
		name string
		path []byte
		want uint8
	}{
		{
			name: "empty",
			path: nil,
			want: 0,
		},
		{
			name: "pathname",
			path: []byte("/tmp/source"),
			want: 11,
		},
		{
			name: "terminated pathname",
			path: []byte{'/', 't', 'm', 'p', 0, 'x'},
			want: 4,
		},
		{
			name: "abstract address ignores terminator",
			path: []byte{0, 'a', 'b', 0},
			want: 4,
		},
		{
			name: "maximum fixed address",
			path: bytes.Repeat([]byte{'x'}, UnixAddressSize),
			want: UnixAddressSize,
		},
		{
			name: "oversized input saturates",
			path: bytes.Repeat([]byte{'x'}, 256),
			want: 0xff,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := unixAddressLength(test.path); got != test.want {
				t.Fatalf("unixAddressLength() = %d, want %d", got, test.want)
			}
		})
	}
}
