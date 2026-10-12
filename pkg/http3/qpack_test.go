package http3

import (
	"sync"
	"testing"
)

func TestQPACKDecoder_BlockedCount(t *testing.T) {
	decoder := NewQPACKDecoder()
	decoder.blockedCount.Add(1)
	if decoder.GetBlockedCount() != 1 {
		t.Errorf("expected blocked count 1, got %d", decoder.GetBlockedCount())
	}
}

func TestStreamBufferPool(t *testing.T) {
	buf := GetBuffer()
	if len(buf) != 16384 {
		t.Errorf("expected buffer size 16384, got %d", len(buf))
	}
	PutBuffer(buf)
}

func TestHeaderBlock_Lifecycle(t *testing.T) {
	decoder := NewQPACKDecoder()
	data := []byte{0x80, 0x01, 0x02}

	hb, err := decoder.DecodeHeaderBlock(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(hb.Data) != string(data) {
		t.Errorf("expected %v, got %v", data, hb.Data)
	}

	hb.Release()
}

func TestQPACKDecoder_NegativeCases(t *testing.T) {
	decoder := NewQPACKDecoder()
	_, err := decoder.DecodeHeaderBlock([]byte{0x00}) // Invalid prefix
	if err == nil {
		t.Error("expected error for invalid prefix, got nil")
	}
}

func TestQPACKDecoder_LargeHeader(t *testing.T) {
	decoder := NewQPACKDecoder()
	largeData := make([]byte, 20000)
	largeData[0] = 0x80

	hb, err := decoder.DecodeHeaderBlock(largeData)
	if err != nil {
		t.Fatalf("failed to decode large header: %v", err)
	}
	if len(hb.Data) != 20000 {
		t.Errorf("expected length 20000, got %d", len(hb.Data))
	}
	hb.Release()
}

func TestQPACKDecoder_ConcurrentAccess(t *testing.T) {
	decoder := NewQPACKDecoder()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hb, _ := decoder.DecodeHeaderBlock([]byte{0x80, 0x01})
			hb.Release()
		}()
	}
	wg.Wait()
}
