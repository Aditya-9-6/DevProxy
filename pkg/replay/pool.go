package replay

import (
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return NewReplayBuffer(32 * 1024)
	},
}

func AcquireBuffer() *ReplayBuffer {
	buf := bufferPool.Get().(*ReplayBuffer)
	buf.Reset()
	return buf
}

func ReleaseBuffer(buf *ReplayBuffer) {
	if buf != nil {
		bufferPool.Put(buf)
	}
}
