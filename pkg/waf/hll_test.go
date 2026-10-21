package waf

import (
	"errors"
	"net/netip"
	"sync"
	"testing"
)

func TestNewHyperLogLogChecked(t *testing.T) {
	tests := []struct {
		name      string
		precision int
		wantError bool
	}{
		{name: "minimum", precision: MinHLLPrecision},
		{name: "default", precision: DefaultHLLPrecision},
		{name: "maximum", precision: MaxHLLPrecision},
		{name: "below minimum", precision: MinHLLPrecision - 1, wantError: true},
		{name: "above maximum", precision: MaxHLLPrecision + 1, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker, err := NewHyperLogLogChecked(test.precision)
			if test.wantError {
				if err == nil {
					t.Fatalf("NewHyperLogLogChecked(%d) returned no error", test.precision)
				}
				if !errors.Is(err, ErrInvalidHLLPrecision) {
					t.Fatalf("NewHyperLogLogChecked(%d) error = %v, want ErrInvalidHLLPrecision", test.precision, err)
				}
				if NewHyperLogLog(test.precision) != nil {
					t.Fatalf("NewHyperLogLog(%d) returned a tracker for invalid precision", test.precision)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewHyperLogLogChecked(%d) error = %v", test.precision, err)
			}
			if tracker.Precision() != test.precision {
				t.Fatalf("Precision() = %d, want %d", tracker.Precision(), test.precision)
			}
			if tracker.RegisterCount() != 1<<test.precision {
				t.Fatalf("RegisterCount() = %d, want %d", tracker.RegisterCount(), 1<<test.precision)
			}
		})
	}
}

func TestHyperLogLogCardinality(t *testing.T) {
	tests := []struct {
		name         string
		count        int
		repeatFirst  bool
		minimum      uint64
		maximum      uint64
		minimumFloat float64
		maximumFloat float64
	}{
		{name: "empty", minimum: 0, maximum: 0, minimumFloat: 0, maximumFloat: 0},
		{name: "single", count: 1, minimum: 1, maximum: 1, minimumFloat: 0.5, maximumFloat: 1.5},
		{name: "duplicate", count: 1, repeatFirst: true, minimum: 1, maximum: 1, minimumFloat: 0.5, maximumFloat: 1.5},
		{name: "one hundred", count: 100, minimum: 75, maximum: 125, minimumFloat: 75, maximumFloat: 125},
		{name: "one thousand", count: 1000, minimum: 750, maximum: 1250, minimumFloat: 750, maximumFloat: 1250},
		{name: "ten thousand", count: 10000, minimum: 8000, maximum: 12000, minimumFloat: 8000, maximumFloat: 12000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := NewHyperLogLog(DefaultHLLPrecision)
			for index := 0; index < test.count; index++ {
				tracker.AddAddr(hllTestAddress(index))
			}
			if test.repeatFirst {
				tracker.AddAddr(hllTestAddress(0))
			}

			estimate := tracker.Estimate()
			if estimate < test.minimumFloat || estimate > test.maximumFloat {
				t.Fatalf("Estimate() = %.2f, want range %.2f..%.2f", estimate, test.minimumFloat, test.maximumFloat)
			}

			cardinality := tracker.Count()
			if cardinality < test.minimum || cardinality > test.maximum {
				t.Fatalf("Count() = %d, want range %d..%d", cardinality, test.minimum, test.maximum)
			}
		})
	}
}

func TestHyperLogLogMergeSnapshotAndReset(t *testing.T) {
	left := NewHyperLogLog(12)
	right := NewHyperLogLog(12)

	for index := 0; index < 1000; index++ {
		left.AddAddr(hllTestAddress(index))
		right.AddAddr(hllTestAddressInBlock(11, index))
	}

	if err := left.Merge(right); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	mergedCount := left.Count()
	if mergedCount < 1600 || mergedCount > 2400 {
		t.Fatalf("merged Count() = %d, want range 1600..2400", mergedCount)
	}

	snapshot := left.Snapshot()
	if len(snapshot) != left.RegisterCount() {
		t.Fatalf("len(Snapshot()) = %d, want %d", len(snapshot), left.RegisterCount())
	}

	restored := NewHyperLogLog(12)
	if err := restored.MergeSnapshot(snapshot); err != nil {
		t.Fatalf("MergeSnapshot() error = %v", err)
	}
	if restored.Count() != mergedCount {
		t.Fatalf("restored Count() = %d, want %d", restored.Count(), mergedCount)
	}

	clone, err := left.Clone()
	if err != nil {
		t.Fatalf("Clone() error = %v", err)
	}
	if clone.Count() != left.Count() {
		t.Fatalf("clone Count() = %d, want %d", clone.Count(), left.Count())
	}

	incompatible := NewHyperLogLog(10)
	if err := left.Merge(incompatible); err == nil {
		t.Fatal("Merge() with incompatible precision returned no error")
	}

	left.Reset()
	if count := left.Count(); count != 0 {
		t.Fatalf("Count() after Reset = %d, want 0", count)
	}
}

func TestHyperLogLogConcurrentUpdates(t *testing.T) {
	const (
		workers        = 16
		itemsPerWorker = 1000
	)

	tracker := NewHyperLogLog(DefaultHLLPrecision)
	var waitGroup sync.WaitGroup
	waitGroup.Add(workers)

	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer waitGroup.Done()
			for index := 0; index < itemsPerWorker; index++ {
				addr := netip.AddrFrom4([4]byte{
					10,
					byte(worker),
					byte(index >> 8),
					byte(index),
				})
				tracker.AddAddr(addr)
			}
		}(worker)
	}

	waitGroup.Wait()

	const expected = workers * itemsPerWorker
	if observations := tracker.Observations(); observations != expected {
		t.Fatalf("Observations() = %d, want %d", observations, expected)
	}

	count := tracker.Count()
	if count < 14000 || count > 18000 {
		t.Fatalf("Count() = %d, want range 14000..18000", count)
	}
}

func TestHyperLogLogInvalidAddresses(t *testing.T) {
	tracker := NewHyperLogLog(DefaultHLLPrecision)

	if tracker.AddAddr(netip.Addr{}) {
		t.Fatal("AddAddr() accepted an invalid address")
	}
	if tracker.AddIP(nil) {
		t.Fatal("AddIP() accepted a nil address")
	}
	if count := tracker.Count(); count != 0 {
		t.Fatalf("Count() = %d, want 0", count)
	}
}

func TestHyperLogLogHotPathAllocations(t *testing.T) {
	tracker := NewHyperLogLog(DefaultHLLPrecision)
	addr := netip.MustParseAddr("192.0.2.10")

	allocations := testing.AllocsPerRun(1000, func() {
		tracker.AddAddr(addr)
		_ = tracker.Count()
	})
	if allocations != 0 {
		t.Fatalf("hot path allocations = %.2f, want 0", allocations)
	}
}

func TestIPHeavyHitterTracker(t *testing.T) {
	tracker := NewIPHeavyHitterTracker(2)
	first := netip.MustParseAddr("198.51.100.1")
	second := netip.MustParseAddr("198.51.100.2")
	third := netip.MustParseAddr("198.51.100.3")

	count, anomaly := tracker.ObserveAddr(first)
	if count != 1 || anomaly {
		t.Fatalf("first ObserveAddr() = (%d, %t), want (1, false)", count, anomaly)
	}

	count, anomaly = tracker.Observe(first)
	if count != 1 || anomaly {
		t.Fatalf("duplicate Observe() = (%d, %t), want (1, false)", count, anomaly)
	}

	count, anomaly = tracker.ObserveAddr(second)
	if count != 2 || !anomaly {
		t.Fatalf("second ObserveAddr() = (%d, %t), want (2, true)", count, anomaly)
	}

	tracker.SetThreshold(3)
	count, anomaly = tracker.ObserveAddr(third)
	if count != 3 || !anomaly {
		t.Fatalf("third ObserveAddr() = (%d, %t), want (3, true)", count, anomaly)
	}
}

func hllTestAddress(index int) netip.Addr {
	return netip.AddrFrom4([4]byte{
		10,
		byte(index >> 16),
		byte(index >> 8),
		byte(index),
	})
}

func hllTestAddressInBlock(second byte, index int) netip.Addr {
	return netip.AddrFrom4([4]byte{
		10,
		second,
		byte(index >> 8),
		byte(index),
	})
}
