package proxy

import (
	"bufio"
	"github.com/Aditya-9-6/DevProxy/pkg/httpwire"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

// HTTP1FramingAdapter adapts raw stream parsing with security smuggling inspection.
type HTTP1FramingAdapter struct {
	Policy *security.SmugglingPolicy
}

// NewHTTP1FramingAdapter creates a new adapter.
func NewHTTP1FramingAdapter(policy *security.SmugglingPolicy) *HTTP1FramingAdapter {
	return &HTTP1FramingAdapter{
		Policy: policy,
	}
}

// AdaptRequest parses and validates an incoming HTTP/1.1 request buffer.
func (a *HTTP1FramingAdapter) AdaptRequest(r *bufio.Reader) (*httpwire.Frame, error) {
	frame, err := httpwire.ParseRequestFraming(r)
	if err != nil {
		return nil, err
	}

	if a.Policy != nil {
		err = a.Policy.Inspect(frame)
		if err != nil {
			return nil, err
		}
	}

	return frame, nil
}
