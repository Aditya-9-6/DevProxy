package proxy

import (
	"sync"
	"testing"
)

func TestConnectionShard(t *testing.T) {
	shard := NewConnectionShard(2)
	defer shard.Close()

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		shard.Dispatch(func() {
			wg.Done()
		})
	}
	wg.Wait()
}
