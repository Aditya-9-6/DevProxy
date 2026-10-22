package waf

import (
	"strconv"
	"strings"
)

// ValidateRangeHeader checks for range overlaps and excessive range counts.
func ValidateRangeHeader(header string) bool {
	header = strings.TrimPrefix(header, "bytes=")

	// Track intervals to detect overlaps without O(N) bitset iteration
	type interval struct{ start, end int }
	// Pre-allocate to 16, as that's the max allowed parts
	intervals := make([]interval, 0, 16)

	partCount := 0
	for {
		idx := strings.IndexByte(header, ',')
		var p string
		if idx == -1 {
			p = header
		} else {
			p = header[:idx]
		}

		partCount++
		if partCount > 16 {
			return false
		}

		rangeStr := strings.TrimSpace(p)
		start, end := parseRange(rangeStr)
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

		if idx == -1 {
			break
		}
		header = header[idx+1:]
	}
	return true
}

func parseRange(s string) (int, int) {
	idx := strings.IndexByte(s, '-')
	if idx == -1 {
		return -1, -1
	}
	startStr := s[:idx]
	endStr := s[idx+1:]

	if strings.IndexByte(endStr, '-') != -1 {
		return -1, -1
	}

	start, err1 := strconv.Atoi(startStr)
	if err1 != nil {
		return -1, -1
	}
	if endStr == "" {
		return start, -1
	}
	end, err2 := strconv.Atoi(endStr)
	if err2 != nil {
		return -1, -1
	}
	return start, end
}
