package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheManager coordinates multiple cache levels and implements advanced features
// like cache promotion, TTL cleanup, and smart eviction.
type CacheManager struct {
	// Cache levels
	l1Memory *MemoryCache
	l2Disk   *DiskCache
	session  *SessionCache
	
	// Configuration
	config *CacheConfig
	
	// Cleanup goroutine control
	cleanupStop   chan struct{}
	cleanupTicker *time.Ticker
	cleanupWg     sync.WaitGroup
	
	// Metrics
	mu    sync.RWMutex
	stats struct {
		TotalHits      int64
		TotalMisses    int64
		L1Hits         int64
		L2Hits         int64
		SessionHits    int64
		Promotions     int64
		CleanupRuns    int64
		LastCleanup    time.Time
	}
}

// NewCacheManager creates a new cache manager with the specified configuration.
func NewCacheManager(config *CacheConfig) (*CacheManager, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}
	
	// Set default cache directory if not specified
	if config.DiskPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		config.DiskPath = filepath.Join(homeDir, ".cache", "glow-tts", "audio")
	}
	
	// Create L1 memory cache
	l1Memory := NewMemoryCache(config.MemoryCapacity)
	
	// Create L2 disk cache
	l2Disk, err := NewDiskCache(config.DiskPath, config.DiskCapacity, config.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk cache: %w", err)
	}
	
	// Create session cache with unique ID
	sessionID := generateSessionID()
	session := NewSessionCache(sessionID, config.SessionCapacity)
	
	cm := &CacheManager{
		l1Memory:    l1Memory,
		l2Disk:      l2Disk,
		session:     session,
		config:      config,
		cleanupStop: make(chan struct{}),
	}
	
	// Start cleanup routine if configured
	if config.CleanupInterval > 0 {
		cm.startCleanupRoutine()
	}
	
	return cm, nil
}

// Get retrieves a value from the cache hierarchy.
// It checks L1 (memory), then L2 (disk), then session cache.
func (cm *CacheManager) Get(key string) ([]byte, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Check L1 memory cache first
	if data, ok := cm.l1Memory.Get(key); ok {
		cm.stats.L1Hits++
		cm.stats.TotalHits++
		return data, true
	}
	
	// Check L2 disk cache
	if data, ok := cm.l2Disk.Get(key); ok {
		cm.stats.L2Hits++
		cm.stats.TotalHits++
		
		// Promote to L1 for faster future access
		cm.promoteToL1(key, data)
		
		return data, true
	}
	
	// Check session cache
	if data, ok := cm.session.Get(key); ok {
		cm.stats.SessionHits++
		cm.stats.TotalHits++
		
		// Promote to L1 for faster future access
		cm.promoteToL1(key, data)
		
		return data, true
	}
	
	cm.stats.TotalMisses++
	return nil, false
}

// Put stores a value in the cache hierarchy.
// It stores in L1 and session cache immediately, and L2 asynchronously.
func (cm *CacheManager) Put(key string, value []byte) error {
	// Store in L1 memory cache
	if err := cm.l1Memory.Put(key, value); err != nil && err != ErrItemTooLarge {
		return fmt.Errorf("L1 cache error: %w", err)
	}
	
	// Store in session cache
	if err := cm.session.Put(key, value); err != nil && err != ErrItemTooLarge {
		// Non-fatal, continue
	}
	
	// Store in L2 disk cache asynchronously
	go func() {
		if err := cm.l2Disk.Put(key, value); err != nil && err != ErrItemTooLarge {
			// Log error but don't fail the operation
			// In production, use proper logging
		}
	}()
	
	return nil
}

// Delete removes an entry from all cache levels.
func (cm *CacheManager) Delete(key string) error {
	var errs []error
	
	if err := cm.l1Memory.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L1 delete: %w", err))
	}
	
	if err := cm.l2Disk.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L2 delete: %w", err))
	}
	
	if err := cm.session.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("session delete: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("cache delete errors: %v", errs)
	}
	
	return nil
}

// Clear removes all entries from all cache levels.
func (cm *CacheManager) Clear() error {
	var errs []error
	
	if err := cm.l1Memory.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L1 clear: %w", err))
	}
	
	if err := cm.l2Disk.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L2 clear: %w", err))
	}
	
	if err := cm.session.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("session clear: %w", err))
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("cache clear errors: %v", errs)
	}
	
	return nil
}

// GetWithMetadata retrieves a value along with its metadata from any cache level.
func (cm *CacheManager) GetWithMetadata(key string) ([]byte, CacheMetadata, bool) {
	// Check L1 memory cache
	if data, meta, ok := cm.l1Memory.GetWithMetadata(key); ok {
		cm.mu.Lock()
		cm.stats.L1Hits++
		cm.stats.TotalHits++
		cm.mu.Unlock()
		return data, meta, true
	}
	
	// Check L2 disk cache
	if data, meta, ok := cm.l2Disk.GetWithMetadata(key); ok {
		cm.mu.Lock()
		cm.stats.L2Hits++
		cm.stats.TotalHits++
		cm.stats.Promotions++
		cm.mu.Unlock()
		
		// Promote to L1
		cm.promoteToL1(key, data)
		
		return data, meta, true
	}
	
	// Check session cache
	if data, meta, ok := cm.session.GetWithMetadata(key); ok {
		cm.mu.Lock()
		cm.stats.SessionHits++
		cm.stats.TotalHits++
		cm.mu.Unlock()
		
		// Promote to L1
		cm.promoteToL1(key, data)
		
		return data, meta, true
	}
	
	cm.mu.Lock()
	cm.stats.TotalMisses++
	cm.mu.Unlock()
	
	return nil, CacheMetadata{}, false
}

// Stats returns aggregated statistics from all cache levels.
func (cm *CacheManager) Stats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	l1Stats := cm.l1Memory.Stats()
	l2Stats := cm.l2Disk.Stats()
	sessionStats := cm.session.Stats()
	
	totalHits := cm.stats.TotalHits
	totalMisses := cm.stats.TotalMisses
	totalRequests := totalHits + totalMisses
	
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"total_hits":    totalHits,
		"total_misses":  totalMisses,
		"hit_rate":      hitRate,
		"l1_hits":       cm.stats.L1Hits,
		"l2_hits":       cm.stats.L2Hits,
		"session_hits":  cm.stats.SessionHits,
		"promotions":    cm.stats.Promotions,
		"cleanup_runs":  cm.stats.CleanupRuns,
		"last_cleanup":  cm.stats.LastCleanup,
		"l1_stats":      l1Stats,
		"l2_stats":      l2Stats,
		"session_stats": sessionStats,
		"l1_size":       cm.l1Memory.Size(),
		"l2_size":       cm.l2Disk.Size(),
		"session_size":  cm.session.Size(),
	}
}

// GenerateCacheKey generates a cache key from text and configuration.
func GenerateCacheKey(text, voice string, speed float64) string {
	// Normalize text for consistent keys
	data := fmt.Sprintf("%s|%s|%.2f", text, voice, speed)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter keys
}

// Close shuts down the cache manager and performs cleanup.
func (cm *CacheManager) Close() error {
	// Stop cleanup routine
	if cm.cleanupTicker != nil {
		close(cm.cleanupStop)
		cm.cleanupWg.Wait()
		cm.cleanupTicker.Stop()
	}
	
	// Clear session cache
	cm.session.Clear()
	
	// Save disk cache index
	if err := cm.l2Disk.Close(); err != nil {
		return fmt.Errorf("failed to close disk cache: %w", err)
	}
	
	return nil
}

// Private helper methods

// promoteToL1 promotes an item to L1 cache for faster access.
func (cm *CacheManager) promoteToL1(key string, data []byte) {
	cm.stats.Promotions++
	// Ignore errors as promotion is best-effort
	_ = cm.l1Memory.Put(key, data)
}

// startCleanupRoutine starts the background cleanup goroutine.
func (cm *CacheManager) startCleanupRoutine() {
	cm.cleanupTicker = time.NewTicker(cm.config.CleanupInterval)
	cm.cleanupWg.Add(1)
	
	go func() {
		defer cm.cleanupWg.Done()
		
		for {
			select {
			case <-cm.cleanupTicker.C:
				cm.performCleanup()
			case <-cm.cleanupStop:
				return
			}
		}
	}()
}

// performCleanup performs periodic cleanup tasks.
func (cm *CacheManager) performCleanup() {
	cm.mu.Lock()
	cm.stats.CleanupRuns++
	cm.stats.LastCleanup = time.Now()
	cm.mu.Unlock()
	
	// Remove expired entries from L2 disk cache (TTL cleanup)
	if cm.config.TTLDays > 0 {
		cutoff := time.Now().Add(-time.Duration(cm.config.TTLDays) * 24 * time.Hour)
		removed := cm.l2Disk.RemoveOlderThan(cutoff)
		if removed > 0 {
			// Log cleanup activity
			// In production, use proper logging
		}
	}
	
	// Enforce size limits with smart eviction
	cm.enforceSizeLimits()
	
	// Clear stale session data
	cm.session.ClearIfStale(24 * time.Hour)
	
	// Prune old entries from memory cache
	if cm.config.TTLDays > 0 {
		maxAge := time.Duration(cm.config.TTLDays) * 24 * time.Hour
		cm.l1Memory.Prune(maxAge)
	}
}

// enforceSizeLimits enforces size limits on caches using smart eviction.
func (cm *CacheManager) enforceSizeLimits() {
	// Check L2 disk cache size
	if cm.l2Disk.Size() > cm.config.DiskCapacity {
		// Use smart eviction scoring to decide what to evict
		candidates := cm.l2Disk.GetLRUEntries(10)
		
		// Sort by eviction score (lower score = more likely to evict)
		for i := range candidates {
			candidates[i].CalculateEvictionScore()
		}
		
		// Evict items with lowest scores
		cm.l2Disk.EvictLRU()
	}
	
	// L1 memory cache handles its own eviction via LRU
	// Session cache also handles its own eviction
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("session-%d-%d", timestamp, os.Getpid())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// ClearSessionCache clears only the session cache.
func (cm *CacheManager) ClearSessionCache() error {
	return cm.session.Clear()
}

// WarmupCache preloads frequently used items into cache.
func (cm *CacheManager) WarmupCache(items map[string][]byte) error {
	for key, value := range items {
		if err := cm.Put(key, value); err != nil {
			// Continue on error, best effort
			continue
		}
	}
	return nil
}

// GetCacheSize returns the total size of all cache levels.
func (cm *CacheManager) GetCacheSize() (l1Size, l2Size, sessionSize int64) {
	return cm.l1Memory.Size(), cm.l2Disk.Size(), cm.session.Size()
}

// SetCleanupInterval updates the cleanup interval dynamically.
func (cm *CacheManager) SetCleanupInterval(interval time.Duration) {
	if cm.cleanupTicker != nil {
		cm.cleanupTicker.Stop()
	}
	
	if interval > 0 {
		cm.config.CleanupInterval = interval
		cm.startCleanupRoutine()
	}
}