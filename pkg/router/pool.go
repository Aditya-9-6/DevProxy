package router

import "sync"

// pathPool provides a pool of string slices to reduce allocations during path processing.
var pathPool = sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 16)
	},
}

// GetPathBuffer retrieves a string slice from the pool.
func GetPathBuffer() []string {
	return pathPool.Get().([]string)
}

// PutPathBuffer returns a string slice to the pool after resetting its length.
func PutPathBuffer(b []string) {
	pathPool.Put(b[:0])
}
