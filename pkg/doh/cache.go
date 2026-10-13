package doh

import (
	"container/list"
	"sync"
	"time"
)

type cacheEntry struct {
	key       string
	data      []byte
	expiresAt time.Time
}

// Cache provides a thread-safe, LRU-based store.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*list.Element
	list  *list.List
	limit int
}

func NewCache() *Cache {
	return &Cache{
		items: make(map[string]*list.Element),
		list:  list.New(),
		limit: 1000,
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		if time.Now().After(el.Value.(*cacheEntry).expiresAt) {
			c.removeElement(el)
			return nil, false
		}
		c.list.MoveToFront(el)
		return el.Value.(*cacheEntry).data, true
	}
	return nil, false
}

func (c *Cache) Set(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.list.MoveToFront(el)
		el.Value.(*cacheEntry).data = data
		el.Value.(*cacheEntry).expiresAt = time.Now().Add(ttl)
		return
	}
	if c.list.Len() >= c.limit {
		c.removeElement(c.list.Back())
	}
	entry := &cacheEntry{key: key, data: data, expiresAt: time.Now().Add(ttl)}
	c.items[key] = c.list.PushFront(entry)
}

func (c *Cache) removeElement(el *list.Element) {
	c.list.Remove(el)
	delete(c.items, el.Value.(*cacheEntry).key)
}
