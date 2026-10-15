package align

// CacheLinePad prevents false sharing by padding structures to 64 bytes.
type CacheLinePad [8]uint64
