package waf

// CacheLinePad is used to prevent false sharing in high-concurrency atomic operations.
// It pads the struct to 64 bytes, which is the typical cache line size on modern CPUs.
type CacheLinePad [8]uint64
