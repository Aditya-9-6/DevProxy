package proxy

import (
	"compress/flate"
	"io"
	"sync/atomic"
)

// CacheLinePadDeflate prevents false sharing on hot atomic performance metrics.
type CacheLinePadDeflate [8]uint64

// DeflatePipeStats records throughput and chunk count atomically without locks.
type DeflatePipeStats struct {
	BytesProcessed  CacheLinePadDeflate
	ChunksProcessed CacheLinePadDeflate
	ErrorsCount     CacheLinePadDeflate
}

// DeflatePipe implements an atomic, cache-aligned, streaming HTTP chunked transfer-encoding
// compression pipe utilizing flate and zero-allocation buffer recycling.
type DeflatePipe struct {
	stats  DeflatePipeStats
	level  int
	active atomic.Bool
	closed atomic.Bool
}

// NewDeflatePipe creates a new DeflatePipe with the specified compression level (e.g. flate.BestSpeed).
func NewDeflatePipe(level int) *DeflatePipe {
	if level < flate.DefaultCompression || level > flate.BestCompression {
		level = flate.DefaultCompression
	}
	p := &DeflatePipe{
		level: level,
	}
	p.active.Store(true)
	return p
}

// Stats returns current atomic performance counters.
func (p *DeflatePipe) Stats() (bytesProc, chunksProc, errorsCount uint64) {
	return atomic.LoadUint64(&p.stats.BytesProcessed[0]),
		atomic.LoadUint64(&p.stats.ChunksProcessed[0]),
		atomic.LoadUint64(&p.stats.ErrorsCount[0])
}

// CompressStream streams data from src through a flate writer into dst, updating atomic stats.
func (p *DeflatePipe) CompressStream(dst io.Writer, src io.Reader) error {
	if !p.active.Load() || p.closed.Load() {
		return io.ErrClosedPipe
	}

	bufPtr := DefaultDeflatePool.Get()
	defer DefaultDeflatePool.Put(bufPtr)

	w, err := flate.NewWriter(dst, p.level)
	if err != nil {
		atomic.AddUint64(&p.stats.ErrorsCount[0], 1)
		return err
	}
	defer w.Close()

	// Ensure buffer has adequate capacity and slice length for reading
	if cap(*bufPtr) < 16384 {
		*bufPtr = make([]byte, 16384)
	} else {
		*bufPtr = (*bufPtr)[:16384]
	}

	for {
		n, rErr := src.Read(*bufPtr)
		if n > 0 {
			_, wErr := w.Write((*bufPtr)[:n])
			if wErr != nil {
				atomic.AddUint64(&p.stats.ErrorsCount[0], 1)
				return wErr
			}
			atomic.AddUint64(&p.stats.BytesProcessed[0], uint64(n))
			atomic.AddUint64(&p.stats.ChunksProcessed[0], 1)
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			atomic.AddUint64(&p.stats.ErrorsCount[0], 1)
			return rErr
		}
	}

	return w.Flush()
}

// DecompressStream inflates a compressed stream from src into dst.
func (p *DeflatePipe) DecompressStream(dst io.Writer, src io.Reader) error {
	if !p.active.Load() || p.closed.Load() {
		return io.ErrClosedPipe
	}

	r := flate.NewReader(src)
	defer r.Close()

	var buf [16384]byte
	for {
		n, rErr := r.Read(buf[:])
		if n > 0 {
			_, wErr := dst.Write(buf[:n])
			if wErr != nil {
				atomic.AddUint64(&p.stats.ErrorsCount[0], 1)
				return wErr
			}
			atomic.AddUint64(&p.stats.BytesProcessed[0], uint64(n))
			atomic.AddUint64(&p.stats.ChunksProcessed[0], 1)
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			atomic.AddUint64(&p.stats.ErrorsCount[0], 1)
			return rErr
		}
	}

	return nil
}

// Close shuts down the deflate pipe.
func (p *DeflatePipe) Close() error {
	p.closed.Store(true)
	p.active.Store(false)
	return nil
}
