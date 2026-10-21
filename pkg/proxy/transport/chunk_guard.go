package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/proxy/parser"
)

var (
	ErrChunkGuardLimit           = errors.New("transport: chunk guard limit exceeded")
	ErrChunkGuardClosed          = errors.New("transport: chunk guard reader is closed")
	ErrReadDeadlineUnsupported   = errors.New("transport: source does not support read deadlines")
	ErrNilChunkGuardReader       = errors.New("transport: nil chunk guard reader")
	ErrInvalidChunkGuardReadSize = errors.New("transport: source returned an invalid read count")
)

const (
	cacheLineSize      = 64
	chunkGuardReadSize = 32 * 1024
)

// ChunkGuardLimits bounds decoded data, chunk count, framing overhead, and
// read latency. A zero byte or duration limit disables that specific limit.
type ChunkGuardLimits struct {
	MaxChunkBytes   uint64
	MaxTotalBytes   uint64
	MaxChunks       uint64
	MaxHeaderBytes  uint64
	MaxTrailerBytes uint64
	MaxEncodedBytes uint64
	IdleTimeout     time.Duration
	TotalTimeout    time.Duration
}

// DefaultChunkGuardLimits returns limits suitable for an untrusted HTTP/1.1
// peer, including idle and whole-transfer deadlines.
func DefaultChunkGuardLimits() ChunkGuardLimits {
	return ChunkGuardLimits{
		MaxChunkBytes:   parser.DefaultMaxChunkBytes,
		MaxTotalBytes:   parser.DefaultMaxTotalBytes,
		MaxChunks:       parser.DefaultMaxChunks,
		MaxHeaderBytes:  parser.DefaultMaxHeaderBytes,
		MaxTrailerBytes: parser.DefaultMaxTrailerBytes,
		MaxEncodedBytes: parser.DefaultMaxEncodedBytes,
		IdleTimeout:     30 * time.Second,
		TotalTimeout:    5 * time.Minute,
	}
}

// UnlimitedChunkGuardLimits disables all size, count, and duration limits.
func UnlimitedChunkGuardLimits() ChunkGuardLimits {
	return ChunkGuardLimits{}
}

// cacheAlignedCounter isolates independently updated counters into separate
// 64-byte cache lines. The atomic value must not be copied after first use.
type cacheAlignedCounter struct {
	value atomic.Uint64
	_     [cacheLineSize - 8]byte
}

func (c *cacheAlignedCounter) load() uint64 {
	return c.value.Load()
}

func (c *cacheAlignedCounter) add(value uint64) {
	c.value.Add(value)
}

// ChunkGuardStats is a race-safe snapshot of guard accounting counters.
type ChunkGuardStats struct {
	ChunkCount        uint64
	DecodedBytes      uint64
	EncodedBytes      uint64
	LargestChunkBytes uint64
	RemainingBytes    uint64
	RejectedChunks    uint64
	Complete          bool
}

// ChunkGuard validates one chunked HTTP/1.1 body and atomically accounts for
// its decoded size, encoded size, and chunk count.
type ChunkGuard struct {
	decodedBytes      cacheAlignedCounter
	encodedBytes      cacheAlignedCounter
	chunkCount        cacheAlignedCounter
	largestChunkBytes cacheAlignedCounter
	rejectedChunks    cacheAlignedCounter

	limits    ChunkGuardLimits
	observer  *chunkGuardObserver
	validator *parser.ChunkValidator
}

type chunkGuardObserver struct {
	guard *ChunkGuard
}

// NewChunkGuard constructs a guard using exactly the supplied limits.
func NewChunkGuard(limits ChunkGuardLimits) *ChunkGuard {
	guard := &ChunkGuard{limits: limits}
	guard.observer = &chunkGuardObserver{guard: guard}
	guard.validator = parser.NewChunkValidatorWithObserver(parser.ChunkLimits{
		MaxHeaderBytes:  limits.MaxHeaderBytes,
		MaxTrailerBytes: limits.MaxTrailerBytes,
		MaxEncodedBytes: limits.MaxEncodedBytes,
	}, guard.observer)
	return guard
}

// NewDefaultChunkGuard constructs a guard with conservative default limits.
func NewDefaultChunkGuard() *ChunkGuard {
	return NewChunkGuard(DefaultChunkGuardLimits())
}

// Limits returns the immutable guard limits.
func (g *ChunkGuard) Limits() ChunkGuardLimits {
	return g.limits
}

// Complete reports whether the guarded chunked body has ended.
func (g *ChunkGuard) Complete() bool {
	return g.validator.Complete()
}

// Reset clears all guard and validator accounting for reuse on a new stream.
func (g *ChunkGuard) Reset() error {
	if err := g.validator.Reset(); err != nil {
		return err
	}
	g.decodedBytes.value.Store(0)
	g.encodedBytes.value.Store(0)
	g.chunkCount.value.Store(0)
	g.largestChunkBytes.value.Store(0)
	g.rejectedChunks.value.Store(0)
	return nil
}

// Stats returns a race-safe accounting snapshot. DecodedBytes is reserved
// when a chunk size line is accepted, preventing a declared chunk from
// crossing the total-size budget before its payload is read.
func (g *ChunkGuard) Stats() ChunkGuardStats {
	decoded := g.decodedBytes.load()
	remaining := uint64(0)
	if limit := g.limits.MaxTotalBytes; limit != 0 && decoded < limit {
		remaining = limit - decoded
	}
	return ChunkGuardStats{
		ChunkCount:        g.chunkCount.load(),
		DecodedBytes:      decoded,
		EncodedBytes:      g.encodedBytes.load(),
		LargestChunkBytes: g.largestChunkBytes.load(),
		RemainingBytes:    remaining,
		RejectedChunks:    g.rejectedChunks.load(),
		Complete:          g.validator.Complete(),
	}
}

// Consume validates and accounts for a chunked-body fragment. It returns the
// exact number of bytes belonging to the guarded body, allowing callers to
// retain trailing bytes for the next pipelined HTTP message.
func (g *ChunkGuard) Consume(ctx context.Context, p []byte) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if g.validator.Complete() {
		return 0, nil
	}
	return g.consume(p)
}

func (g *ChunkGuard) consume(p []byte) (int, error) {
	consumed, err := g.validator.Consume(p)
	if consumed > 0 {
		g.encodedBytes.add(uint64(consumed))
	}
	if err == nil {
		return consumed, nil
	}
	if isGuardSizeError(err) {
		g.rejectedChunks.add(1)
		return consumed, fmt.Errorf("%w: %w", ErrChunkGuardLimit, err)
	}
	return consumed, err
}

func isGuardSizeError(err error) bool {
	return errors.Is(err, parser.ErrEncodedBodyTooLarge) ||
		errors.Is(err, parser.ErrChunkHeaderTooLarge) ||
		errors.Is(err, parser.ErrTrailerTooLarge)
}

func (g *ChunkGuard) limitError(format string, args ...any) error {
	g.rejectedChunks.add(1)
	return fmt.Errorf("%w: %s", ErrChunkGuardLimit, fmt.Sprintf(format, args...))
}

func (o *chunkGuardObserver) ObserveChunk(event parser.ChunkEvent) error {
	guard := o.guard
	if limit := guard.limits.MaxChunkBytes; limit != 0 && event.Size > limit {
		return guard.limitError("chunk size %d exceeds limit %d", event.Size, limit)
	}

	currentTotal := guard.decodedBytes.load()
	currentCount := guard.chunkCount.load()
	if limit := guard.limits.MaxTotalBytes; limit != 0 {
		if currentTotal > limit || event.Size > limit-currentTotal {
			return guard.limitError("decoded body exceeds limit %d", limit)
		}
	}
	if limit := guard.limits.MaxChunks; limit != 0 && currentCount >= limit {
		return guard.limitError("chunk count exceeds limit %d", limit)
	}

	for {
		if limit := guard.limits.MaxTotalBytes; limit != 0 {
			if currentTotal > limit || event.Size > limit-currentTotal {
				return guard.limitError("decoded body exceeds limit %d", limit)
			}
		}
		if guard.decodedBytes.value.CompareAndSwap(currentTotal, currentTotal+event.Size) {
			break
		}
		currentTotal = guard.decodedBytes.load()
	}

	for {
		if limit := guard.limits.MaxChunks; limit != 0 && currentCount >= limit {
			return guard.limitError("chunk count exceeds limit %d", limit)
		}
		if guard.chunkCount.value.CompareAndSwap(currentCount, currentCount+1) {
			break
		}
		currentCount = guard.chunkCount.load()
	}

	for {
		largest := guard.largestChunkBytes.load()
		if event.Size <= largest || guard.largestChunkBytes.value.CompareAndSwap(largest, event.Size) {
			break
		}
	}
	return nil
}

// ChunkGuardReader validates the first chunked body in a raw HTTP/1.1 stream.
// Bytes after that body are forwarded without validation so a pipelined next
// message remains available to the caller.
type ChunkGuardReader struct {
	source io.Reader
	guard  *ChunkGuard
	ctx    context.Context

	bodyComplete atomic.Bool
	closed       atomic.Bool
	startOnce    sync.Once
	startedAt    time.Time
}

// WrapReader wraps r with a context-free chunk guard reader.
func (g *ChunkGuard) WrapReader(r io.Reader) *ChunkGuardReader {
	return g.WrapReaderContext(context.Background(), r)
}

// WrapReaderContext wraps r and propagates cancellation to the guarded read.
func (g *ChunkGuard) WrapReaderContext(ctx context.Context, r io.Reader) *ChunkGuardReader {
	if ctx == nil {
		ctx = context.Background()
	}
	reader := &ChunkGuardReader{source: r, guard: g, ctx: ctx}
	reader.bodyComplete.Store(g.validator.Complete())
	return reader
}

// BodyComplete reports whether the guarded body has ended.
func (r *ChunkGuardReader) BodyComplete() bool {
	return r.bodyComplete.Load()
}

// Read implements io.Reader. Read deadlines are applied when configured; the
// source must implement SetReadDeadline for hard idle-timeout enforcement.
func (r *ChunkGuardReader) Read(p []byte) (int, error) {
	if r.closed.Load() {
		return 0, ErrChunkGuardClosed
	}
	if r.source == nil {
		return 0, ErrNilChunkGuardReader
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.bodyComplete.Load() {
		return r.readThrough(p)
	}

	r.startOnce.Do(func() {
		r.startedAt = time.Now()
	})
	if err := r.guard.applyReadDeadline(r.source, r.startedAt, r.ctx); err != nil {
		return 0, err
	}

	readBuffer := p
	if len(readBuffer) > chunkGuardReadSize {
		readBuffer = readBuffer[:chunkGuardReadSize]
	}
	n, readErr := r.source.Read(readBuffer)
	if n < 0 || n > len(readBuffer) {
		return 0, ErrInvalidChunkGuardReadSize
	}
	if n > 0 {
		consumed, validationErr := r.guard.consume(readBuffer[:n])
		if validationErr != nil {
			return consumed, validationErr
		}
		if r.guard.validator.Complete() {
			r.bodyComplete.Store(true)
		}
		if consumed != n && !r.bodyComplete.Load() {
			return consumed, io.ErrShortWrite
		}
	}
	return r.finishRead(n, readErr)
}

func (r *ChunkGuardReader) readThrough(p []byte) (int, error) {
	n, err := r.source.Read(p)
	if n < 0 || n > len(p) {
		return 0, ErrInvalidChunkGuardReadSize
	}
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		if n > 0 {
			return n, ctxErr
		}
		return 0, ctxErr
	}
	return n, err
}

func (r *ChunkGuardReader) finishRead(n int, readErr error) (int, error) {
	if ctxErr := r.ctx.Err(); ctxErr != nil {
		if n > 0 {
			return n, ctxErr
		}
		return 0, ctxErr
	}
	if readErr == nil {
		return n, nil
	}
	if errors.Is(readErr, io.EOF) {
		if r.guard.validator.Complete() {
			return n, nil
		}
		return n, fmt.Errorf("%w: transfer-encoding body ended before the final chunk", parser.ErrIncompleteChunk)
	}
	return n, readErr
}

// Close closes the underlying source when it implements io.Closer.
func (r *ChunkGuardReader) Close() error {
	if r.closed.Swap(true) {
		return nil
	}
	if closer, ok := r.source.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type readDeadlineSetter interface {
	SetReadDeadline(time.Time) error
}

func (g *ChunkGuard) applyReadDeadline(source io.Reader, startedAt time.Time, ctx context.Context) error {
	deadline, required := g.readDeadline(startedAt, ctx)
	if !required {
		return nil
	}
	setter, ok := source.(readDeadlineSetter)
	if !ok {
		return fmt.Errorf("%w: cannot enforce configured transfer deadlines", ErrReadDeadlineUnsupported)
	}
	if err := setter.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("transport: set chunk read deadline: %w", err)
	}
	return nil
}

func (g *ChunkGuard) readDeadline(startedAt time.Time, ctx context.Context) (time.Time, bool) {
	var deadline time.Time
	setDeadline := func(candidate time.Time) {
		if deadline.IsZero() || candidate.Before(deadline) {
			deadline = candidate
		}
	}

	now := time.Now()
	if g.limits.IdleTimeout > 0 {
		setDeadline(now.Add(g.limits.IdleTimeout))
	}
	if g.limits.TotalTimeout > 0 {
		setDeadline(startedAt.Add(g.limits.TotalTimeout))
	}
	if contextDeadline, ok := ctx.Deadline(); ok {
		setDeadline(contextDeadline)
	}
	return deadline, !deadline.IsZero()
}

var guardBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, chunkGuardReadSize)
		return &buffer
	},
}

// Verify consumes and validates an isolated chunked body.
func (g *ChunkGuard) Verify(r io.Reader) error {
	return g.VerifyContext(context.Background(), r)
}

// VerifyContext consumes and validates an isolated chunked body while
// enforcing configured read deadlines and context cancellation.
func (g *ChunkGuard) VerifyContext(ctx context.Context, r io.Reader) error {
	if ctx == nil {
		ctx = context.Background()
	}
	reader := g.WrapReaderContext(ctx, r)
	bufferPointer := guardBufferPool.Get().(*[]byte)
	defer guardBufferPool.Put(bufferPointer)

	emptyReads := 0
	for {
		n, err := reader.Read(*bufferPointer)
		if n > 0 {
			emptyReads = 0
		} else if err == nil {
			emptyReads++
			if emptyReads >= 100 {
				return io.ErrNoProgress
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) && reader.BodyComplete() {
				return nil
			}
			return err
		}
		if reader.BodyComplete() {
			return nil
		}
	}
}
