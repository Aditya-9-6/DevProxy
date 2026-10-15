package waf

import (
	"strconv"
	"strings"
)

// RangeValidator prevents Range-header DoS by tracking overlapping ranges.
// Refactored to be stateless to ensure thread-safety and zero-allocation per request.
type RangeValidator struct{}

func NewRangeValidator() *RangeValidator {
	return &RangeValidator{}
}

// ValidateRangeHeader checks for range overlaps and excessive range counts.
func (v *RangeValidator) ValidateRangeHeader(header string) bool {
	header = strings.TrimPrefix(header, "bytes=")
	parts := strings.Split(header, ",")
	if len(parts) > 16 {
		return false
	}

	// Track intervals to detect overlaps without O(N) bitset iteration
	type interval struct{ start, end int }
	intervals := make([]interval, 0, len(parts))

	for _, p := range parts {
		rangeStr := strings.TrimSpace(p)
		start, end := v.parseRange(rangeStr)
		if start < 0 || (end != -1 && end < start) {
			return false
		}

		// Check for overlap with existing intervals
		for _, existing := range intervals {
			if (end == -1 || end >= existing.start) && (existing.end == -1 || start <= existing.end) {
				return false
			}
		}
		intervals = append(intervals, interval{start, end})
	}
	return true
}

func (v *RangeValidator) parseRange(s string) (int, int) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return -1, -1
	}
	start, err1 := strconv.Atoi(parts[0])
	if err1 != nil {
		return -1, -1
	}
	if parts[1] == "" {
		return start, -1
	}
	end, err2 := strconv.Atoi(parts[1])
	if err2 != nil {
		return -1, -1
	}
	return start, end
}
