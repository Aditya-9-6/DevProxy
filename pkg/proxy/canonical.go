package proxy

import (
	"bytes"
	"sync"
	"sync/atomic"
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
	// fastIndexTable is a fixed cache-line padded index table for O(1) fast-path checks
	fastIndexTable [256]CacheLinePad
	// bufferPool provides zero-allocation byte buffers for header formatting
	bufferPool sync.Pool
}

// NewCanonicalEngine initializes and returns a new CanonicalEngine with default rules.
func NewCanonicalEngine() *CanonicalEngine {
	engine := &CanonicalEngine{
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
	}

	initialRules := map[string]string{
		"x-powered-by":           "DevProxy-Secure",
		"server":                 "DevProxy/1.0.0",
		"content-type":           "application/json; charset=utf-8",
		"x-content-type-options": "nosniff",
		"x-frame-options":        "DENY",
	}

	engine.atomicRules.Store(&initialRules)
	engine.rebuildFastIndex(initialRules)
	return engine
}

// SetRules atomically replaces header canonicalization rules with zero locks on hot paths.
func (e *CanonicalEngine) SetRules(newRules map[string]string) {
	copied := make(map[string]string, len(newRules))
	for k, v := range newRules {
		copied[k] = v
	}
	e.atomicRules.Store(&copied)
	e.rebuildFastIndex(copied)
}

func (e *CanonicalEngine) rebuildFastIndex(rules map[string]string) {
	for k := range rules {
		if len(k) > 0 {
			// Compute simple hash index for fast lookup bucket
			hash := uint64(k[0]) % 256
			atomic.AddUint64(&e.fastIndexTable[hash][0], 1)
		}
	}
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

// CanonicalizeBytes processes header bytes using pre-allocated buffers with zero heap allocations.
func (e *CanonicalEngine) CanonicalizeBytes(headerName string, val []byte) []byte {
	rulesPtr := e.atomicRules.Load()
	if rulesPtr == nil {
		return val
	}

	if canonicalVal, ok := (*rulesPtr)[headerName]; ok {
		buf := e.bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.WriteString(canonicalVal)
		result := buf.Bytes()
		e.bufferPool.Put(buf)
		return result
	}

	return val
}
