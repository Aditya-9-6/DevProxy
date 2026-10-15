package waf

import (
	"strconv"
	"strings"
	"sync/atomic"
)

// RangeValidator prevents Range-header DoS by tracking overlapping ranges.
// Uses a cache-line padded atomic bitset to track range indices without locks.
type RangeValidator struct {
	_      [56]byte
	bitset [1024]atomic.Uint64 // 64k bits for range tracking
	_      [56]byte
}

func NewRangeValidator() *RangeValidator {
	return &RangeValidator{}
}

// ValidateRangeHeader checks for range overlaps and excessive range counts.
func (v *RangeValidator) ValidateRangeHeader(header string) bool {
	// Strip prefix if present
	header = strings.TrimPrefix(header, "bytes=")
	parts := strings.Split(header, ",")
	if len(parts) > 16 {
		return false
	}
	for _, p := range parts {
		rangeStr := strings.TrimSpace(p)
		start, end := v.parseRange(rangeStr)
		if start < 0 || (end != -1 && end < start) {
			return false
		}
		if !v.markRange(start, end) {
			return false
		}
	}
	return true
}

func (v *RangeValidator) parseRange(s string) (int, int) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return -1, -1
	}
	start, err1 := strconv.Atoi(parts[0])
	end := -1
	var err2 error
	if parts[1] != "" {
		end, err2 = strconv.Atoi(parts[1])
	}
	if err1 != nil || (parts[1] != "" && err2 != nil) {
		return -1, -1
	}
	return start, end
}

func (v *RangeValidator) markRange(start, end int) bool {
	limit := end
	if limit == -1 || limit >= 65535 {
		limit = 65535
	}
	for i := start; i <= limit; i++ {
		idx := i / 64
		if idx >= 1024 {
			return false
		}
		bit := uint64(1) << (uint(i) % 64)
		prev := v.bitset[idx].Load()
		if (prev & bit) != 0 {
			return false
		}
		v.bitset[idx].Or(bit)
		if end == -1 {
			break
		}
	}
	return true
}
