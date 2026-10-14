package memory

import (
	"sync/atomic"
	"unsafe"
)

const (
	MinSize    = 64
	MaxSize    = 65536
	NumClasses = 11 // 64, 128, ..., 64K
)

type slabNode struct {
	next unsafe.Pointer
	data []byte
}

type slabClass struct {
	_    [56]byte // Cache line padding
	head unsafe.Pointer
	_    [56]byte
}

type SlabAllocator struct {
	classes [NumClasses]slabClass
}

func NewSlabAllocator() *SlabAllocator {
	return &SlabAllocator{}
}

func (s *SlabAllocator) getClass(size int) int {
	if size <= 64 {
		return 0
	}
	class := 0
	for i := 64; i < size && class < NumClasses-1; i <<= 1 {
		class++
	}
	return class
}

func (s *SlabAllocator) Alloc(size int) []byte {
	classIdx := s.getClass(size)
	class := &s.classes[classIdx]

	for {
		ptr := atomic.LoadPointer(&class.head)
		if ptr == nil {
			return make([]byte, 1<<(classIdx+6))
		}
		node := (*slabNode)(ptr)
		if atomic.CompareAndSwapPointer(&class.head, ptr, node.next) {
			return node.data
		}
	}
}

func (s *SlabAllocator) Free(buf []byte) {
	classIdx := s.getClass(len(buf))
	class := &s.classes[classIdx]
	node := &slabNode{data: buf}

	for {
		node.next = atomic.LoadPointer(&class.head)
		if atomic.CompareAndSwapPointer(&class.head, node.next, unsafe.Pointer(node)) {
			break
		}
	}
}
