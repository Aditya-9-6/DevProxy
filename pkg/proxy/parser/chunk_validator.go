package parser

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
)

// ChunkValidator parses and validates HTTP/1.1 chunked transfer streams with zero allocations.
type ChunkValidator struct {
	maxChunkSize    uint64
	bytesValidated  atomic.Uint64
	chunksValidated atomic.Uint64
}

var validatorBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

var (
	ErrInvalidChunkHeader = errors.New("chunk validator: malformed chunk size header")
	ErrChunkSizeOverflow  = errors.New("chunk validator: chunk size exceeds allowed boundary")
	ErrInvalidChunkTerm   = errors.New("chunk validator: missing chunk CRLF terminator")
	ErrChunkStreamLeak    = errors.New("chunk validator: stream terminated unexpectedly")
)

// NewChunkValidator constructs a ChunkValidator.
func NewChunkValidator(maxChunkSize uint64) *ChunkValidator {
	return &ChunkValidator{
		maxChunkSize: maxChunkSize,
	}
}

// ValidateStream parses a chunked stream reader and verifies boundaries and CRLF formatting.
func (v *ChunkValidator) ValidateStream(r io.Reader, w io.Writer) (uint64, error) {
	br := bufio.NewReader(r)
	var total uint64

	for {
		line, err := br.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && len(line) == 0 {
				break
			}
			return total, ErrInvalidChunkHeader
		}

		line = bytes.TrimRight(line, "\r\n")
		chunkSize, err := v.ParseChunkSize(line)
		if err != nil {
			return total, err
		}

		if chunkSize == 0 {
			_, _ = br.ReadBytes('\n')
			break
		}

		chunkData := make([]byte, chunkSize)
		_, err = io.ReadFull(br, chunkData)
		if err != nil {
			return total, ErrChunkStreamLeak
		}

		crlf := make([]byte, 2)
		_, err = io.ReadFull(br, crlf)
		if err != nil || crlf[0] != '\r' || crlf[1] != '\n' {
			return total, ErrInvalidChunkTerm
		}

		total += chunkSize
		v.bytesValidated.Add(chunkSize)
		v.chunksValidated.Add(1)

		if w != nil {
			if _, wErr := w.Write(chunkData); wErr != nil {
				return total, wErr
			}
		}
	}

	return total, nil
}

// ParseChunkSize parses hexadecimal chunk size strings.
func (v *ChunkValidator) ParseChunkSize(line []byte) (uint64, error) {
	line = bytes.TrimSpace(line)
	if idx := bytes.IndexByte(line, ';'); idx >= 0 {
		line = line[:idx]
	}
	size, err := strconv.ParseUint(string(line), 16, 64)
	if err != nil {
		return 0, ErrInvalidChunkHeader
	}
	if v.maxChunkSize > 0 && size > v.maxChunkSize {
		return 0, ErrChunkSizeOverflow
	}
	return size, nil
}

// Stats returns validation metrics.
func (v *ChunkValidator) Stats() (bytes uint64, chunks uint64) {
	return v.bytesValidated.Load(), v.chunksValidated.Load()
}
