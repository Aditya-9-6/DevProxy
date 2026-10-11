package proxy

import (
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCoalescer(t *testing.T) {
	c := NewCoalescer()
	var callCount atomic.Int32

	req := &http.Request{Method: "GET", URL: &url.URL{Path: "/test"}}
	key := c.GetKey(req)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Do(key, func() (*http.Response, error) {
				callCount.Add(1)
				time.Sleep(50 * time.Millisecond)
				return &http.Response{StatusCode: 200}, nil
			})
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Errorf("Expected 1 call, got %d", callCount.Load())
	}
}
