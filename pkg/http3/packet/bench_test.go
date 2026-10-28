package packet

import (
	"testing"
)

func BenchmarkHeaderProtectionRemoval(b *testing.B) {
	key := make([]byte, 16)
	sample := make([]byte, 16)
	fb := byte(0xc0)
	pn := []byte{0x01, 0x02, 0x03, 0x04}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RemoveHeaderProtection(key, sample, &fb, pn)
	}
}
