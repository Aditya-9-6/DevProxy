package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
)

var (
	ErrInvalidChunkSize = errors.New("invalid chunk size hex representation")
	ErrChunkLimitExceeded = errors.New("parsed chunk size exceeds bound")
)

// ChunkValidator parses and validates HTTP/1.1 chunked streams with strict size bounds.
type ChunkValidator struct {
	maxChunkSize  int64
	maxTotalSize  int64
	totalParsed   atomic.Int64
}

// NewChunkValidator instantiates a ChunkValidator with upper bounds.
func NewChunkValidator(maxChunkSize, maxTotalSize int64) *ChunkValidator {
	return &ChunkValidator{
		maxChunkSize: maxChunkSize,
		maxTotalSize: maxTotalSize,
	}
}

// ParseStream parses a raw chunked stream reader and verifies bounds securely.
func (v *ChunkValidator) ParseStream(r io.Reader, w io.Writer) (int64, error) {
	br := bufio.NewReader(r)
	var total int64

	for {
		line, err := br.ReadBytes('\n')
		data := bytes.TrimSpace(line)
		if len(data) == 0 {
			if err != nil {
				break
			}
			continue
		}

		// Parse hex chunk size (ignore chunk extensions if any)
		if idx := bytes.IndexByte(data, ';'); idx >= 0 {
			data = data[:idx]
		}

		sizeVal, parseErr := strconv.ParseInt(string(bytes.TrimSpace(data)), 16, 64)
		if parseErr != nil {
			return total, fmt.Errorf("%w: %v", ErrInvalidChunkSize, parseErr)
		}

		if v.maxChunkSize > 0 && sizeVal > v.maxChunkSize {
			return total, ErrChunkLimitExceeded
		}

		if sizeVal == 0 {
			// Terminal chunk
			// Consume trailing CRLF if present
			_, _ = br.ReadBytes('\n')
			break
		}

		newTotal := total + sizeVal
		if v.maxTotalSize > 0 && newTotal > v.maxTotalSize {
			return total, ErrChunkLimitExceeded
		}

		chunkBuf := make([]byte, sizeVal)
		if _, rErr := io.ReadFull(br, chunkBuf); rErr != nil {
			return total, rErr
		}

		if _, wErr := w.Write(chunkBuf); wErr != nil {
			return total, wErr
		}

		// Consume trailing CRLF after chunk data
		_, _ = br.ReadBytes('\n')

		total = newTotal
		v.totalParsed.Store(total)

		if err != nil {
			break
		}
	}

	return total, nil
}

// TotalParsed returns atomically the number of parsed bytes.
func (v *ChunkValidator) TotalParsed() int64 {
	return v.totalParsed.Load()
}
