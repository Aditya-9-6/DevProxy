package waf

// CacheLinePad provides padding to prevent false sharing in concurrent structures.
type CacheLinePad [8]uint64
