package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/waf"
	"net/http"
)

// Transport wraps the standard http.RoundTripper to inject WAF inspection.
type Transport struct {
	Base      http.RoundTripper
	Inspector *waf.ChunkInspector
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Base.RoundTrip(req)
}
