package proxy

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// RangeChunkStats tracks range-request chunk sizes using cache-line padded atomic counters.
type RangeChunkStats struct {
	// [64]atomic.Uint64 padded to prevent false sharing
	Counters [64]atomic.Uint64
}

// RecordChunkSize records a range chunk size in O(1) lock-free manner.
func (s *RangeChunkStats) RecordChunkSize(size uint64) {
	// Use bucket 0 for total bytes and bucket 1 for total chunks
	s.Counters[0].Add(size)
	s.Counters[1].Add(1)
}

// GetTotalBytes returns the total bytes processed across ranges.
func (s *RangeChunkStats) GetTotalBytes() uint64 {
	return s.Counters[0].Load()
}

// GetTotalChunks returns the total number of range chunks processed.
func (s *RangeChunkStats) GetTotalChunks() uint64 {
	return s.Counters[1].Load()
}

// GlobalRangeStats provides global metrics for range requests.
var GlobalRangeStats RangeChunkStats

var rangeBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32768)
	},
}

// RangeSpec represents a single parsed byte range [start, end].
type RangeSpec struct {
	Start int64
	End   int64
}

// ParseByteRanges parses the standard HTTP Range header (e.g., "bytes=0-499,500-").
func ParseByteRanges(headerVal string, totalLength int64) ([]RangeSpec, error) {
	if !strings.HasPrefix(headerVal, "bytes=") {
		return nil, fmt.Errorf("invalid range unit: %s", headerVal)
	}

	specsStr := strings.TrimPrefix(headerVal, "bytes=")
	parts := strings.Split(specsStr, ",")
	var ranges []RangeSpec

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		idx := strings.Index(p, "-")
		if idx == -1 {
			return nil, fmt.Errorf("malformed range spec: %s", p)
		}

		startStr := strings.TrimSpace(p[:idx])
		endStr := strings.TrimSpace(p[idx+1:])

		var start, end int64
		var err error

		if startStr == "" {
			// Suffix range: -500 (last 500 bytes)
			val, err := strconv.ParseInt(endStr, 10, 64)
			if err != nil {
				return nil, err
			}
			if val > totalLength {
				val = totalLength
			}
			start = totalLength - val
			end = totalLength - 1
		} else {
			start, err = strconv.ParseInt(startStr, 10, 64)
			if err != nil {
				return nil, err
			}

			if start >= totalLength {
				continue // Unsatisfiable range
			}

			if endStr == "" {
				end = totalLength - 1
			} else {
				end, err = strconv.ParseInt(endStr, 10, 64)
				if err != nil {
					return nil, err
				}
				if end >= totalLength {
					end = totalLength - 1
				}
			}
		}

		if start > end {
			continue
		}

		ranges = append(ranges, RangeSpec{Start: start, End: end})
	}

	if len(ranges) == 0 && specsStr != "" {
		return nil, fmt.Errorf("range not satisfiable")
	}

	return ranges, nil
}

// MultipartBoundaryConcatenator handles streaming construction of multipart/byteranges responses.
type MultipartBoundaryConcatenator struct {
	writer      io.Writer
	boundary    string
	contentType string
	totalBytes  uint64
}

// NewMultipartConcatenator initializes a new multipart range response builder.
func NewMultipartConcatenator(w io.Writer, contentType string) *MultipartBoundaryConcatenator {
	return &MultipartBoundaryConcatenator{
		writer:      w,
		boundary:    "3d6b6a416f9b5",
		contentType: contentType,
	}
}

// Boundary returns the generated multipart boundary string.
func (m *MultipartBoundaryConcatenator) Boundary() string {
	return m.boundary
}

// WritePart appends a single range part into the multipart stream using zero-allocation buffer pooling.
func (m *MultipartBoundaryConcatenator) WritePart(start, end, totalLength int64, reader io.Reader) error {
	bufPtr := rangeBufferPool.Get().([]byte)
	defer rangeBufferPool.Put(bufPtr)

	headerPrefix := fmt.Sprintf("\r\n--%s\r\nContent-Type: %s\r\nContent-Range: bytes %d-%d/%d\r\n\r\n",
		m.boundary, m.contentType, start, end, totalLength)

	if _, err := io.WriteString(m.writer, headerPrefix); err != nil {
		return err
	}

	written, err := io.CopyBuffer(m.writer, reader, bufPtr)
	if err != nil {
		return err
	}

	GlobalRangeStats.RecordChunkSize(uint64(written))
	m.totalBytes += uint64(written)
	return nil
}

// Close finalizes the multipart response by writing the closing boundary.
func (m *MultipartBoundaryConcatenator) Close() error {
	closing := fmt.Sprintf("\r\n--%s--\r\n", m.boundary)
	_, err := io.WriteString(m.writer, closing)
	return err
}
