package cache

import (
	"errors"
	"time"
)

// Common errors for cache operations
var (
	// ErrItemTooLarge is returned when an item exceeds the cache capacity
	ErrItemTooLarge = errors.New("item too large for cache")

	// ErrCacheMiss is returned when an item is not found in cache
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheCorrupted is returned when cache data is corrupted
	ErrCacheCorrupted = errors.New("cache data corrupted")
)

// CacheLevel represents the cache tier
type CacheLevel int

const (
	// CacheLevelL1 represents the memory cache (fastest)
	CacheLevelL1 CacheLevel = iota

	// CacheLevelL2 represents the disk cache (persistent)
	CacheLevelL2

	// CacheLevelSession represents the session-specific cache
	CacheLevelSession
)

// String returns the string representation of the cache level
func (l CacheLevel) String() string {
	switch l {
	case CacheLevelL1:
		return "L1-Memory"
	case CacheLevelL2:
		return "L2-Disk"
	case CacheLevelSession:
		return "Session"
	default:
		return "Unknown"
	}
}

// CacheStats holds cache performance metrics
type CacheStats struct {
	// Configuration
	Capacity int64 // Maximum capacity in bytes

	// Current state
	Size      int64 // Current size in bytes
	ItemCount int64 // Number of items in cache

	// Performance metrics
	Hits      int64   // Number of cache hits
	Misses    int64   // Number of cache misses
	Evictions int64   // Number of evictions
	HitRate   float64 // Calculated hit rate (hits / (hits + misses))

	// Timing
	LastAccess time.Time     // Last access time
	LastEvict  time.Time     // Last eviction time
	AvgLatency time.Duration // Average access latency
}

// CacheMetadata contains metadata about a cached item
type CacheMetadata struct {
	Key        string     // Cache key
	Size       int64      // Size in bytes
	Timestamp  time.Time  // When item was cached
	LastAccess time.Time  // Last access time
	Hits       int64      // Number of times accessed
	Level      CacheLevel // Which cache level this is from

	// For smart eviction scoring
	Score float64 // Eviction score (lower = more likely to evict)
}

// CalculateEvictionScore calculates the eviction score for smart eviction
// Score = (age_hours * size_mb) / (hits + 1)
// Lower score = more likely to evict
func (m *CacheMetadata) CalculateEvictionScore() float64 {
	age := time.Since(m.Timestamp).Hours()
	sizeMB := float64(m.Size) / (1024 * 1024)
	hits := float64(m.Hits + 1) // +1 to avoid division by zero

	m.Score = (age * sizeMB) / hits
	return m.Score
}

// CacheConfig holds configuration for cache instances
type CacheConfig struct {
	// Memory cache (L1)
	MemoryCapacity int64 // Bytes

	// Disk cache (L2)
	DiskCapacity     int64  // Bytes
	DiskPath         string // Directory for cache files
	CompressionLevel int    // Zstd compression level (1-22, default 3)

	// Session cache
	SessionCapacity int64 // Bytes

	// Cleanup settings
	TTLDays         int           // Days before items expire (default 7)
	CleanupInterval time.Duration // How often to run cleanup (default 1 hour)

	// Performance tuning
	EnableMetrics     bool // Track detailed metrics
	EnableCompression bool // Enable disk compression
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MemoryCapacity:    100 * 1024 * 1024,  // 100MB
		DiskCapacity:      1024 * 1024 * 1024, // 1GB
		SessionCapacity:   50 * 1024 * 1024,   // 50MB
		TTLDays:           7,
		CleanupInterval:   time.Hour,
		CompressionLevel:  3, // Balanced compression
		EnableMetrics:     true,
		EnableCompression: true,
	}
}

// CacheKey represents a cache key with its components
type CacheKey struct {
	Text     string
	Voice    string
	Speed    float64
	Language string
}

// Cache defines the interface for cache implementations
type Cache interface {
	// Basic operations
	Get(key string) ([]byte, bool)
	Put(key string, value []byte) error
	Delete(key string) error
	Clear() error

	// Size management
	Size() int64
	Contains(key string) bool

	// Statistics
	Stats() CacheStats
}

// ExtendedCache defines additional operations for advanced caches
type ExtendedCache interface {
	Cache

	// Advanced operations
	GetWithMetadata(key string) ([]byte, CacheMetadata, bool)
	GetLRUEntries(n int) []CacheMetadata
	Prune(maxAge time.Duration) int
	Keys() []string
}
