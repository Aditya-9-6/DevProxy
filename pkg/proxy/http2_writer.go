package proxy

import (
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/http2flow"
)

// HTTP2Writer handles high performance HTTP/2 writing with flow control.
type HTTP2Writer struct {
	ledger *http2flow.Ledger
}

// NewHTTP2Writer creates a new HTTP2Writer.
func NewHTTP2Writer(ledger *http2flow.Ledger) *HTTP2Writer {
	return &HTTP2Writer{
		ledger: ledger,
	}
}
