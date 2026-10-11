package reaper

import (
	"sync/atomic"
)

// HazardPointer tracks an active reference to a connection.
// It uses atomic integers to store pointer values safely.
type HazardPointer struct {
	ptr atomic.Uintptr
}

func (h *HazardPointer) Set(addr uintptr) {
	h.ptr.Store(addr)
}

func (h *HazardPointer) Clear() {
	h.ptr.Store(0)
}

func (h *HazardPointer) IsActive() bool {
	return h.ptr.Load() != 0
}
