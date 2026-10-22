package router

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

var benchmarkRoute Route

func TestSnapshotLongestPathAndTrustIsolation(t *testing.T) {
	policy, err := spiffe.NewPolicy(
		[]string{"example.org", "other.org"},
		[]spiffe.Route{
			{TrustDomain: "example.org", Path: "/api", Target: "api"},
			{TrustDomain: "example.org", Path: "/api/v1", Target: "v1"},
			{TrustDomain: "other.org", Path: "/api", Target: "other"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(policy)
	if err != nil {
		t.Fatal(err)
	}

	route, ok := router.Lookup(spiffe.ID{TrustDomain: "example.org", Path: "/api/v1/users"})
	if !ok || route.Target != "v1" {
		t.Fatalf("longest lookup = %+v, %v", route, ok)
	}
	if _, ok := router.Lookup(spiffe.ID{TrustDomain: "example.org.evil", Path: "/api/v1"}); ok {
		t.Fatal("lookalike trust domain was routed")
	}
	if _, ok := router.Lookup(spiffe.ID{TrustDomain: "other.org", Path: "/api"}); !ok {
		t.Fatal("other exact trust domain was not routed")
	}
}

func TestRouterPublishKeepsOldSnapshotImmutable(t *testing.T) {
	router, err := NewRouter(policyForTarget("old"))
	if err != nil {
		t.Fatal(err)
	}
	oldSnapshot := router.Current()
	if err := router.Publish(policyForTarget("new")); err != nil {
		t.Fatal(err)
	}

	oldRoute, oldOK := oldSnapshot.Lookup(spiffe.ID{TrustDomain: "example.org", Path: "/api"})
	newRoute, newOK := router.Lookup(spiffe.ID{TrustDomain: "example.org", Path: "/api"})
	if !oldOK || oldRoute.Target != "old" {
		t.Fatalf("old snapshot changed: %+v", oldRoute)
	}
	if !newOK || newRoute.Target != "new" {
		t.Fatalf("new snapshot not published: %+v", newRoute)
	}
}

func TestRouterInvalidPublishDoesNotReplaceSnapshot(t *testing.T) {
	router, err := NewRouter(policyForTarget("stable"))
	if err != nil {
		t.Fatal(err)
	}
	if err := router.Publish(nil); err == nil {
		t.Fatal("nil policy was accepted")
	}
	route, ok := router.Lookup(spiffe.ID{TrustDomain: "example.org", Path: "/api"})
	if !ok || route.Target != "stable" {
		t.Fatalf("failed publish replaced snapshot: %+v", route)
	}
}

func TestRouterConcurrentSnapshotUpdate(t *testing.T) {
	router, err := NewRouter(policyForTarget("old"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wait sync.WaitGroup
	var failed atomic.Bool

	for reader := 0; reader < 8; reader++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					route, ok := router.Lookup(spiffe.ID{TrustDomain: "example.org", Path: "/api"})
					if !ok || route.Target != "old" && route.Target != "new" {
						failed.Store(true)
						return
					}
				}
			}
		}()
	}

	for i := 0; i < 100; i++ {
		target := "old"
		if i%2 == 0 {
			target = "new"
		}
		if err := router.Publish(policyForTarget(target)); err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	wait.Wait()
	if failed.Load() {
		t.Fatal("reader observed a partially published snapshot")
	}
}

func TestRouterLookupAllocations(t *testing.T) {
	router, err := NewRouter(policyForTarget("backend"))
	if err != nil {
		t.Fatal(err)
	}
	identity := spiffe.ID{TrustDomain: "example.org", Path: "/api/v1"}
	allocs := testing.AllocsPerRun(1000, func() {
		route, ok := router.Lookup(identity)
		if !ok {
			panic("route not found")
		}
		benchmarkRoute = route
	})
	if allocs != 0 {
		t.Fatalf("Router.Lookup allocated %v objects, want 0", allocs)
	}
}

func BenchmarkRouterLookup(b *testing.B) {
	policy, err := spiffe.NewPolicy([]string{"example.org"}, []spiffe.Route{{TrustDomain: "example.org", Path: "/api/v1", Target: "backend"}})
	if err != nil {
		b.Fatal(err)
	}
	router, err := NewRouter(policy)
	if err != nil {
		b.Fatal(err)
	}
	identity := spiffe.ID{TrustDomain: "example.org", Path: "/api/v1"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchmarkRoute, _ = router.Lookup(identity)
	}
}

func policyForTarget(target string) *spiffe.Policy {
	policy, err := spiffe.NewPolicy(
		[]string{"example.org"},
		[]spiffe.Route{{TrustDomain: "example.org", Path: "/api", Target: target}},
	)
	if err != nil {
		panic(err)
	}
	return policy
}
