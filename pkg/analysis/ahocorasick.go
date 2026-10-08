package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"sync"
)

type node struct {
	children map[byte]*node
	fail     *node
	pattern  string
}

type SecretScanner struct {
	root *node
	mu   sync.RWMutex
}

func NewSecretScanner(patterns []string) *SecretScanner {
	root := &node{children: make(map[byte]*node)}
	for _, p := range patterns {
		curr := root
		for i := 0; i < len(p); i++ {
			if _, ok := curr.children[p[i]]; !ok {
				curr.children[p[i]] = &node{children: make(map[byte]*node)}
			}
			curr = curr.children[p[i]]
		}
		curr.pattern = p
	}
	queue := make([]*node, 0)
	for _, child := range root.children {
		child.fail = root
		queue = append(queue, child)
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for b, child := range curr.children {
			f := curr.fail
			for f != nil && f.children[b] == nil {
				f = f.fail
			}
			if f == nil {
				child.fail = root
			} else {
				child.fail = f.children[b]
			}
			queue = append(queue, child)
		}
	}
	return &SecretScanner{root: root}
}

func (s *SecretScanner) Name() string { return "SecretScanner" }

func (s *SecretScanner) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	data := append(event.ReqBody, event.RespBody...)
	curr := s.root
	for _, b := range data {
		for curr != s.root && curr.children[b] == nil {
			curr = curr.fail
		}
		if next, ok := curr.children[b]; ok {
			curr = next
		}
		if curr.pattern != "" {
			findings = append(findings, &Finding{Title: "Secret Detected", RuleName: s.Name(), Severity: SeverityCritical, Description: "Found pattern: " + curr.pattern})
		}
	}
	return findings
}
