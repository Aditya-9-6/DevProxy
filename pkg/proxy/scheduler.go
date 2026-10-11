package proxy

import (
	"context"
	"net"
)

// ConnectionScheduler handles the migration of raw TCP connections to worker threads.
type ConnectionScheduler struct {
	shard *ConnectionShard
}

func NewConnectionScheduler(shard *ConnectionShard) *ConnectionScheduler {
	return &ConnectionScheduler{shard: shard}
}

// ScheduleConnection pins the connection handling to a specific worker core.
func (s *ConnectionScheduler) ScheduleConnection(ctx context.Context, conn net.Conn, handler func(net.Conn)) {
	dispatched := s.shard.Dispatch(func() {
		select {
		case <-ctx.Done():
			conn.Close()
		default:
			handler(conn)
		}
	})
	if !dispatched {
		conn.Close()
	}
}
