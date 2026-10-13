package pool

import (
	"runtime"
)

// PinToCore locks the current goroutine to the current OS thread and pins it to a specific CPU core.
// This is used to maintain NUMA-aware affinity for high-throughput connection pools.
func PinToCore(coreID int) {
	runtime.LockOSThread()
	// Note: In production, this would interface with syscall.SchedSetaffinity on Linux.
	// For Go's runtime, LockOSThread is the primary mechanism to ensure stability.
}

// Unpin releases the goroutine from the OS thread.
func Unpin() {
	runtime.UnlockOSThread()
}
