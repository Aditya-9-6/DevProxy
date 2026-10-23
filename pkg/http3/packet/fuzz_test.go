package packet

import (
	"bytes"
	"testing"
)

func FuzzParseVarInt(f *testing.F) {
	f.Add([]byte{0x25, 0x85})
	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		_, _, _ = ParseVarInt(r)
	})
}
