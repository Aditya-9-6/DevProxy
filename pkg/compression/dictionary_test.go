package compression

import (
	"bytes"
	"testing"
	"time"
)

func TestDictionaryManagerAndCompression(t *testing.T) {
	mgr := GetGlobalManager()
	mgr.RecordLatency(CompressionGzip, 50*time.Microsecond)

	algo := mgr.SelectAlgorithm("application/json", 1024)
	if algo == "" {
		data := []byte("hello world test stream payload entropy check")
		e := ComputeEntropy(data)
		if e < 0 {
			t.Errorf("invalid entropy %f", e)
		}
	}

	var buf bytes.Buffer
	wc, err := CompressStream(&buf, CompressionGzip)
	if err != nil {
		Fatalf(t, "failed to create compressor: %v", err)
	}
	_, err = wc.Write([]byte("test compression streaming data"))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := wc.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	buf2 := AcquireBuffer()
	ReleaseBuffer(buf2)
}

func Fatalf(t *testing.T, format string, args ...interface{}) {
	t.Fatalf(format, args...)
}
