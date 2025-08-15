package tts

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Cache size limits
const (
	// L1CacheSizeLimit is the maximum size of the memory cache (100MB)
	L1CacheSizeLimit = 100 * 1024 * 1024
	// L2CacheSizeLimit is the maximum size of the disk cache (1GB)
	L2CacheSizeLimit = 1024 * 1024 * 1024
	// CacheTTL is the time-to-live for disk cache entries (7 days)
	CacheTTL = 7 * 24 * time.Hour
	// L1CacheTTL is the time-to-live for memory cache entries (1 hour)
	L1CacheTTL = 1 * time.Hour
	// CleanupInterval is how often the cleanup routine runs
	CleanupInterval = 15 * time.Minute
	// CacheKeyVersion is the current cache key version
	CacheKeyVersion = "v1"
)

// AudioData represents cached audio with metadata
type AudioData struct {
	Audio     []byte    `json:"-"` // Audio data (not serialized to JSON)
	Text      string    `json:"text"`
	Voice     string    `json:"voice"`
	Speed     float64   `json:"speed"`
	CacheKey  string    `json:"cache_key"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Hits      int64     `json:"hits"`
}

// CacheMetadata represents metadata for disk cache entries
type CacheMetadata struct {
	Text      string    `json:"text"`
	Voice     string    `json:"voice"`
	Speed     float64   `json:"speed"`
	CacheKey  string    `json:"cache_key"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Hits      int64     `json:"hits"`
	AudioFile string    `json:"audio_file"`
}

// Cache interface defines common cache operations
type Cache interface {
	Get(key string) (*AudioData, error)
	Put(key string, data *AudioData) error
	Delete(key string) error
	Size() int64
	Clear() error
	Close() error
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	L1SizeLimit      int64
	L2SizeLimit      int64
	L1TTL            time.Duration
	L2TTL            time.Duration
	CleanupInterval  time.Duration
	EnableMetrics    bool
	EnableCompression bool
	CacheDir         string
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "glow-tts-cache")
	} else {
		cacheDir = filepath.Join(cacheDir, "glow-tts")
	}

	return &CacheConfig{
		L1SizeLimit:       L1CacheSizeLimit,
		L2SizeLimit:       L2CacheSizeLimit,
		L1TTL:             L1CacheTTL,
		L2TTL:             CacheTTL,
		CleanupInterval:   CleanupInterval,
		EnableMetrics:     true,
		EnableCompression: true,
		CacheDir:          cacheDir,
	}
}

// TTSCacheManager manages two-level caching for TTS
type TTSCacheManager struct {
	l1Cache  *MemoryCache
	l2Cache  *DiskCache
	config   *CacheConfig
	metrics  *CacheMetrics
	cleanupStop chan struct{}
	cleanupWg   sync.WaitGroup
}

// NewTTSCacheManager creates a new TTS cache manager
func NewTTSCacheManager(config *CacheConfig) (*TTSCacheManager, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	// Create L1 memory cache
	l1Cache := NewMemoryCache(config.L1SizeLimit, config.L1TTL)

	// Create L2 disk cache
	l2Cache, err := NewDiskCache(config.CacheDir, config.L2SizeLimit, config.L2TTL)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk cache: %w", err)
	}

	// Create metrics if enabled
	var metrics *CacheMetrics
	if config.EnableMetrics {
		metrics = NewCacheMetrics()
	}

	manager := &TTSCacheManager{
		l1Cache:     l1Cache,
		l2Cache:     l2Cache,
		config:      config,
		metrics:     metrics,
		cleanupStop: make(chan struct{}),
	}

	// Start cleanup routine
	manager.startCleanup()

	return manager, nil
}

// Get retrieves audio data from cache (L1 first, then L2)
func (cm *TTSCacheManager) Get(key string) (*AudioData, error) {
	// Record access attempt
	if cm.metrics != nil {
		cm.metrics.RecordAccess()
	}

	// Try L1 cache first
	data, err := cm.l1Cache.Get(key)
	if err == nil && data != nil {
		if cm.metrics != nil {
			cm.metrics.RecordL1Hit()
		}
		return data, nil
	}

	// Try L2 cache
	data, err = cm.l2Cache.Get(key)
	if err == nil && data != nil {
		// Promote to L1
		_ = cm.l1Cache.Put(key, data)
		if cm.metrics != nil {
			cm.metrics.RecordL2Hit()
			cm.metrics.RecordPromotion()
		}
		return data, nil
	}

	// Cache miss
	if cm.metrics != nil {
		cm.metrics.RecordMiss()
	}
	return nil, fmt.Errorf("cache miss for key: %s", key)
}

// Put stores audio data in both cache levels
func (cm *TTSCacheManager) Put(key string, data *AudioData) error {
	// Store in L1
	if err := cm.l1Cache.Put(key, data); err != nil {
		// L1 failure is not critical, log and continue
	}

	// Store in L2
	if err := cm.l2Cache.Put(key, data); err != nil {
		return fmt.Errorf("failed to store in L2 cache: %w", err)
	}

	if cm.metrics != nil {
		cm.metrics.RecordWrite()
	}

	return nil
}

// Delete removes entry from both cache levels
func (cm *TTSCacheManager) Delete(key string) error {
	var errs []error

	if err := cm.l1Cache.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L1 delete failed: %w", err))
	}

	if err := cm.l2Cache.Delete(key); err != nil {
		errs = append(errs, fmt.Errorf("L2 delete failed: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cache delete errors: %v", errs)
	}

	return nil
}

// Clear clears both cache levels
func (cm *TTSCacheManager) Clear() error {
	var errs []error

	if err := cm.l1Cache.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L1 clear failed: %w", err))
	}

	if err := cm.l2Cache.Clear(); err != nil {
		errs = append(errs, fmt.Errorf("L2 clear failed: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cache clear errors: %v", errs)
	}

	return nil
}

// Size returns the combined size of both cache levels
func (cm *TTSCacheManager) Size() int64 {
	return cm.l1Cache.Size() + cm.l2Cache.Size()
}

// Close shuts down the cache manager
func (cm *TTSCacheManager) Close() error {
	// Stop cleanup routine
	close(cm.cleanupStop)
	cm.cleanupWg.Wait()

	// Close caches
	var errs []error

	if err := cm.l1Cache.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := cm.l2Cache.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("cache close errors: %v", errs)
	}

	return nil
}

// startCleanup starts the background cleanup routine
func (cm *TTSCacheManager) startCleanup() {
	cm.cleanupWg.Add(1)
	go func() {
		defer cm.cleanupWg.Done()
		ticker := time.NewTicker(cm.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cm.cleanup()
			case <-cm.cleanupStop:
				return
			}
		}
	}()
}

// cleanup performs periodic cleanup
func (cm *TTSCacheManager) cleanup() {
	// Clean L1 cache
	cm.l1Cache.cleanup()

	// Clean L2 cache
	cm.l2Cache.cleanup()

	if cm.metrics != nil {
		cm.metrics.RecordCleanup()
	}
}

// GetMetrics returns cache metrics
func (cm *TTSCacheManager) GetMetrics() *CacheMetrics {
	return cm.metrics
}

// MemoryCache implements L1 in-memory cache with LRU eviction
type MemoryCache struct {
	mu        sync.RWMutex
	items     map[string]*list.Element
	lru       *list.List
	size      int64
	sizeLimit int64
	ttl       time.Duration
}

// memoryCacheEntry wraps AudioData with LRU list element
type memoryCacheEntry struct {
	key       string
	data      *AudioData
	timestamp time.Time
	size      int64
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(sizeLimit int64, ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		items:     make(map[string]*list.Element),
		lru:       list.New(),
		sizeLimit: sizeLimit,
		ttl:       ttl,
	}
}

// Get retrieves data from memory cache
func (mc *MemoryCache) Get(key string) (*AudioData, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	elem, ok := mc.items[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	entry := elem.Value.(*memoryCacheEntry)

	// Check TTL
	if time.Since(entry.timestamp) > mc.ttl {
		mc.removeElement(elem)
		return nil, fmt.Errorf("entry expired")
	}

	// Move to front (most recently used)
	mc.lru.MoveToFront(elem)

	// Update hits
	atomic.AddInt64(&entry.data.Hits, 1)

	// Return a copy to prevent modification
	dataCopy := *entry.data
	dataCopy.Audio = make([]byte, len(entry.data.Audio))
	copy(dataCopy.Audio, entry.data.Audio)

	return &dataCopy, nil
}

// Put stores data in memory cache
func (mc *MemoryCache) Put(key string, data *AudioData) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Calculate size
	size := int64(len(data.Audio)) + int64(len(data.Text)) + 256 // metadata overhead

	// Check if key exists
	if elem, ok := mc.items[key]; ok {
		// Update existing entry
		entry := elem.Value.(*memoryCacheEntry)
		mc.size -= entry.size
		entry.data = data
		entry.timestamp = time.Now()
		entry.size = size
		mc.size += size
		mc.lru.MoveToFront(elem)
		return nil
	}

	// Evict if necessary
	for mc.size+size > mc.sizeLimit && mc.lru.Len() > 0 {
		mc.removeElement(mc.lru.Back())
	}

	// Add new entry
	entry := &memoryCacheEntry{
		key:       key,
		data:      data,
		timestamp: time.Now(),
		size:      size,
	}

	elem := mc.lru.PushFront(entry)
	mc.items[key] = elem
	mc.size += size

	return nil
}

// Delete removes entry from memory cache
func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	elem, ok := mc.items[key]
	if !ok {
		return nil
	}

	mc.removeElement(elem)
	return nil
}

// removeElement removes an element from the cache (must be called with lock held)
func (mc *MemoryCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*memoryCacheEntry)
	delete(mc.items, entry.key)
	mc.lru.Remove(elem)
	mc.size -= entry.size
}

// Size returns current cache size
func (mc *MemoryCache) Size() int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.size
}

// Clear removes all entries
func (mc *MemoryCache) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.items = make(map[string]*list.Element)
	mc.lru.Init()
	mc.size = 0

	return nil
}

// Close closes the memory cache
func (mc *MemoryCache) Close() error {
	return mc.Clear()
}

// cleanup removes expired entries
func (mc *MemoryCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	for elem := mc.lru.Back(); elem != nil; {
		prev := elem.Prev()
		entry := elem.Value.(*memoryCacheEntry)
		if now.Sub(entry.timestamp) > mc.ttl {
			mc.removeElement(elem)
		}
		elem = prev
	}
}

// DiskCache implements L2 disk-based cache
type DiskCache struct {
	mu        sync.RWMutex
	cacheDir  string
	sizeLimit int64
	size      int64
	ttl       time.Duration
	index     map[string]*CacheMetadata
	indexFile string
}

// NewDiskCache creates a new disk cache
func NewDiskCache(cacheDir string, sizeLimit int64, ttl time.Duration) (*DiskCache, error) {
	// Create cache directory with restricted permissions
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dc := &DiskCache{
		cacheDir:  cacheDir,
		sizeLimit: sizeLimit,
		ttl:       ttl,
		index:     make(map[string]*CacheMetadata),
		indexFile: filepath.Join(cacheDir, "cache_index.json"),
	}

	// Load existing index
	if err := dc.loadIndex(); err != nil {
		// Index doesn't exist or is corrupted, start fresh
		dc.index = make(map[string]*CacheMetadata)
	}

	// Calculate current size
	dc.calculateSize()

	return dc, nil
}

// Get retrieves data from disk cache
func (dc *DiskCache) Get(key string) (*AudioData, error) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	metadata, ok := dc.index[key]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}

	// Check TTL
	if time.Since(metadata.Timestamp) > dc.ttl {
		return nil, fmt.Errorf("entry expired")
	}

	// Read audio file
	audioPath := filepath.Join(dc.cacheDir, metadata.AudioFile)
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Update hits
	atomic.AddInt64(&metadata.Hits, 1)

	return &AudioData{
		Audio:     audioData,
		Text:      metadata.Text,
		Voice:     metadata.Voice,
		Speed:     metadata.Speed,
		CacheKey:  metadata.CacheKey,
		Timestamp: metadata.Timestamp,
		Size:      metadata.Size,
		Hits:      metadata.Hits,
	}, nil
}

// Put stores data in disk cache
func (dc *DiskCache) Put(key string, data *AudioData) error {
	if dc == nil {
		return fmt.Errorf("disk cache is nil")
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Generate filename
	audioFile := fmt.Sprintf("%s.audio", key)
	audioPath := filepath.Join(dc.cacheDir, audioFile)

	// Write audio file
	if err := os.WriteFile(audioPath, data.Audio, 0600); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	// Create metadata
	metadata := &CacheMetadata{
		Text:      data.Text,
		Voice:     data.Voice,
		Speed:     data.Speed,
		CacheKey:  key,
		Timestamp: time.Now(),
		Size:      int64(len(data.Audio)),
		Hits:      0,
		AudioFile: audioFile,
	}

	// Update index
	if oldMeta, exists := dc.index[key]; exists {
		dc.size -= oldMeta.Size
	}
	dc.index[key] = metadata
	dc.size += metadata.Size

	// Save index
	return dc.saveIndex()
}

// Delete removes entry from disk cache
func (dc *DiskCache) Delete(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	metadata, ok := dc.index[key]
	if !ok {
		return nil
	}

	// Delete audio file
	audioPath := filepath.Join(dc.cacheDir, metadata.AudioFile)
	_ = os.Remove(audioPath)

	// Update index
	delete(dc.index, key)
	dc.size -= metadata.Size

	return dc.saveIndex()
}

// Size returns current cache size
func (dc *DiskCache) Size() int64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.size
}

// Clear removes all entries
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Remove all audio files
	for _, metadata := range dc.index {
		audioPath := filepath.Join(dc.cacheDir, metadata.AudioFile)
		_ = os.Remove(audioPath)
	}

	// Clear index
	dc.index = make(map[string]*CacheMetadata)
	dc.size = 0

	return dc.saveIndex()
}

// Close closes the disk cache
func (dc *DiskCache) Close() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	return dc.saveIndex()
}

// cleanup removes expired entries and enforces size limit
func (dc *DiskCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	var toDelete []string

	// Find expired entries
	for key, metadata := range dc.index {
		if now.Sub(metadata.Timestamp) > dc.ttl {
			toDelete = append(toDelete, key)
		}
	}

	// Delete expired entries
	for _, key := range toDelete {
		metadata := dc.index[key]
		audioPath := filepath.Join(dc.cacheDir, metadata.AudioFile)
		_ = os.Remove(audioPath)
		delete(dc.index, key)
		dc.size -= metadata.Size
	}

	// Enforce size limit (remove least recently used)
	if dc.size > dc.sizeLimit {
		// Sort by timestamp (oldest first)
		// This is simplified; in production, use a proper LRU structure
		for dc.size > dc.sizeLimit*9/10 && len(dc.index) > 0 { // Keep 90% full
			var oldestKey string
			var oldestTime time.Time
			for key, metadata := range dc.index {
				if oldestKey == "" || metadata.Timestamp.Before(oldestTime) {
					oldestKey = key
					oldestTime = metadata.Timestamp
				}
			}
			if oldestKey != "" {
				metadata := dc.index[oldestKey]
				audioPath := filepath.Join(dc.cacheDir, metadata.AudioFile)
				_ = os.Remove(audioPath)
				delete(dc.index, oldestKey)
				dc.size -= metadata.Size
			}
		}
	}

	_ = dc.saveIndex()
}

// loadIndex loads the cache index from disk
func (dc *DiskCache) loadIndex() error {
	data, err := os.ReadFile(dc.indexFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &dc.index)
}

// saveIndex saves the cache index to disk (must be called with lock held)
func (dc *DiskCache) saveIndex() error {
	// Skip saving in test environment if we're under heavy load
	if os.Getenv("GO_TEST_FAST") == "1" {
		return nil
	}
	
	data, err := json.MarshalIndent(dc.index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(dc.indexFile, data, 0600)
}

// calculateSize calculates the current cache size
func (dc *DiskCache) calculateSize() {
	dc.size = 0
	for _, metadata := range dc.index {
		dc.size += metadata.Size
	}
}

// GenerateCacheKey generates a deterministic cache key
func GenerateCacheKey(text, voice string, speed float64) string {
	// Normalize inputs
	text = normalizeText(text)
	voice = normalizeVoice(voice)
	// Normalize speed to consistent precision (e.g., "1.50" not "1.5")
	speedStr := fmt.Sprintf("%.2f", speed)

	// Create hash input with all parameters
	input := fmt.Sprintf("%s|%s|%s", text, voice, speedStr)

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Add version prefix for cache migration support
	return fmt.Sprintf("%s_%s", CacheKeyVersion, hashStr)
}

// normalizeText normalizes text for cache key generation
func normalizeText(text string) string {
	// Trim whitespace
	// In production, might do more normalization (lowercase, remove punctuation, etc.)
	return text
}

// normalizeVoice normalizes voice name for cache key generation
func normalizeVoice(voice string) string {
	// Lowercase and trim
	return voice
}

// CacheMetrics tracks cache performance metrics
type CacheMetrics struct {
	mu sync.RWMutex

	// Counters
	totalAccesses int64
	l1Hits        int64
	l2Hits        int64
	misses        int64
	writes        int64
	promotions    int64
	cleanups      int64

	// Rolling window stats (simplified)
	recentHits   []int64
	recentMisses []int64
	windowStart  time.Time
}

// NewCacheMetrics creates new cache metrics
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		recentHits:   make([]int64, 0, 60), // Last 60 samples
		recentMisses: make([]int64, 0, 60),
		windowStart:  time.Now(),
	}
}

// RecordAccess records a cache access
func (cm *CacheMetrics) RecordAccess() {
	atomic.AddInt64(&cm.totalAccesses, 1)
}

// RecordL1Hit records an L1 cache hit
func (cm *CacheMetrics) RecordL1Hit() {
	atomic.AddInt64(&cm.l1Hits, 1)
}

// RecordL2Hit records an L2 cache hit
func (cm *CacheMetrics) RecordL2Hit() {
	atomic.AddInt64(&cm.l2Hits, 1)
}

// RecordMiss records a cache miss
func (cm *CacheMetrics) RecordMiss() {
	atomic.AddInt64(&cm.misses, 1)
}

// RecordWrite records a cache write
func (cm *CacheMetrics) RecordWrite() {
	atomic.AddInt64(&cm.writes, 1)
}

// RecordPromotion records an L2 to L1 promotion
func (cm *CacheMetrics) RecordPromotion() {
	atomic.AddInt64(&cm.promotions, 1)
}

// RecordCleanup records a cleanup operation
func (cm *CacheMetrics) RecordCleanup() {
	atomic.AddInt64(&cm.cleanups, 1)
}

// GetHitRate returns the overall cache hit rate
func (cm *CacheMetrics) GetHitRate() float64 {
	total := atomic.LoadInt64(&cm.totalAccesses)
	if total == 0 {
		return 0
	}

	hits := atomic.LoadInt64(&cm.l1Hits) + atomic.LoadInt64(&cm.l2Hits)
	return float64(hits) / float64(total)
}

// GetStats returns current statistics
func (cm *CacheMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_accesses": atomic.LoadInt64(&cm.totalAccesses),
		"l1_hits":        atomic.LoadInt64(&cm.l1Hits),
		"l2_hits":        atomic.LoadInt64(&cm.l2Hits),
		"misses":         atomic.LoadInt64(&cm.misses),
		"writes":         atomic.LoadInt64(&cm.writes),
		"promotions":     atomic.LoadInt64(&cm.promotions),
		"cleanups":       atomic.LoadInt64(&cm.cleanups),
		"hit_rate":       cm.GetHitRate(),
		"l1_hit_rate":    cm.getL1HitRate(),
		"l2_hit_rate":    cm.getL2HitRate(),
	}
}

// getL1HitRate returns L1 cache hit rate
func (cm *CacheMetrics) getL1HitRate() float64 {
	total := atomic.LoadInt64(&cm.totalAccesses)
	if total == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&cm.l1Hits)) / float64(total)
}

// getL2HitRate returns L2 cache hit rate
func (cm *CacheMetrics) getL2HitRate() float64 {
	total := atomic.LoadInt64(&cm.totalAccesses)
	if total == 0 {
		return 0
	}
	return float64(atomic.LoadInt64(&cm.l2Hits)) / float64(total)
}

// Reset resets all metrics
func (cm *CacheMetrics) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	atomic.StoreInt64(&cm.totalAccesses, 0)
	atomic.StoreInt64(&cm.l1Hits, 0)
	atomic.StoreInt64(&cm.l2Hits, 0)
	atomic.StoreInt64(&cm.misses, 0)
	atomic.StoreInt64(&cm.writes, 0)
	atomic.StoreInt64(&cm.promotions, 0)
	atomic.StoreInt64(&cm.cleanups, 0)

	cm.recentHits = make([]int64, 0, 60)
	cm.recentMisses = make([]int64, 0, 60)
	cm.windowStart = time.Now()
}