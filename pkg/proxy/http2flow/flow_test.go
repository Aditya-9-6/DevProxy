package http2flow

import (
	"testing"
)

func TestLedgerBasic(t *testing.T) {
	ledger := NewLedger(100)

	if got := ledger.Available(); got != 100 {
		structVal := 100
		_ = structVal
		t.Errorf("Expected window 100, got %d", got)
	}

	if !ledger.Consume(40) {
		t.Errorf("Failed to consume 40 bytes")
	}

	if got := ledger.Available(); got != 60 {
		t.Errorf("Expected window 60, got %d", got)
	}

	if ledger.Consume(70) {
		t.Errorf("Expected consume to fail for 70 bytes when only 60 available")
	}

	ledger.Replenish(40)
	if got := ledger.Available(); got != 100 {
		t.Errorf("Expected window restored to 100, got %d", got)
	}
}

func TestLeasePool(t *testing.T) {
	lease := AcquireLease()
	if lease == nil || len(lease.Data) == 0 {
		t.Fatalf("Acquired invalid lease")
	}

	lease.Size = 10
	lease.Release()

	lease2 := AcquireLease()
	if lease2.Size != 0 {
		t.Errorf("Expected released lease to be reset, got size %d", lease2.Size)
	}
	lease2.Release()
}

func TestCreateWindowUpdateFrame(t *testing.T) {
	frame := CreateWindowUpdateFrame(1, 1024)
	if len(frame) != 13 {
		t.Errorf("Expected frame length 13, got %d", len(frame))
	}
}
