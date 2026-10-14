package waf

import (
	"hash/fnv"
	"sync"
)

// CookieValidator implements a lock-free bloom filter for cookie integrity.
// It uses sync.Pool to manage bitsets without heap allocations on the hot path.
type CookieValidator struct {
	pool sync.Pool
	size uint32
}

func NewCookieValidator(size uint32) *CookieValidator {
	return &CookieValidator{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size/8+1)
			},
		},
	}
}

func (cv *CookieValidator) hash(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32() % cv.size
}

func (cv *CookieValidator) IsTampered(cookieData []byte) bool {
	buf := cv.pool.Get().([]byte)
	defer cv.pool.Put(buf)

	h := cv.hash(cookieData)
	return (buf[h/8] & (1 << (h % 8))) == 0
}

func (cv *CookieValidator) MarkValid(cookieData []byte) {
	buf := cv.pool.Get().([]byte)
	defer cv.pool.Put(buf)

	h := cv.hash(cookieData)
	buf[h/8] |= (1 << (h % 8))
}
