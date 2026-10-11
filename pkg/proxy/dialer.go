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
}

func NewHappyEyeballsDialer() *HappyEyeballsDialer {
	return &HappyEyeballsDialer{
		resolver: dns.NewResolver(),
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

	return d.raceDials(ctx, network, addrs, port)
}

func (d *HappyEyeballsDialer) raceDials(ctx context.Context, network string, addrs []string, port string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	connChan := make(chan net.Conn, len(addrs))
	var wg sync.WaitGroup
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

			dialer := &net.Dialer{}
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
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

	select {
	case conn, ok := <-connChan:
		if !ok {
			return nil, &net.OpError{Op: "dial", Net: network, Err: context.DeadlineExceeded}
		}
		go func() {
			for extra := range connChan {
				extra.Close()
			}
		}()
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
