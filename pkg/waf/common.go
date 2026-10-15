package waf

// CacheLinePad prevents false sharing.
type CacheLinePad [8]uint64
