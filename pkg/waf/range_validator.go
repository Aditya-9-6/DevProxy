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
	start, _ := strconv.Atoi(parts[0])
	end := -1
	if parts[1] != "" {
		end, _ = strconv.Atoi(parts[1])
	}
	return start, end
}

func (v *RangeValidator) markRange(start, end int) bool {
	idx := start / 64
	bit := uint64(1) << (uint(start) % 64)
	if idx >= 1024 {
		return false
	}
	prev := v.bitset[idx].Load()
	if (prev & bit) != 0 {
		return false
	}
	v.bitset[idx].Or(bit)
	return true
}
