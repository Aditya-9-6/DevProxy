package replay

import (
	"context"
	"net/http"
	"sync"
	"testing"
)

func TestSanitizer(t *testing.T) {
	t.Parallel()
	s := NewSanitizer()
	body := []byte("password=12345&user=admin")
	sanitized := s.SanitizeBody(body)
	if string(sanitized) != "password=[REDACTED]&user=admin" {
		t.Errorf("Expected redaction, got %s", string(sanitized))
	}
}

func TestRingBuffer(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(1024)
	success := rb.Push("test")
	if !success {
		t.Error("Failed to push to ring buffer")
	}
	val := rb.GetBufferForTest(0)
	if val != "test" {
		t.Errorf("Expected 'test', got %v", val)
	}
}

func TestRecorderConcurrent(t *testing.T) {
	t.Parallel()
	rb := NewRingBuffer(256)
	recorder := NewRecorder(rb)
	ctx := context.Background()

	var wg sync.WaitGroup
	workers := 10
	its := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < its; j++ {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				req.Header.Set("Authorization", "Bearer secrettoken")
				recorder.Record(ctx, req, []byte("password=secret"))
			}
		}(i)
	}
	wg.Wait()
}
