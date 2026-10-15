package proxy

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var simpleChunkBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

type ChunkState uint32

const (
	StateReadingSize ChunkState = iota
	StateReadingData
	StateReadingTrailer
	StateFinished
)

type SimpleChunkDecoder struct {
	state     atomic.Uint32
	remaining int64
	reader    *bufio.Reader
	mu        sync.Mutex
}

func NewSimpleChunkDecoder(r io.Reader) *SimpleChunkDecoder {
	return &SimpleChunkDecoder{
		reader: bufio.NewReader(r),
	}
}

func (d *SimpleChunkDecoder) Read(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for n < len(p) && d.state.Load() != uint32(StateFinished) {
		if d.remaining == 0 {
			line, err := d.reader.ReadString('\n')
			if err != nil {
				return n, err
			}
			line = strings.TrimRight(line, "\r\n")
			if idx := strings.IndexByte(line, ';'); idx != -1 {
				line = line[:idx]
			}
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			size, err := strconv.ParseInt(line, 16, 64)
			if err != nil {
				return n, err
			}
			if size == 0 {
				d.state.Store(uint32(StateFinished))
				return n, io.EOF
			}
			d.remaining = size
			d.state.Store(uint32(StateReadingData))
		}

		toRead := int64(len(p) - n)
		if toRead > d.remaining {
			toRead = d.remaining
		}

		read, err := d.reader.Read(p[n : int64(n)+toRead])
		n += read
		d.remaining -= int64(read)

		if d.remaining == 0 {
			_, _ = d.reader.Discard(2)
			d.state.Store(uint32(StateReadingSize))
		}

		if err != nil {
			return n, err
		}
	}
	return n, nil
}
