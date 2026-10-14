package cache

// Node represents a cache entry node with frequency tracking.
type Node struct {
	Key       string
	Value     interface{}
	Frequency uint64
	Next      *Node
	Prev      *Node
}

// NewNode creates a new cache node.
func NewNode(key string, value interface{}) *Node {
	return &Node{Key: key, Value: value, Frequency: 1}
}
