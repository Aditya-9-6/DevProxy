package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
)

// Coalescer manages concurrent identical requests to prevent redundant upstream traffic.
type Coalescer struct {
	flights sync.Map
	pool    sync.Pool
}

type flight struct {
	response *http.Response
	err      error
	wg       sync.WaitGroup
}

func NewCoalescer() *Coalescer {
	return &Coalescer{
		pool: sync.Pool{
			New: func() interface{} { return &flight{} },
		},
	}
}

func (c *Coalescer) GetKey(req *http.Request) string {
	h := sha256.New()
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.String()))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Coalescer) Do(key string, fn func() (*http.Response, error)) (*http.Response, error) {
	f := c.pool.Get().(*flight)
	f.wg.Add(1)

	actual, loaded := c.flights.LoadOrStore(key, f)
	if loaded {
		c.pool.Put(f)
		existing := actual.(*flight)
		existing.wg.Wait()
		return existing.response, existing.err
	}

	defer func() {
		c.flights.Delete(key)
		f.wg.Done()
		c.pool.Put(f)
	}()

	f.response, f.err = fn()
	return f.response, f.err
}
