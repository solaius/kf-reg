// Package cache provides an in-memory LRU cache with TTL for caching
// HTTP responses from discovery and capabilities endpoints.
package cache

import (
	"sync"
	"time"
)

// entry holds a cached value with its expiration time and insertion order.
type entry struct {
	value     []byte
	expiresAt time.Time
	insertedAt time.Time
}

// LRUCache is a thread-safe in-memory cache with TTL and max-size eviction.
// When the cache reaches maxSize, the oldest entry (by insertion time) is
// evicted to make room for new entries. Expired entries are lazily evicted
// on Get.
type LRUCache struct {
	mu      sync.RWMutex
	items   map[string]*entry
	maxSize int
	ttl     time.Duration
}

// NewLRUCache creates a new LRU cache with the given maximum size and TTL.
// maxSize must be >= 1; ttl must be > 0.
func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
	if maxSize < 1 {
		maxSize = 1
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &LRUCache{
		items:   make(map[string]*entry, maxSize),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a cached value by key. Returns (nil, false) if the key is
// missing or expired. Expired entries are lazily deleted.
func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(e.expiresAt) {
		delete(c.items, key)
		return nil, false
	}

	return e.value, true
}

// Set stores a value in the cache. If the cache is at capacity, the oldest
// entry (by insertion time) is evicted before inserting.
func (c *LRUCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// If key already exists, update it in place.
	if _, ok := c.items[key]; ok {
		c.items[key] = &entry{
			value:      value,
			expiresAt:  now.Add(c.ttl),
			insertedAt: now,
		}
		return
	}

	// Evict oldest if at capacity.
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &entry{
		value:      value,
		expiresAt:  now.Add(c.ttl),
		insertedAt: now,
	}
}

// Invalidate removes a specific key from the cache.
func (c *LRUCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// InvalidateAll removes all entries from the cache.
func (c *LRUCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*entry, c.maxSize)
}

// Size returns the number of entries currently in the cache (including
// potentially expired ones that haven't been lazily cleaned).
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictOldest removes the entry with the oldest insertedAt timestamp.
// Must be called with c.mu held.
func (c *LRUCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for k, e := range c.items {
		if first || e.insertedAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = e.insertedAt
			first = false
		}
	}

	if !first {
		delete(c.items, oldestKey)
	}
}
