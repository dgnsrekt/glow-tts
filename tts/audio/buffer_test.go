package audio_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/audio"
)

// createTestAudio creates a test audio with given index.
func createTestAudio(index int) *tts.Audio {
	return &tts.Audio{
		Data:       make([]byte, 1024),
		Format:     tts.FormatPCM16,
		SampleRate: 44100,
		Channels:   2,
		Duration:   time.Second,
	}
}

// TestBufferCreation tests buffer creation with various configurations.
func TestBufferCreation(t *testing.T) {
	tests := []struct {
		name   string
		config audio.BufferConfig
		want   int // expected capacity
	}{
		{
			name:   "default config",
			config: audio.DefaultBufferConfig(),
			want:   10,
		},
		{
			name: "custom capacity",
			config: audio.BufferConfig{
				Capacity:      20,
				MaxItemAge:    10 * time.Minute,
				EnableMetrics: true,
				PreAllocate:   false,
			},
			want: 20,
		},
		{
			name: "zero capacity uses default",
			config: audio.BufferConfig{
				Capacity: 0,
			},
			want: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := audio.NewBuffer(tt.config)
			if buf == nil {
				t.Fatal("NewBuffer returned nil")
			}
			if buf.Capacity() != tt.want {
				t.Errorf("Capacity() = %d, want %d", buf.Capacity(), tt.want)
			}
			if buf.Size() != 0 {
				t.Errorf("Initial Size() = %d, want 0", buf.Size())
			}
			if !buf.IsEmpty() {
				t.Error("New buffer should be empty")
			}
		})
	}
}

// TestBufferAddAndGet tests adding and retrieving items.
func TestBufferAddAndGet(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)

	// Add items
	for i := 0; i < 3; i++ {
		audio := createTestAudio(i)
		err := buf.Add(audio, i)
		if err != nil {
			t.Errorf("Add failed for item %d: %v", i, err)
		}
	}

	// Check size
	if buf.Size() != 3 {
		t.Errorf("Size() = %d, want 3", buf.Size())
	}

	// Get items by index
	for i := 0; i < 3; i++ {
		audio, found := buf.Get(i)
		if !found {
			t.Errorf("Get(%d) not found", i)
		}
		if audio == nil {
			t.Errorf("Get(%d) returned nil audio", i)
		}
	}

	// Try to get non-existent item
	_, found := buf.Get(99)
	if found {
		t.Error("Get(99) should not be found")
	}
}

// TestBufferGetNext tests FIFO retrieval.
func TestBufferGetNext(t *testing.T) {
	config := audio.BufferConfig{
		Capacity: 5,
	}
	buf := audio.NewBuffer(config)

	// Add items
	items := []int{10, 20, 30}
	for _, idx := range items {
		audio := createTestAudio(idx)
		buf.Add(audio, idx)
	}

	// Get items in FIFO order
	for _, expectedIdx := range items {
		audio, idx, err := buf.GetNext()
		if err != nil {
			t.Errorf("GetNext failed: %v", err)
		}
		if idx != expectedIdx {
			t.Errorf("GetNext index = %d, want %d", idx, expectedIdx)
		}
		if audio == nil {
			t.Error("GetNext returned nil audio")
		}
	}

	// Buffer should be empty now
	if !buf.IsEmpty() {
		t.Error("Buffer should be empty after retrieving all items")
	}

	// GetNext on empty buffer should wait (or return error if non-blocking)
	// We'll test with a timeout
	done := make(chan bool)
	go func() {
		_, _, err := buf.GetNext()
		done <- (err != nil)
	}()

	select {
	case <-done:
		// GetNext returned (likely with error or empty result)
	case <-time.After(100 * time.Millisecond):
		// GetNext is blocking as expected
	}
}

// TestBufferPeek tests peeking without removal.
func TestBufferPeek(t *testing.T) {
	config := audio.BufferConfig{
		Capacity: 5,
	}
	buf := audio.NewBuffer(config)

	// Peek empty buffer
	_, _, found := buf.Peek()
	if found {
		t.Error("Peek should return false for empty buffer")
	}

	// Add item
	testAudio := createTestAudio(42)
	buf.Add(testAudio, 42)

	// Peek should return item without removing
	audio, idx, found := buf.Peek()
	if !found {
		t.Error("Peek should find item")
	}
	if idx != 42 {
		t.Errorf("Peek index = %d, want 42", idx)
	}
	if audio == nil {
		t.Error("Peek returned nil audio")
	}

	// Size should remain unchanged
	if buf.Size() != 1 {
		t.Errorf("Size after Peek = %d, want 1", buf.Size())
	}
}

// TestBufferOverflowPolicies tests different overflow handling.
func TestBufferOverflowPolicies(t *testing.T) {
	t.Run("OverflowDrop", func(t *testing.T) {
		config := audio.BufferConfig{
			Capacity:      3,
			EnableMetrics: true,
		}
		buf := audio.NewBuffer(config)
		buf.SetOverflowPolicy(audio.OverflowDrop)

		// Fill buffer
		for i := 0; i < 5; i++ {
			err := buf.Add(createTestAudio(i), i)
			if err != nil {
				t.Errorf("Add failed: %v", err)
			}
		}

		// Should have dropped oldest items
		if buf.Size() != 3 {
			t.Errorf("Size = %d, want 3", buf.Size())
		}

		// Check stats
		stats := buf.GetStats()
		if stats.TotalDropped == 0 {
			t.Error("Expected dropped items in stats")
		}
	})

	t.Run("OverflowReject", func(t *testing.T) {
		config := audio.BufferConfig{
			Capacity: 3,
		}
		buf := audio.NewBuffer(config)
		buf.SetOverflowPolicy(audio.OverflowReject)

		// Fill buffer
		for i := 0; i < 3; i++ {
			err := buf.Add(createTestAudio(i), i)
			if err != nil {
				t.Errorf("Add failed for item %d: %v", i, err)
			}
		}

		// Next add should be rejected
		err := buf.Add(createTestAudio(99), 99)
		if err == nil {
			t.Error("Expected error when buffer full with OverflowReject")
		}
	})

	t.Run("OverflowBlock", func(t *testing.T) {
		config := audio.BufferConfig{
			Capacity: 2,
		}
		buf := audio.NewBuffer(config)
		buf.SetOverflowPolicy(audio.OverflowBlock)

		// Fill buffer
		for i := 0; i < 2; i++ {
			buf.Add(createTestAudio(i), i)
		}

		// Try to add when full (should block)
		done := make(chan bool)
		go func() {
			err := buf.Add(createTestAudio(99), 99)
			done <- (err == nil)
		}()

		// Should be blocking
		select {
		case <-done:
			t.Error("Add should block when buffer is full")
		case <-time.After(100 * time.Millisecond):
			// Good, it's blocking
		}

		// Remove an item to unblock
		buf.GetNext()

		// Now the add should complete
		select {
		case success := <-done:
			if !success {
				t.Error("Add should succeed after space available")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Add should unblock after space available")
		}
	})
}

// TestBufferClear tests clearing the buffer.
func TestBufferClear(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)

	// Add items
	for i := 0; i < 3; i++ {
		buf.Add(createTestAudio(i), i)
	}

	// Clear buffer
	buf.Clear()

	// Check state
	if buf.Size() != 0 {
		t.Errorf("Size after Clear = %d, want 0", buf.Size())
	}
	if !buf.IsEmpty() {
		t.Error("Buffer should be empty after Clear")
	}

	// Stats should reflect clear
	stats := buf.GetStats()
	if stats.CurrentSize != 0 {
		t.Errorf("Stats CurrentSize = %d, want 0", stats.CurrentSize)
	}
}

// TestBufferEvictOldItems tests age-based eviction.
func TestBufferEvictOldItems(t *testing.T) {
	config := audio.BufferConfig{
		Capacity: 5,
	}
	buf := audio.NewBuffer(config)

	// Add items
	for i := 0; i < 3; i++ {
		buf.Add(createTestAudio(i), i)
	}

	// Evict items older than 1 microsecond (should evict all)
	time.Sleep(time.Millisecond)
	evicted := buf.EvictOldItems(time.Microsecond)

	if evicted != 3 {
		t.Errorf("EvictOldItems = %d, want 3", evicted)
	}

	if buf.Size() != 0 {
		t.Errorf("Size after eviction = %d, want 0", buf.Size())
	}
}

// TestBufferClose tests closing the buffer.
func TestBufferClose(t *testing.T) {
	config := audio.BufferConfig{
		Capacity: 5,
	}
	buf := audio.NewBuffer(config)

	// Add some items
	buf.Add(createTestAudio(1), 1)

	// Close buffer
	err := buf.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Operations should fail after close
	err = buf.Add(createTestAudio(2), 2)
	if err == nil {
		t.Error("Add should fail after Close")
	}

	_, found := buf.Get(1)
	if found {
		t.Error("Get should fail after Close")
	}

	// Double close should error
	err = buf.Close()
	if err == nil {
		t.Error("Double Close should return error")
	}
}

// TestBufferConcurrency tests thread-safe operations.
func TestBufferConcurrency(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      10,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)
	buf.SetOverflowPolicy(audio.OverflowDrop) // Use drop policy to avoid blocking

	const numProducers = 5
	const numConsumers = 3
	const itemsPerProducer = 20

	var producerWg sync.WaitGroup
	var consumerWg sync.WaitGroup
	stopConsumers := make(chan struct{})
	produced := int32(0)
	consumed := int32(0)

	// Producers
	for p := 0; p < numProducers; p++ {
		producerWg.Add(1)
		go func(id int) {
			defer producerWg.Done()
			for i := 0; i < itemsPerProducer; i++ {
				audio := createTestAudio(id*100 + i)
				err := buf.Add(audio, id*100+i)
				if err == nil {
					atomic.AddInt32(&produced, 1)
				}
				// Small delay to simulate realistic production
				if i%10 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}(p)
	}

	// Consumers
	for c := 0; c < numConsumers; c++ {
		consumerWg.Add(1)
		go func() {
			defer consumerWg.Done()
			for {
				select {
				case <-stopConsumers:
					// Drain remaining items with timeout
					deadline := time.After(100 * time.Millisecond)
					for {
						select {
						case <-deadline:
							return
						default:
							if buf.IsEmpty() {
								return
							}
							_, _, err := buf.GetNext()
							if err == nil {
								atomic.AddInt32(&consumed, 1)
							}
						}
					}
				default:
					// Use non-blocking approach with small timeout
					done := make(chan bool)
					go func() {
						_, _, err := buf.GetNext()
						if err == nil {
							atomic.AddInt32(&consumed, 1)
						}
						done <- true
					}()
					
					select {
					case <-done:
						// Successfully consumed
					case <-time.After(10 * time.Millisecond):
						// Timeout, check if we should stop
						if buf.IsEmpty() {
							select {
							case <-stopConsumers:
								return
							default:
								// Continue trying
							}
						}
					}
				}
			}
		}()
	}

	// Wait for all producers to finish
	producerWg.Wait()
	
	// Give consumers time to process remaining items
	time.Sleep(50 * time.Millisecond)
	
	// Signal consumers to stop
	close(stopConsumers)
	
	// Wait for consumers to finish
	consumerWg.Wait()

	// Verify results
	finalProduced := atomic.LoadInt32(&produced)
	finalConsumed := atomic.LoadInt32(&consumed)

	t.Logf("Produced: %d, Consumed: %d", finalProduced, finalConsumed)

	// Due to overflow policies, consumed might be less than produced
	if finalConsumed > finalProduced {
		t.Errorf("Consumed (%d) > Produced (%d)", finalConsumed, finalProduced)
	}

	// Check stats
	stats := buf.GetStats()
	t.Logf("Buffer stats: %s", stats.String())
}

// TestProducerConsumerBuffer tests the channel-based wrapper.
func TestProducerConsumerBuffer(t *testing.T) {
	pcb := audio.NewProducerConsumerBuffer(5)

	// Test produce
	err := pcb.Produce(createTestAudio(1), 1)
	if err != nil {
		t.Errorf("Produce failed: %v", err)
	}

	// Test consume
	done := make(chan bool)
	go func() {
		audio, idx, err := pcb.Consume()
		if err != nil {
			t.Errorf("Consume failed: %v", err)
		}
		if idx != 1 {
			t.Errorf("Consume index = %d, want 1", idx)
		}
		if audio == nil {
			t.Error("Consume returned nil audio")
		}
		done <- true
	}()

	select {
	case <-done:
		// Good
	case <-time.After(time.Second):
		t.Error("Consume timed out")
	}

	// Test close
	err = pcb.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Operations should fail after close
	err = pcb.Produce(createTestAudio(2), 2)
	if err == nil {
		t.Error("Produce should fail after Close")
	}

	_, _, err = pcb.Consume()
	if err == nil {
		t.Error("Consume should fail after Close")
	}
}

// TestBufferStats tests statistics tracking.
func TestBufferStats(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)

	// Add items
	for i := 0; i < 3; i++ {
		buf.Add(createTestAudio(i), i)
	}

	// Get some items
	buf.GetNext()
	buf.Get(1)

	// Check stats
	stats := buf.GetStats()
	if stats.TotalAdded != 3 {
		t.Errorf("TotalAdded = %d, want 3", stats.TotalAdded)
	}
	if stats.TotalRetrieved < 1 {
		t.Errorf("TotalRetrieved = %d, want at least 1", stats.TotalRetrieved)
	}
	if stats.CurrentSize != 2 {
		t.Errorf("CurrentSize = %d, want 2", stats.CurrentSize)
	}
	if stats.PeakSize != 3 {
		t.Errorf("PeakSize = %d, want 3", stats.PeakSize)
	}

	// Test string representation
	str := stats.String()
	if str == "" {
		t.Error("Stats.String() returned empty string")
	}
}

// TestBufferMemoryPooling tests sync.Pool usage.
func TestBufferMemoryPooling(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
		PreAllocate:   true,
	}
	buf := audio.NewBuffer(config)

	// Add and remove items multiple times
	for cycle := 0; cycle < 10; cycle++ {
		// Add items
		for i := 0; i < 3; i++ {
			buf.Add(createTestAudio(i), i)
		}
		// Remove items
		for i := 0; i < 3; i++ {
			buf.GetNext()
		}
	}

	// Check recycling stats
	stats := buf.GetStats()
	if stats.TotalRecycled == 0 {
		t.Error("Expected items to be recycled")
	}
	t.Logf("Total recycled: %d", stats.TotalRecycled)
}

// BenchmarkBufferAdd benchmarks Add operation.
func BenchmarkBufferAdd(b *testing.B) {
	config := audio.BufferConfig{
		Capacity:      100,
		EnableMetrics: false,
	}
	buf := audio.NewBuffer(config)
	audio := createTestAudio(0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Add(audio, i)
		if i%100 == 99 {
			buf.Clear() // Clear periodically to avoid overflow
		}
	}
}

// BenchmarkBufferGet benchmarks Get operation.
func BenchmarkBufferGet(b *testing.B) {
	config := audio.BufferConfig{
		Capacity:      100,
		EnableMetrics: false,
	}
	buf := audio.NewBuffer(config)

	// Pre-fill buffer
	for i := 0; i < 50; i++ {
		buf.Add(createTestAudio(i), i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Get(i % 50)
	}
}

// BenchmarkBufferConcurrent benchmarks concurrent operations.
func BenchmarkBufferConcurrent(b *testing.B) {
	config := audio.BufferConfig{
		Capacity:      100,
		EnableMetrics: false,
	}
	buf := audio.NewBuffer(config)

	b.RunParallel(func(pb *testing.PB) {
		audio := createTestAudio(0)
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				buf.Add(audio, i)
			} else {
				buf.GetNext()
			}
			i++
		}
	})
}

// TestBufferEdgeCases tests edge cases and error conditions.
func TestBufferEdgeCases(t *testing.T) {
	t.Run("NilAudio", func(t *testing.T) {
		buf := audio.NewBuffer(audio.DefaultBufferConfig())
		err := buf.Add(nil, 0)
		if err != nil {
			t.Errorf("Add(nil) returned error: %v", err)
		}
		// Should handle nil gracefully
	})

	t.Run("NegativeIndex", func(t *testing.T) {
		buf := audio.NewBuffer(audio.DefaultBufferConfig())
		audio := createTestAudio(-1)
		err := buf.Add(audio, -1)
		if err != nil {
			t.Errorf("Add with negative index failed: %v", err)
		}

		// Should be retrievable
		found, ok := buf.Get(-1)
		if !ok {
			t.Error("Get(-1) not found")
		}
		if found == nil {
			t.Error("Get(-1) returned nil")
		}
	})

	t.Run("LargeCapacity", func(t *testing.T) {
		config := audio.BufferConfig{
			Capacity: 10000,
		}
		buf := audio.NewBuffer(config)
		if buf.Capacity() != 10000 {
			t.Errorf("Capacity = %d, want 10000", buf.Capacity())
		}
	})
}

// TestBufferConsistency tests internal consistency.
func TestBufferConsistency(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)

	// Perform various operations
	for i := 0; i < 3; i++ {
		buf.Add(createTestAudio(i), i)
	}

	// Check consistency
	if buf.IsEmpty() && buf.Size() > 0 {
		t.Error("Inconsistent: IsEmpty=true but Size>0")
	}

	if buf.IsFull() && buf.Size() < buf.Capacity() {
		t.Error("Inconsistent: IsFull=true but Size<Capacity")
	}

	// Clear and check again
	buf.Clear()
	if !buf.IsEmpty() {
		t.Error("Buffer should be empty after Clear")
	}
	if buf.Size() != 0 {
		t.Error("Size should be 0 after Clear")
	}
}

// TestBufferErrorRecovery tests recovery from error conditions.
func TestBufferErrorRecovery(t *testing.T) {
	config := audio.BufferConfig{
		Capacity:      5,
		EnableMetrics: true,
	}
	buf := audio.NewBuffer(config)

	// Normal operation
	buf.Add(createTestAudio(1), 1)

	// Simulate various error conditions and recovery
	t.Run("RecoverFromFullBuffer", func(t *testing.T) {
		// Fill buffer
		for i := 0; i < 5; i++ {
			buf.Add(createTestAudio(i), i)
		}

		// Clear to recover
		buf.Clear()

		// Should work normally again
		err := buf.Add(createTestAudio(99), 99)
		if err != nil {
			t.Errorf("Add after recovery failed: %v", err)
		}
	})
}