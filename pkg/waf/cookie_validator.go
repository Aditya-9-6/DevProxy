package waf

import (
	"hash"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

// CookieValidator implements a thread-safe bloom filter for cookie integrity.
// Uses double hashing to minimize collision probability.
// False positive rate is approximately (1 - e^(-kn/m))^k where k=2.
type CookieValidator struct {
	_        [56]byte
	bits     []uint64
	size     uint32
	hashPool sync.Pool
	_        [56]byte
}

func NewCookieValidator(size uint32) *CookieValidator {
	return &CookieValidator{
		size: size,
		bits: make([]uint64, (size/64)+1),
		hashPool: sync.Pool{
			New: func() interface{} {
				return struct {
					h1 hash.Hash32
					h2 hash.Hash32
				}{
					h1: fnv.New32a(),
					h2: fnv.New32(),
				}
			},
		},
	}
}

func (cv *CookieValidator) hashes(data []byte) (uint64, uint64) {
	h := cv.hashPool.Get().(struct{ h1, h2 hash.Hash32 })
	h.h1.Reset()
	h.h2.Reset()
	h.h1.Write(data)
	h.h2.Write(data)
	s1, s2 := h.h1.Sum32(), h.h2.Sum32()
	cv.hashPool.Put(h)
	return uint64(s1 % cv.size), uint64(s2 % cv.size)
}

func (cv *CookieValidator) IsTampered(cookieData []byte) bool {
	h1, h2 := cv.hashes(cookieData)
	idx1, idx2 := h1/64, h2/64
	bit1, bit2 := uint64(1)<<(h1%64), uint64(1)<<(h2%64)
	// Check bits atomically. If either bit is 0, it is considered tampered.
	return (atomic.LoadUint64(&cv.bits[idx1])&bit1) == 0 || (atomic.LoadUint64(&cv.bits[idx2])&bit2) == 0
}

func (cv *CookieValidator) MarkValid(cookieData []byte) {
	h1, h2 := cv.hashes(cookieData)
	idx1, idx2 := h1/64, h2/64
	bit1, bit2 := uint64(1)<<(h1%64), uint64(1)<<(h2%64)
	for _, pair := range [][2]uint64{{idx1, bit1}, {idx2, bit2}} {
		idx, bit := pair[0], pair[1]
		for {
			old := atomic.LoadUint64(&cv.bits[idx])
			if (old & bit) != 0 {
				break
			}
			new := old | bit
			if atomic.CompareAndSwapUint64(&cv.bits[idx], old, new) {
				break
			}
		}
	}
}
