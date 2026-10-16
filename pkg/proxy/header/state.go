package header

import (
	"sync"
	"sync/atomic"
)

const ShardCount = 32

type HeaderBlob struct {
	Data []byte
	Refs atomic.Int64
}

var blobPool = sync.Pool{
	New: func() interface{} {
		return &HeaderBlob{}
	},
}

type StateMachine struct {
	shards [ShardCount]*sync.Map
}

func NewStateMachine() *StateMachine {
	s := &StateMachine{}
	for i := 0; i < ShardCount; i++ {
		s.shards[i] = &sync.Map{}
	}
	return s
}

func (s *StateMachine) getShard(hash uint64) *sync.Map {
	return s.shards[hash%ShardCount]
}

func (s *StateMachine) GetOrStore(hash uint64, data []byte) *HeaderBlob {
	shard := s.getShard(hash)
	if val, ok := shard.Load(hash); ok {
		return val.(*HeaderBlob)
	}

	newBlob := blobPool.Get().(*HeaderBlob)
	buf := make([]byte, len(data))
	copy(buf, data)
	newBlob.Data = buf
	newBlob.Refs.Store(1)

	actual, loaded := shard.LoadOrStore(hash, newBlob)
	if loaded {
		newBlob.Data = nil
		blobPool.Put(newBlob)
		return actual.(*HeaderBlob)
	}
	return newBlob
}
