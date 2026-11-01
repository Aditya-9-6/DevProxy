package frame

import (
	"testing"
)

func FuzzDecoder(f *testing.F) {
	f.Add([]byte("hello"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = ValidateUTF8(data)
	})
}