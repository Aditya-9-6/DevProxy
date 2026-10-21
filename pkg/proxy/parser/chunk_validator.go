package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
)

var (
	ErrInvalidChunkSize       = errors.New("parser: invalid chunk size")
	ErrChunkSizeOverflow      = errors.New("parser: chunk size overflow")
	ErrChunkTooLarge          = errors.New("parser: chunk exceeds configured limit")
	ErrBodyTooLarge           = errors.New("parser: decoded body exceeds configured limit")
	ErrTooManyChunks          = errors.New("parser: chunk count exceeds configured limit")
	ErrChunkHeaderTooLarge    = errors.New("parser: chunk header exceeds configured limit")
	ErrTrailerTooLarge        = errors.New("parser: trailer section exceeds configured limit")
	ErrEncodedBodyTooLarge    = errors.New("parser: encoded body exceeds configured limit")
	ErrInvalidChunkExtension  = errors.New("parser: invalid chunk extension")
	ErrInvalidChunkTerminator = errors.New("parser: invalid chunk terminator")
	ErrInvalidTrailer         = errors.New("parser: invalid trailer")
	ErrIncompleteChunk        = errors.New("parser: incomplete chunked body")
	ErrConcurrentUse          = errors.New("parser: chunk validator is already in use")
)

const (
	DefaultMaxChunkBytes   uint64 = 16 << 20
	DefaultMaxTotalBytes   uint64 = 64 << 20
	DefaultMaxChunks       uint64 = 100_000
	DefaultMaxHeaderBytes  uint64 = 8 << 10
	DefaultMaxTrailerBytes uint64 = 16 << 10
	DefaultMaxEncodedBytes uint64 = 128 << 20
)

// ChunkLimits bounds the resources accepted while parsing a chunked body. A
// zero value disables the corresponding limit.
type ChunkLimits struct {
	MaxChunkBytes   uint64
	MaxTotalBytes   uint64
	MaxChunks       uint64
	MaxHeaderBytes  uint64
	MaxTrailerBytes uint64
	MaxEncodedBytes uint64
}

// DefaultChunkLimits returns conservative limits suitable for HTTP/1.1 bodies.
func DefaultChunkLimits() ChunkLimits {
	return ChunkLimits{
		MaxChunkBytes:   DefaultMaxChunkBytes,
		MaxTotalBytes:   DefaultMaxTotalBytes,
		MaxChunks:       DefaultMaxChunks,
		MaxHeaderBytes:  DefaultMaxHeaderBytes,
		MaxTrailerBytes: DefaultMaxTrailerBytes,
		MaxEncodedBytes: DefaultMaxEncodedBytes,
	}
}

// UnlimitedChunkLimits disables all size and count limits.
func UnlimitedChunkLimits() ChunkLimits {
	return ChunkLimits{}
}

// ChunkStats is a point-in-time snapshot of validator accounting counters.
type ChunkStats struct {
	ChunkCount     uint64
	TotalBytes     uint64
	EncodedBytes   uint64
	LastChunkBytes uint64
	Complete       bool
}

// ChunkEvent describes a nonzero chunk when its size line is accepted.
type ChunkEvent struct {
	Index      uint64
	Size       uint64
	TotalBytes uint64
}

// ChunkObserver receives chunk boundaries before their payloads are consumed.
type ChunkObserver interface {
	ObserveChunk(ChunkEvent) error
}

// ChunkObserverFunc adapts a function to ChunkObserver.
type ChunkObserverFunc func(ChunkEvent) error

// ObserveChunk implements ChunkObserver.
func (f ChunkObserverFunc) ObserveChunk(event ChunkEvent) error {
	return f(event)
}

type chunkState uint8

const (
	stateChunkSize chunkState = iota
	stateChunkExtension
	stateChunkSizeLF
	stateChunkData
	stateChunkDataCR
	stateChunkDataLF
	stateTrailerStart
	stateTrailerName
	stateTrailerValue
	stateTrailerCR
	stateComplete
)

type extensionState uint8

const (
	extNameStart extensionState = iota
	extName
	extBeforeEquals
	extValueStart
	extValueToken
	extValueEnd
	extValueQuoted
	extValueQuotedEscape
	extValueAfterQuote
)

// ChunkValidator incrementally validates an HTTP/1.1 chunked body. It never
// retains payload bytes and can consume a message boundary without consuming
// bytes belonging to the next pipelined message.
type ChunkValidator struct {
	limits   ChunkLimits
	observer ChunkObserver

	writer         atomic.Uint32
	state          chunkState
	extensionState extensionState

	lineBytes       uint64
	trailerBytes    uint64
	chunkSize       uint64
	chunkDigits     uint64
	chunkRemaining  uint64
	trailerEndsBody bool

	chunkCount     atomic.Uint64
	totalBytes     atomic.Uint64
	encodedBytes   atomic.Uint64
	lastChunkBytes atomic.Uint64
	complete       atomic.Bool
}

// NewChunkValidator constructs a validator with the supplied limits.
func NewChunkValidator(limits ChunkLimits) *ChunkValidator {
	return NewChunkValidatorWithObserver(limits, nil)
}

// NewChunkValidatorWithObserver constructs a validator that reports chunk
// boundaries through observer.
func NewChunkValidatorWithObserver(limits ChunkLimits, observer ChunkObserver) *ChunkValidator {
	return &ChunkValidator{
		limits:         limits,
		observer:       observer,
		state:          stateChunkSize,
		extensionState: extNameStart,
	}
}

// Limits returns the immutable validator limits.
func (v *ChunkValidator) Limits() ChunkLimits {
	return v.limits
}

// Complete reports whether the terminating zero chunk and trailers were read.
func (v *ChunkValidator) Complete() bool {
	return v.complete.Load()
}

// Stats returns a race-safe accounting snapshot.
func (v *ChunkValidator) Stats() ChunkStats {
	return ChunkStats{
		ChunkCount:     v.chunkCount.Load(),
		TotalBytes:     v.totalBytes.Load(),
		EncodedBytes:   v.encodedBytes.Load(),
		LastChunkBytes: v.lastChunkBytes.Load(),
		Complete:       v.complete.Load(),
	}
}

// Reset returns the validator to its initial state while preserving limits
// and observer configuration.
func (v *ChunkValidator) Reset() error {
	if !v.writer.CompareAndSwap(0, 1) {
		return ErrConcurrentUse
	}
	defer v.writer.Store(0)

	v.state = stateChunkSize
	v.extensionState = extNameStart
	v.lineBytes = 0
	v.trailerBytes = 0
	v.chunkSize = 0
	v.chunkDigits = 0
	v.chunkRemaining = 0
	v.trailerEndsBody = false
	v.chunkCount.Store(0)
	v.totalBytes.Store(0)
	v.encodedBytes.Store(0)
	v.lastChunkBytes.Store(0)
	v.complete.Store(false)
	return nil
}

// Consume validates the next portion of a chunked body. The returned count
// identifies the exact message boundary, so callers handling HTTP pipelines
// can retain any unconsumed bytes for the next message.
func (v *ChunkValidator) Consume(p []byte) (int, error) {
	if !v.writer.CompareAndSwap(0, 1) {
		return 0, ErrConcurrentUse
	}
	defer v.writer.Store(0)

	if v.complete.Load() {
		return 0, nil
	}
	return v.consume(p)
}

// Write implements io.Writer. A short write indicates that the chunked body
// ended before all bytes in p were consumed.
func (v *ChunkValidator) Write(p []byte) (int, error) {
	n, err := v.Consume(p)
	if err == nil && n != len(p) {
		return n, io.ErrShortWrite
	}
	return n, err
}

func (v *ChunkValidator) consume(p []byte) (int, error) {
	offset := 0
	for offset < len(p) {
		if v.complete.Load() {
			break
		}

		if v.state == stateChunkData {
			n, err := v.consumeChunkData(p[offset:])
			offset += n
			if err != nil {
				return offset, err
			}
			continue
		}

		if !v.canConsumeEncoded(1) {
			return offset, ErrEncodedBodyTooLarge
		}
		if err := v.consumeFramingByte(p[offset]); err != nil {
			return offset, err
		}
		v.encodedBytes.Add(1)
		offset++
	}
	return offset, nil
}

func (v *ChunkValidator) canConsumeEncoded(count uint64) bool {
	limit := v.limits.MaxEncodedBytes
	if limit == 0 {
		return true
	}
	current := v.encodedBytes.Load()
	if current >= limit {
		return false
	}
	return count <= limit-current
}

func (v *ChunkValidator) consumeChunkData(data []byte) (int, error) {
	allowed := uint64(len(data))
	if v.chunkRemaining < allowed {
		allowed = v.chunkRemaining
	}

	clipped := false
	if limit := v.limits.MaxEncodedBytes; limit != 0 {
		current := v.encodedBytes.Load()
		if current >= limit {
			return 0, ErrEncodedBodyTooLarge
		}
		available := limit - current
		if allowed > available {
			allowed = available
			clipped = true
		}
	}

	if allowed > 0 {
		v.totalBytes.Add(allowed)
		v.encodedBytes.Add(allowed)
		v.chunkRemaining -= allowed
	}
	if v.chunkRemaining > 0 {
		if clipped {
			return int(allowed), ErrEncodedBodyTooLarge
		}
		return int(allowed), nil
	}

	v.state = stateChunkDataCR
	return int(allowed), nil
}

func (v *ChunkValidator) consumeFramingByte(b byte) error {
	switch v.state {
	case stateChunkSize:
		if err := v.addHeaderByte(); err != nil {
			return err
		}
		return v.consumeChunkSizeByte(b)
	case stateChunkExtension:
		if err := v.addHeaderByte(); err != nil {
			return err
		}
		return v.consumeChunkExtensionByte(b)
	case stateChunkSizeLF:
		if err := v.addHeaderByte(); err != nil {
			return err
		}
		if b != '\n' {
			return ErrInvalidChunkTerminator
		}
		return v.finishChunkLine()
	case stateChunkDataCR:
		if b != '\r' {
			return ErrInvalidChunkTerminator
		}
		v.state = stateChunkDataLF
		return nil
	case stateChunkDataLF:
		if b != '\n' {
			return ErrInvalidChunkTerminator
		}
		v.state = stateChunkSize
		v.resetChunkLine()
		return nil
	case stateTrailerStart, stateTrailerName, stateTrailerValue, stateTrailerCR:
		if err := v.addTrailerByte(); err != nil {
			return err
		}
		return v.consumeTrailerByte(b)
	default:
		return ErrInvalidChunkTerminator
	}
}

func (v *ChunkValidator) addHeaderByte() error {
	v.lineBytes++
	if limit := v.limits.MaxHeaderBytes; limit != 0 && v.lineBytes > limit {
		return ErrChunkHeaderTooLarge
	}
	return nil
}

func (v *ChunkValidator) addTrailerByte() error {
	v.trailerBytes++
	if limit := v.limits.MaxTrailerBytes; limit != 0 && v.trailerBytes > limit {
		return ErrTrailerTooLarge
	}
	return nil
}

func (v *ChunkValidator) consumeChunkSizeByte(b byte) error {
	switch {
	case isHexDigit(b):
		digit := uint64(hexValue(b))
		if v.chunkSize > (^uint64(0)-digit)/16 {
			return ErrChunkSizeOverflow
		}
		v.chunkSize = v.chunkSize*16 + digit
		v.chunkDigits++
		if limit := v.limits.MaxChunkBytes; limit != 0 && v.chunkSize > limit {
			return ErrChunkTooLarge
		}
	case b == ';':
		if v.chunkDigits == 0 {
			return ErrInvalidChunkSize
		}
		v.extensionState = extNameStart
		v.state = stateChunkExtension
	case b == '\r':
		if v.chunkDigits == 0 {
			return ErrInvalidChunkSize
		}
		v.state = stateChunkSizeLF
	case b == '\n':
		return ErrInvalidChunkTerminator
	default:
		return ErrInvalidChunkSize
	}
	return nil
}

func (v *ChunkValidator) consumeChunkExtensionByte(b byte) error {
	switch v.extensionState {
	case extNameStart:
		if isBWS(b) {
			return nil
		}
		if !isTokenChar(b) {
			return ErrInvalidChunkExtension
		}
		v.extensionState = extName
	case extName:
		if isTokenChar(b) {
			return nil
		}
		switch b {
		case '=':
			v.extensionState = extValueStart
		case ';':
			v.extensionState = extNameStart
		case '\r':
			v.state = stateChunkSizeLF
		default:
			if !isBWS(b) {
				return ErrInvalidChunkExtension
			}
			v.extensionState = extBeforeEquals
		}
	case extBeforeEquals:
		if isBWS(b) {
			return nil
		}
		switch b {
		case '=':
			v.extensionState = extValueStart
		case ';':
			v.extensionState = extNameStart
		case '\r':
			v.state = stateChunkSizeLF
		default:
			return ErrInvalidChunkExtension
		}
	case extValueStart:
		if isBWS(b) {
			return nil
		}
		switch {
		case b == '"':
			v.extensionState = extValueQuoted
		case isTokenChar(b):
			v.extensionState = extValueToken
		default:
			return ErrInvalidChunkExtension
		}
	case extValueToken:
		if isTokenChar(b) {
			return nil
		}
		switch b {
		case ';':
			v.extensionState = extNameStart
		case '\r':
			v.state = stateChunkSizeLF
		default:
			if !isBWS(b) {
				return ErrInvalidChunkExtension
			}
			v.extensionState = extValueEnd
		}
	case extValueEnd:
		if isBWS(b) {
			return nil
		}
		switch b {
		case ';':
			v.extensionState = extNameStart
		case '\r':
			v.state = stateChunkSizeLF
		default:
			return ErrInvalidChunkExtension
		}
	case extValueQuoted:
		switch {
		case b == '"':
			v.extensionState = extValueAfterQuote
		case b == '\\':
			v.extensionState = extValueQuotedEscape
		case isQuotedText(b):
		default:
			return ErrInvalidChunkExtension
		}
	case extValueQuotedEscape:
		if !isQuotedPair(b) {
			return ErrInvalidChunkExtension
		}
		v.extensionState = extValueQuoted
	case extValueAfterQuote:
		if isBWS(b) {
			return nil
		}
		switch b {
		case ';':
			v.extensionState = extNameStart
		case '\r':
			v.state = stateChunkSizeLF
		default:
			return ErrInvalidChunkExtension
		}
	}
	return nil
}

func (v *ChunkValidator) finishChunkLine() error {
	size := v.chunkSize
	if size == 0 {
		v.state = stateTrailerStart
		v.resetChunkLine()
		return nil
	}

	currentTotal := v.totalBytes.Load()
	if size > ^uint64(0)-currentTotal {
		return ErrBodyTooLarge
	}
	if limit := v.limits.MaxTotalBytes; limit != 0 {
		if currentTotal > limit || size > limit-currentTotal {
			return ErrBodyTooLarge
		}
	}

	count := v.chunkCount.Load()
	if limit := v.limits.MaxChunks; limit != 0 && count >= limit {
		return ErrTooManyChunks
	}

	event := ChunkEvent{Index: count + 1, Size: size, TotalBytes: currentTotal + size}
	if err := v.observe(event); err != nil {
		return err
	}

	v.chunkCount.Add(1)
	v.lastChunkBytes.Store(size)
	v.chunkRemaining = size
	v.state = stateChunkData
	v.resetChunkLine()
	return nil
}

func (v *ChunkValidator) observe(event ChunkEvent) error {
	if v.observer == nil {
		return nil
	}
	if observer, ok := v.observer.(ChunkObserverFunc); ok && observer == nil {
		return nil
	}
	return v.observer.ObserveChunk(event)
}

func (v *ChunkValidator) resetChunkLine() {
	v.lineBytes = 0
	v.chunkSize = 0
	v.chunkDigits = 0
	v.extensionState = extNameStart
}

func (v *ChunkValidator) consumeTrailerByte(b byte) error {
	switch v.state {
	case stateTrailerStart:
		switch {
		case b == '\r':
			v.trailerEndsBody = true
			v.state = stateTrailerCR
		case isTokenChar(b):
			v.state = stateTrailerName
		default:
			return ErrInvalidTrailer
		}
	case stateTrailerName:
		if b == ':' {
			v.state = stateTrailerValue
			return nil
		}
		if !isTokenChar(b) {
			return ErrInvalidTrailer
		}
	case stateTrailerValue:
		if b == '\r' {
			v.trailerEndsBody = false
			v.state = stateTrailerCR
			return nil
		}
		if b == '\n' || b < 0x20 && b != '\t' || b == 0x7f {
			return ErrInvalidTrailer
		}
	case stateTrailerCR:
		if b != '\n' {
			return ErrInvalidTrailer
		}
		if v.trailerEndsBody {
			v.state = stateComplete
			v.complete.Store(true)
			return nil
		}
		v.state = stateTrailerStart
	default:
		return ErrInvalidTrailer
	}
	return nil
}

var validatorBufferPool = sync.Pool{
	New: func() any {
		buffer := make([]byte, 32*1024)
		return &buffer
	},
}

// Validate consumes and validates a complete chunked body from r. It is
// intended for isolated bodies; use Consume when preserving a pipelined
// message boundary is required.
func (v *ChunkValidator) Validate(r io.Reader) error {
	return v.ValidateContext(context.Background(), r)
}

// ValidateContext consumes and validates a complete chunked body while
// observing cancellation between reads.
func (v *ChunkValidator) ValidateContext(ctx context.Context, r io.Reader) error {
	if ctx == nil {
		ctx = context.Background()
	}
	bufferPointer := validatorBufferPool.Get().(*[]byte)
	defer validatorBufferPool.Put(bufferPointer)
	buffer := *bufferPointer

	emptyReads := 0
	for !v.Complete() {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := r.Read(buffer)
		if n < 0 || n > len(buffer) {
			return fmt.Errorf("parser: invalid read count %d", n)
		}
		if n > 0 {
			consumed, validationErr := v.Consume(buffer[:n])
			if validationErr != nil {
				return validationErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if consumed != n {
				if v.Complete() && (readErr == nil || errors.Is(readErr, io.EOF)) {
					return nil
				}
				if readErr != nil && !errors.Is(readErr, io.EOF) {
					return readErr
				}
				return io.ErrShortWrite
			}
			emptyReads = 0
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if v.Complete() {
					return nil
				}
				return fmt.Errorf("%w: transfer-encoding body ended before the final chunk", ErrIncompleteChunk)
			}
			return readErr
		}
		if n == 0 {
			emptyReads++
			if emptyReads >= 100 {
				return io.ErrNoProgress
			}
		}
	}
	return nil
}

// ValidateChunked validates an isolated chunked body with the supplied limits.
func ValidateChunked(r io.Reader, limits ChunkLimits) error {
	return NewChunkValidator(limits).Validate(r)
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func hexValue(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	default:
		return int(b-'A') + 10
	}
}

func isBWS(b byte) bool {
	return b == ' ' || b == '\t'
}

func isTokenChar(b byte) bool {
	if (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
		return true
	}
	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}

func isQuotedText(b byte) bool {
	return b == '\t' || b == ' ' || b == 0x21 || (b >= 0x23 && b <= 0x5b) || (b >= 0x5d && b <= 0x7e) || b >= 0x80
}

func isQuotedPair(b byte) bool {
	return b == '\t' || b == ' ' || (b >= 0x21 && b <= 0x7e) || b >= 0x80
}
