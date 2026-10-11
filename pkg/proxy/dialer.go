package proxy

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/dns"
)

type HappyEyeballsDialer struct {
	resolver *dns.Resolver
	pool     sync.Pool
}

func NewHappyEyeballsDialer() *HappyEyeballsDialer {
	return &HappyEyeballsDialer{
		resolver: dns.NewResolver(),
		pool: sync.Pool{
			New: func() interface{} { return new(net.Dialer) },
		},
	}
}

func (d *HappyEyeballsDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}

	addrs, err := d.resolver.Resolve(ctx, host)
	if err != nil {
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	connChan := make(chan net.Conn, len(addrs))

	// Concurrency limiter to prevent FD exhaustion
	sem := make(chan struct{}, 8)

	for _, addr := range addrs {
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			dialer := d.pool.Get().(*net.Dialer)
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
			d.pool.Put(dialer)

			if err == nil {
				select {
				case connChan <- conn:
				default:
					conn.Close()
				}
			}
		}(addr)
	}

	go func() {
		wg.Wait()
		close(connChan)
	}()

	var establishedConns []net.Conn
	defer func() {
		for _, c := range establishedConns {
			c.Close()
		}
	}()

	for {
		select {
		case conn, ok := <-connChan:
			if !ok {
				return nil, &net.OpError{Op: "dial", Net: network, Source: nil, Addr: nil, Err: context.DeadlineExceeded}
			}
			// Return the first successful connection and buffer/close others
			for extra := range connChan {
				extra.Close()
			}
			return conn, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
