package queue

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/internal/tts"
)

var (
	// ErrQueueFull is returned when the queue is at capacity
	ErrQueueFull = errors.New("queue is full")
	
	// ErrQueueClosed is returned when operations are attempted on a closed queue
	ErrQueueClosed = errors.New("queue is closed")
)

// AudioQueue manages sentence processing order with priority support and lookahead buffering.
// It provides thread-safe operations for enqueueing and dequeueing sentences,
// with support for high-priority items (user navigation) and normal priority items (sequential reading).
type AudioQueue struct {
	// Priority queue for high-priority items (user navigation)
	priorityQueue *priorityQueue
	
	// Regular queue for normal priority items
	regularQueue []tts.Sentence
	
	// Configuration
	maxSize       int           // Maximum queue size
	lookahead     int           // Number of sentences to preprocess
	memoryLimit   int64         // Maximum memory usage in bytes
	currentMemory int64         // Current memory usage estimate
	
	// Synchronization
	mu       sync.RWMutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
	
	// State
	closed bool
	stats  Stats
	
	// Channels for async operations
	processRequests chan processRequest
	done           chan struct{}
}

// processRequest represents a lookahead processing request
type processRequest struct {
	sentence tts.Sentence
	callback func(tts.Sentence)
}

// Stats tracks queue performance metrics
type Stats struct {
	TotalEnqueued   int64
	TotalDequeued   int64
	TotalDropped    int64
	HighPriorityCount int64
	CurrentSize     int
	PeakSize        int
	LastEnqueue     time.Time
	LastDequeue     time.Time
	AverageWaitTime time.Duration
}

// NewAudioQueue creates a new audio queue with the specified configuration.
func NewAudioQueue(maxSize int, lookahead int, memoryLimit int64) *AudioQueue {
	q := &AudioQueue{
		priorityQueue:   &priorityQueue{},
		regularQueue:    make([]tts.Sentence, 0, maxSize),
		maxSize:        maxSize,
		lookahead:      lookahead,
		memoryLimit:    memoryLimit,
		processRequests: make(chan processRequest, lookahead),
		done:           make(chan struct{}),
	}
	
	heap.Init(q.priorityQueue)
	
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	
	// Start lookahead processor
	go q.processLookahead()
	
	return q
}

// Enqueue adds a sentence to the queue with optional priority.
// High-priority items (from user navigation) are processed before regular items.
func (q *AudioQueue) Enqueue(sentence tts.Sentence, priority bool) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return ErrQueueClosed
	}
	
	// Check if queue is full
	totalSize := q.priorityQueue.Len() + len(q.regularQueue)
	if totalSize >= q.maxSize {
		// Apply backpressure - wait for space
		for totalSize >= q.maxSize && !q.closed {
			q.notFull.Wait()
			totalSize = q.priorityQueue.Len() + len(q.regularQueue)
		}
		
		if q.closed {
			return ErrQueueClosed
		}
	}
	
	// Check memory limit
	estimatedMemory := int64(len(sentence.Text) * 2) // Rough estimate
	if q.currentMemory+estimatedMemory > q.memoryLimit {
		q.stats.TotalDropped++
		return ErrQueueFull
	}
	
	// Add to appropriate queue
	if priority {
		heap.Push(q.priorityQueue, &queueItem{
			sentence: sentence,
			priority: int(tts.PriorityHigh),
			index:    0,
		})
		q.stats.HighPriorityCount++
	} else {
		q.regularQueue = append(q.regularQueue, sentence)
	}
	
	q.currentMemory += estimatedMemory
	q.stats.TotalEnqueued++
	q.stats.LastEnqueue = time.Now()
	
	// Update peak size
	currentSize := q.priorityQueue.Len() + len(q.regularQueue)
	if currentSize > q.stats.PeakSize {
		q.stats.PeakSize = currentSize
	}
	q.stats.CurrentSize = currentSize
	
	// Signal that queue is not empty
	q.notEmpty.Signal()
	
	// Trigger lookahead processing for non-priority items
	if !priority && q.lookahead > 0 {
		select {
		case q.processRequests <- processRequest{sentence: sentence}:
		default:
			// Lookahead processor is busy, skip
		}
	}
	
	return nil
}

// Dequeue removes and returns the next sentence to process.
// Priority items are returned first, followed by regular items in FIFO order.
func (q *AudioQueue) Dequeue() (tts.Sentence, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return tts.Sentence{}, ErrQueueClosed
	}
	
	// Wait while queue is empty
	for q.priorityQueue.Len() == 0 && len(q.regularQueue) == 0 && !q.closed {
		q.notEmpty.Wait()
	}
	
	if q.closed {
		return tts.Sentence{}, ErrQueueClosed
	}
	
	var sentence tts.Sentence
	var memoryFreed int64
	
	// Check priority queue first
	if q.priorityQueue.Len() > 0 {
		item := heap.Pop(q.priorityQueue).(*queueItem)
		sentence = item.sentence
		memoryFreed = int64(len(sentence.Text) * 2)
	} else if len(q.regularQueue) > 0 {
		// Get from regular queue
		sentence = q.regularQueue[0]
		q.regularQueue = q.regularQueue[1:]
		memoryFreed = int64(len(sentence.Text) * 2)
	} else {
		return tts.Sentence{}, tts.ErrQueueEmpty
	}
	
	q.currentMemory -= memoryFreed
	if q.currentMemory < 0 {
		q.currentMemory = 0
	}
	
	q.stats.TotalDequeued++
	q.stats.LastDequeue = time.Now()
	q.stats.CurrentSize = q.priorityQueue.Len() + len(q.regularQueue)
	
	// Signal that queue has space
	q.notFull.Signal()
	
	return sentence, nil
}

// Peek returns the next sentence without removing it from the queue.
func (q *AudioQueue) Peek() (tts.Sentence, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	if q.closed {
		return tts.Sentence{}, ErrQueueClosed
	}
	
	// Check priority queue first
	if q.priorityQueue.Len() > 0 {
		return (*q.priorityQueue)[0].sentence, nil
	}
	
	// Check regular queue
	if len(q.regularQueue) > 0 {
		return q.regularQueue[0], nil
	}
	
	return tts.Sentence{}, tts.ErrQueueEmpty
}

// Size returns the current number of sentences in the queue.
func (q *AudioQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	return q.priorityQueue.Len() + len(q.regularQueue)
}

// Clear removes all sentences from the queue.
func (q *AudioQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	// Clear both queues
	q.priorityQueue = &priorityQueue{}
	heap.Init(q.priorityQueue)
	q.regularQueue = q.regularQueue[:0]
	q.currentMemory = 0
	q.stats.CurrentSize = 0
	
	// Signal that queue has space
	q.notFull.Broadcast()
}

// SetLookahead sets the number of sentences to preprocess.
func (q *AudioQueue) SetLookahead(count int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.lookahead = count
}

// GetStats returns current queue statistics.
func (q *AudioQueue) GetStats() Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	stats := q.stats
	stats.CurrentSize = q.priorityQueue.Len() + len(q.regularQueue)
	
	// Calculate average wait time if we have data
	if q.stats.TotalDequeued > 0 && !q.stats.LastEnqueue.IsZero() && !q.stats.LastDequeue.IsZero() {
		if q.stats.LastDequeue.After(q.stats.LastEnqueue) {
			totalWait := q.stats.LastDequeue.Sub(q.stats.LastEnqueue)
			stats.AverageWaitTime = totalWait / time.Duration(q.stats.TotalDequeued)
		}
	}
	
	return stats
}

// GetLookahead returns upcoming sentences for preprocessing.
// This doesn't remove them from the queue.
func (q *AudioQueue) GetLookahead() []tts.Sentence {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	lookahead := make([]tts.Sentence, 0, q.lookahead)
	
	// First add priority items
	for i := 0; i < q.priorityQueue.Len() && len(lookahead) < q.lookahead; i++ {
		lookahead = append(lookahead, (*q.priorityQueue)[i].sentence)
	}
	
	// Then add regular items
	remaining := q.lookahead - len(lookahead)
	if remaining > 0 && len(q.regularQueue) > 0 {
		end := remaining
		if end > len(q.regularQueue) {
			end = len(q.regularQueue)
		}
		lookahead = append(lookahead, q.regularQueue[:end]...)
	}
	
	return lookahead
}

// Close gracefully shuts down the queue.
func (q *AudioQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return nil
	}
	
	q.closed = true
	close(q.done)
	
	// Wake up any waiting goroutines
	q.notEmpty.Broadcast()
	q.notFull.Broadcast()
	
	return nil
}

// processLookahead handles background preprocessing of upcoming sentences.
func (q *AudioQueue) processLookahead() {
	for {
		select {
		case <-q.done:
			return
		case req := <-q.processRequests:
			// In a real implementation, this would trigger synthesis
			// For now, we just track that it was requested
			if req.callback != nil {
				req.callback(req.sentence)
			}
		}
	}
}

// Priority queue implementation using a heap
type queueItem struct {
	sentence tts.Sentence
	priority int
	index    int // Index in the heap
}

type priorityQueue []*queueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Higher priority items come first
	return pq[i].priority > pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*queueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	item.index = -1 // For safety
	*pq = old[0 : n-1]
	return item
}

// EnqueueBatch adds multiple sentences to the queue efficiently.
func (q *AudioQueue) EnqueueBatch(sentences []tts.Sentence, priority bool) error {
	if len(sentences) == 0 {
		return nil
	}
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return ErrQueueClosed
	}
	
	// Check if we have space for the batch
	totalSize := q.priorityQueue.Len() + len(q.regularQueue)
	spaceAvailable := q.maxSize - totalSize
	
	if spaceAvailable < len(sentences) {
		// Can only add partial batch
		sentences = sentences[:spaceAvailable]
		if len(sentences) == 0 {
			return ErrQueueFull
		}
	}
	
	// Add all sentences
	for _, sentence := range sentences {
		estimatedMemory := int64(len(sentence.Text) * 2)
		if q.currentMemory+estimatedMemory > q.memoryLimit {
			// Stop adding if we hit memory limit
			break
		}
		
		if priority {
			heap.Push(q.priorityQueue, &queueItem{
				sentence: sentence,
				priority: int(tts.PriorityHigh),
				index:    0,
			})
			q.stats.HighPriorityCount++
		} else {
			q.regularQueue = append(q.regularQueue, sentence)
		}
		
		q.currentMemory += estimatedMemory
		q.stats.TotalEnqueued++
	}
	
	q.stats.LastEnqueue = time.Now()
	
	// Update stats
	currentSize := q.priorityQueue.Len() + len(q.regularQueue)
	if currentSize > q.stats.PeakSize {
		q.stats.PeakSize = currentSize
	}
	q.stats.CurrentSize = currentSize
	
	// Signal that queue is not empty
	q.notEmpty.Broadcast()
	
	return nil
}

// DrainTo drains up to n sentences from the queue into the provided slice.
func (q *AudioQueue) DrainTo(sentences []tts.Sentence, n int) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return 0
	}
	
	count := 0
	for count < n && count < len(sentences) {
		// Try priority queue first
		if q.priorityQueue.Len() > 0 {
			item := heap.Pop(q.priorityQueue).(*queueItem)
			sentences[count] = item.sentence
			q.currentMemory -= int64(len(item.sentence.Text) * 2)
			count++
		} else if len(q.regularQueue) > 0 {
			sentences[count] = q.regularQueue[0]
			q.regularQueue = q.regularQueue[1:]
			q.currentMemory -= int64(len(sentences[count].Text) * 2)
			count++
		} else {
			break
		}
		q.stats.TotalDequeued++
	}
	
	if count > 0 {
		q.stats.LastDequeue = time.Now()
		q.stats.CurrentSize = q.priorityQueue.Len() + len(q.regularQueue)
		q.notFull.Broadcast()
	}
	
	return count
}

// WaitForSpace blocks until there is space in the queue or the context is cancelled.
func (q *AudioQueue) WaitForSpace(ctx context.Context) error {
	// Check if there's already space without locking
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return ErrQueueClosed
	}
	
	totalSize := q.priorityQueue.Len() + len(q.regularQueue)
	if totalSize < q.maxSize {
		q.mu.Unlock()
		return nil
	}
	q.mu.Unlock()
	
	// Use a channel to handle context cancellation properly
	done := make(chan error, 1)
	go func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		
		for {
			if q.closed {
				done <- ErrQueueClosed
				return
			}
			
			totalSize := q.priorityQueue.Len() + len(q.regularQueue)
			if totalSize < q.maxSize {
				done <- nil
				return
			}
			
			q.notFull.Wait()
		}
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Wake up the waiting goroutine to avoid leak
		q.mu.Lock()
		q.notFull.Broadcast()
		q.mu.Unlock()
		return ctx.Err()
	}
}