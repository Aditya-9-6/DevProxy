package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const MaxChunkSize = 1024 * 1024 // 1MB limit to prevent OOM

var chunkBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// ChunkDecoder implements io.Reader to decode HTTP chunked transfer encoding.
type ChunkDecoder struct {
	r    *bufio.Reader
	curr []byte
	pos  int
	err  error
}

func NewChunkDecoder(r io.Reader) *ChunkDecoder {
	return &ChunkDecoder{r: bufio.NewReader(r)}
}

func (d *ChunkDecoder) Read(p []byte) (int, error) {
	if d.err != nil {
		return 0, d.err
	}
	if d.pos >= len(d.curr) {
		line, err := d.r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
		if err != nil || size < 0 || size > MaxChunkSize {
			d.err = io.ErrUnexpectedEOF
			return 0, d.err
		}
		if size == 0 {
			d.err = io.EOF
			return 0, io.EOF
		}
		buf := chunkBufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Grow(int(size))
		_, err = io.CopyN(buf, d.r, size)
		d.r.Discard(2) // Discard \r\n
		d.curr = buf.Bytes()
		d.pos = 0
		if err != nil {
			chunkBufferPool.Put(buf)
			return 0, err
		}
		chunkBufferPool.Put(buf)
	}
	n := copy(p, d.curr[d.pos:])
	d.pos += n
	return n, nil
}

// ChunkEncoder implements io.Writer to encode data into HTTP chunked format.
type ChunkEncoder struct {
	w io.Writer
}

func NewChunkEncoder(w io.Writer) *ChunkEncoder {
	return &ChunkEncoder{w: w}
}

func (e *ChunkEncoder) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	_, err := fmt.Fprintf(e.w, "%x\r\n", len(p))
	if err != nil {
		return 0, err
	}
	_, err = e.w.Write(p)
	if err != nil {
		return 0, err
	}
	_, err = e.w.Write([]byte("\r\n"))
	return len(p), err
}
