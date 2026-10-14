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
	r         *bufio.Reader
	curr      []byte
	pos       int
	err       error
	activeBuf *bytes.Buffer
}

func NewChunkDecoder(r io.Reader) *ChunkDecoder {
	return &ChunkDecoder{r: bufio.NewReader(r)}
}

func (d *ChunkDecoder) Read(p []byte) (int, error) {
	if d.err != nil {
		return 0, d.err
	}

	for d.pos >= len(d.curr) {
		if d.activeBuf != nil {
			chunkBufferPool.Put(d.activeBuf)
			d.activeBuf = nil
		}

		line, err := d.r.ReadString('\n')
		if err != nil {
			d.err = err
			return 0, err
		}
		sizeStr := strings.TrimSpace(line)
		if idx := strings.IndexByte(sizeStr, ';'); idx != -1 {
			sizeStr = sizeStr[:idx]
		}
		size, err := strconv.ParseInt(sizeStr, 16, 64)
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
		if err != nil {
			chunkBufferPool.Put(buf)
			d.err = err
			return 0, err
		}

		// Verify and discard trailing \r\n
		cr, err := d.r.ReadByte()
		lf, err2 := d.r.ReadByte()
		if err != nil || err2 != nil || cr != '\r' || lf != '\n' {
			chunkBufferPool.Put(buf)
			d.err = io.ErrUnexpectedEOF
			return 0, d.err
		}

		d.activeBuf = buf
		d.curr = buf.Bytes()
		d.pos = 0
	}

	n := copy(p, d.curr[d.pos:])
	d.pos += n
	return n, nil
}

func (d *ChunkDecoder) Close() error {
	if d.activeBuf != nil {
		chunkBufferPool.Put(d.activeBuf)
		d.activeBuf = nil
	}
	return nil
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

func (e *ChunkEncoder) Close() error {
	_, err := e.w.Write([]byte("0\r\n\r\n"))
	return err
}
