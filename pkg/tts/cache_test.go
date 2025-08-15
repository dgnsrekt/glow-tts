package tts

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		voice  string
		speed  float64
		text2  string
		voice2 string
		speed2 float64
		same   bool
	}{
		{
			name:   "identical inputs",
			text:   "Hello world",
			voice:  "en-US",
			speed:  1.0,
			text2:  "Hello world",
			voice2: "en-US",
			speed2: 1.0,
			same:   true,
		},
		{
			name:   "different text",
			text:   "Hello world",
			voice:  "en-US",
			speed:  1.0,
			text2:  "Hello World",
			voice2: "en-US",
			speed2: 1.0,
			same:   false,
		},
		{
			name:   "different voice",
			text:   "Hello world",
			voice:  "en-US",
			speed:  1.0,
			text2:  "Hello world",
			voice2: "en-GB",
			speed2: 1.0,
			same:   false,
		},
		{
			name:   "different speed",
			text:   "Hello world",
			voice:  "en-US",
			speed:  1.0,
			text2:  "Hello world",
			voice2: "en-US",
			speed2: 1.5,
			same:   false,
		},
		{
			name:   "speed normalization",
			text:   "Hello world",
			voice:  "en-US",
			speed:  1.50,
			text2:  "Hello world",
			voice2: "en-US",
			speed2: 1.5,
			same:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := GenerateCacheKey(tt.text, tt.voice, tt.speed)
			key2 := GenerateCacheKey(tt.text2, tt.voice2, tt.speed2)

			if tt.same && key1 != key2 {
				t.Errorf("Expected same keys, got different: %s != %s", key1, key2)
			}
			if !tt.same && key1 == key2 {
				t.Errorf("Expected different keys, got same: %s", key1)
			}

			// Check key format
			if len(key1) != 67 { // "v1_" + 64 hex chars
				t.Errorf("Invalid key length: %d", len(key1))
			}
			if key1[:3] != "v1_" {
				t.Errorf("Key should start with version prefix")
			}
		})
	}
}

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(1024*1024, time.Hour) // 1MB limit

	t.Run("basic operations", func(t *testing.T) {
		// Test Put and Get
		data := &AudioData{
			Audio:    []byte("test audio data"),
			Text:     "test text",
			Voice:    "test-voice",
			Speed:    1.0,
			CacheKey: "test-key",
		}

		err := cache.Put("key1", data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		retrieved, err := cache.Get("key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if string(retrieved.Audio) != string(data.Audio) {
			t.Error("Retrieved data doesn't match")
		}

		// Test miss
		_, err = cache.Get("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent key")
		}
	})

	t.Run("LRU eviction", func(t *testing.T) {
		// Test that cache enforces size limits
		smallCache := NewMemoryCache(1000, time.Hour) // 1KB cache

		// Fill cache with data
		totalSize := int64(0)
		for i := 0; i < 10; i++ {
			data := &AudioData{
				Audio: make([]byte, 200), // 200 bytes each
				Text:  fmt.Sprintf("text%d", i),
			}
			_ = smallCache.Put(fmt.Sprintf("key%d", i), data)
			totalSize += int64(len(data.Audio)) + 256 // account for overhead
		}

		// Cache size should be under limit
		if smallCache.Size() > 1000 {
			t.Errorf("Cache size %d exceeds limit 1000", smallCache.Size())
		}

		// At least some items should have been evicted
		found := 0
		for i := 0; i < 10; i++ {
			if _, err := smallCache.Get(fmt.Sprintf("key%d", i)); err == nil {
				found++
			}
		}

		if found == 10 {
			t.Error("No items were evicted despite exceeding size limit")
		}

		if found == 0 {
			t.Error("All items were evicted, cache is empty")
		}

		t.Logf("Cache retained %d out of 10 items after eviction", found)
	})

	t.Run("TTL expiration", func(t *testing.T) {
		shortTTLCache := NewMemoryCache(1024, 100*time.Millisecond)

		data := &AudioData{
			Audio: []byte("test"),
		}
		_ = shortTTLCache.Put("ttl-key", data)

		// Should exist immediately
		_, err := shortTTLCache.Get("ttl-key")
		if err != nil {
			t.Error("Key should exist immediately after put")
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired
		_, err = shortTTLCache.Get("ttl-key")
		if err == nil {
			t.Error("Key should have expired")
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		var wg sync.WaitGroup
		concurrent := 100

		for i := 0; i < concurrent; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("concurrent-%d", id)
				data := &AudioData{
					Audio: []byte(fmt.Sprintf("data-%d", id)),
				}
				_ = cache.Put(key, data)
				_, _ = cache.Get(key)
			}(i)
		}

		wg.Wait()
		// If we get here without deadlock/panic, concurrent access works
	})
}

func TestDiskCache(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewDiskCache(tempDir, 1024*1024, time.Hour) // 1MB limit
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	defer cache.Close()

	t.Run("basic operations", func(t *testing.T) {
		data := &AudioData{
			Audio:    []byte("test audio data for disk"),
			Text:     "disk test",
			Voice:    "disk-voice",
			Speed:    1.5,
			CacheKey: "disk-key",
		}

		err := cache.Put("disk1", data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		retrieved, err := cache.Get("disk1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if string(retrieved.Audio) != string(data.Audio) {
			t.Error("Retrieved data doesn't match")
		}
		if retrieved.Speed != data.Speed {
			t.Error("Metadata doesn't match")
		}
	})

	t.Run("persistence", func(t *testing.T) {
		// Store data
		persistData := &AudioData{
			Audio: []byte("persistent data"),
			Text:  "persist test",
		}
		_ = cache.Put("persist-key", persistData)

		// Close and reopen cache
		cache.Close()
		cache2, err := NewDiskCache(tempDir, 1024*1024, time.Hour)
		if err != nil {
			t.Fatalf("Failed to reopen cache: %v", err)
		}
		defer cache2.Close()

		// Data should still exist
		retrieved, err := cache2.Get("persist-key")
		if err != nil {
			t.Fatalf("Failed to get persistent data: %v", err)
		}

		if string(retrieved.Audio) != string(persistData.Audio) {
			t.Error("Persistent data doesn't match")
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		// Store data and check file permissions
		_ = cache.Put("perm-key", &AudioData{Audio: []byte("test")})
		audioFile := filepath.Join(tempDir, "perm-key.audio")
		info, err := os.Stat(audioFile)
		if err != nil {
			t.Fatalf("Failed to stat audio file: %v", err)
		}

		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Audio file has wrong permissions: %o", mode)
		}
	})
}

func TestTTSCacheManager(t *testing.T) {
	config := &CacheConfig{
		L1SizeLimit:     1024,
		L2SizeLimit:     10240,
		L1TTL:           time.Hour,
		L2TTL:           time.Hour,
		CleanupInterval: time.Minute,
		EnableMetrics:   true,
		CacheDir:        t.TempDir(),
	}

	manager, err := NewTTSCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	t.Run("two-level caching", func(t *testing.T) {
		data := &AudioData{
			Audio: []byte("test audio"),
			Text:  "test",
		}

		// Put data
		err := manager.Put("test-key", data)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Get should hit L1
		retrieved, err := manager.Get("test-key")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if string(retrieved.Audio) != string(data.Audio) {
			t.Error("Data doesn't match")
		}

		// Check metrics
		metrics := manager.GetMetrics()
		stats := metrics.GetStats()
		if stats["l1_hits"].(int64) != 1 {
			t.Error("Expected L1 hit")
		}
	})

	t.Run("L2 to L1 promotion", func(t *testing.T) {
		// Clear L1 cache
		manager.l1Cache.Clear()

		// Data should still be in L2
		retrieved, err := manager.Get("test-key")
		if err != nil {
			t.Fatalf("Get from L2 failed: %v", err)
		}

		if retrieved == nil {
			t.Error("Expected data from L2")
		}

		// Check metrics for L2 hit and promotion
		metrics := manager.GetMetrics()
		stats := metrics.GetStats()
		if stats["l2_hits"].(int64) < 1 {
			t.Error("Expected L2 hit")
		}
		if stats["promotions"].(int64) < 1 {
			t.Error("Expected promotion")
		}

		// Next access should hit L1
		_, _ = manager.Get("test-key")
		stats = metrics.GetStats()
		if stats["l1_hits"].(int64) < 2 {
			t.Error("Expected L1 hit after promotion")
		}
	})

	t.Run("cache miss", func(t *testing.T) {
		_, err := manager.Get("nonexistent-key")
		if err == nil {
			t.Error("Expected error for nonexistent key")
		}

		metrics := manager.GetMetrics()
		stats := metrics.GetStats()
		if stats["misses"].(int64) < 1 {
			t.Error("Expected cache miss to be recorded")
		}
	})
}

func TestCacheMetrics(t *testing.T) {
	metrics := NewCacheMetrics()

	// Record various events
	metrics.RecordAccess()
	metrics.RecordL1Hit()
	metrics.RecordAccess()
	metrics.RecordL2Hit()
	metrics.RecordAccess()
	metrics.RecordMiss()
	metrics.RecordWrite()
	metrics.RecordPromotion()
	metrics.RecordCleanup()

	stats := metrics.GetStats()

	if stats["total_accesses"].(int64) != 3 {
		t.Errorf("Expected 3 accesses, got %d", stats["total_accesses"])
	}

	if stats["l1_hits"].(int64) != 1 {
		t.Errorf("Expected 1 L1 hit, got %d", stats["l1_hits"])
	}

	if stats["l2_hits"].(int64) != 1 {
		t.Errorf("Expected 1 L2 hit, got %d", stats["l2_hits"])
	}

	if stats["misses"].(int64) != 1 {
		t.Errorf("Expected 1 miss, got %d", stats["misses"])
	}

	hitRate := metrics.GetHitRate()
	expectedRate := 2.0 / 3.0
	if hitRate != expectedRate {
		t.Errorf("Expected hit rate %.2f, got %.2f", expectedRate, hitRate)
	}

	// Test reset
	metrics.Reset()
	stats = metrics.GetStats()
	if stats["total_accesses"].(int64) != 0 {
		t.Error("Metrics not reset")
	}
}

func BenchmarkCacheKeyGeneration(b *testing.B) {
	text := "This is a sample text for benchmarking cache key generation"
	voice := "en-US-neural"
	speed := 1.25

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateCacheKey(text, voice, speed)
	}
}

func BenchmarkMemoryCacheAccess(b *testing.B) {
	cache := NewMemoryCache(10*1024*1024, time.Hour) // 10MB

	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		data := &AudioData{
			Audio: make([]byte, 1024), // 1KB each
			Text:  fmt.Sprintf("text-%d", i),
		}
		_ = cache.Put(fmt.Sprintf("key-%d", i), data)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			_, _ = cache.Get(key)
			i++
		}
	})
}

func BenchmarkDiskCacheAccess(b *testing.B) {
	tempDir := b.TempDir()
	cache, _ := NewDiskCache(tempDir, 10*1024*1024, time.Hour)
	defer cache.Close()

	// Pre-populate cache
	for i := 0; i < 100; i++ {
		data := &AudioData{
			Audio: make([]byte, 10*1024), // 10KB each
			Text:  fmt.Sprintf("text-%d", i),
		}
		_ = cache.Put(fmt.Sprintf("key-%d", i), data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		_, _ = cache.Get(key)
	}
}

func TestCacheCleanup(t *testing.T) {
	t.Run("memory cache cleanup", func(t *testing.T) {
		cache := NewMemoryCache(1024, 100*time.Millisecond)

		// Add data
		for i := 0; i < 5; i++ {
			data := &AudioData{
				Audio: []byte(fmt.Sprintf("data-%d", i)),
			}
			_ = cache.Put(fmt.Sprintf("key-%d", i), data)
		}

		// Wait for TTL
		time.Sleep(150 * time.Millisecond)

		// Run cleanup
		cache.cleanup()

		// All entries should be gone
		for i := 0; i < 5; i++ {
			_, err := cache.Get(fmt.Sprintf("key-%d", i))
			if err == nil {
				t.Error("Expected expired entry to be cleaned up")
			}
		}
	})

	t.Run("disk cache size limit", func(t *testing.T) {
		tempDir := t.TempDir()
		cache, _ := NewDiskCache(tempDir, 1000, time.Hour) // Very small limit
		defer cache.Close()

		// Add data exceeding limit
		for i := 0; i < 10; i++ {
			data := &AudioData{
				Audio: make([]byte, 200), // 200 bytes each
			}
			_ = cache.Put(fmt.Sprintf("key-%d", i), data)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Run cleanup
		cache.cleanup()

		// Size should be under limit
		if cache.Size() > 1000 {
			t.Errorf("Cache size %d exceeds limit", cache.Size())
		}

		// Newer entries should remain
		_, err := cache.Get("key-9")
		if err != nil {
			t.Error("Newest entry should not be evicted")
		}
	})
}

func TestConcurrentTTSCacheManager(t *testing.T) {
	// Skip index saves during heavy concurrent testing
	os.Setenv("GO_TEST_FAST", "1")
	defer os.Unsetenv("GO_TEST_FAST")
	
	config := DefaultCacheConfig()
	config.CacheDir = t.TempDir()
	
	manager, err := NewTTSCacheManager(config)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				data := &AudioData{
					Audio: []byte(fmt.Sprintf("data-%d-%d", id, j)),
					Text:  fmt.Sprintf("text-%d-%d", id, j),
				}
				_ = manager.Put(key, data)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, _ = manager.Get(key)
			}
		}(i)
	}

	wg.Wait()

	// Check metrics
	if manager.metrics != nil {
		stats := manager.metrics.GetStats()
		t.Logf("Concurrent test stats: %+v", stats)
		
		if stats["total_accesses"].(int64) == 0 {
			t.Error("No accesses recorded")
		}
	}
}