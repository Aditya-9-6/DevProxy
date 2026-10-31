package router

import (
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

func TestTrieBoundaryAndMalformedPaths(t *testing.T) {
	root := NewNode()

	// Insert IDs with empty components, trailing slashes, and complex paths
	id1, _ := spiffe.ParseID("spiffe://example.org///service//a/")
	id2, _ := spiffe.ParseID("spiffe://example.org/")
	id3, _ := spiffe.ParseID("spiffe://example.org/service/b")

	root.Insert(id1)
	root.Insert(id2)
	root.Insert(id3)

	// Test matching against malformed/edge paths
	if !root.Match(id1) {
		T.Errorf("expected match for id1")
	}

	if !root.Match(id2) {
		t.Errorf("expected match for id2")
	}

	if !root.Match(id3) {
		t.Errorf("expected match for id3")
	}

	// Test non-matching path
	badID, _ := spiffe.ParseID("spiffe://example.org/nonexistent")
	if root.Match(badID) {
		t.Errorf("expected no match for badID")
	}
}
