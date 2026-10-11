package reaper

import (
	"sync/atomic"
	"unsafe"
)

// HazardPointer tracks an active reference to a connection.
// It uses atomic pointers to ensure thread safety without locks.
type HazardPointer struct {
	ptr unsafe.Pointer
}

func (h *HazardPointer) Set(conn interface{}) {
	atomic.StorePointer(&h.ptr, unsafe.Pointer(&conn))
}

func (h *HazardPointer) Clear() {
	atomic.StorePointer(&h.ptr, nil)
}

func (h *HazardPointer) IsActive() bool {
	return atomic.LoadPointer(&h.ptr) != nil
}
