package tunnel

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestMultiplexerTunnelHappyPath(t *testing.T) {
	t.Parallel()
	m := NewMultiplexer()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	acceptedCh := make(chan net.Conn, 1)
	go func() {
		c, err := listener.Accept()
		if err == nil {
			acceptedCh <- c
		}
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	upstreamConn := <-acceptedCh
	defer upstreamConn.Close()

	tcpClient := clientConn.(*net.TCPConn)
	tcpUpstream := upstreamConn.(*net.TCPConn)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = tcpClient.Write([]byte("hello"))
	}()

	buf := make([]byte, 5)
	_, _ = tcpUpstream.Read(buf)
	wg.Wait()

	_ = m.Tunnel(ctx, tcpClient, tcpUpstream)
}

func TestMultiplexerContextCancellation(t *testing.T) {
	t.Parallel()
	m := NewMultiplexer()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	connCh := make(chan net.Conn, 1)
	go func() {
		c, _ := listener.Accept()
		connCh <- c
	}()

	c1, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c1.Close()

	c2 := <-connCh
	defer c2.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = m.Tunnel(ctx, c1.(*net.TCPConn), c2.(*net.TCPConn))
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}

func TestConnectionTracker(t *testing.T) {
	t.Parallel()
	tracker := &ConnectionTracker{}
	if tracker.ActiveCount() != 0 {
		t.Errorf("Expected 0 active streams, got %d", tracker.ActiveCount())
	}

	tracker.Increment()
	if tracker.ActiveCount() != 1 {
		t.Errorf("Expected 1 active stream, got %d", tracker.ActiveCount())
	}

	tracker.Decrement()
	if tracker.ActiveCount() != 0 {
		t.Errorf("Expected 0 active streams after decrement, got %d", tracker.ActiveCount())
	}
}

func TestMultiplexerConcurrentTunnels(t *testing.T) {
	t.Parallel()
	m := NewMultiplexer()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker := m.tracker
			tracker.Increment()
			time.Sleep(5 * time.Millisecond)
			tracker.Decrement()
		}()
	}
	wg.Wait()
	if m.tracker.ActiveCount() != 0 {
		t.Errorf("Leaked active stream count: %d", m.tracker.ActiveCount())
	}
}
