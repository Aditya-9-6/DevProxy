package http2guard

import (
	"sync"
	"testing"
	"time"
)

func TestVerdictString(t *testing.T) {
	if VerdictAllow.String() != "ALLOW" {
		t.Errorf("expected ALLOW, got %s", VerdictAllow.String())
	}
	if VerdictRateLimited.String() != "RATE_LIMITED" {
		t.Errorf("expected RATE_LIMITED, got %s", VerdictRateLimited.String())
	}
	if VerdictBlocked.String() != "BLOCKED" {
		t.Errorf("expected BLOCKED, got %s", VerdictBlocked.String())
	}
	if Verdict(99).String() != "UNKNOWN" {
		t.Errorf("expected UNKNOWN, got %s", Verdict(99).String())
	}
}

func TestGuardRapidResetAndSettings(t *testing.T) {
	cfg := GuardConfig{
		MaxRapidResets: 2,
		MaxSettings:    2,
		RefillWindow:   100 * time.Millisecond,
		ShardsCount:    4,
	}
	guard := NewGuard(cfg)

	conn := "conn-123"

	if v := guard.CheckReset(conn); v != VerdictAllow {
		t.Errorf("expected ALLOW, got %v", v)
	}
	if v := guard.CheckReset(conn); v != VerdictAllow {
		t.Errorf("expected ALLOW, got %v", v)
	}
	if v := guard.CheckReset(conn); v != VerdictRateLimited {
		t.Errorf("expected RATE_LIMITED, got %v", v)
	}

	if v := guard.CheckSettings(conn); v != VerdictAllow {
		t.Errorf("expected ALLOW, got %v", v)
	}
	if v := guard.CheckSettings(conn); v != VerdictAllow {
		t.Errorf("expected ALLOW, got %v", v)
	}
	if v := guard.CheckSettings(conn); v != VerdictRateLimited {
		t.Errorf("expected RATE_LIMITED, got %v", v)
	}

	// Wait for refill
	time.Sleep(120 * time.Millisecond)
	if v := guard.CheckReset(conn); v != VerdictAllow {
		t.Errorf("expected ALLOW after refill, got %v", v)
	}

	guard.Evict(conn)
}

func TestGuardConcurrency(t *testing.T) {
	cfg := DefaultGuardConfig()
	guard := NewGuard(cfg)
	var wg sync.WaitGroup
	workers := 10
	its := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := "conn-parallel"
			for j := 0; j < its; j++ {
				guard.CheckReset(conn)
				guard.CheckSettings(conn)
			}
		}(i)
	}
	wg.Wait()
}
