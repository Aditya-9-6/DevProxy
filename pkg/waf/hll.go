package waf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"net"
	"net/netip"
	"sync/atomic"
)

const (
	// DefaultHLLPrecision uses 16,384 registers and approximately 0.8% error.
	DefaultHLLPrecision = 14
	MinHLLPrecision     = 4
	MaxHLLPrecision     = 18

	cacheLineSize     = 64
	hllThirtyTwoRange = float64(1 << 30)
)

var (
	ErrInvalidHLLPrecision = errors.New("invalid HyperLogLog precision")
	ErrNilHyperLogLog      = errors.New("nil HyperLogLog")
)

// hllRegister occupies one cache line so concurrently updated registers do not
// cause false sharing. Only the low rank bits are used by the estimator.
type hllRegister struct {
	value atomic.Uint64
	_     [cacheLineSize/8 - 1]uint64
}

// hllState is immutable after publication except for its atomic registers and
// histogram. Publishing a new state makes resets linearizable without locks.
type hllState struct {
	registers    []hllRegister
	histogram    [64]atomic.Int64
	observations atomic.Uint64
}

// HyperLogLog estimates the number of distinct values observed. Updates are
// lock-free, and the fixed register and histogram storage is allocated once.
type HyperLogLog struct {
	precision     int
	registerCount int
	suffixMask    uint64
	state         atomic.Pointer[hllState]
}

// NewHyperLogLog constructs an estimator. With no argument it uses the default
// precision. It returns nil when the precision is invalid; callers that prefer
// explicit error handling should use NewHyperLogLogChecked.
func NewHyperLogLog(precision ...int) *HyperLogLog {
	if len(precision) > 1 {
		return nil
	}

	p := DefaultHLLPrecision
	if len(precision) == 1 {
		p = precision[0]
	}

	tracker, err := NewHyperLogLogChecked(p)
	if err != nil {
		return nil
	}
	return tracker
}

// NewHLL is a concise alias for NewHyperLogLog.
func NewHLL(precision ...int) *HyperLogLog {
	return NewHyperLogLog(precision...)
}

// NewHyperLogLogChecked constructs an estimator and reports invalid precision.
func NewHyperLogLogChecked(precision int) (*HyperLogLog, error) {
	if precision < MinHLLPrecision || precision > MaxHLLPrecision {
		return nil, fmt.Errorf(
			"%w: %d (supported range is %d-%d)",
			ErrInvalidHLLPrecision,
			precision,
			MinHLLPrecision,
			MaxHLLPrecision,
		)
	}

	registerCount := 1 << precision
	tracker := &HyperLogLog{
		precision:     precision,
		registerCount: registerCount,
		suffixMask:    (uint64(1) << uint(64-precision)) - 1,
	}
	tracker.state.Store(newHLLState(registerCount))
	return tracker, nil
}

// MustNewHyperLogLog constructs an estimator and panics on invalid precision.
func MustNewHyperLogLog(precision int) *HyperLogLog {
	tracker, err := NewHyperLogLogChecked(precision)
	if err != nil {
		panic(err)
	}
	return tracker
}

// MustNewHLL is a concise alias for MustNewHyperLogLog.
func MustNewHLL(precision int) *HyperLogLog {
	return MustNewHyperLogLog(precision)
}

func newHLLState(registerCount int) *hllState {
	state := &hllState{registers: make([]hllRegister, registerCount)}
	state.histogram[0].Store(int64(registerCount))
	return state
}

// Add hashes and records an arbitrary byte sequence. It performs no heap
// allocation and returns false only for an invalid tracker.
func (h *HyperLogLog) Add(data []byte) bool {
	if h == nil {
		return false
	}

	state := h.state.Load()
	if state == nil {
		return false
	}

	h.addHash(state, hllHashBytes(data))
	return true
}

// AddIP records a client address represented by net.IP.
func (h *HyperLogLog) AddIP(ip net.IP) bool {
	if h == nil || len(ip) == 0 {
		return false
	}

	state := h.state.Load()
	if state == nil {
		return false
	}

	if ip4 := ip.To4(); ip4 != nil {
		h.addHash(state, hllHashBytes(ip4))
		return true
	}
	if ip16 := ip.To16(); ip16 != nil {
		h.addHash(state, hllHashBytes(ip16))
		return true
	}
	return false
}

// AddAddr records a client address without converting it to a string or slice.
func (h *HyperLogLog) AddAddr(addr netip.Addr) bool {
	if h == nil || !addr.IsValid() {
		return false
	}

	state := h.state.Load()
	if state == nil {
		return false
	}

	addr = addr.Unmap().WithZone("")
	if addr.Is4() {
		bytes := addr.As4()
		h.addHash(state, hllHashBytes(bytes[:]))
		return true
	}

	bytes := addr.As16()
	h.addHash(state, hllHashBytes(bytes[:]))
	return true
}

// ObserveAddr is an alias for AddAddr.
func (h *HyperLogLog) ObserveAddr(addr netip.Addr) bool {
	return h.AddAddr(addr)
}

func (h *HyperLogLog) addHash(state *hllState, hash uint64) {
	state.observations.Add(1)
	index, rank := h.position(hash)
	h.updateRegister(state, index, rank)
}

func (h *HyperLogLog) position(hash uint64) (uint32, uint64) {
	index := hash >> uint(64-h.precision)
	suffix := hash & h.suffixMask

	// Shift the suffix into the high bits so LeadingZeros64 counts only the
	// non-index portion of the hash. LeadingZeros64 is a compiler intrinsic.
	rank := uint64(bits.LeadingZeros64(suffix<<uint(h.precision))) + 1
	maximumRank := uint64(h.precision + 1)
	if rank > maximumRank {
		rank = maximumRank
	}
	return uint32(index), rank
}

func (h *HyperLogLog) updateRegister(state *hllState, index uint32, rank uint64) {
	register := &state.registers[index]
	for {
		current := register.value.Load()
		if current >= rank {
			return
		}
		if register.value.CompareAndSwap(current, rank) {
			state.moveHistogram(current, rank)
			return
		}
	}
}

func (state *hllState) moveHistogram(oldRank, newRank uint64) {
	if oldRank == newRank {
		return
	}
	state.histogram[oldRank].Add(-1)
	state.histogram[newRank].Add(1)
}

// Estimate returns the current approximate cardinality. It reads a fixed-size
// rank histogram, making the calculation independent of register count.
func (h *HyperLogLog) Estimate() float64 {
	if h == nil {
		return 0
	}

	state := h.state.Load()
	if state == nil {
		return 0
	}

	sum, zeroRegisters := state.histogramTotals()
	if sum <= 0 {
		return 0
	}

	m := float64(h.registerCount)
	estimate := h.alpha() * m * m / sum

	if estimate <= 2.5*m && zeroRegisters > 0 {
		if zeroRegisters > int64(h.registerCount) {
			zeroRegisters = int64(h.registerCount)
		}
		estimate = m * math.Log(m/float64(zeroRegisters))
	}

	if h.registerCount == 32 && estimate > hllThirtyTwoRange/30 && estimate < hllThirtyTwoRange {
		corrected := -hllThirtyTwoRange * math.Log1p(-estimate/hllThirtyTwoRange)
		if corrected > 0 {
			estimate = corrected
		}
	}
	return estimate
}

func (state *hllState) histogramTotals() (float64, int64) {
	var sum float64
	var zeroRegisters int64

	for rank := range state.histogram {
		count := state.histogram[rank].Load()
		if count <= 0 {
			continue
		}
		if rank == 0 {
			zeroRegisters += count
		}
		sum += float64(count) * math.Ldexp(1, -rank)
	}
	return sum, zeroRegisters
}

func (h *HyperLogLog) alpha() float64 {
	switch h.registerCount {
	case 16:
		return 0.673
	case 32:
		return 0.697
	case 64:
		return 0.709
	default:
		m := float64(h.registerCount)
		return 0.7213 / (1 + 1.079/m)
	}
}

// Count returns the rounded cardinality estimate.
func (h *HyperLogLog) Count() uint64 {
	estimate := h.Estimate()
	if estimate <= 0 {
		return 0
	}

	maximum := ^uint64(0)
	if estimate >= float64(maximum) {
		return maximum
	}
	return uint64(math.Floor(estimate + 0.5))
}

// Cardinality is an alias for Count.
func (h *HyperLogLog) Cardinality() uint64 {
	return h.Count()
}

// Observations returns the number of accepted observations, including
// duplicates. It is not a distinct-cardinality count.
func (h *HyperLogLog) Observations() uint64 {
	if h == nil {
		return 0
	}
	state := h.state.Load()
	if state == nil {
		return 0
	}
	return state.observations.Load()
}

// Precision returns the configured HLL precision.
func (h *HyperLogLog) Precision() int {
	if h == nil {
		return 0
	}
	return h.precision
}

// RegisterCount returns the number of HLL registers.
func (h *HyperLogLog) RegisterCount() int {
	if h == nil {
		return 0
	}
	return h.registerCount
}

// Merge incorporates another sketch using register-wise maxima. Both sketches
// must use the same precision and deterministic hash function.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h == nil || other == nil {
		return ErrNilHyperLogLog
	}
	if h.precision != other.precision {
		return fmt.Errorf("cannot merge HyperLogLog precision %d with %d", h.precision, other.precision)
	}
	if h == other {
		return nil
	}

	state := h.state.Load()
	otherState := other.state.Load()
	if state == nil || otherState == nil {
		return ErrNilHyperLogLog
	}

	for index := range state.registers {
		h.updateRegister(state, uint32(index), otherState.registers[index].value.Load())
	}
	return nil
}

// Snapshot returns register values suitable for transporting a sketch between
// distributed proxy instances. The allocation is intentional and is not used
// on the request hot path.
func (h *HyperLogLog) Snapshot() []uint64 {
	if h == nil {
		return nil
	}
	state := h.state.Load()
	if state == nil {
		return nil
	}

	values := make([]uint64, len(state.registers))
	for index := range state.registers {
		values[index] = state.registers[index].value.Load()
	}
	return values
}

// MergeSnapshot incorporates register values produced by Snapshot.
func (h *HyperLogLog) MergeSnapshot(values []uint64) error {
	if h == nil {
		return ErrNilHyperLogLog
	}
	if len(values) != h.registerCount {
		return fmt.Errorf("invalid HyperLogLog snapshot length %d, expected %d", len(values), h.registerCount)
	}

	maximumRank := uint64(h.precision + 1)
	for _, value := range values {
		if value > maximumRank {
			return fmt.Errorf("invalid HyperLogLog register rank %d", value)
		}
	}

	state := h.state.Load()
	if state == nil {
		return ErrNilHyperLogLog
	}
	for index, value := range values {
		h.updateRegister(state, uint32(index), value)
	}
	return nil
}

// Clone returns an independent copy of the sketch.
func (h *HyperLogLog) Clone() (*HyperLogLog, error) {
	if h == nil {
		return nil, ErrNilHyperLogLog
	}

	source := h.state.Load()
	if source == nil {
		return nil, ErrNilHyperLogLog
	}

	clone, err := NewHyperLogLogChecked(h.precision)
	if err != nil {
		return nil, err
	}
	target := clone.state.Load()

	for index := range source.registers {
		rank := source.registers[index].value.Load()
		target.registers[index].value.Store(rank)
		target.moveHistogram(0, rank)
	}
	target.observations.Store(source.observations.Load())
	return clone, nil
}

// Reset atomically publishes a fresh sketch. An observation that loaded the
// previous state before publication may finish against that old state and is
// discarded, providing Reset with a clear linearization point.
func (h *HyperLogLog) Reset() {
	if h == nil {
		return
	}
	h.state.Store(newHLLState(h.registerCount))
}

const (
	hllPrime1 uint64 = 11400714785074694791
	hllPrime2 uint64 = 14029467366897019727
	hllPrime3 uint64 = 1609587929392839161
	hllPrime4 uint64 = 9650029242287828579
	hllPrime5 uint64 = 2870177450012600261
)

// hllHashBytes is a deterministic XXH64 implementation. A fixed hash is
// required so independently collected distributed sketches can be merged.
func hllHashBytes(data []byte) uint64 {
	length := len(data)
	var hash uint64

	if len(data) >= 32 {
		// These must remain runtime operations. Go rejects overflowing constant
		// expressions, while XXH64 requires unsigned modulo-2^64 arithmetic.
		v1 := hllPrime1
		v1 += hllPrime2
		v2 := hllPrime2
		v3 := uint64(0)
		v4 := uint64(0)
		v4 -= hllPrime1

		for len(data) >= 32 {
			v1 = hllRound(v1, binary.LittleEndian.Uint64(data[0:8]))
			v2 = hllRound(v2, binary.LittleEndian.Uint64(data[8:16]))
			v3 = hllRound(v3, binary.LittleEndian.Uint64(data[16:24]))
			v4 = hllRound(v4, binary.LittleEndian.Uint64(data[24:32]))
			data = data[32:]
		}

		hash = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) +
			bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		hash = hllMergeRound(hash, v1)
		hash = hllMergeRound(hash, v2)
		hash = hllMergeRound(hash, v3)
		hash = hllMergeRound(hash, v4)
	} else {
		hash = hllPrime5
	}

	hash += uint64(length)
	for len(data) >= 8 {
		hash ^= hllRound(0, binary.LittleEndian.Uint64(data[0:8]))
		hash = bits.RotateLeft64(hash, 27)*hllPrime1 + hllPrime4
		data = data[8:]
	}
	if len(data) >= 4 {
		hash ^= uint64(binary.LittleEndian.Uint32(data[0:4])) * hllPrime1
		hash = bits.RotateLeft64(hash, 23)*hllPrime2 + hllPrime3
		data = data[4:]
	}
	for _, value := range data {
		hash ^= uint64(value) * hllPrime5
		hash = bits.RotateLeft64(hash, 11) * hllPrime1
	}

	hash ^= hash >> 33
	hash *= hllPrime2
	hash ^= hash >> 29
	hash *= hllPrime3
	hash ^= hash >> 32
	return hash
}

func hllRound(accumulator, input uint64) uint64 {
	accumulator += input * hllPrime2
	accumulator = bits.RotateLeft64(accumulator, 31)
	accumulator *= hllPrime1
	return accumulator
}

func hllMergeRound(accumulator, value uint64) uint64 {
	value = hllRound(0, value)
	accumulator ^= value
	accumulator = accumulator*hllPrime1 + hllPrime4
	return accumulator
}

// IPHeavyHitterTracker uses HLL cardinality to identify high-volume client
// diversity, which is a signal of distributed scanning activity. It estimates
// cardinality; it does not claim to identify an individual top talker.
type IPHeavyHitterTracker struct {
	sketch    *HyperLogLog
	threshold atomic.Uint64
}

// IPCardinalityTracker is an alias emphasizing the tracker's cardinality role.
type IPCardinalityTracker = IPHeavyHitterTracker

// NewIPHeavyHitterTracker constructs a tracker with the supplied anomaly
// threshold. A threshold of zero disables anomaly reporting.
func NewIPHeavyHitterTracker(threshold uint64) *IPHeavyHitterTracker {
	tracker := &IPHeavyHitterTracker{sketch: NewHyperLogLog()}
	tracker.threshold.Store(threshold)
	return tracker
}

// NewIPCardinalityTracker is an alias for NewIPHeavyHitterTracker.
func NewIPCardinalityTracker(threshold uint64) *IPCardinalityTracker {
	return NewIPHeavyHitterTracker(threshold)
}

// NewCardinalityTracker is an alias for NewIPHeavyHitterTracker.
func NewCardinalityTracker(threshold uint64) *IPHeavyHitterTracker {
	return NewIPHeavyHitterTracker(threshold)
}

// ObserveAddr records an address and reports whether the current cardinality
// has reached the configured threshold.
func (t *IPHeavyHitterTracker) ObserveAddr(addr netip.Addr) (uint64, bool) {
	if t == nil || t.sketch == nil || !addr.IsValid() {
		return 0, false
	}

	t.sketch.AddAddr(addr)
	count := t.sketch.Count()
	threshold := t.threshold.Load()
	return count, threshold > 0 && count >= threshold
}

// Observe is an alias for ObserveAddr.
func (t *IPHeavyHitterTracker) Observe(addr netip.Addr) (uint64, bool) {
	return t.ObserveAddr(addr)
}

// ObserveIP records a net.IP client address.
func (t *IPHeavyHitterTracker) ObserveIP(ip net.IP) (uint64, bool) {
	if t == nil || t.sketch == nil || !t.sketch.AddIP(ip) {
		return 0, false
	}

	count := t.sketch.Count()
	threshold := t.threshold.Load()
	return count, threshold > 0 && count >= threshold
}

// ObserveBytes records an arbitrary byte sequence, which is useful for tests
// and for generic cardinality tracking.
func (t *IPHeavyHitterTracker) ObserveBytes(data []byte) (uint64, bool) {
	if t == nil || t.sketch == nil || !t.sketch.Add(data) {
		return 0, false
	}

	count := t.sketch.Count()
	threshold := t.threshold.Load()
	return count, threshold > 0 && count >= threshold
}

// Count returns the current cardinality estimate.
func (t *IPHeavyHitterTracker) Count() uint64 {
	if t == nil || t.sketch == nil {
		return 0
	}
	return t.sketch.Count()
}

// Estimate returns the unrounded cardinality estimate.
func (t *IPHeavyHitterTracker) Estimate() float64 {
	if t == nil || t.sketch == nil {
		return 0
	}
	return t.sketch.Estimate()
}

// Threshold returns the active anomaly threshold.
func (t *IPHeavyHitterTracker) Threshold() uint64 {
	if t == nil {
		return 0
	}
	return t.threshold.Load()
}

// SetThreshold updates the anomaly threshold without locking the hot path.
func (t *IPHeavyHitterTracker) SetThreshold(threshold uint64) {
	if t != nil {
		t.threshold.Store(threshold)
	}
}

// Merge incorporates another tracker or distributed proxy sketch.
func (t *IPHeavyHitterTracker) Merge(other *IPHeavyHitterTracker) error {
	if t == nil || other == nil || t.sketch == nil || other.sketch == nil {
		return ErrNilHyperLogLog
	}
	return t.sketch.Merge(other.sketch)
}

// Sketch exposes the underlying sketch for snapshot-based distribution.
func (t *IPHeavyHitterTracker) Sketch() *HyperLogLog {
	if t == nil {
		return nil
	}
	return t.sketch
}

// Reset starts a new cardinality window while retaining the threshold.
func (t *IPHeavyHitterTracker) Reset() {
	if t != nil && t.sketch != nil {
		t.sketch.Reset()
	}
}
