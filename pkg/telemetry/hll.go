package telemetry

import (
	"math"
	"sync/atomic"
)

const (
	// HLLPrecision determines the number of registers ($m = 2^p$).
	// p = 11 gives 2048 registers, standard error ~ 1 / sqrt(2048) ≈ 2.2%
	HLLPrecision = 11
	HLLRegisters = 1 << HLLPrecision
)

// cacheLinePad provides 64-byte cache padding to prevent false sharing across CPU cores.
type cacheLinePad struct {
	val atomic.Uint32
	_   [7]uint64 // pads to 64 bytes on 64-bit arch
}

// HyperLogLogEstimator implements a lock-free, cache-aligned HLL cardinality estimator.
type HyperLogLogEstimator struct {
	registers [HLLRegisters]cacheLinePad
}

// NewHyperLogLogEstimator constructs a new HLL cardinality estimator.
func NewHyperLogLogEstimator() *HyperLogLogEstimator {
	return &HyperLogLogEstimator{}
}

// Add hashes and registers an item (e.g. client IP string or bytes) into the HLL estimator.
// Operates with zero heap allocations on the hot path.
func (h *HyperLogLogEstimator) Add(data []byte) {
	hash := xxHash64(data)
	// Use upper bits for register index
	regIdx := uint(hash >> (64 - HLLPrecision))
	// Remaining bits for leading zeros count
	valBits := (hash << HLLPrecision) | (1 << (HLLPrecision - 1))

	lz := bitsLeadingZeros64(valBits)
	var zeros uint8
	if lz >= 255 {
		zeros = 255
	} else if lz < 0 {
		zeros = 0
	} else {
		zeros = uint8(lz) + 1
	}

	// Lock-free atomic max update
	reg := &h.registers[regIdx]
	for {
		old := reg.val.Load()
		if uint32(zeros) <= old {
			break
		}
		if reg.val.CompareAndSwap(old, uint32(zeros)) {
			break
		}
	}
}

// AddString convenience wrapper for string keys (such as client IP addresses).
func (h *HyperLogLogEstimator) AddString(s string) {
	// Convert string to bytes without heap allocation via unsafe or standard conversion
	h.Add([]byte(s))
}

// Estimate computes the approximate cardinality of distinct items.
func (h *HyperLogLogEstimator) Estimate() uint64 {
	m := float64(HLLRegisters)
	sum := 0.0
	zerosCount := 0

	for i := 0; i < HLLRegisters; i++ {
		val := h.registers[i].val.Load()
		sum += 1.0 / math.Pow(2.0, float64(val))
		if val == 0 {
			zerosCount++
		}
	}

	// Alpha correction constant
	var alpha float64
	switch HLLRegisters {
	case 16:
		alpha = 0.673
	case 32:
		alpha = 0.697
	case 64:
		alpha = 0.709
	default:
		alpha = 0.7213 / (1.0 + 1.079/m)
	}

	rawEstimate := alpha * m * m / sum

	// Small range correction (LinearCounting)
	if rawEstimate <= 2.5*m && zerosCount > 0 {
		return uint64(m * math.Log(m/float64(zerosCount)))
	}

	return uint64(rawEstimate)
}

// Reset clears all register states.
func (h *HyperLogLogEstimator) Reset() {
	for i := 0; i < HLLRegisters; i++ {
		h.registers[i].val.Store(0)
	}
}

// xxHash64 implements a fast, non-cryptographic hash function (xxHash variant) with zero allocations.
func xxHash64(input []byte) uint64 {
	const prime64_1 = 11400714785032798589
	const prime64_2 = 14029467366897019727
	const prime64_3 = 1609587929392839161
	const prime64_4 = 9650029242287828079
	const prime64_5 = 2870177450012600261

	hash := uint64(0)
	length := len(input)
	index := 0

	if length >= 8 {
		limit := length - 8
		hash = prime64_5
		for index <= limit {
			k1 := getUint64LE(input[index:])
			hash ^= round64(0, k1)
			hash = hash*prime64_1 + prime64_4
			index += 8
		}
	} else {
		hash = prime64_5 + uint64(length)
	}

	for index+4 <= length {
		k1 := uint64(getUint32LE(input[index:]))
		hash ^= k1 * prime64_1
		hash = hash*prime64_2 + prime64_3
		index += 4
	}

	for index < length {
		hash ^= uint64(input[index]) * prime64_5
		hash = hash*prime64_1 + prime64_4
		index++
	}

	hash ^= hash >> 33
	hash *= prime64_2
	hash ^= hash >> 29
	hash *= prime64_3
	hash ^= hash >> 32

	return hash
}

func round64(acc, input uint64) uint64 {
	const prime64_2 = 14029467366897019727
	const prime64_1 = 11400714785032798589
	acc += input * prime64_2
	acc = rotl64(acc, 31)
	return acc * prime64_1
}

func rotl64(x uint64, r uint) uint64 {
	return (x << r) | (x >> (64 - r))
}

func getUint64LE(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func getUint32LE(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func bitsLeadingZeros64(v uint64) int {
	if v == 0 {
		return 64
	}
	n := 0
	if v <= 0x00000000FFFFFFFF {
		n += 32
		v <<= 32
	}
	if v <= 0x0000FFFFFFFFFFFF {
		n += 16
		v <<= 16
	}
	if v <= 0x00FFFFFFFFFFFFFF {
		n += 8
		v <<= 8
	}
	if v <= 0x0FFFFFFFFFFFFFFF {
		n += 4
		v <<= 4
	}
	if v <= 0x3FFFFFFFFFFFFFFF {
		n += 2
		v <<= 2
	}
	if v <= 0x7FFFFFFFFFFFFFFF {
		n += 1
	}
	return n
}
