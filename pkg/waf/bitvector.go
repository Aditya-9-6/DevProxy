package waf

import "sync/atomic"

// BitVector provides a cache-aligned, lock-free structure for state tracking.
type BitVector struct {
	_    [56]byte // Cache line padding
	bits atomic.Uint64
	_    [56]byte
}

func (b *BitVector) Set(bit uint) {
	b.bits.Store(b.bits.Load() | (1 << bit))
}

func (b *BitVector) Get(bit uint) bool {
	return (b.bits.Load() & (1 << bit)) != 0
}

func (b *BitVector) Reset() {
	b.bits.Store(0)
}
