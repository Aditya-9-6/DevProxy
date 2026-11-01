package http2guard

import (
	"sync"
	"sync/atomic"
	"time"
)

// Guard provides sharded, cache-line padded HTTP/2 abuse protection.
type GuardConfig struct {
	MaxRapidResets int64
	MaxSettings    int64
	RefillWindow   time.Duration
	ShardsCount    int
}

func DefaultGuardConfig() GuardConfig {
	return GuardConfig{
		MaxRapidResets: 100,
		MaxSettings:    50,
		RefillWindow:   time.Second,
		ShardsCount:    16,
	}
}

type shard struct {
	budgets map[string]*Budget
	mu      sync.Mutex
	_pad    [56]byte // Cache line padding simulation
}

type Guard struct {
	cfg    GuardConfig
	shards []shard
	active atomic.Int64
}

func NewGuard(cfg GuardConfig) *Guard{
	if cfg.ShardsCount <= 0 {
		cfg.ShardsCount = 16
	}
	shards := make([]shard, cfg.ShardsCount)
	for i := range shards {
		shards[i].budgets = make(map[string]*Budget)
	}
	return &Guard{
		cfg:    cfg,
		shards: shards,
	}
}

func (g *Guard) getShard(connID string) *shard {
	hash := uint32(0)
	for i := 0; i < len(connID); i++ {
		hash = hash*31 + uint32(connID[i])
	}
	return &g.shards[hash%uint32(len(g.shards))]
}

func (g *Guard) getOrCreateBudget(s *shard, connID string) *Budget {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, exists := s.budgets[connID]
	if !exists {
		b = NewBudget(g.cfg.MaxRapidResets, g.cfg.MaxSettings, g.cfg.RefillWindow)
		s.budgets[connID] = b
	}
	return b
}

func (g *Guard) CheckReset(connID string) Verdict {
	s := g.getShard(connID)
	b := g.getOrCreateBudget(s, connID)
	now := time.Now().UnixNano()
	if !b.ConsumeReset(now) {
		return VerdictRateLimited
	}
	return VerdictAllow
}

func (g *Guard) CheckSettings(connID string) Verdict {
	s := g.getShard(connID)
	b := g.getOrCreateBudget(s, connID)
	now := time.Now().UnixNano()
	if !b.ConsumeSettings(now) {
		return VerdictRateLimited
	}
	return VerdictAllow
}

func (g *Guard) Evict(connID string) {
	s := g.getShard(connID)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.budgets, connID)
}
