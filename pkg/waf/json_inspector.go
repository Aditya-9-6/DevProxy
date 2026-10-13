package waf

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing on hot atomic counters.
type CacheLinePad [8]uint64

// JSONInspector provides zero-allocation structural analysis.
type JSONInspector struct {
	MaxDepth      uint64
	DepthCounter  CacheLinePad
	CollisionFlag CacheLinePad
}

func NewJSONInspector(maxDepth uint64) *JSONInspector {
	return &JSONInspector{MaxDepth: maxDepth}
}

// InspectStream processes a byte slice without allocations.
func (j *JSONInspector) InspectStream(data []byte) bool {
	var depth uint64
	var inString bool
	var escaped bool

	for i := 0; i < len(data); i++ {
		b := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if b == '\\' {
				escaped = true
				continue
			}
			if b == '"' {
				inString = false
			}
			continue
		}

		switch b {
		case '"':
			inString = true
		case '{', '[':
			depth++
			if depth > j.MaxDepth {
				atomic.StoreUint64(&j.DepthCounter[0], depth)
				return false
			}
		case '}', ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return true
}
