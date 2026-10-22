package proxy

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	wire "github.com/Aditya-9-6/DevProxy/pkg/proxy/proxyproto"
	security "github.com/Aditya-9-6/DevProxy/pkg/security/proxyproto"
)

func TestProxyProtocolListenerAcceptsValidatedFrame(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	listener := NewProxyProtocolListener(&oneShotListener{conn: serverConn}, security.DefaultPolicy())
	payload := []byte("GET / HTTP/1.1\r\nHost: example.test\r\n\r\n")
	frame := listenerTestFrame(t)
	wireBytes := make([]byte, 0, len(frame)+len(payload))
	wireBytes = append(wireBytes, frame...)
	wireBytes = append(wireBytes, payload...)

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- writeProxyProtocolTestBytes(clientConn, wireBytes)
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	metadata, ok := ProxyProtocolFrameFromConn(conn)
	if !ok {
		t.Fatal("ProxyProtocolFrameFromConn() did not find metadata")
	}
	if !bytes.Equal(metadata.SourceIPBytes(), []byte{192, 0, 2, 10}) {
		t.Fatalf("source = %v", metadata.SourceIPBytes())
	}
	if metadata.SourcePort != 12345 || metadata.DestinationPort != 443 {
		t.Fatalf("ports = %d/%d", metadata.SourcePort, metadata.DestinationPort)
	}

	received := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, received); err != nil {
		t.Fatalf("ReadFull() error = %v", err)
	}
	if !bytes.Equal(received, payload) {
		t.Fatalf("payload = %q, want %q", received, payload)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("client Write() error = %v", err)
	}
}

func TestProxyProtocolListenerRejectsInvalidFrames(t *testing.T) {
	t.Parallel()

	valid := listenerTestFrame(t)
	badChecksum := append([]byte(nil), valid...)
	badChecksum[len(badChecksum)-1] ^= 0xff

	tests := []struct {
		name  string
		frame []byte
	}{
		{name: "checksum mismatch", frame: badChecksum},
		{name: "policy rejects tlv", frame: listenerTestFrameWithTLV(t)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertProxyProtocolFrameRejected(t, test.frame)
		})
	}
}

func TestProxyProtocolListenerReadTimeout(t *testing.T) {
	t.Parallel()

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	listener := NewProxyProtocolListenerWithConfig(&oneShotListener{conn: serverConn}, ProxyProtocolListenerConfig{
		ReadTimeout: 20 * time.Millisecond,
	})
	_, err := listener.Accept()
	if err == nil {
		t.Fatal("Accept() unexpectedly succeeded")
	}

	var acceptError *ProxyProtocolAcceptError
	if !errors.As(err, &acceptError) || !acceptError.Timeout() {
		t.Fatalf("Accept() error = %v, want timeout", err)
	}
}

func assertProxyProtocolFrameRejected(t *testing.T, frame []byte) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	listener := NewProxyProtocolListener(&oneShotListener{conn: serverConn}, security.DefaultPolicy())
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- writeProxyProtocolTestBytes(clientConn, frame)
	}()

	conn, err := listener.Accept()
	if err == nil {
		_ = conn.Close()
		t.Fatal("Accept() unexpectedly succeeded")
	}

	var acceptError *ProxyProtocolAcceptError
	if !errors.As(err, &acceptError) {
		t.Fatalf("Accept() error type = %T, want *ProxyProtocolAcceptError", err)
	}
	if !acceptError.Temporary() {
		t.Fatal("ProxyProtocolAcceptError.Temporary() = false")
	}

	select {
	case writeErr := <-writeDone:
		if writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) && !errors.Is(writeErr, net.ErrClosed) {
			t.Fatalf("client Write() error = %v", writeErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("client Write() did not finish")
	}
}

func writeProxyProtocolTestBytes(conn net.Conn, payload []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return err
	}

	for len(payload) > 0 {
		written, err := conn.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrUnexpectedEOF
		}
		payload = payload[written:]
	}
	return nil
}

func listenerTestFrame(t *testing.T) []byte {
	t.Helper()
	return encodeListenerTestFrame(t, listenerTestFrameValue())
}

func listenerTestFrameWithTLV(t *testing.T) []byte {
	t.Helper()
	frame := listenerTestFrameValue()
	frame.TLVData = []byte{0x02, 0x00, 0x02, 'h', '2'}
	return encodeListenerTestFrame(t, frame)
}

func listenerTestFrameValue() wire.Frame {
	return wire.Frame{
		Command:            wire.CommandProxy,
		Family:             wire.AddressFamilyINET,
		Transport:          wire.TransportStream,
		SourceAddress:      [16]byte{192, 0, 2, 10},
		DestinationAddress: [16]byte{198, 51, 100, 20},
		SourcePort:         12345,
		DestinationPort:    443,
	}
}

func encodeListenerTestFrame(t *testing.T, frame wire.Frame) []byte {
	t.Helper()
	buffer := make([]byte, wire.MaxFrameSize)
	written, err := wire.Encode(buffer, frame)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return buffer[:written]
}

type oneShotListener struct {
	mu   sync.Mutex
	conn net.Conn
}

func (l *oneShotListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return nil, net.ErrClosed
	}
	conn := l.conn
	l.conn = nil
	return conn, nil
}

func (l *oneShotListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return nil
	}
	err := l.conn.Close()
	l.conn = nil
	return err
}

func (l *oneShotListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
}
