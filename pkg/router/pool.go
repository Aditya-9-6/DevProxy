package router

import "sync"

var pathPool = sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 16)
	},
}

func GetPathBuffer() []string {
	return pathPool.Get().([]string)
}

func PutPathBuffer(b []string) {
	pathPool.Put(b[:0])
}
