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
	data := []byte("test-header")

	hb, err := decoder.DecodeHeaderBlock(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(hb.Data) != string(data) {
		t.Errorf("expected %s, got %s", data, hb.Data)
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
			hb, _ := decoder.DecodeHeaderBlock([]byte("data"))
			hb.Release()
		}()
	}
	wg.Wait()
}
