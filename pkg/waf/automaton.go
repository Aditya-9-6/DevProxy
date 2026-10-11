package waf

type ACNode struct {
	children map[byte]*ACNode
	fail     *ACNode
	outputs  []string
}

type Automaton struct {
	root *ACNode
}

func NewAutomaton(patterns []string) *Automaton {
	root := &ACNode{children: make(map[byte]*ACNode)}
	for _, p := range patterns {
		curr := root
		for i := 0; i < len(p); i++ {
			b := p[i]
			if curr.children[b] == nil {
				curr.children[b] = &ACNode{children: make(map[byte]*ACNode)}
			}
			curr = curr.children[b]
		}
		curr.outputs = append(curr.outputs, p)
	}
	queue := make([]*ACNode, 0)
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
			if f != nil {
				child.fail = f.children[b]
			} else {
				child.fail = root
			}
			if child.fail != nil {
				child.outputs = append(child.outputs, child.fail.outputs...)
			}
			queue = append(queue, child)
		}
	}
	return &Automaton{root: root}
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
