package service

type lruEntry struct {
	key   string
	value any
	prev  *lruEntry
	next  *lruEntry
}

type DetailCache struct {
	maxEntries int
	items      map[string]*lruEntry
	head       *lruEntry
	tail       *lruEntry
}

func NewDetailCache(maxEntries int) *DetailCache {
	return &DetailCache{
		maxEntries: maxEntries,
		items:      make(map[string]*lruEntry),
	}
}

func (c *DetailCache) Get(key string) (any, bool) {
	if entry, ok := c.items[key]; ok {
		c.moveToFront(entry)
		return entry.value, true
	}
	return nil, false
}

func (c *DetailCache) Set(key string, value any) {
	if entry, ok := c.items[key]; ok {
		entry.value = value
		c.moveToFront(entry)
		return
	}
	if len(c.items) >= c.maxEntries {
		c.evict()
	}
	entry := &lruEntry{key: key, value: value}
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
