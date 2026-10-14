package waf

import (
	"hash"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

// CookieValidator implements a thread-safe bloom filter for cookie integrity.
type CookieValidator struct {
	bits     []uint64
	size     uint32
	hashPool sync.Pool
}

func NewCookieValidator(size uint32) *CookieValidator {
	return &CookieValidator{
		size: size,
		bits: make([]uint64, (size/64)+1),
		hashPool: sync.Pool{
			New: func() interface{} {
				return fnv.New32a()
			},
		},
	}
}

func (cv *CookieValidator) hash(data []byte) uint32 {
	h := cv.hashPool.Get().(hash.Hash32)
	h.Reset()
	h.Write(data)
	sum := h.Sum32()
	cv.hashPool.Put(h)
	return sum % cv.size
}

func (cv *CookieValidator) IsTampered(cookieData []byte) bool {
	h := cv.hash(cookieData)
	idx := h / 64
	bit := uint64(1) << (h % 64)
	return (atomic.LoadUint64(&cv.bits[idx]) & bit) == 0
}

func (cv *CookieValidator) MarkValid(cookieData []byte) {
	h := cv.hash(cookieData)
	idx := h / 64
	bit := uint64(1) << (h % 64)
	for {
		old := atomic.LoadUint64(&cv.bits[idx])
		new := old | bit
		if atomic.CompareAndSwapUint64(&cv.bits[idx], old, new) {
			break
		}
	}
}
