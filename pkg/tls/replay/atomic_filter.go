package replay

import (
	"sync/atomic"
)

// AtomicBloomFilter provides a thread-safe, lock-free membership test.
// Uses a bit-array of uint64 to minimize memory footprint.
type AtomicBloomFilter struct {
	bits []atomic.Uint64
	size uint64
}

func NewAtomicBloomFilter(size uint64) *AtomicBloomFilter {
	return &AtomicBloomFilter{
		bits: make([]atomic.Uint64, (size+63)/64),
		size: size,
	}
}

func (f *AtomicBloomFilter) Add(hash uint64) {
	idx := hash % f.size
	f.bits[idx/64].Or(1 << (idx % 64))
}

func (f *AtomicBloomFilter) Test(hash uint64) bool {
	idx := hash % f.size
	return (f.bits[idx/64].Load() & (1 << (idx % 64))) != 0
}

func (f *AtomicBloomFilter) Reset() {
	for i := range f.bits {
		f.bits[i].Store(0)
	}
}
