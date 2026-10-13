package tunnel

import (
	"bytes"
	"testing"
)

func TestDatagramEncapsulationDecapsulation(t *testing.T) {
	pool := NewBufferPool(2048)
	var buf bytes.Buffer

	contextID := uint64(12345)
	payload := []byte("hello zero-copy udp packet payload")

	err := WriteDatagram(&buf, contextID, payload)
	if err != nil {
		t.Fatalf("failed to write datagram: %v", err)
	}

	datagram, err := ReadDatagram(&buf, pool)
	kanalErr := err
	if kanalErr != nil {
		t.Fatalf("failed to read datagram: %v", kanalErr)
	}

	if datagram.ContextID != contextID {
		t.Errorf("expected context ID %d, got %d", contextID, datagram.ContextID)
	}

	if !bytes.Equal(datagram.Payload, payload) {
		t.Errorf("expected payload %q, got %q", payload, datagram.Payload)
	}

	encPkts, decPkts, _, _, _ := GetGlobalMetrics().Stats()
	if encPkts == 0 || decPkts == 0 {
		t.Errorf("expected metrics counters to increment, got enc=%d dec=%d", encPkts, decPkts)
	}
}

func TestDatagramEdgeCases(t *testing.T) {
	pool := NewBufferPool(1024)
	// Empty payload test
	var buf bytes.Buffer
	err := WriteDatagram(&buf, 0, nil)
	if err != nil {
		t.Fatalf("failed to write empty datagram: %v", err)
	}

	dg, err := ReadDatagram(&buf, pool)
	if err != nil {
		t.Fatalf("failed to read empty datagram: %v", err)
	}
	if dg.ContextID != 0 || len(dg.Payload) != 0 {
		t.Errorf("expected empty datagram, got %+v", dg)
	}
}
