// Package audio provides audio buffering and playback functionality for TTS.
package audio

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// BufferItem wraps audio data with metadata for buffering.
type BufferItem struct {
	Audio    *tts.Audio
	Index    int       // Sentence index
	Created  time.Time // When the item was created
	Accessed time.Time // Last access time
}

// Buffer implements a thread-safe ring buffer for audio data.
type Buffer struct {
	// Ring buffer storage
	items    []*BufferItem
	capacity int
	size     int32 // atomic for lock-free reads

	// Ring buffer indices
	head int // write position
	tail int // read position

	// Synchronization
	mu       sync.RWMutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	// Memory pool for buffer items
	itemPool *sync.Pool

	// Statistics
	stats BufferStats

	// Configuration
	config BufferConfig

	// Overflow handling
	overflowPolicy OverflowPolicy

	// Closed flag
	closed int32 // atomic
}

// BufferStats tracks buffer performance metrics.
type BufferStats struct {
	TotalAdded     uint64
	TotalRetrieved uint64
	TotalDropped   uint64
	TotalRecycled  uint64
	CurrentSize    int
	PeakSize       int
	AvgWaitTime    time.Duration
}

// BufferConfig holds buffer configuration.
type BufferConfig struct {
	Capacity      int           // Maximum number of items
	MaxItemAge    time.Duration // Maximum age before eviction
	EnableMetrics bool          // Track performance metrics
	PreAllocate   bool          // Pre-allocate buffer space
}

// OverflowPolicy defines how to handle buffer overflow.
type OverflowPolicy int

const (
	// OverflowDrop drops the oldest item when buffer is full.
	OverflowDrop OverflowPolicy = iota
	// OverflowBlock blocks until space is available.
	OverflowBlock
	// OverflowReject rejects new items when buffer is full.
	OverflowReject
)

// DefaultBufferConfig returns sensible defaults.
func DefaultBufferConfig() BufferConfig {
	return BufferConfig{
		Capacity:      10,
		MaxItemAge:    5 * time.Minute,
		EnableMetrics: true,
		PreAllocate:   true,
	}
}

// NewBuffer creates a new audio buffer with the given configuration.
func NewBuffer(config BufferConfig) *Buffer {
	if config.Capacity <= 0 {
		config.Capacity = 10
	}

	b := &Buffer{
		items:          make([]*BufferItem, config.Capacity),
		capacity:       config.Capacity,
		config:         config,
		overflowPolicy: OverflowDrop, // Default policy
	}

	// Initialize condition variables
	b.notEmpty = sync.NewCond(&b.mu)
	b.notFull = sync.NewCond(&b.mu)

	// Setup memory pool
	b.itemPool = &sync.Pool{
		New: func() interface{} {
			return &BufferItem{}
		},
	}

	// Pre-allocate if requested
	if config.PreAllocate {
		for i := 0; i < config.Capacity; i++ {
			b.items[i] = b.itemPool.Get().(*BufferItem)
		}
	}

	return b
}

// Add adds audio to the buffer.
func (b *Buffer) Add(audio *tts.Audio, index int) error {
	if atomic.LoadInt32(&b.closed) == 1 {
		return errors.New("buffer is closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Check if buffer is full
	if b.isFull() {
		switch b.overflowPolicy {
		case OverflowDrop:
			// Drop oldest item
			b.dropOldest()
		case OverflowBlock:
			// Wait for space
			for b.isFull() && atomic.LoadInt32(&b.closed) == 0 {
				b.notFull.Wait()
			}
			if atomic.LoadInt32(&b.closed) == 1 {
				return errors.New("buffer closed while waiting")
			}
		case OverflowReject:
			// Reject new item
			atomic.AddUint64(&b.stats.TotalDropped, 1)
			return errors.New("buffer full")
		}
	}

	// Get or create item
	item := b.getOrCreateItem()
	item.Audio = audio
	item.Index = index
	item.Created = time.Now()
	item.Accessed = time.Time{}

	// Add to ring buffer
	b.items[b.head] = item
	b.head = (b.head + 1) % b.capacity
	atomic.AddInt32(&b.size, 1)

	// Update statistics
	if b.config.EnableMetrics {
		atomic.AddUint64(&b.stats.TotalAdded, 1)
		currentSize := int(atomic.LoadInt32(&b.size))
		b.stats.CurrentSize = currentSize
		if currentSize > b.stats.PeakSize {
			b.stats.PeakSize = currentSize
		}
	}

	// Signal that buffer is not empty
	b.notEmpty.Signal()

	return nil
}

// Get retrieves audio by index from the buffer.
func (b *Buffer) Get(index int) (*tts.Audio, bool) {
	if atomic.LoadInt32(&b.closed) == 1 {
		return nil, false
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	// Search for item with matching index
	for i := 0; i < int(atomic.LoadInt32(&b.size)); i++ {
		pos := (b.tail + i) % b.capacity
		item := b.items[pos]
		if item != nil && item.Index == index {
			item.Accessed = time.Now()
			if b.config.EnableMetrics {
				atomic.AddUint64(&b.stats.TotalRetrieved, 1)
			}
			return item.Audio, true
		}
	}

	return nil, false
}

// GetNext retrieves the next audio item from the buffer (FIFO).
func (b *Buffer) GetNext() (*tts.Audio, int, error) {
	if atomic.LoadInt32(&b.closed) == 1 {
		return nil, -1, errors.New("buffer is closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Wait for items if buffer is empty
	for b.isEmpty() && atomic.LoadInt32(&b.closed) == 0 {
		b.notEmpty.Wait()
	}

	if atomic.LoadInt32(&b.closed) == 1 {
		return nil, -1, errors.New("buffer closed while waiting")
	}

	// Get item from tail
	item := b.items[b.tail]
	if item == nil {
		return nil, -1, errors.New("nil item in buffer")
	}

	audio := item.Audio
	index := item.Index

	// Clear and recycle item
	b.recycleItem(item)
	b.items[b.tail] = nil

	// Update indices
	b.tail = (b.tail + 1) % b.capacity
	atomic.AddInt32(&b.size, -1)

	// Update statistics
	if b.config.EnableMetrics {
		atomic.AddUint64(&b.stats.TotalRetrieved, 1)
		b.stats.CurrentSize = int(atomic.LoadInt32(&b.size))
	}

	// Signal that buffer is not full
	b.notFull.Signal()

	return audio, index, nil
}

// Peek returns the next audio without removing it.
func (b *Buffer) Peek() (*tts.Audio, int, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.isEmpty() {
		return nil, -1, false
	}

	item := b.items[b.tail]
	if item == nil {
		return nil, -1, false
	}

	return item.Audio, item.Index, true
}

// Size returns the current number of items in the buffer.
func (b *Buffer) Size() int {
	return int(atomic.LoadInt32(&b.size))
}

// Capacity returns the maximum capacity of the buffer.
func (b *Buffer) Capacity() int {
	return b.capacity
}

// IsFull returns true if the buffer is at capacity.
func (b *Buffer) IsFull() bool {
	return b.isFull()
}

// IsEmpty returns true if the buffer has no items.
func (b *Buffer) IsEmpty() bool {
	return b.isEmpty()
}

// Clear removes all items from the buffer.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Recycle all items
	for i := 0; i < b.capacity; i++ {
		if b.items[i] != nil {
			b.recycleItem(b.items[i])
			b.items[i] = nil
		}
	}

	// Reset indices
	b.head = 0
	b.tail = 0
	atomic.StoreInt32(&b.size, 0)

	// Update stats
	if b.config.EnableMetrics {
		b.stats.CurrentSize = 0
	}

	// Signal that buffer is not full
	b.notFull.Broadcast()
}

// SetOverflowPolicy sets the buffer overflow handling policy.
func (b *Buffer) SetOverflowPolicy(policy OverflowPolicy) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.overflowPolicy = policy
}

// GetStats returns buffer statistics.
func (b *Buffer) GetStats() BufferStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := b.stats
	stats.CurrentSize = int(atomic.LoadInt32(&b.size))
	return stats
}

// EvictOldItems removes items older than maxAge.
func (b *Buffer) EvictOldItems(maxAge time.Duration) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	evicted := 0
	newItems := make([]*BufferItem, 0, b.capacity)

	// Collect non-evicted items
	for i := 0; i < int(atomic.LoadInt32(&b.size)); i++ {
		pos := (b.tail + i) % b.capacity
		item := b.items[pos]
		if item != nil {
			if now.Sub(item.Created) > maxAge {
				b.recycleItem(item)
				evicted++
			} else {
				newItems = append(newItems, item)
			}
		}
	}

	// Clear all items
	for i := 0; i < b.capacity; i++ {
		b.items[i] = nil
	}

	// Re-add non-evicted items
	b.tail = 0
	b.head = len(newItems)
	for i, item := range newItems {
		b.items[i] = item
	}
	
	// Update size
	atomic.StoreInt32(&b.size, int32(len(newItems)))

	if evicted > 0 {
		b.notFull.Broadcast()
	}

	return evicted
}

// Close closes the buffer and releases resources.
func (b *Buffer) Close() error {
	if !atomic.CompareAndSwapInt32(&b.closed, 0, 1) {
		return errors.New("buffer already closed")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Wake up any waiting goroutines
	b.notEmpty.Broadcast()
	b.notFull.Broadcast()

	// Clear all items
	for i := 0; i < b.capacity; i++ {
		if b.items[i] != nil {
			b.recycleItem(b.items[i])
			b.items[i] = nil
		}
	}

	return nil
}

// Private helper methods

func (b *Buffer) isFull() bool {
	return int(atomic.LoadInt32(&b.size)) >= b.capacity
}

func (b *Buffer) isEmpty() bool {
	return atomic.LoadInt32(&b.size) == 0
}

func (b *Buffer) getOrCreateItem() *BufferItem {
	item := b.itemPool.Get().(*BufferItem)
	// Reset the item
	*item = BufferItem{}
	return item
}

func (b *Buffer) recycleItem(item *BufferItem) {
	if item == nil {
		return
	}

	// Clear references to allow GC
	item.Audio = nil
	item.Index = 0
	item.Created = time.Time{}
	item.Accessed = time.Time{}

	// Return to pool
	b.itemPool.Put(item)

	if b.config.EnableMetrics {
		atomic.AddUint64(&b.stats.TotalRecycled, 1)
	}
}

func (b *Buffer) dropOldest() {
	if b.isEmpty() {
		return
	}

	// Remove item at tail
	item := b.items[b.tail]
	if item != nil {
		b.recycleItem(item)
		b.items[b.tail] = nil
	}

	// Update indices
	b.tail = (b.tail + 1) % b.capacity
	atomic.AddInt32(&b.size, -1)

	if b.config.EnableMetrics {
		atomic.AddUint64(&b.stats.TotalDropped, 1)
	}
}


// ProducerConsumerBuffer wraps Buffer with channel-based producer/consumer pattern.
type ProducerConsumerBuffer struct {
	buffer   *Buffer
	inputCh  chan *BufferItem
	outputCh chan *BufferItem
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewProducerConsumerBuffer creates a buffer with channel-based interface.
func NewProducerConsumerBuffer(capacity int) *ProducerConsumerBuffer {
	config := DefaultBufferConfig()
	config.Capacity = capacity

	pcb := &ProducerConsumerBuffer{
		buffer:   NewBuffer(config),
		inputCh:  make(chan *BufferItem, capacity),
		outputCh: make(chan *BufferItem, capacity),
		done:     make(chan struct{}),
	}

	// Start producer goroutine
	pcb.wg.Add(1)
	go pcb.producer()

	// Start consumer goroutine
	pcb.wg.Add(1)
	go pcb.consumer()

	return pcb
}

// Produce sends audio to the buffer.
func (pcb *ProducerConsumerBuffer) Produce(audio *tts.Audio, index int) error {
	// Check if already closed first
	select {
	case <-pcb.done:
		return errors.New("buffer closed")
	default:
	}
	
	// Try to send
	select {
	case pcb.inputCh <- &BufferItem{Audio: audio, Index: index, Created: time.Now()}:
		return nil
	case <-pcb.done:
		return errors.New("buffer closed")
	}
}

// Consume receives audio from the buffer.
func (pcb *ProducerConsumerBuffer) Consume() (*tts.Audio, int, error) {
	select {
	case item := <-pcb.outputCh:
		if item == nil {
			return nil, -1, errors.New("received nil item")
		}
		return item.Audio, item.Index, nil
	case <-pcb.done:
		return nil, -1, errors.New("buffer closed")
	}
}

// Close shuts down the producer/consumer buffer.
func (pcb *ProducerConsumerBuffer) Close() error {
	// Signal done first
	select {
	case <-pcb.done:
		// Already closed
		return errors.New("already closed")
	default:
		close(pcb.done)
	}
	
	// Wait for goroutines to finish
	pcb.wg.Wait()
	
	// Then close channels
	close(pcb.inputCh)
	close(pcb.outputCh)
	
	// Finally close the buffer
	return pcb.buffer.Close()
}

func (pcb *ProducerConsumerBuffer) producer() {
	defer pcb.wg.Done()

	for {
		select {
		case item := <-pcb.inputCh:
			if item != nil {
				pcb.buffer.Add(item.Audio, item.Index)
			}
		case <-pcb.done:
			return
		}
	}
}

func (pcb *ProducerConsumerBuffer) consumer() {
	defer pcb.wg.Done()

	for {
		// Check if we should stop first
		select {
		case <-pcb.done:
			return
		default:
		}

		// Try to get next item with a timeout
		done := make(chan struct{})
		var audio *tts.Audio
		var index int
		var err error
		
		go func() {
			audio, index, err = pcb.buffer.GetNext()
			close(done)
		}()

		select {
		case <-done:
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			// Send the item
			select {
			case pcb.outputCh <- &BufferItem{Audio: audio, Index: index}:
			case <-pcb.done:
				return
			}
		case <-pcb.done:
			// Just return, don't close the buffer here as it will be closed in Close()
			return
		}
	}
}

// String returns a string representation of buffer stats.
func (s BufferStats) String() string {
	return fmt.Sprintf("Buffer Stats: Added=%d, Retrieved=%d, Dropped=%d, Recycled=%d, Current=%d, Peak=%d",
		s.TotalAdded, s.TotalRetrieved, s.TotalDropped, s.TotalRecycled, s.CurrentSize, s.PeakSize)
}