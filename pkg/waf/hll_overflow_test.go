package waf

import (
	"encoding/binary"
	"math/bits"
	"testing"
)

const (
	hllRegressionPrime1 uint64 = 11400714785074694791
	hllRegressionPrime2 uint64 = 14029467366897019727
	hllRegressionPrime3 uint64 = 1609587929392839161
	hllRegressionPrime4 uint64 = 9650029242287828579
	hllRegressionPrime5 uint64 = 2870177450012600261
)

func TestHLLHashBytesXXH64BranchBoundaries(t *testing.T) {
	const emptyHash uint64 = 0xef46db3751d8e999
	if got := hllHashBytes(nil); got != emptyHash {
		t.Fatalf("empty input hash: got %#x, want %#x", got, emptyHash)
	}

	lengths := []int{1, 3, 4, 7, 8, 31, 32, 33, 64}
	for _, length := range lengths {
		data := make([]byte, length)
		for index := range data {
			data[index] = byte((index*29 + 11) & 0xff)
		}

		got := hllHashBytes(data)
		want := hllRegressionXXH64(data, 0)
		if got != want {
			t.Errorf("length %d: got %#016x, want %#016x", length, got, want)
		}
	}
}

func TestHLLPositionRankExcludesIndexBits(t *testing.T) {
	tracker := MustNewHLL(MinHLLPrecision)

	index, rank := tracker.position(uint64(1) << 59)
	if index != 0 {
		t.Fatalf("register index: got %d, want 0", index)
	}
	if rank != 1 {
		t.Fatalf("register rank: got %d, want 1", rank)
	}

	_, rank = tracker.position(0)
	maximumRank := uint64(MinHLLPrecision + 1)
	if rank != maximumRank {
		t.Fatalf("maximum register rank: got %d, want %d", rank, maximumRank)
	}
}

func TestHLLPositionUsesConfiguredPrecision(t *testing.T) {
	tracker := MustNewHLL(MaxHLLPrecision)

	index, _ := tracker.position(^uint64(0))
	const wantIndex uint64 = (uint64(1) << MaxHLLPrecision) - 1
	if index != wantIndex {
		t.Fatalf("maximum register index: got %d, want %d", index, wantIndex)
	}

	_, rank := tracker.position(0)
	const wantMaximumRank uint64 = MaxHLLPrecision + 1
	if rank != wantMaximumRank {
		t.Fatalf("maximum register rank: got %d, want %d", rank, wantMaximumRank)
	}
}

func hllRegressionXXH64(data []byte, seed uint64) uint64 {
	length := uint64(len(data))
	var hash uint64

	if len(data) >= 32 {
		v1 := seed + hllRegressionPrime1
		v1 += hllRegressionPrime2
		v2 := seed + hllRegressionPrime2
		v3 := seed
		v4 := seed
		v4 -= hllRegressionPrime1

		for len(data) >= 32 {
			v1 = hllRegressionXXHRound(v1, binary.LittleEndian.Uint64(data[0:8]))
			v2 = hllRegressionXXHRound(v2, binary.LittleEndian.Uint64(data[8:16]))
			v3 = hllRegressionXXHRound(v3, binary.LittleEndian.Uint64(data[16:24]))
			v4 = hllRegressionXXHRound(v4, binary.LittleEndian.Uint64(data[24:32]))
			data = data[32:]
		}

		hash = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) +
			bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		hash = hllRegressionXXHMergeRound(hash, v1)
		hash = hllRegressionXXHMergeRound(hash, v2)
		hash = hllRegressionXXHMergeRound(hash, v3)
		hash = hllRegressionXXHMergeRound(hash, v4)
	} else {
		hash = seed + hllRegressionPrime5
	}

	hash += length
	for len(data) >= 8 {
		hash ^= hllRegressionXXHRound(0, binary.LittleEndian.Uint64(data[0:8]))
		hash = bits.RotateLeft64(hash, 27)*hllRegressionPrime1 + hllRegressionPrime4
		data = data[8:]
	}
	if len(data) >= 4 {
		hash ^= uint64(binary.LittleEndian.Uint32(data[0:4])) * hllRegressionPrime1
		hash = bits.RotateLeft64(hash, 23)*hllRegressionPrime2 + hllRegressionPrime3
		data = data[4:]
	}
	for _, value := range data {
		hash ^= uint64(value) * hllRegressionPrime5
		hash = bits.RotateLeft64(hash, 11) * hllRegressionPrime1
	}

	hash ^= hash >> 33
	hash *= hllRegressionPrime2
	hash ^= hash >> 29
	hash *= hllRegressionPrime3
	hash ^= hash >> 32
	return hash
}

func hllRegressionXXHRound(accumulator, input uint64) uint64 {
	accumulator += input * hllRegressionPrime2
	accumulator = bits.RotateLeft64(accumulator, 31)
	accumulator *= hllRegressionPrime1
	return accumulator
}

func hllRegressionXXHMergeRound(accumulator, value uint64) uint64 {
	value = hllRegressionXXHRound(0, value)
	accumulator ^= value
	accumulator = accumulator*hllRegressionPrime1 + hllRegressionPrime4
	return accumulator
}
