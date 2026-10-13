package storage

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// ARCCache implements an Adaptive Replacement Cache policy.
// It maintains two lists (T1: Recency, T2: Frequency) and two ghost lists (B1, B2).
type ARCCache struct {
	oc       int64
	capacity int64
	p        int64
	mu       sync.Mutex

	t1, t2 *list.List
	b1, b2 *list.List
	items  map[interface{}]*entry
	ghosts map[interface{}]*entry

	hits   atomic.Uint64
	misses atomic.Uint64

	pool sync.Pool
}

type entry struct {
	key, value interface{}
	list       *list.List
	elem       *list.Element
	isGhost    bool
}

func NewARCCache(capacity int) *ARCCache {
	cap := int64(capacity)
	if cap < 1 {
		cap = 1
	}
	return &ARCCache{
		oc:       cap,
		capacity: cap,
		p:        0,
		t1:       list.New(),
		t2:       list.New(),
		b1:       list.New(),
		b2:       list.New(),
		items:    make(map[interface{}]*entry),
		ghosts:   make(map[interface{}]*entry),
		pool: sync.Pool{
			New: func() interface{} {
				return &entry{}
			},
		},
	}
}

func (c *ARCCache) Get(key interface{}) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Case I: Key is in T1 or T2 (Cache Hit)
	if e, ok := c.items[key]; ok {
		c.hits.Add(1)
		// Move to front of T2
		e.list.Remove(e.elem)
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		return e.value, true
	}

	// Case II: Key is in B1 (Ghost Recency Hit)
	if e, ok := c.ghosts[key]; ok && e.isGhost && e.list == c.b1 {
		c.misses.Add(1)
		// Adapt p upwards
		delta := int64(1)
		if c.b1.Len() > 0 && c.b2.Len() > 0 {
			// delta = max(1, b2.Len() / b1.Len())
			d2 := int64(c.b2.Len() / c.b1.Len())
			if d2 > 1 {
				delta = d2
			}
		}
		c.p = min(c.p+delta, c.capacity)

		c.replace(false)

		// Move from B1 to T2
		delete(c.ghosts, key)
		e.list.Remove(e.elem)
		e.isGhost = false
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		c.items[key] = e
		return e.value, false // Found in ghost, treated as miss for value retrieval but cache state updated
	}

	// Case III: Key is in B2 (Ghost Frequency Hit)
	if e, ok := c.ghosts[key]; ok && e.isGhost && e.list == c.b2 {
		c.misses.Add(1)
		// Adapt p downwards
		delta := int64(1)
		if c.b1.Len() > 0 && c.b2.Len() > 0 {
			d1 := int64(c.b1.Len() / c.b2.Len())
			if d1 > 1 {
				delta = d1
			}
		}
		if c.p > delta {
			c.p -= delta
		} else {
			c.p = 0
		}

		c.replace(true)

		// Move from B2 to T2
		delete(c.ghosts, key)
		e.list.Remove(e.elem)
		e.isGhost = false
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		c.items[key] = e
		return e.value, false
	}

	c.misses.Add(1)
	return nil, false
}

func (c *ARCCache) Set(key, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Case I: Key is already in T1 or T2
	if e, ok := c.items[key]; ok {
		e.value = value
		e.list.Remove(e.elem)
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		return
	}

	// Case II: Key is in B1
	if e, ok := c.ghosts[key]; ok && e.isGhost && e.list == c.b1 {
		delta := int64(1)
		if c.b1.Len() > 0 && c.b2.Len() > 0 {
			d2 := int64(c.b2.Len() / c.b1.Len())
			if d2 > 1 {
				delta = d2
			}
		}
		c.p = min(c.p+delta, c.capacity)

		c.replace(false)

		delete(c.ghosts, key)
		e.list.Remove(e.elem)
		e.value = value
		e.isGhost = false
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		c.items[key] = e
		return
	}

	// Case III: Key is in B2
	if e, ok := c.ghosts[key]; ok && e.isGhost && e.list == c.b2 {
		delta := int64(1)
		if c.b1.Len() > 0 && c.b2.Len() > 0 {
			d1 := int64(c.b1.Len() / c.b2.Len())
			if d1 > 1 {
				delta = d1
			}
		}
		if c.p > delta {
			c.p -= delta
		} else {
			c.p = 0
		}

		c.replace(true)

		delete(c.ghosts, key)
		e.list.Remove(e.elem)
		e.value = value
		e.isGhost = false
		e.list = c.t2
		e.elem = c.t2.PushFront(e)
		c.items[key] = e
		return
	}

	// Case IV: Completely new item
	// L1 size (T1 + B1) = capacity
	if int64(c.t1.Len()+c.b1.Len()) == c.capacity {
		if int64(c.t1.Len()) < c.capacity {
			if c.b1.Len() > 0 {
				oldest := c.b1.Back()
				if oldest != nil {
					ae := oldest.Value.(*entry)
					c.b1.Remove(oldest)
					delete(c.ghosts, ae.key)
					c.pool.Put(ae)
				}
			}
			c.replace(false)
		} else {
			oldest := c.t1.Back()
			if oldest != nil {
				ae := oldest.Value.(*entry)
				c.t1.Remove(oldest)
				delete(c.items, ae.key)
				ae.isGhost = true
				ae.list = c.b1
				ae.elem = c.b1.PushFront(ae)
				c.ghosts[ae.key] = ae
			}
		}
	} else if int64(c.t1.Len()+c.t2.Len()+c.b1.Len()+c.b2.Len()) >= c.capacity {
		totalLen := int64(c.t1.Len() + c.t2.Len() + c.b1.Len() + c.b2.Len())
		if totalLen >= 2*c.capacity {
			if c.b2.Len() > 0 {
				oldest := c.b2.Back()
				if oldest != nil {
					ae := oldest.Value.(*entry)
					c.b2.Remove(oldest)
					delete(c.ghosts, ae.key)
					c.pool.Put(ae)
				}
			}
		}
		c.replace(false)
	}

	e := c.pool.Get().(*entry)
	e.key = key
	e.value = value
	e.isGhost = false
	e.list = c.t1
	e.elem = c.t1.PushFront(e)
	c.items[key] = e
}

func (c *ARCCache) replace(inB2 bool) {
	if c.t1.Len() > 0 && ((int64(c.t1.Len()) > c.p) || (inB2 && int64(c.t1.Len()) == c.p)) {
		oldest := c.t1.Back()
		if oldest != nil {
			e := oldest.Value.(*entry)
			c.t1.Remove(oldest)
			delete(c.items, e.key)
			e.isGhost = true
			e.list = c.b1
			e.elem = c.b1.PushFront(e)
			c.ghosts[e.key] = e
		}
	} else {
		if c.t2.Len() > 0 {
			oldest := c.t2.Back()
			if oldest != nil {
				e := oldest.Value.(*entry)
				c.t2.Remove(oldest)
				delete(c.items, e.key)
				e.isGhost = true
				e.list = c.b2
				e.elem = c.b2.PushFront(e)
				c.ghosts[e.key] = e
			}
		}
	}
}

func min(a, b int64) int64 {
	client := a
	if b < client {
		client = b
	}
	return client
}
