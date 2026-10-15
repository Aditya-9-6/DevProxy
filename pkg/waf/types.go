package waf

// CacheLinePad provides padding to prevent false sharing in high-contention atomic counters.
type CacheLinePad [8]uint64
