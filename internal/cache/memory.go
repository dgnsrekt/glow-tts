package cache

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache implements an L1 in-memory cache with LRU eviction.
// It provides fast access to frequently used audio data with a configurable size limit.
type MemoryCache struct {
	capacity int64 // Maximum size in bytes
	size     int64 // Current size in bytes

	// LRU implementation
	items    map[string]*list.Element
	eviction *list.List

	// Synchronization
	mu sync.RWMutex

	// Metrics
	stats CacheStats
}

// memoryCacheEntry represents an entry in the memory cache
type memoryCacheEntry struct {
	key       string
	value     []byte
	size      int64
	timestamp time.Time
	hits      int64
}

// NewMemoryCache creates a new memory cache with the specified capacity in bytes.
func NewMemoryCache(capacity int64) *MemoryCache {
	return &MemoryCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		eviction: list.New(),
		stats: CacheStats{
			Capacity: capacity,
		},
	}
}

// Get retrieves a value from the cache.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.eviction.MoveToFront(elem)
	entry := elem.Value.(*memoryCacheEntry)
	entry.hits++

	c.stats.Hits++
	return entry.value, true
}

// Put stores a value in the cache.
func (c *MemoryCache) Put(key string, value []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	valueSize := int64(len(value))

	// Check if key already exists
	if elem, ok := c.items[key]; ok {
		// Update existing entry
		c.eviction.MoveToFront(elem)
		entry := elem.Value.(*memoryCacheEntry)

		// Update size
		c.size += valueSize - entry.size

		// Update entry
		entry.value = value
		entry.size = valueSize
		entry.timestamp = time.Now()

		c.stats.Size = c.size
		return nil
	}

	// Check if value is too large for cache
	if valueSize > c.capacity {
		return ErrItemTooLarge
	}

	// Evict items if necessary
	for c.size+valueSize > c.capacity && c.eviction.Len() > 0 {
		c.evictOldest()
	}

	// Add new entry
	entry := &memoryCacheEntry{
		key:       key,
		value:     value,
		size:      valueSize,
		timestamp: time.Now(),
		hits:      0,
	}

	elem := c.eviction.PushFront(entry)
	c.items[key] = elem
	c.size += valueSize

	c.stats.Size = c.size
	return nil
}

// Delete removes an entry from the cache.
func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil
	}

	c.removeElement(elem)
	return nil
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.eviction.Init()
	c.size = 0
	c.stats.Size = 0

	return nil
}

// Size returns the current cache size in bytes.
func (c *MemoryCache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.size
}

// Stats returns cache statistics.
func (c *MemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := c.stats
	stats.Size = c.size
	stats.ItemCount = int64(len(c.items))

	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}

	return stats
}

// GetWithMetadata retrieves a value along with its metadata.
func (c *MemoryCache) GetWithMetadata(key string) ([]byte, CacheMetadata, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		c.stats.Misses++
		return nil, CacheMetadata{}, false
	}

	// Move to front (most recently used)
	c.eviction.MoveToFront(elem)
	entry := elem.Value.(*memoryCacheEntry)
	entry.hits++

	c.stats.Hits++

	metadata := CacheMetadata{
		Key:       entry.key,
		Size:      entry.size,
		Timestamp: entry.timestamp,
		Hits:      entry.hits,
		Level:     CacheLevelL1,
	}

	return entry.value, metadata, true
}

// GetLRUEntries returns the n least recently used entries for eviction or promotion.
func (c *MemoryCache) GetLRUEntries(n int) []CacheMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]CacheMetadata, 0, n)

	// Start from the back (least recently used)
	elem := c.eviction.Back()
	for i := 0; i < n && elem != nil; i++ {
		entry := elem.Value.(*memoryCacheEntry)
		entries = append(entries, CacheMetadata{
			Key:       entry.key,
			Size:      entry.size,
			Timestamp: entry.timestamp,
			Hits:      entry.hits,
			Level:     CacheLevelL1,
		})
		elem = elem.Prev()
	}

	return entries
}

// Resize changes the cache capacity.
func (c *MemoryCache) Resize(newCapacity int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = newCapacity
	c.stats.Capacity = newCapacity

	// Evict if necessary
	for c.size > c.capacity && c.eviction.Len() > 0 {
		c.evictOldest()
	}
}

// evictOldest removes the least recently used item (must be called with lock held).
func (c *MemoryCache) evictOldest() {
	elem := c.eviction.Back()
	if elem != nil {
		c.removeElement(elem)
		c.stats.Evictions++
	}
}

// removeElement removes an element from the cache (must be called with lock held).
func (c *MemoryCache) removeElement(elem *list.Element) {
	c.eviction.Remove(elem)
	entry := elem.Value.(*memoryCacheEntry)
	delete(c.items, entry.key)
	c.size -= entry.size
}

// Contains checks if a key exists in the cache without updating LRU.
func (c *MemoryCache) Contains(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.items[key]
	return ok
}

// Keys returns all keys in the cache.
func (c *MemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	return keys
}

// Prune removes entries older than the specified duration.
func (c *MemoryCache) Prune(maxAge time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	pruned := 0

	// Start from the back (oldest entries)
	elem := c.eviction.Back()
	for elem != nil {
		prev := elem.Prev()
		entry := elem.Value.(*memoryCacheEntry)

		if entry.timestamp.Before(cutoff) {
			c.removeElement(elem)
			pruned++
		}

		elem = prev
	}

	return pruned
}
