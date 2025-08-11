package cache

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
)

// DiskCache implements an L2 disk-based cache with optional compression.
// It provides persistent storage for audio data across sessions.
type DiskCache struct {
	basePath string
	capacity int64 // Maximum size in bytes
	size     int64 // Current size in bytes

	// Compression
	compressionLevel int
	encoder          *zstd.Encoder
	decoder          *zstd.Decoder

	// Index for fast lookups
	index map[string]*diskCacheEntry

	// Synchronization
	mu sync.RWMutex

	// Metrics
	stats CacheStats

	// Configuration
	enableCompression bool
}

// diskCacheEntry represents an entry in the disk cache index
type diskCacheEntry struct {
	Key          string
	FilePath     string
	Size         int64 // Size on disk (compressed)
	OriginalSize int64 // Original size (uncompressed)
	Timestamp    time.Time
	LastAccess   time.Time
	Hits         int64
	Compressed   bool
}

// NewDiskCache creates a new disk cache with the specified path and capacity.
func NewDiskCache(basePath string, capacity int64, compressionLevel int) (*DiskCache, error) {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	dc := &DiskCache{
		basePath:          basePath,
		capacity:          capacity,
		compressionLevel:  compressionLevel,
		index:             make(map[string]*diskCacheEntry),
		enableCompression: compressionLevel > 0,
		stats: CacheStats{
			Capacity: capacity,
		},
	}

	// Initialize compression if enabled
	if dc.enableCompression {
		var err error
		dc.encoder, err = zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compressionLevel)))
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
		}

		dc.decoder, err = zstd.NewReader(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
		}
	}

	// Load existing cache index
	if err := dc.loadIndex(); err != nil {
		// Non-fatal: just start with empty index
		dc.index = make(map[string]*diskCacheEntry)
	}

	// Calculate current size
	dc.calculateSize()

	return dc, nil
}

// Get retrieves a value from the disk cache.
func (dc *DiskCache) Get(key string) ([]byte, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	entry, ok := dc.index[key]
	if !ok {
		dc.stats.Misses++
		return nil, false
	}

	// Read from disk
	data, err := os.ReadFile(entry.FilePath)
	if err != nil {
		// File missing or corrupted, remove from index
		delete(dc.index, key)
		dc.size -= entry.Size
		dc.stats.Misses++
		return nil, false
	}

	// Decompress if needed
	if entry.Compressed && dc.enableCompression {
		decompressed, err := dc.decoder.DecodeAll(data, nil)
		if err != nil {
			// Decompression failed, remove from cache
			delete(dc.index, key)
			os.Remove(entry.FilePath)
			dc.size -= entry.Size
			dc.stats.Misses++
			return nil, false
		}
		data = decompressed
	}

	// Update access time and hits
	entry.LastAccess = time.Now()
	entry.Hits++

	dc.stats.Hits++
	dc.stats.LastAccess = time.Now()

	return data, true
}

// Put stores a value in the disk cache.
func (dc *DiskCache) Put(key string, value []byte) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	originalSize := int64(len(value))

	// Compress if enabled
	var dataToWrite []byte
	var compressed bool
	if dc.enableCompression && originalSize > 1024 { // Only compress if > 1KB
		compressedData := dc.encoder.EncodeAll(value, nil)
		// Only use compression if it actually reduces size
		if len(compressedData) < len(value) {
			dataToWrite = compressedData
			compressed = true
		} else {
			dataToWrite = value
		}
	} else {
		dataToWrite = value
	}

	diskSize := int64(len(dataToWrite))

	// Check if key already exists
	if existing, ok := dc.index[key]; ok {
		// Update existing entry
		dc.size -= existing.Size
		os.Remove(existing.FilePath)
	}

	// Check capacity
	if diskSize > dc.capacity {
		return ErrItemTooLarge
	}

	// Evict items if necessary
	for dc.size+diskSize > dc.capacity && len(dc.index) > 0 {
		dc.evictOldest()
	}

	// Generate file path
	filePath := dc.generateFilePath(key)

	// Write to disk
	if err := dc.writeFile(filePath, dataToWrite); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Update index
	entry := &diskCacheEntry{
		Key:          key,
		FilePath:     filePath,
		Size:         diskSize,
		OriginalSize: originalSize,
		Timestamp:    time.Now(),
		LastAccess:   time.Now(),
		Hits:         0,
		Compressed:   compressed,
	}

	dc.index[key] = entry
	dc.size += diskSize

	dc.stats.Size = dc.size
	dc.stats.ItemCount = int64(len(dc.index))

	return nil
}

// Delete removes an entry from the disk cache.
func (dc *DiskCache) Delete(key string) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	entry, ok := dc.index[key]
	if !ok {
		return nil
	}

	// Remove file
	os.Remove(entry.FilePath)

	// Update index and size
	delete(dc.index, key)
	dc.size -= entry.Size

	dc.stats.Size = dc.size
	dc.stats.ItemCount = int64(len(dc.index))

	return nil
}

// Clear removes all entries from the disk cache.
func (dc *DiskCache) Clear() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Remove all cache files
	for _, entry := range dc.index {
		os.Remove(entry.FilePath)
	}

	// Clear index
	dc.index = make(map[string]*diskCacheEntry)
	dc.size = 0

	dc.stats.Size = 0
	dc.stats.ItemCount = 0

	// Save empty index
	return dc.saveIndex()
}

// Size returns the current cache size in bytes.
func (dc *DiskCache) Size() int64 {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return dc.size
}

// Stats returns cache statistics.
func (dc *DiskCache) Stats() CacheStats {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	stats := dc.stats
	stats.Size = dc.size
	stats.ItemCount = int64(len(dc.index))

	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}

	return stats
}

// Contains checks if a key exists in the cache without updating access time.
func (dc *DiskCache) Contains(key string) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	_, ok := dc.index[key]
	return ok
}

// GetWithMetadata retrieves a value along with its metadata.
func (dc *DiskCache) GetWithMetadata(key string) ([]byte, CacheMetadata, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	entry, ok := dc.index[key]
	if !ok {
		dc.stats.Misses++
		return nil, CacheMetadata{}, false
	}

	// Read from disk
	data, err := os.ReadFile(entry.FilePath)
	if err != nil {
		delete(dc.index, key)
		dc.size -= entry.Size
		dc.stats.Misses++
		return nil, CacheMetadata{}, false
	}

	// Decompress if needed
	if entry.Compressed && dc.enableCompression {
		decompressed, err := dc.decoder.DecodeAll(data, nil)
		if err != nil {
			delete(dc.index, key)
			os.Remove(entry.FilePath)
			dc.size -= entry.Size
			dc.stats.Misses++
			return nil, CacheMetadata{}, false
		}
		data = decompressed
	}

	// Update access time and hits
	entry.LastAccess = time.Now()
	entry.Hits++

	dc.stats.Hits++

	metadata := CacheMetadata{
		Key:        entry.Key,
		Size:       entry.OriginalSize,
		Timestamp:  entry.Timestamp,
		LastAccess: entry.LastAccess,
		Hits:       entry.Hits,
		Level:      CacheLevelL2,
	}
	metadata.CalculateEvictionScore()

	return data, metadata, true
}

// RemoveOlderThan removes entries older than the specified time.
func (dc *DiskCache) RemoveOlderThan(cutoff time.Time) int {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	removed := 0
	for key, entry := range dc.index {
		if entry.Timestamp.Before(cutoff) {
			os.Remove(entry.FilePath)
			dc.size -= entry.Size
			delete(dc.index, key)
			removed++
		}
	}

	dc.stats.Size = dc.size
	dc.stats.ItemCount = int64(len(dc.index))

	return removed
}

// EvictLRU evicts least recently used items to free space.
func (dc *DiskCache) EvictLRU() int {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	targetSize := dc.capacity * 90 / 100 // Free to 90% capacity
	evicted := 0

	for dc.size > targetSize && len(dc.index) > 0 {
		dc.evictOldest()
		evicted++
	}

	return evicted
}

// GetLRUEntries returns the n least recently used entries.
func (dc *DiskCache) GetLRUEntries(n int) []CacheMetadata {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// Convert index to slice for sorting
	entries := make([]*diskCacheEntry, 0, len(dc.index))
	for _, entry := range dc.index {
		entries = append(entries, entry)
	}

	// Sort by last access time (oldest first)
	// Note: In production, use sort.Slice for efficiency
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LastAccess.After(entries[j].LastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Return up to n entries
	result := make([]CacheMetadata, 0, n)
	for i := 0; i < n && i < len(entries); i++ {
		entry := entries[i]
		meta := CacheMetadata{
			Key:        entry.Key,
			Size:       entry.OriginalSize,
			Timestamp:  entry.Timestamp,
			LastAccess: entry.LastAccess,
			Hits:       entry.Hits,
			Level:      CacheLevelL2,
		}
		meta.CalculateEvictionScore()
		result = append(result, meta)
	}

	return result
}

// Private helper methods

func (dc *DiskCache) generateFilePath(key string) string {
	// Use SHA256 hash of key for filename
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:16]) + ".cache"
	return filepath.Join(dc.basePath, filename)
}

func (dc *DiskCache) writeFile(path string, data []byte) error {
	// Write to temp file first, then rename (atomic on most systems)
	tempPath := path + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	closeErr := file.Close()

	if err != nil {
		os.Remove(tempPath)
		return err
	}
	if closeErr != nil {
		os.Remove(tempPath)
		return closeErr
	}

	// Atomic rename
	return os.Rename(tempPath, path)
}

func (dc *DiskCache) evictOldest() {
	// Find oldest entry by last access time
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range dc.index {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		entry := dc.index[oldestKey]
		os.Remove(entry.FilePath)
		dc.size -= entry.Size
		delete(dc.index, oldestKey)
		dc.stats.Evictions++
		dc.stats.LastEvict = time.Now()
	}
}

func (dc *DiskCache) loadIndex() error {
	indexPath := filepath.Join(dc.basePath, "cache.index")

	file, err := os.Open(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No index file yet
		}
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(&dc.index)
}

func (dc *DiskCache) saveIndex() error {
	indexPath := filepath.Join(dc.basePath, "cache.index")
	tempPath := indexPath + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(dc.index)
	closeErr := file.Close()

	if err != nil {
		os.Remove(tempPath)
		return err
	}
	if closeErr != nil {
		os.Remove(tempPath)
		return closeErr
	}

	return os.Rename(tempPath, indexPath)
}

func (dc *DiskCache) calculateSize() {
	dc.size = 0
	for _, entry := range dc.index {
		dc.size += entry.Size
	}
	dc.stats.Size = dc.size
	dc.stats.ItemCount = int64(len(dc.index))
}

// Close closes the disk cache, saving the index.
func (dc *DiskCache) Close() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return dc.saveIndex()
}
