package proxy

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"time"

	wire "github.com/Aditya-9-6/DevProxy/pkg/proxy/proxyproto"
	security "github.com/Aditya-9-6/DevProxy/pkg/security/proxyproto"
)

const defaultProxyProtocolReadTimeout = 5 * time.Second

var proxyProtocolFramePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, wire.MaxFrameSize)
	},
}

// ErrProxyProtocolUnavailable indicates that a listener was not configured.
var ErrProxyProtocolUnavailable = errors.New("proxy: PROXY protocol listener unavailable")

// ProxyProtocolListenerConfig configures ingress validation.
type ProxyProtocolListenerConfig struct {
	// Policy is the immutable security policy. A nil policy selects the strict default.
	Policy *security.Policy
	// ReadTimeout bounds a malicious or stalled peer during frame negotiation.
	// A negative value disables the deadline.
	ReadTimeout time.Duration
}

// ProxyProtocolListener validates a PROXY protocol v2 frame before exposing
// the accepted connection to the HTTP server.
type ProxyProtocolListener struct {
	net.Listener
	policy      *security.Policy
	readTimeout time.Duration
}

// NewProxyProtocolListener wraps listener with strict PROXY protocol v2 ingress validation.
func NewProxyProtocolListener(listener net.Listener, policy *security.Policy) *ProxyProtocolListener {
	return NewProxyProtocolListenerWithConfig(listener, ProxyProtocolListenerConfig{
		Policy:      policy,
		ReadTimeout: defaultProxyProtocolReadTimeout,
	})
}

// NewProxyProtocolListenerWithConfig wraps listener with configured validation.
func NewProxyProtocolListenerWithConfig(listener net.Listener, config ProxyProtocolListenerConfig) *ProxyProtocolListener {
	if config.Policy == nil {
		config.Policy = security.DefaultPolicy()
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = defaultProxyProtocolReadTimeout
	}
	return &ProxyProtocolListener{
		Listener:    listener,
		policy:      config.Policy,
		readTimeout: config.ReadTimeout,
	}
}

// ListenProxyProtocol creates a TCP listener with PROXY protocol v2 validation.
func ListenProxyProtocol(network, address string, policy *security.Policy) (net.Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return NewProxyProtocolListener(listener, policy), nil
}

// ProxyProtocolConn carries validated connection metadata.
type ProxyProtocolConn struct {
	net.Conn
	frame wire.Frame
}

// ProxyProtocolFrame returns a copy of the validated frame metadata.
func (c *ProxyProtocolConn) ProxyProtocolFrame() wire.Frame {
	if c == nil {
		return wire.Frame{}
	}
	return c.frame
}

// ProxyProtocolFrameFromConn returns validated metadata when conn originated
// from a ProxyProtocolListener.
func ProxyProtocolFrameFromConn(conn net.Conn) (wire.Frame, bool) {
	provider, ok := conn.(interface{ ProxyProtocolFrame() wire.Frame })
	if !ok {
		return wire.Frame{}, false
	}
	return provider.ProxyProtocolFrame(), true
}

// Accept accepts and validates one PROXY protocol v2 connection.
func (l *ProxyProtocolListener) Accept() (net.Conn, error) {
	if l == nil || l.Listener == nil {
		return nil, ErrProxyProtocolUnavailable
	}
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	frame, err := l.readAndValidateFrame(conn)
	if err != nil {
		_ = conn.Close()
		return nil, &ProxyProtocolAcceptError{Err: err}
	}
	return &ProxyProtocolConn{Conn: conn, frame: frame}, nil
}

func (l *ProxyProtocolListener) readAndValidateFrame(conn net.Conn) (wire.Frame, error) {
	buffer := proxyProtocolFramePool.Get().([]byte)
	defer proxyProtocolFramePool.Put(buffer)
	if l.readTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(l.readTimeout)); err != nil {
			return wire.Frame{}, err
		}
		defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
	}
	decoder := wire.StreamDecoder{}
	frame, err := decoder.Read(context.Background(), conn, buffer)
	if err != nil {
		return wire.Frame{}, err
	}
	policy := l.policy
	if policy == nil {
		policy = security.DefaultPolicy()
	}
	if err := policy.Validate(frame); err != nil {
		return wire.Frame{}, err
	}
	frame.TLVData = nil
	return frame, nil
}

// ProxyProtocolAcceptError reports a rejected PROXY protocol frame.
type ProxyProtocolAcceptError struct {
	Err error
}

// Error implements error.
func (e *ProxyProtocolAcceptError) Error() string {
	if e == nil || e.Err == nil {
		return "proxy: invalid PROXY protocol frame"
	}
	return "proxy: invalid PROXY protocol frame: " + e.Err.Error()
}

// Unwrap exposes the underlying validation error.
func (e *ProxyProtocolAcceptError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Temporary reports that the listener can continue accepting other clients.
func (e *ProxyProtocolAcceptError) Temporary() bool {
	return true
}

// Timeout reports whether rejection was caused by a read deadline.
func (e *ProxyProtocolAcceptError) Timeout() bool {
	return e != nil && errors.Is(e.Err, os.ErrDeadlineExceeded)
}
