package relay

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestFullDuplexRelay(t *testing.T) {
	c1, s1 := net.Pipe()
	defer c1.Close()
	defer s1.Close()

	c2, s2 := net.Pipe()
	defer c2.Close()
	defer s2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	relayDone := make(chan error, 1)
	go func() {
		relayDone <- FullDuplexRelay(ctx, s1, c2, Config{IdleTimeout: 1 * time.Second})
	}()

	msg := []byte("hello proxy")
	go func() {
		_, _ = c1.Write(msg)
	}()

	buf := make([]byte, 64)
	n, err := s2.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if string(buf[:n]) != "hello proxy" {
		t.Fatalf("expected 'hello proxy', got '%s'", string(buf[:n]))
	}

	cancel()
	<-relayDone
}

func TestDuplexStateTermination(t *testing.T) {
	st := NewDuplexState()
	if st.IsTerminated() {
		t.Fatal("expected not terminated initially")
	}

	st.SetHalfClosed(true)
	if st.IsTerminated() {
		t.Fatal("expected not terminated after one direction")
	}

	st.SetHalfClosed(false)
	if !st.IsTerminated() {
		t.Fatal("expected terminated after both directions")
	}
}
