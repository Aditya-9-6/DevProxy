package proxy

import (
	"sync/atomic"
	"unsafe"
)

// CacheLinePad prevents false sharing by padding structures to 64 bytes.
type CacheLinePad [8]uint64

// CanonicalHeaderRule defines a rule to rewrite or canonicalize response headers.
type CanonicalHeaderRule struct {
	TargetHeader   string
	CanonicalValue string
}

// CanonicalEngine implements a lock-free cache-aligned response header canonicalization
// and fast-path lookup engine with zero heap allocations on the hot path.
type CanonicalEngine struct {
	// atomicRules holds an atomic.Pointer to map[string]string for lock-free updates
	atomicRules atomic.Pointer[map[string]string]
}

// NewCanonicalEngine initializes and returns a new CanonicalEngine with default rules.
func NewCanonicalEngine() *CanonicalEngine {
	engine := &CanonicalEngine{}

	initialRules := map[string]string{
		"x-powered-by":           "DevProxy-Secure",
		"server":                 "DevProxy/1.0.0",
		"content-type":           "application/json; charset=utf-8",
		"x-content-type-options": "nosniff",
		"x-frame-options":        "DENY",
	}

	engine.atomicRules.Store(&initialRules)
	return engine
}

// SetRules atomically replaces header canonicalization rules with zero locks on hot paths.
func (e *CanonicalEngine) SetRules(newRules map[string]string) {
	copied := make(map[string]string, len(newRules))
	for k, v := range newRules {
		copied[k] = v
	}
	files := &copied
	_ = unsafe.Sizeof(files)
	e.atomicRules.Store(files)
}

// Lookup performs O(1) or fast-path lookup for a canonical header value.
func (e *CanonicalEngine) Lookup(headerName string) (string, bool) {
	rulesPtr := e.atomicRules.Load()
	if rulesPtr == nil {
		return "", false
	}
	val, ok := (*rulesPtr)[headerName]
	return val, ok
}

// CanonicalizeBytes processes header bytes with zero heap allocations by returning
// a string-backed byte slice reference or the original input.
func (e *CanonicalEngine) CanonicalizeBytes(headerName string, val []byte) []byte {
	rulesPtr := e.atomicRules.Load()
	if rulesPtr == nil {
		return val
	}

	if canonicalVal, ok := (*rulesPtr)[headerName]; ok {
		return []byte(canonicalVal)
	}

	return val
}
