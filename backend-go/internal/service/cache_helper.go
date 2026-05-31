package service

import (
	"sync"
	"time"
)

type lruEntry struct {
	key      string
	value    any
	cachedAt time.Time
	prev     *lruEntry
	next     *lruEntry
}

type DetailCache struct {
	mu         sync.Mutex
	maxEntries int
	ttl        time.Duration
	items      map[string]*lruEntry
	head       *lruEntry
	tail       *lruEntry
}

func NewDetailCache(maxEntries int, ttl time.Duration) *DetailCache {
	return &DetailCache{
		maxEntries: maxEntries,
		ttl:        ttl,
		items:      make(map[string]*lruEntry),
	}
}

func (c *DetailCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.items[key]; ok {
		if time.Since(entry.cachedAt) > c.ttl {
			c.evict()
			return nil, false
		}
		c.moveToFront(entry)
		return entry.value, true
	}
	return nil, false
}

func (c *DetailCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.items[key]; ok {
		entry.value = value
		entry.cachedAt = time.Now()
		c.moveToFront(entry)
		return
	}
	if len(c.items) >= c.maxEntries {
		c.evict()
	}
	entry := &lruEntry{key: key, value: value, cachedAt: time.Now()}
	c.items[key] = entry
	c.pushFront(entry)
}

func (c *DetailCache) moveToFront(e *lruEntry) {
	if e == c.head {
		return
	}
	c.remove(e)
	c.pushFront(e)
}

func (c *DetailCache) pushFront(e *lruEntry) {
	e.prev = nil
	e.next = c.head
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
	if c.tail == nil {
		c.tail = e
	}
}

func (c *DetailCache) remove(e *lruEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
}

func (c *DetailCache) evict() {
	if c.tail == nil {
		return
	}
	delete(c.items, c.tail.key)
	c.remove(c.tail)
}
