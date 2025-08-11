package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	cache := NewMemoryCache(1024) // 1KB capacity
	
	// Test Put and Get
	key := "test-key"
	value := []byte("test-value")
	
	err := cache.Put(key, value)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	
	retrieved, ok := cache.Get(key)
	if !ok {
		t.Fatal("Get failed: key not found")
	}
	
	if string(retrieved) != string(value) {
		t.Errorf("Retrieved value mismatch: got %s, want %s", retrieved, value)
	}
	
	// Test Contains
	if !cache.Contains(key) {
		t.Error("Contains returned false for existing key")
	}
	
	// Test Size
	expectedSize := int64(len(value))
	if cache.Size() != expectedSize {
		t.Errorf("Size mismatch: got %d, want %d", cache.Size(), expectedSize)
	}
	
	// Test Delete
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	
	if cache.Contains(key) {
		t.Error("Key still exists after delete")
	}
	
	if cache.Size() != 0 {
		t.Errorf("Size not zero after delete: %d", cache.Size())
	}
}

func TestMemoryCache_LRUEviction(t *testing.T) {
	cache := NewMemoryCache(100) // Small capacity for testing
	
	// Add items until capacity is reached
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := make([]byte, 20) // Each item is 20 bytes
		err := cache.Put(key, value)
		if err != nil {
			t.Fatalf("Put failed for key %s: %v", key, err)
		}
	}
	
	// Access key-0 and key-1 to make them recently used
	cache.Get("key-0")
	cache.Get("key-1")
	
	// Add new item that should trigger eviction
	newKey := "key-new"
	newValue := make([]byte, 30)
	err := cache.Put(newKey, newValue)
	if err != nil {
		t.Fatalf("Put failed for new key: %v", err)
	}
	
	// key-2 should be evicted (least recently used)
	if cache.Contains("key-2") {
		t.Error("key-2 should have been evicted")
	}
	
	// key-0 and key-1 should still exist (recently accessed)
	if !cache.Contains("key-0") {
		t.Error("key-0 should not have been evicted")
	}
	if !cache.Contains("key-1") {
		t.Error("key-1 should not have been evicted")
	}
}

func TestMemoryCache_ItemTooLarge(t *testing.T) {
	cache := NewMemoryCache(100)
	
	// Try to add item larger than capacity
	largeValue := make([]byte, 200)
	err := cache.Put("large-key", largeValue)
	
	if err != ErrItemTooLarge {
		t.Errorf("Expected ErrItemTooLarge, got %v", err)
	}
}

func TestMemoryCache_UpdateExisting(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	key := "update-key"
	value1 := []byte("original")
	value2 := []byte("updated-value")
	
	// Add original
	err := cache.Put(key, value1)
	if err != nil {
		t.Fatalf("First Put failed: %v", err)
	}
	
	// Update with new value
	err = cache.Put(key, value2)
	if err != nil {
		t.Fatalf("Update Put failed: %v", err)
	}
	
	// Check updated value
	retrieved, ok := cache.Get(key)
	if !ok {
		t.Fatal("Key not found after update")
	}
	
	if string(retrieved) != string(value2) {
		t.Errorf("Value not updated: got %s, want %s", retrieved, value2)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	// Add multiple items
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Put(key, value)
	}
	
	// Clear cache
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	
	// Check all items are gone
	if cache.Size() != 0 {
		t.Errorf("Size not zero after clear: %d", cache.Size())
	}
	
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		if cache.Contains(key) {
			t.Errorf("Key %s still exists after clear", key)
		}
	}
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	// Initial stats
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Initial stats should be zero")
	}
	
	// Add item and get it
	cache.Put("key1", []byte("value1"))
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss
	
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestMemoryCache_GetWithMetadata(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	key := "meta-key"
	value := []byte("meta-value")
	
	cache.Put(key, value)
	
	// Access multiple times to increase hits
	cache.Get(key)
	cache.Get(key)
	
	retrieved, meta, ok := cache.GetWithMetadata(key)
	if !ok {
		t.Fatal("GetWithMetadata failed")
	}
	
	if string(retrieved) != string(value) {
		t.Errorf("Value mismatch: got %s, want %s", retrieved, value)
	}
	
	if meta.Key != key {
		t.Errorf("Metadata key mismatch: got %s, want %s", meta.Key, key)
	}
	
	if meta.Size != int64(len(value)) {
		t.Errorf("Metadata size mismatch: got %d, want %d", meta.Size, len(value))
	}
	
	if meta.Hits != 3 { // 2 Gets + 1 GetWithMetadata
		t.Errorf("Metadata hits mismatch: got %d, want 3", meta.Hits)
	}
	
	if meta.Level != CacheLevelL1 {
		t.Errorf("Metadata level mismatch: got %v, want %v", meta.Level, CacheLevelL1)
	}
}

func TestMemoryCache_Prune(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	// Add items with delays to create different timestamps
	cache.Put("old-1", []byte("value1"))
	time.Sleep(10 * time.Millisecond)
	cache.Put("old-2", []byte("value2"))
	time.Sleep(10 * time.Millisecond)
	cache.Put("new-1", []byte("value3"))
	
	// Prune items older than 15ms
	pruned := cache.Prune(15 * time.Millisecond)
	
	if pruned != 2 {
		t.Errorf("Expected 2 items pruned, got %d", pruned)
	}
	
	// Old items should be gone
	if cache.Contains("old-1") || cache.Contains("old-2") {
		t.Error("Old items should have been pruned")
	}
	
	// New item should remain
	if !cache.Contains("new-1") {
		t.Error("New item should not have been pruned")
	}
}

func TestMemoryCache_GetLRUEntries(t *testing.T) {
	cache := NewMemoryCache(1024)
	
	// Add items
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Put(key, value)
		time.Sleep(5 * time.Millisecond) // Ensure different timestamps
	}
	
	// Access some items to change LRU order
	cache.Get("key-0") // Make key-0 most recently used
	cache.Get("key-1") // Make key-1 second most recently used
	
	// Get 3 LRU entries
	lruEntries := cache.GetLRUEntries(3)
	
	if len(lruEntries) != 3 {
		t.Fatalf("Expected 3 LRU entries, got %d", len(lruEntries))
	}
	
	// The LRU entries should be key-2, key-3, key-4 (not accessed)
	expectedKeys := []string{"key-2", "key-3", "key-4"}
	for i, entry := range lruEntries {
		if entry.Key != expectedKeys[i] {
			t.Errorf("LRU entry %d: expected key %s, got %s", i, expectedKeys[i], entry.Key)
		}
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(10240) // 10KB
	
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	
	// Multiple writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("writer-%d-key-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))
				if err := cache.Put(key, value); err != nil {
					errors <- fmt.Errorf("writer %d: %v", id, err)
				}
			}
		}(i)
	}
	
	// Multiple readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("writer-%d-key-%d", id, j)
				// Some reads might miss if write hasn't happened yet
				cache.Get(key)
			}
		}(i)
	}
	
	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Success
	case err := <-errors:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestMemoryCache_Resize(t *testing.T) {
	cache := NewMemoryCache(100)
	
	// Fill cache
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := make([]byte, 20)
		cache.Put(key, value)
	}
	
	// Resize to smaller capacity
	cache.Resize(50)
	
	// Cache should have evicted items to fit new capacity
	if cache.Size() > 50 {
		t.Errorf("Size exceeds new capacity: %d > 50", cache.Size())
	}
	
	// Resize to larger capacity
	cache.Resize(200)
	
	// Should be able to add more items now
	err := cache.Put("new-key", make([]byte, 100))
	if err != nil {
		t.Errorf("Failed to add item after resize: %v", err)
	}
}

// Benchmark tests
func BenchmarkMemoryCache_Put(b *testing.B) {
	cache := NewMemoryCache(1024 * 1024) // 1MB
	value := make([]byte, 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Put(key, value)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache(1024 * 1024)
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := make([]byte, 100)
		cache.Put(key, value)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkMemoryCache_ConcurrentPutGet(b *testing.B) {
	cache := NewMemoryCache(1024 * 1024)
	value := make([]byte, 100)
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			if i%2 == 0 {
				cache.Put(key, value)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}