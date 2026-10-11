package waf

import "errors"

const MaxPatterns = 1000

type ACNode struct {
	children [256]*ACNode
	fail     *ACNode
	outputs  []string
}

type Automaton struct {
	root *ACNode
}

func NewAutomaton(patterns []string) (*Automaton, error) {
	if len(patterns) > MaxPatterns {
		return nil, errors.New("too many patterns")
	}
	root := &ACNode{}
	for _, p := range patterns {
		curr := root
		for i := 0; i < len(p); i++ {
			b := p[i]
			if curr.children[b] == nil {
				curr.children[b] = &ACNode{}
			}
			curr = curr.children[b]
		}
		curr.outputs = append(curr.outputs, p)
	}
	queue := make([]*ACNode, 0)
	for i := 0; i < 256; i++ {
		if root.children[i] != nil {
			root.children[i].fail = root
			queue = append(queue, root.children[i])
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for i := 0; i < 256; i++ {
			if child := curr.children[i]; child != nil {
				f := curr.fail
				for f != nil && f.children[i] == nil {
					f = f.fail
				}
				if f != nil {
					child.fail = f.children[i]
				} else {
					child.fail = root
				}
				child.outputs = append(child.outputs, child.fail.outputs...)
				queue = append(queue, child)
			}
		}
	}
	return &Automaton{root: root}, nil
}

func (a *Automaton) Scan(data []byte, callback func(match string)) {
	curr := a.root
	for _, b := range data {
		for curr != a.root && curr.children[b] == nil {
			curr = curr.fail
		}
		if curr.children[b] != nil {
			curr = curr.children[b]
			for _, out := range curr.outputs {
				callback(out)
			}
		}
	}
}
