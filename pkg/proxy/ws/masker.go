package ws

import (
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

// MaskXOR performs an in-place XOR transformation on the payload using the 4-byte masking key.
// This implementation is optimized for performance and avoids heap allocations.
func MaskXOR(payload []byte, maskKey []byte) {
	if len(maskKey) != 4 {
		return
	}
	for i := 0; i < len(payload); i++ {
		payload[i] ^= maskKey[i%4]
	}
}
