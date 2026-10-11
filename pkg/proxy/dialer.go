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
	host, port, _ := net.SplitHostPort(address)
	addrs, err := d.resolver.Resolve(ctx, host)
	if err != nil {
		return net.Dial(network, address)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	connChan := make(chan net.Conn, len(addrs))

	for _, addr := range addrs {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
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

	select {
	case conn := <-connChan:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
