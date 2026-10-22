package router

import (
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

func TestPathTrieLongestMatch(t *testing.T) {
	trie := newPathTrie()
	if err := trie.insert(spiffe.Route{Path: "/api", Target: "api"}); err != nil {
		t.Fatal(err)
	}
	if err := trie.insert(spiffe.Route{Path: "/api/v1", Target: "v1"}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		wantTarget string
		wantOK     bool
	}{
		{name: "exact", path: "/api", wantTarget: "api", wantOK: true},
		{name: "descendant", path: "/api/users", wantTarget: "api", wantOK: true},
		{name: "longest", path: "/api/v1/users", wantTarget: "v1", wantOK: true},
		{name: "prefix boundary", path: "/apix", wantOK: false},
		{name: "other prefix", path: "/other", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			route, ok := trie.lookup(test.path)
			if ok != test.wantOK {
				t.Fatalf("lookup(%q) ok = %v, want %v", test.path, ok, test.wantOK)
			}
			if ok && route.Target != test.wantTarget {
				t.Fatalf("lookup(%q) target = %q, want %q", test.path, route.Target, test.wantTarget)
			}
		})
	}
}

func TestPathTrieRootRoute(t *testing.T) {
	trie := newPathTrie()
	if err := trie.insert(spiffe.Route{Path: "/", Target: "root"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/anything", "/nested/path"} {
		route, ok := trie.lookup(path)
		if !ok || route.Target != "root" {
			t.Fatalf("lookup(%q) = %+v, %v", path, route, ok)
		}
	}
}

func TestPathTrieRejectsDuplicate(t *testing.T) {
	trie := newPathTrie()
	if err := trie.insert(spiffe.Route{Path: "/api"}); err != nil {
		t.Fatal(err)
	}
	if err := trie.insert(spiffe.Route{Path: "/api"}); err == nil {
		t.Fatal("duplicate route was accepted")
	}
}

func TestPathTrieLookupAllocations(t *testing.T) {
	trie := newPathTrie()
	if err := trie.insert(spiffe.Route{Path: "/api/v1", Target: "v1"}); err != nil {
		t.Fatal(err)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		if _, ok := trie.lookup("/api/v1/users"); !ok {
			panic("route did not match")
		}
	})
	if allocs != 0 {
		t.Fatalf("pathTrie.lookup allocated %v objects, want 0", allocs)
	}
}
