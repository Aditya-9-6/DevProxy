package storage

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// ARCCache implements an Adaptive Replacement Cache policy.
// It maintains two lists (T1: Recency, T2: Frequency) and two ghost lists (B1, B2).
type ARCCache struct {
	capacity int64
	p        int64
	mu       sync.Mutex

	t1, t2 *list.List
	b1, b2 *list.List
	items  map[interface{}]*entry

	hits   atomic.Uint64
	misses atomic.Uint64
}

type entry struct {
	key, value interface{}
	list       *list.List
	elem       *list.Element
}

func NewARCCache(capacity int) *ARCCache {
	return &ARCCache{
		capacity: int64(capacity),
		t1:       list.New(),
		t2:       list.New(),
		b1:       list.New(),
		b2:       list.New(),
		items:    make(map[interface{}]*entry),
	}
}

func (c *ARCCache) Get(key interface{}) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		c.hits.Add(1)
		c.moveToT2(e)
		return e.value, true
	}
	c.misses.Add(1)
	return nil, false
}

func (c *ARCCache) Set(key, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.items[key]; ok {
		e.value = value
		c.moveToT2(e)
		return
	}

	e := &entry{key: key, value: value}
	c.items[key] = e
	c.t1.PushFront(e)
	e.list = c.t1
	e.elem = c.t1.Front()

	if int64(c.t1.Len()+c.t2.Len()) > c.capacity {
		c.evict()
	}
}

func (c *ARCCache) moveToT2(e *entry) {
	e.list.Remove(e.elem)
	c.t2.PushFront(e)
	e.list = c.t2
	e.elem = c.t2.Front()
}

func (c *ARCCache) evict() {
	if c.t1.Len() > 0 {
		e := c.t1.Back()
		entry := e.Value.(*entry)
		c.t1.Remove(e)
		delete(c.items, entry.key)
	}
}
