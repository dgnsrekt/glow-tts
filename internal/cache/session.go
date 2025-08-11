package cache

import (
	"sync"
	"time"
)

// SessionCache implements a session-specific cache that is cleared on exit.
// It's designed for quick access to recently used items in the current session.
type SessionCache struct {
	capacity int64 // Maximum size in bytes
	size     int64 // Current size in bytes

	// Simple map-based storage (no LRU for simplicity/speed)
	items map[string]*sessionCacheEntry

	// Session tracking
	sessionID string
	startTime time.Time
	lastClear time.Time

	// Synchronization
	mu sync.RWMutex

	// Metrics
	stats CacheStats
}

// sessionCacheEntry represents an entry in the session cache
type sessionCacheEntry struct {
	key       string
	value     []byte
	size      int64
	timestamp time.Time
	hits      int64
}

// NewSessionCache creates a new session cache with the specified capacity.
func NewSessionCache(sessionID string, capacity int64) *SessionCache {
	return &SessionCache{
		capacity:  capacity,
		items:     make(map[string]*sessionCacheEntry),
		sessionID: sessionID,
		startTime: time.Now(),
		lastClear: time.Now(),
		stats: CacheStats{
			Capacity: capacity,
		},
	}
}

// Get retrieves a value from the session cache.
func (sc *SessionCache) Get(key string) ([]byte, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, ok := sc.items[key]
	if !ok {
		sc.stats.Misses++
		return nil, false
	}

	entry.hits++
	sc.stats.Hits++
	sc.stats.LastAccess = time.Now()

	return entry.value, true
}

// Put stores a value in the session cache.
func (sc *SessionCache) Put(key string, value []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	valueSize := int64(len(value))

	// Check if key already exists
	if existing, ok := sc.items[key]; ok {
		// Update existing entry
		sc.size -= existing.size
		existing.value = value
		existing.size = valueSize
		existing.timestamp = time.Now()
		sc.size += valueSize

		sc.stats.Size = sc.size
		return nil
	}

	// Check if value is too large
	if valueSize > sc.capacity {
		return ErrItemTooLarge
	}

	// Evict items if necessary (simple FIFO for session cache)
	for sc.size+valueSize > sc.capacity && len(sc.items) > 0 {
		sc.evictOldest()
	}

	// Add new entry
	entry := &sessionCacheEntry{
		key:       key,
		value:     value,
		size:      valueSize,
		timestamp: time.Now(),
		hits:      0,
	}

	sc.items[key] = entry
	sc.size += valueSize

	sc.stats.Size = sc.size
	sc.stats.ItemCount = int64(len(sc.items))

	return nil
}

// Delete removes an entry from the session cache.
func (sc *SessionCache) Delete(key string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	entry, ok := sc.items[key]
	if !ok {
		return nil
	}

	delete(sc.items, key)
	sc.size -= entry.size

	sc.stats.Size = sc.size
	sc.stats.ItemCount = int64(len(sc.items))

	return nil
}

// Clear removes all entries from the session cache.
func (sc *SessionCache) Clear() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.items = make(map[string]*sessionCacheEntry)
	sc.size = 0
	sc.lastClear = time.Now()

	sc.stats.Size = 0
	sc.stats.ItemCount = 0

	return nil
}

// Size returns the current cache size in bytes.
func (sc *SessionCache) Size() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.size
}

// Stats returns cache statistics.
func (sc *SessionCache) Stats() CacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	stats := sc.stats
	stats.Size = sc.size
	stats.ItemCount = int64(len(sc.items))

	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}

	return stats
}

// Contains checks if a key exists in the cache.
func (sc *SessionCache) Contains(key string) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	_, ok := sc.items[key]
	return ok
}

// GetWithMetadata retrieves a value along with its metadata.
func (sc *SessionCache) GetWithMetadata(key string) ([]byte, CacheMetadata, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	entry, ok := sc.items[key]
	if !ok {
		sc.stats.Misses++
		return nil, CacheMetadata{}, false
	}

	entry.hits++
	sc.stats.Hits++

	metadata := CacheMetadata{
		Key:       entry.key,
		Size:      entry.size,
		Timestamp: entry.timestamp,
		Hits:      entry.hits,
		Level:     CacheLevelSession,
	}

	return entry.value, metadata, true
}

// Keys returns all keys in the cache.
func (sc *SessionCache) Keys() []string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	keys := make([]string, 0, len(sc.items))
	for key := range sc.items {
		keys = append(keys, key)
	}
	return keys
}

// ClearIfStale clears the cache if it hasn't been cleared recently.
func (sc *SessionCache) ClearIfStale(maxAge time.Duration) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if time.Since(sc.lastClear) > maxAge {
		sc.items = make(map[string]*sessionCacheEntry)
		sc.size = 0
		sc.lastClear = time.Now()

		sc.stats.Size = 0
		sc.stats.ItemCount = 0

		return true
	}

	return false
}

// GetSessionInfo returns information about the current session.
func (sc *SessionCache) GetSessionInfo() (sessionID string, duration time.Duration, itemCount int) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	return sc.sessionID, time.Since(sc.startTime), len(sc.items)
}

// evictOldest removes the oldest item from the cache (by timestamp).
func (sc *SessionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range sc.items {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		entry := sc.items[oldestKey]
		delete(sc.items, oldestKey)
		sc.size -= entry.size
		sc.stats.Evictions++
		sc.stats.LastEvict = time.Now()
	}
}

// Prune removes entries older than the specified duration.
func (sc *SessionCache) Prune(maxAge time.Duration) int {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	pruned := 0

	for key, entry := range sc.items {
		if entry.timestamp.Before(cutoff) {
			delete(sc.items, key)
			sc.size -= entry.size
			pruned++
		}
	}

	sc.stats.Size = sc.size
	sc.stats.ItemCount = int64(len(sc.items))

	return pruned
}
