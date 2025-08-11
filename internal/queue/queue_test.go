package queue

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

func TestAudioQueue_BasicOperations(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024) // 10 items max, 3 lookahead, 1MB memory
	defer q.Close()

	// Test empty queue
	if size := q.Size(); size != 0 {
		t.Errorf("Expected empty queue, got size %d", size)
	}

	// Test peek on empty queue
	_, err := q.Peek()
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty, got %v", err)
	}

	// Test enqueue
	sentence := ttypes.Sentence{
		ID:   "s1",
		Text: "Test sentence",
	}

	err = q.Enqueue(sentence, false)
	if err != nil {
		t.Errorf("Enqueue failed: %v", err)
	}

	if size := q.Size(); size != 1 {
		t.Errorf("Expected size 1, got %d", size)
	}

	// Test peek
	peeked, err := q.Peek()
	if err != nil {
		t.Errorf("Peek failed: %v", err)
	}
	if peeked.ID != sentence.ID {
		t.Errorf("Peeked wrong sentence: %v", peeked)
	}

	// Test dequeue
	dequeued, err := q.Dequeue()
	if err != nil {
		t.Errorf("Dequeue failed: %v", err)
	}
	if dequeued.ID != sentence.ID {
		t.Errorf("Dequeued wrong sentence: %v", dequeued)
	}

	if size := q.Size(); size != 0 {
		t.Errorf("Expected empty queue after dequeue, got size %d", size)
	}
}

func TestAudioQueue_PriorityHandling(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024)
	defer q.Close()

	// Add regular sentences
	for i := 0; i < 5; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("regular-%d", i),
			Text: fmt.Sprintf("Regular sentence %d", i),
		}
		if err := q.Enqueue(sentence, false); err != nil {
			t.Fatalf("Failed to enqueue regular sentence: %v", err)
		}
	}

	// Add priority sentences
	for i := 0; i < 3; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("priority-%d", i),
			Text: fmt.Sprintf("Priority sentence %d", i),
		}
		if err := q.Enqueue(sentence, true); err != nil {
			t.Fatalf("Failed to enqueue priority sentence: %v", err)
		}
	}

	// Priority sentences should be dequeued first
	for i := 0; i < 3; i++ {
		sentence, err := q.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue failed: %v", err)
		}
		if sentence.ID[:8] != "priority" {
			t.Errorf("Expected priority sentence, got %s", sentence.ID)
		}
	}

	// Then regular sentences
	for i := 0; i < 5; i++ {
		sentence, err := q.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue failed: %v", err)
		}
		if sentence.ID[:7] != "regular" {
			t.Errorf("Expected regular sentence, got %s", sentence.ID)
		}
	}
}

func TestAudioQueue_Backpressure(t *testing.T) {
	q := NewAudioQueue(3, 1, 1024*1024) // Small queue for testing backpressure
	defer q.Close()

	// Fill the queue
	for i := 0; i < 3; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("s%d", i),
			Text: fmt.Sprintf("Sentence %d", i),
		}
		if err := q.Enqueue(sentence, false); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	// Try to enqueue when full - should block
	done := make(chan bool)
	go func() {
		sentence := ttypes.Sentence{
			ID:   "s4",
			Text: "Sentence 4",
		}
		err := q.Enqueue(sentence, false)
		if err != nil {
			t.Errorf("Enqueue after space available failed: %v", err)
		}
		done <- true
	}()

	// Should not complete immediately
	select {
	case <-done:
		t.Error("Enqueue should have blocked on full queue")
	case <-time.After(100 * time.Millisecond):
		// Expected behavior
	}

	// Dequeue to make space
	_, err := q.Dequeue()
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}

	// Now enqueue should complete
	select {
	case <-done:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("Enqueue should have completed after space was made")
	}
}

func TestAudioQueue_MemoryLimit(t *testing.T) {
	// Very small memory limit for testing
	q := NewAudioQueue(100, 3, 100) // 100 bytes memory limit
	defer q.Close()

	// Try to add a sentence that exceeds memory limit
	largeSentence := ttypes.Sentence{
		ID:   "large",
		Text: string(make([]byte, 100)), // 200 bytes estimated (x2)
	}

	err := q.Enqueue(largeSentence, false)
	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull for memory limit, got %v", err)
	}

	// Add smaller sentence that fits
	smallSentence := ttypes.Sentence{
		ID:   "small",
		Text: "Hi", // 4 bytes estimated
	}

	err = q.Enqueue(smallSentence, false)
	if err != nil {
		t.Errorf("Failed to enqueue small sentence: %v", err)
	}
}

func TestAudioQueue_Lookahead(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024)
	defer q.Close()

	// Add sentences
	for i := 0; i < 5; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("s%d", i),
			Text: fmt.Sprintf("Sentence %d", i),
		}
		if err := q.Enqueue(sentence, false); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	// Get lookahead
	lookahead := q.GetLookahead()
	if len(lookahead) != 3 {
		t.Errorf("Expected 3 lookahead items, got %d", len(lookahead))
	}

	// Verify lookahead doesn't remove items
	if size := q.Size(); size != 5 {
		t.Errorf("Lookahead changed queue size: %d", size)
	}

	// Verify lookahead order
	for i, s := range lookahead {
		expectedID := fmt.Sprintf("s%d", i)
		if s.ID != expectedID {
			t.Errorf("Lookahead[%d] = %s, expected %s", i, s.ID, expectedID)
		}
	}
}

func TestAudioQueue_ConcurrentAccess(t *testing.T) {
	q := NewAudioQueue(100, 3, 1024*1024)
	defer q.Close()

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Multiple producers
	for p := 0; p < 5; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				sentence := ttypes.Sentence{
					ID:   fmt.Sprintf("p%d-s%d", producerID, i),
					Text: fmt.Sprintf("Producer %d, Sentence %d", producerID, i),
				}
				if err := q.Enqueue(sentence, producerID%2 == 0); err != nil {
					errors <- fmt.Errorf("producer %d enqueue failed: %v", producerID, err)
				}
			}
		}(p)
	}

	// Multiple consumers
	for c := 0; c < 3; c++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			for i := 0; i < 16; i++ { // ~50 items total / 3 consumers
				_, err := q.Dequeue()
				if err != nil && err != ErrQueueEmpty {
					errors <- fmt.Errorf("consumer %d dequeue failed: %v", consumerID, err)
				}
				time.Sleep(time.Millisecond) // Simulate processing
			}
		}(c)
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
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	case err := <-errors:
		t.Fatal(err)
	}

	// Check for any remaining errors
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}

func TestAudioQueue_Clear(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024)
	defer q.Close()

	// Add items
	for i := 0; i < 5; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("s%d", i),
			Text: fmt.Sprintf("Sentence %d", i),
		}
		if err := q.Enqueue(sentence, i%2 == 0); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	if size := q.Size(); size != 5 {
		t.Errorf("Expected size 5, got %d", size)
	}

	// Clear queue
	q.Clear()

	if size := q.Size(); size != 0 {
		t.Errorf("Expected empty queue after clear, got size %d", size)
	}

	// Queue should be empty after clear
	_, err := q.Peek()
	if err != ErrQueueEmpty {
		t.Errorf("Expected ErrQueueEmpty after clear, got %v", err)
	}
}

func TestAudioQueue_Stats(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024)
	defer q.Close()

	// Enqueue items
	for i := 0; i < 5; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("s%d", i),
			Text: fmt.Sprintf("Sentence %d", i),
		}
		if err := q.Enqueue(sentence, i == 0); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	// Dequeue some items
	for i := 0; i < 2; i++ {
		if _, err := q.Dequeue(); err != nil {
			t.Fatalf("Failed to dequeue: %v", err)
		}
	}

	stats := q.GetStats()

	if stats.TotalEnqueued != 5 {
		t.Errorf("Expected 5 enqueued, got %d", stats.TotalEnqueued)
	}

	if stats.TotalDequeued != 2 {
		t.Errorf("Expected 2 dequeued, got %d", stats.TotalDequeued)
	}

	if stats.CurrentSize != 3 {
		t.Errorf("Expected current size 3, got %d", stats.CurrentSize)
	}

	if stats.PeakSize < 5 {
		t.Errorf("Expected peak size >= 5, got %d", stats.PeakSize)
	}

	if stats.HighPriorityCount != 1 {
		t.Errorf("Expected 1 high priority item, got %d", stats.HighPriorityCount)
	}
}

func TestAudioQueue_BatchOperations(t *testing.T) {
	q := NewAudioQueue(20, 3, 1024*1024)
	defer q.Close()

	// Create batch
	batch := make([]ttypes.Sentence, 10)
	for i := 0; i < 10; i++ {
		batch[i] = ttypes.Sentence{
			ID:   fmt.Sprintf("batch-%d", i),
			Text: fmt.Sprintf("Batch sentence %d", i),
		}
	}

	// Enqueue batch
	if err := q.EnqueueBatch(batch, false); err != nil {
		t.Fatalf("EnqueueBatch failed: %v", err)
	}

	if size := q.Size(); size != 10 {
		t.Errorf("Expected size 10 after batch enqueue, got %d", size)
	}

	// Drain to slice
	drained := make([]ttypes.Sentence, 5)
	count := q.DrainTo(drained, 5)

	if count != 5 {
		t.Errorf("Expected to drain 5 items, got %d", count)
	}

	if size := q.Size(); size != 5 {
		t.Errorf("Expected size 5 after drain, got %d", size)
	}

	// Verify drained items
	for i := 0; i < count; i++ {
		expectedID := fmt.Sprintf("batch-%d", i)
		if drained[i].ID != expectedID {
			t.Errorf("Drained[%d] = %s, expected %s", i, drained[i].ID, expectedID)
		}
	}
}

func TestAudioQueue_WaitForSpace(t *testing.T) {
	q := NewAudioQueue(2, 1, 1024*1024) // Small queue
	defer q.Close()

	// Fill queue
	for i := 0; i < 2; i++ {
		sentence := ttypes.Sentence{
			ID:   fmt.Sprintf("s%d", i),
			Text: fmt.Sprintf("Sentence %d", i),
		}
		if err := q.Enqueue(sentence, false); err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should timeout waiting for space
	err := q.WaitForSpace(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected timeout waiting for space, got %v", err)
	}

	// Make space
	if _, err := q.Dequeue(); err != nil {
		t.Fatalf("Failed to dequeue: %v", err)
	}

	// Now should succeed immediately
	ctx2 := context.Background()
	if err := q.WaitForSpace(ctx2); err != nil {
		t.Errorf("WaitForSpace failed after space available: %v", err)
	}
}

func TestAudioQueue_CloseHandling(t *testing.T) {
	q := NewAudioQueue(10, 3, 1024*1024)

	// Close the queue
	if err := q.Close(); err != nil {
		t.Fatalf("Failed to close queue: %v", err)
	}

	// Operations should fail after close
	sentence := ttypes.Sentence{
		ID:   "test",
		Text: "Test",
	}

	if err := q.Enqueue(sentence, false); err != ErrQueueClosed {
		t.Errorf("Expected ErrQueueClosed on enqueue after close, got %v", err)
	}

	if _, err := q.Dequeue(); err != ErrQueueClosed {
		t.Errorf("Expected ErrQueueClosed on dequeue after close, got %v", err)
	}

	if _, err := q.Peek(); err != ErrQueueClosed {
		t.Errorf("Expected ErrQueueClosed on peek after close, got %v", err)
	}

	// Double close should be safe
	if err := q.Close(); err != nil {
		t.Errorf("Double close failed: %v", err)
	}
}

// Benchmark tests
func BenchmarkAudioQueue_Enqueue(b *testing.B) {
	q := NewAudioQueue(1000, 10, 10*1024*1024)
	defer q.Close()

	sentence := ttypes.Sentence{
		ID:   "bench",
		Text: "Benchmark sentence for testing performance",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = q.Enqueue(sentence, false)
		if q.Size() > 900 {
			q.Clear()
		}
	}
}

func BenchmarkAudioQueue_EnqueueDequeue(b *testing.B) {
	q := NewAudioQueue(100, 10, 10*1024*1024)
	defer q.Close()

	sentence := ttypes.Sentence{
		ID:   "bench",
		Text: "Benchmark sentence",
	}

	// Pre-fill
	for i := 0; i < 50; i++ {
		_ = q.Enqueue(sentence, false)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			_ = q.Enqueue(sentence, false)
		} else {
			_, _ = q.Dequeue()
		}
	}
}

func BenchmarkAudioQueue_Priority(b *testing.B) {
	q := NewAudioQueue(1000, 10, 10*1024*1024)
	defer q.Close()

	regular := ttypes.Sentence{
		ID:   "regular",
		Text: "Regular sentence",
	}
	priority := ttypes.Sentence{
		ID:   "priority",
		Text: "Priority sentence",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%3 == 0 {
			_ = q.Enqueue(priority, true)
		} else {
			_ = q.Enqueue(regular, false)
		}

		if q.Size() > 900 {
			for j := 0; j < 100; j++ {
				_, _ = q.Dequeue()
			}
		}
	}
}

func BenchmarkAudioQueue_Concurrent(b *testing.B) {
	q := NewAudioQueue(1000, 10, 10*1024*1024)
	defer q.Close()

	sentence := ttypes.Sentence{
		ID:   "bench",
		Text: "Benchmark sentence",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := q.Enqueue(sentence, false); err == nil {
				_, _ = q.Dequeue()
			}
		}
	})
}
