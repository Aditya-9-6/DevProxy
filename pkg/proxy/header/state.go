package header

import (
	"sync/atomic"
)

const CacheLineSize = 64

type HeaderBlob struct {
	Data []byte
	Refs atomic.Int64
}

type StateMachine struct {
	_        [CacheLineSize]byte
	registry atomic.Pointer[map[uint64]*HeaderBlob]
	_        [CacheLineSize]byte
}

func NewStateMachine() *StateMachine {
	s := &StateMachine{}
	s.registry.Store(&map[uint64]*HeaderBlob{})
	return s
}

func (s *StateMachine) GetOrStore(hash uint64, data []byte) *HeaderBlob {
	for {
		reg := s.registry.Load()
		if blob, ok := (*reg)[hash]; ok {
			return blob
		}

		newBlob := &HeaderBlob{Data: data}
		newBlob.Refs.Store(1)

		newReg := make(map[uint64]*HeaderBlob, len(*reg)+1)
		for k, v := range *reg {
			newReg[k] = v
		}
		newReg[hash] = newBlob

		if s.registry.CompareAndSwap(reg, &newReg) {
			return newBlob
		}
	}
}
