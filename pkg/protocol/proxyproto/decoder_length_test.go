package proxyproto

import "testing"

func TestUnixAddressLengthBoundaryBehavior(t *testing.T) {
	abstract := make([]byte, UnixAddressSize)
	abstract[10] = 0

	tests := []struct {
		name string
		path []byte
		want int
	}{
		{
			name: "empty",
			want: 0,
		},
		{
			name: "pathname stops at NUL",
			path: []byte("/tmp/socket\x00ignored"),
			want: 11,
		},
		{
			name: "unterminated pathname",
			path: make([]byte, UnixAddressSize),
			want: UnixAddressSize,
		},
		{
			name: "abstract address retains embedded NUL",
			path: abstract,
			want: UnixAddressSize,
		},
		{
			name: "maximum representable length",
			path: make([]byte, 255),
			want: 255,
		},
		{
			name: "oversized input saturates",
			path: make([]byte, 256),
			want: 255,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := int(unixAddressLength(test.path)); got != test.want {
				t.Fatalf("unixAddressLength() = %d, want %d", got, test.want)
			}
		})
	}
}
