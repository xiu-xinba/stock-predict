// Package cache 实现了基于 LRU（最近最少使用）策略的内存缓存。
package cache

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

// DetailCache 是带 TTL 过期策略的 LRU 内存缓存，支持自定义最大条目数和过期时间。
type DetailCache struct {
	mu         sync.RWMutex  // 保护并发访问的读写锁
	maxEntries int           // 缓存最大条目数
	ttl        time.Duration // 默认缓存过期时间
	items      map[string]*lruEntry // 键到缓存条目的映射
	head       *lruEntry     // 双向链表头节点（最近访问）
	tail       *lruEntry     // 双向链表尾节点（最久未访问）
}

// NewDetailCache 创建一个新的 LRU 缓存实例，指定最大条目数和默认 TTL。
func NewDetailCache(maxEntries int, ttl time.Duration) *DetailCache {
	return &DetailCache{
		maxEntries: maxEntries,
		ttl:        ttl,
		items:      make(map[string]*lruEntry),
	}
}

// Get 从缓存中获取指定键的值，使用默认 TTL 检查过期。
// 若键不存在或已过期，返回 nil, false。
func (c *DetailCache) Get(key string) (any, bool) {
	if c.ttl <= 0 {
		return nil, false
	}
	return c.get(key, c.ttl, true)
}

// GetWithMaxAge 从缓存中获取指定键的值，使用自定义 maxAge 检查过期。
// 若键不存在或已超过 maxAge，返回 nil, false。
func (c *DetailCache) GetWithMaxAge(key string, maxAge time.Duration) (any, bool) {
	if maxAge <= 0 {
		return nil, false
	}
	return c.get(key, maxAge, true)
}

// GetStale 从缓存中获取指定键的值，不检查过期时间（允许返回过期数据）。
func (c *DetailCache) GetStale(key string) (any, bool) {
	return c.get(key, 0, false)
}

// TTL 返回缓存的默认过期时间。
func (c *DetailCache) TTL() time.Duration {
	return c.ttl
}

// Peek 查看缓存中指定键的值，不更新 LRU 顺序也不检查过期。
func (c *DetailCache) Peek(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	return entry.value, true
}

// Backdate 将指定键的缓存时间向前回退 age 时长，使该条目更快过期。
// 若键不存在返回 false。
func (c *DetailCache) Backdate(key string, age time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return false
	}
	entry.cachedAt = time.Now().Add(-age)
	return true
}

func (c *DetailCache) get(key string, maxAge time.Duration, enforceAge bool) (any, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if enforceAge && time.Since(entry.cachedAt) > maxAge {
		c.mu.RUnlock()
		return nil, false
	}
	if entry == c.head {
		c.mu.RUnlock()
		return entry.value, true
	}
	c.mu.RUnlock()
	c.mu.Lock()
	c.moveToFront(entry)
	c.mu.Unlock()
	return entry.value, true
}

// Set 向缓存中写入键值对，若键已存在则更新值和缓存时间并移至链表头部；
// 若缓存已满则淘汰最久未使用的条目。
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
