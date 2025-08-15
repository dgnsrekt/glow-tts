package tts

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"github.com/charmbracelet/log"
)

// QueueState represents the current state of the audio queue
type QueueState int

const (
	// QueueStateIdle indicates the queue is idle
	QueueStateIdle QueueState = iota
	// QueueStateProcessing indicates the queue is processing audio
	QueueStateProcessing
	// QueueStatePlaying indicates audio is currently playing
	QueueStatePlaying
	// QueueStatePaused indicates playback is paused
	QueueStatePaused
	// QueueStateStopped indicates the queue is stopped
	QueueStateStopped
)

// Default queue configuration
const (
	DefaultLookaheadSize   = 3        // Number of sentences to synthesize ahead
	DefaultWorkerCount     = 2        // Number of synthesis workers
	DefaultMaxMemoryMB     = 50       // Maximum memory usage in MB
	DefaultRetentionPeriod = 5        // Sentences to retain after playback
	DefaultCrossfadeMs     = 50       // Crossfade duration in milliseconds
	SilenceThreshold       = 0.01     // Amplitude threshold for silence detection
	MaxQueueSize           = 100      // Maximum number of segments in queue
)

// TextSegment represents a text segment to be synthesized
type TextSegment struct {
	ID       string
	Text     string
	Position int
	Priority int
}

// AudioSegment represents a synthesized audio segment
type AudioSegment struct {
	ID           string
	Text         string
	Audio        []byte
	ProcessedAudio []byte // After preprocessing
	Position     int
	Duration     time.Duration
	Synthesized  time.Time
	LastAccessed time.Time
	Playing      bool
	Played       bool
}

// QueueConfig contains configuration for the audio queue
type QueueConfig struct {
	LookaheadSize      int
	WorkerCount        int
	MaxMemoryMB        int
	RetentionPeriod    int
	CrossfadeDurationMs int
	CacheManager       *TTSCacheManager
	Engine             TTSEngine
	Parser             TextParser
}

// DefaultQueueConfig returns default queue configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		LookaheadSize:       DefaultLookaheadSize,
		WorkerCount:         DefaultWorkerCount,
		MaxMemoryMB:         DefaultMaxMemoryMB,
		RetentionPeriod:     DefaultRetentionPeriod,
		CrossfadeDurationMs: DefaultCrossfadeMs,
	}
}

// TTSAudioQueue manages audio preprocessing and playback order for TTS
type TTSAudioQueue struct {
	// Core components
	config   *QueueConfig
	segments map[string]*AudioSegment // Map of segment ID to audio segment
	order    []string                 // Ordered list of segment IDs
	
	// State management
	state          QueueState
	currentIndex   int
	totalProcessed int64
	totalPlayed    int64
	
	// Processing pipeline
	textQueue      chan TextSegment
	synthesisQueue chan string // Segment IDs to synthesize
	workers        []*queueWorker
	
	// Memory management
	memoryUsage    int64
	maxMemory      int64
	segmentPool    sync.Pool
	
	// Synchronization
	mu            sync.RWMutex
	stateMu       sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	workerWg      sync.WaitGroup
	
	// Callbacks
	onStateChange func(QueueState)
	onProgress    func(current, total int)
	onError       func(error)
	
	// Metrics
	metrics *QueueMetrics
}

// queueWorker represents a background synthesis worker
type queueWorker struct {
	id       int
	queue    *TTSAudioQueue
	ctx      context.Context
	cancel   context.CancelFunc
}

// QueueMetrics tracks queue performance metrics
type QueueMetrics struct {
	mu               sync.RWMutex
	synthesisCount   int64
	synthesisTime    time.Duration
	bufferHits       int64
	bufferMisses     int64
	avgSynthesisTime time.Duration
	queueDepth       int
	memoryUsage      int64
	lastUpdate       time.Time
}

// NewAudioQueue creates a new audio queue
func NewAudioQueue(config *QueueConfig) (*TTSAudioQueue, error) {
	if config == nil {
		config = DefaultQueueConfig()
	}
	
	// Validate configuration
	if config.Engine == nil {
		return nil, fmt.Errorf("TTSEngine is required")
	}
	
	if config.Parser == nil {
		return nil, fmt.Errorf("TextParser is required")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	aq := &TTSAudioQueue{
		config:         config,
		segments:       make(map[string]*AudioSegment),
		order:          make([]string, 0, MaxQueueSize),
		state:          QueueStateIdle,
		currentIndex:   -1,
		textQueue:      make(chan TextSegment, MaxQueueSize),
		synthesisQueue: make(chan string, config.LookaheadSize*2),
		workers:        make([]*queueWorker, 0, config.WorkerCount),
		maxMemory:      int64(config.MaxMemoryMB * 1024 * 1024),
		ctx:            ctx,
		cancel:         cancel,
		metrics:        &QueueMetrics{lastUpdate: time.Now()},
	}
	
	// Initialize segment pool
	aq.segmentPool = sync.Pool{
		New: func() interface{} {
			return &AudioSegment{}
		},
	}
	
	// Start workers
	aq.startWorkers()
	
	// Start background processors
	go aq.processTextQueue()
	go aq.memoryManager()
	
	return aq, nil
}

// SetCallbacks sets callback functions for queue events
func (aq *TTSAudioQueue) SetCallbacks(onStateChange func(QueueState), onProgress func(int, int), onError func(error)) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	aq.onStateChange = onStateChange
	aq.onProgress = onProgress
	aq.onError = onError
}

// AddText adds text to the queue for processing
func (aq *TTSAudioQueue) AddText(text string) error {
	// Parse text into sentences
	sentences, err := aq.config.Parser.ParseSentences(text)
	if err != nil {
		return fmt.Errorf("failed to parse text: %w", err)
	}
	
	log.Debug("TTS Queue: Adding sentences", "count", len(sentences))
	
	// Add each sentence to the queue
	for i, sentence := range sentences {
		// Lock for reading aq.order length
		aq.mu.RLock()
		position := len(aq.order) + i
		aq.mu.RUnlock()
		
		segment := TextSegment{
			ID:       fmt.Sprintf("seg-%d-%d", time.Now().UnixNano(), i),
			Text:     sentence.Text,
			Position: position,
			Priority: 0,
		}
		
		log.Debug("TTS Queue: Adding segment", 
			"index", i, 
			"position", segment.Position,
			"text_preview", sentence.Text[:min(30, len(sentence.Text))])
		
		select {
		case aq.textQueue <- segment:
		case <-aq.ctx.Done():
			return fmt.Errorf("queue is shutting down")
		default:
			return fmt.Errorf("text queue is full")
		}
	}
	
	return nil
}

// processTextQueue processes incoming text segments
func (aq *TTSAudioQueue) processTextQueue() {
	for {
		select {
		case segment := <-aq.textQueue:
			aq.mu.Lock()
			
			// Create audio segment
			audioSeg := aq.segmentPool.Get().(*AudioSegment)
			audioSeg.ID = segment.ID
			audioSeg.Text = segment.Text
			audioSeg.Position = segment.Position
			audioSeg.Synthesized = time.Time{}
			audioSeg.Playing = false
			audioSeg.Played = false
			
			// Add to segments map and order
			aq.segments[segment.ID] = audioSeg
			aq.order = append(aq.order, segment.ID)
			
			// Initialize currentIndex to 0 for the first segment
			if aq.currentIndex < 0 && len(aq.order) == 1 {
				aq.currentIndex = 0
				log.Debug("TTS Queue: Initialized currentIndex to 0")
			}
			
			// Queue for synthesis if within lookahead window
			// Use max(0, currentIndex) to handle initial state
			effectiveIndex := aq.currentIndex
			if effectiveIndex < 0 {
				effectiveIndex = 0
			}
			if segment.Position <= effectiveIndex+aq.config.LookaheadSize {
				log.Debug("TTS Queue: Queueing segment for synthesis",
					"segmentID", segment.ID,
					"position", segment.Position,
					"effectiveIndex", effectiveIndex,
					"lookaheadSize", aq.config.LookaheadSize)
				select {
				case aq.synthesisQueue <- segment.ID:
					log.Debug("TTS Queue: Segment queued for synthesis", "segmentID", segment.ID)
				default:
					log.Warn("TTS Queue: Synthesis queue full", "segmentID", segment.ID)
				}
			} else {
				log.Debug("TTS Queue: Segment outside lookahead window",
					"position", segment.Position,
					"window", effectiveIndex+aq.config.LookaheadSize)
			}
			
			aq.mu.Unlock()
			
			// Update state
			aq.setState(QueueStateProcessing)
			
		case <-aq.ctx.Done():
			return
		}
	}
}

// startWorkers starts synthesis worker goroutines
func (aq *TTSAudioQueue) startWorkers() {
	for i := 0; i < aq.config.WorkerCount; i++ {
		ctx, cancel := context.WithCancel(aq.ctx)
		worker := &queueWorker{
			id:     i,
			queue:  aq,
			ctx:    ctx,
			cancel: cancel,
		}
		aq.workers = append(aq.workers, worker)
		
		aq.workerWg.Add(1)
		go worker.run()
	}
}

// run is the main loop for a synthesis worker
func (w *queueWorker) run() {
	defer w.queue.workerWg.Done()
	
	for {
		select {
		case segmentID := <-w.queue.synthesisQueue:
			w.synthesizeSegment(segmentID)
			
		case <-w.ctx.Done():
			return
		}
	}
}

// synthesizeSegment synthesizes audio for a segment
func (w *queueWorker) synthesizeSegment(segmentID string) {
	log.Debug("TTS Worker: Starting synthesis", "workerID", w.id, "segmentID", segmentID)
	
	// Get segment data safely - copy what we need while holding the lock
	w.queue.mu.RLock()
	segment, exists := w.queue.segments[segmentID]
	if !exists {
		w.queue.mu.RUnlock()
		log.Debug("TTS Worker: Segment not found", "segmentID", segmentID)
		return
	}
	
	// Check if already synthesized while we have the lock
	if segment.Audio != nil {
		w.queue.mu.RUnlock()
		log.Debug("TTS Worker: Segment already synthesized", "segmentID", segmentID)
		return
	}
	
	// Copy the text we need to synthesize while holding the lock
	textToSynthesize := segment.Text
	w.queue.mu.RUnlock()
	
	start := time.Now()
	
	// Check cache first
	var audioData []byte
	var err error
	
	if w.queue.config.CacheManager != nil {
		cacheKey := GenerateCacheKey(textToSynthesize, w.queue.config.Engine.GetName(), 1.0)
		cached, cacheErr := w.queue.config.CacheManager.Get(cacheKey)
		if cacheErr == nil && cached != nil {
			audioData = cached.Audio
			atomic.AddInt64(&w.queue.metrics.bufferHits, 1)
		} else {
			atomic.AddInt64(&w.queue.metrics.bufferMisses, 1)
		}
	}
	
	// Synthesize if not cached
	if audioData == nil {
		log.Debug("TTS Worker: Synthesizing text", "segmentID", segmentID, "textLen", len(textToSynthesize))
		audioData, err = w.queue.config.Engine.Synthesize(textToSynthesize, 1.0)
		if err != nil {
			log.Error("TTS Worker: Synthesis failed", "segmentID", segmentID, "error", err)
			if w.queue.onError != nil {
				w.queue.onError(fmt.Errorf("synthesis failed for segment %s: %w", segmentID, err))
			}
			return
		}
		log.Debug("TTS Worker: Synthesis complete", "segmentID", segmentID, "audioSize", len(audioData))
		
		// Cache the result
		if w.queue.config.CacheManager != nil && len(audioData) > 0 {
			cacheKey := GenerateCacheKey(textToSynthesize, w.queue.config.Engine.GetName(), 1.0)
			cacheData := &AudioData{
				Audio:    audioData,
				Text:     textToSynthesize,
				Voice:    w.queue.config.Engine.GetName(),
				Speed:    1.0,
				CacheKey: cacheKey,
			}
			_ = w.queue.config.CacheManager.Put(cacheKey, cacheData)
		}
	}
	
	// Preprocess audio
	processedAudio := w.queue.preprocessAudio(audioData)
	
	// Update segment - re-fetch it safely to avoid stale pointer
	w.queue.mu.Lock()
	segment, exists = w.queue.segments[segmentID]
	if !exists {
		// Segment was cleared while we were synthesizing
		w.queue.mu.Unlock()
		log.Debug("TTS Worker: Segment cleared during synthesis", "segmentID", segmentID)
		return
	}
	
	segment.Audio = audioData
	segment.ProcessedAudio = processedAudio
	segment.Synthesized = time.Now()
	segment.Duration = w.queue.calculateDuration(audioData)
	
	// Update memory usage
	audioSize := int64(len(audioData) + len(processedAudio))
	atomic.AddInt64(&w.queue.memoryUsage, audioSize)
	atomic.AddInt64(&w.queue.totalProcessed, 1)
	
	w.queue.mu.Unlock()
	
	// Update metrics
	synthTime := time.Since(start)
	w.queue.updateMetrics(synthTime)
	
	// Trigger lookahead for next segments
	w.queue.checkLookahead()
	
	// Notify progress
	if w.queue.onProgress != nil {
		w.queue.mu.RLock()
		current := w.queue.currentIndex + 1
		total := len(w.queue.order)
		w.queue.mu.RUnlock()
		w.queue.onProgress(current, total)
	}
}

// preprocessAudio preprocesses audio for seamless playback
func (aq *TTSAudioQueue) preprocessAudio(audio []byte) []byte {
	log.Debug("TTS Queue: preprocessAudio called", "inputSize", len(audio))
	
	if len(audio) == 0 {
		log.Warn("TTS Queue: preprocessAudio received empty audio")
		return audio
	}
	
	// Trim silence from beginning and end
	processed := aq.trimSilence(audio)
	log.Debug("TTS Queue: After trimSilence", "processedSize", len(processed))
	
	// Normalize audio levels
	processed = aq.normalizeAudio(processed)
	log.Debug("TTS Queue: After normalizeAudio", "finalSize", len(processed))
	
	// Add crossfade markers (actual crossfading done during playback)
	// For now, just return the processed audio
	return processed
}

// trimSilence removes silence from the beginning and end of audio
func (aq *TTSAudioQueue) trimSilence(audio []byte) []byte {
	if len(audio) < 4 {
		log.Debug("TTS Queue: Audio too small to trim", "size", len(audio))
		return audio
	}
	
	samples := len(audio) / 2 // 16-bit samples
	
	// Find first non-silent sample
	startIdx := -1
	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		amplitude := math.Abs(float64(sample) / 32768.0)
		if amplitude > SilenceThreshold {
			startIdx = i
			break
		}
	}
	
	// If no non-silent samples found, return empty to indicate silence
	if startIdx == -1 {
		log.Warn("TTS Queue: Audio is completely silent")
		// Return original audio instead of empty to avoid issues
		return audio
	}
	
	// Find last non-silent sample
	endIdx := samples - 1
	for i := samples - 1; i >= startIdx; i-- {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		amplitude := math.Abs(float64(sample) / 32768.0)
		if amplitude > SilenceThreshold {
			endIdx = i
			break
		}
	}
	
	log.Debug("TTS Queue: Trimming silence", 
		"startIdx", startIdx, 
		"endIdx", endIdx, 
		"samples", samples,
		"removedStart", startIdx,
		"removedEnd", samples-1-endIdx)
	
	// Return trimmed audio
	if startIdx > endIdx {
		log.Warn("TTS Queue: Invalid trim indices", "startIdx", startIdx, "endIdx", endIdx)
		return audio // Return original if something went wrong
	}
	
	return audio[startIdx*2 : (endIdx+1)*2]
}

// normalizeAudio normalizes audio levels
func (aq *TTSAudioQueue) normalizeAudio(audio []byte) []byte {
	if len(audio) < 4 {
		return audio
	}
	
	samples := len(audio) / 2
	
	// Find peak amplitude
	var peak float64
	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		amplitude := math.Abs(float64(sample))
		if amplitude > peak {
			peak = amplitude
		}
	}
	
	if peak == 0 {
		return audio // Silent audio
	}
	
	// Calculate normalization factor (normalize to 80% of max)
	targetPeak := 32768.0 * 0.8
	factor := targetPeak / peak
	
	// Don't amplify too much (max 2x)
	if factor > 2.0 {
		factor = 2.0
	}
	
	// Apply normalization
	normalized := make([]byte, len(audio))
	for i := 0; i < samples; i++ {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		normalizedSample := int16(float64(sample) * factor)
		binary.LittleEndian.PutUint16(normalized[i*2:i*2+2], uint16(normalizedSample))
	}
	
	return normalized
}

// calculateDuration calculates audio duration from PCM data
func (aq *TTSAudioQueue) calculateDuration(audio []byte) time.Duration {
	// Assuming 22050 Hz, 16-bit mono
	sampleRate := 22050
	bytesPerSample := 2
	samples := len(audio) / bytesPerSample
	seconds := float64(samples) / float64(sampleRate)
	return time.Duration(seconds * float64(time.Second))
}

// checkLookahead ensures lookahead buffer is maintained
func (aq *TTSAudioQueue) checkLookahead() {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	// Calculate target range for synthesis
	startIdx := aq.currentIndex + 1
	endIdx := startIdx + aq.config.LookaheadSize
	
	if endIdx > len(aq.order) {
		endIdx = len(aq.order)
	}
	
	// Queue segments for synthesis
	for i := startIdx; i < endIdx; i++ {
		if i < 0 || i >= len(aq.order) {
			continue
		}
		
		segmentID := aq.order[i]
		segment := aq.segments[segmentID]
		
		if segment != nil && segment.Audio == nil {
			select {
			case aq.synthesisQueue <- segmentID:
			default:
				// Queue full, will be picked up later
			}
		}
	}
}

// GetCurrent returns the current audio segment
func (aq *TTSAudioQueue) GetCurrent() (*AudioSegment, error) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	log.Debug("TTS Queue: GetCurrent", 
		"currentIndex", aq.currentIndex, 
		"orderLen", len(aq.order),
		"segmentsLen", len(aq.segments))
	
	if aq.currentIndex < 0 || aq.currentIndex >= len(aq.order) {
		log.Error("TTS Queue: No current segment", 
			"currentIndex", aq.currentIndex, 
			"orderLen", len(aq.order))
		return nil, fmt.Errorf("no current segment")
	}
	
	segmentID := aq.order[aq.currentIndex]
	segment := aq.segments[segmentID]
	
	if segment == nil {
		log.Error("TTS Queue: Segment not found", "segmentID", segmentID)
		return nil, fmt.Errorf("segment not found")
	}
	
	log.Debug("TTS Queue: Found current segment", 
		"segmentID", segmentID,
		"hasAudio", segment.Audio != nil,
		"hasProcessedAudio", segment.ProcessedAudio != nil)
	
	// Update access time
	segment.LastAccessed = time.Now()
	
	return segment, nil
}

// Next moves to the next segment
func (aq *TTSAudioQueue) Next() (*AudioSegment, error) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	if aq.currentIndex >= len(aq.order)-1 {
		return nil, fmt.Errorf("end of queue")
	}
	
	// Mark current as played
	if aq.currentIndex >= 0 && aq.currentIndex < len(aq.order) {
		segmentID := aq.order[aq.currentIndex]
		if segment := aq.segments[segmentID]; segment != nil {
			segment.Playing = false
			segment.Played = true
			atomic.AddInt64(&aq.totalPlayed, 1)
		}
	}
	
	aq.currentIndex++
	
	segmentID := aq.order[aq.currentIndex]
	segment := aq.segments[segmentID]
	
	if segment == nil {
		return nil, fmt.Errorf("segment not found")
	}
	
	segment.Playing = true
	segment.LastAccessed = time.Now()
	
	// Trigger lookahead
	go aq.checkLookahead()
	
	// Update state
	aq.setState(QueueStatePlaying)
	
	return segment, nil
}

// Previous moves to the previous segment
func (aq *TTSAudioQueue) Previous() (*AudioSegment, error) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	if aq.currentIndex <= 0 {
		return nil, fmt.Errorf("beginning of queue")
	}
	
	// Mark current as not playing
	if aq.currentIndex < len(aq.order) {
		segmentID := aq.order[aq.currentIndex]
		if segment := aq.segments[segmentID]; segment != nil {
			segment.Playing = false
		}
	}
	
	aq.currentIndex--
	
	segmentID := aq.order[aq.currentIndex]
	segment := aq.segments[segmentID]
	
	if segment == nil {
		return nil, fmt.Errorf("segment not found")
	}
	
	segment.Playing = true
	segment.LastAccessed = time.Now()
	
	return segment, nil
}

// Skip skips n segments (positive for forward, negative for backward)
func (aq *TTSAudioQueue) Skip(n int) (*AudioSegment, error) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	newIndex := aq.currentIndex + n
	
	if newIndex < 0 {
		newIndex = 0
	} else if newIndex >= len(aq.order) {
		newIndex = len(aq.order) - 1
	}
	
	// Mark current as not playing
	if aq.currentIndex >= 0 && aq.currentIndex < len(aq.order) {
		segmentID := aq.order[aq.currentIndex]
		if segment := aq.segments[segmentID]; segment != nil {
			segment.Playing = false
			if n > 0 {
				segment.Played = true
			}
		}
	}
	
	aq.currentIndex = newIndex
	
	segmentID := aq.order[aq.currentIndex]
	segment := aq.segments[segmentID]
	
	if segment == nil {
		return nil, fmt.Errorf("segment not found")
	}
	
	segment.Playing = true
	segment.LastAccessed = time.Now()
	
	// Trigger lookahead
	go aq.checkLookahead()
	
	return segment, nil
}

// Clear clears the queue
func (aq *TTSAudioQueue) Clear() {
	// First drain the queues to prevent new work
	// This needs to happen before taking the lock to avoid deadlock
	done := make(chan struct{})
	go func() {
		// Clear text queue
		for len(aq.textQueue) > 0 {
			select {
			case <-aq.textQueue:
			default:
				// If queue is empty, break
				break
			}
		}
		// Clear synthesis queue  
		for len(aq.synthesisQueue) > 0 {
			select {
			case <-aq.synthesisQueue:
			default:
				// If queue is empty, break
				break
			}
		}
		close(done)
	}()
	
	// Wait for queue draining with timeout
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		// Continue anyway if draining takes too long
	}
	
	// Now lock and clear everything
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	// Return segments to pool
	for _, segment := range aq.segments {
		if segment != nil {
			segment.Audio = nil
			segment.ProcessedAudio = nil
			aq.segmentPool.Put(segment)
		}
	}
	
	aq.segments = make(map[string]*AudioSegment)
	aq.order = make([]string, 0, MaxQueueSize)
	aq.currentIndex = -1
	atomic.StoreInt64(&aq.memoryUsage, 0)
	
	// Only set to Idle if we're not already stopped
	aq.stateMu.RLock()
	currentState := aq.state
	aq.stateMu.RUnlock()
	
	if currentState != QueueStateStopped {
		aq.setState(QueueStateIdle)
	}
}

// GetQueueDepth returns the number of segments in the queue
func (aq *TTSAudioQueue) GetQueueDepth() int {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	return len(aq.order)
}

// GetMemoryUsage returns current memory usage in bytes
func (aq *TTSAudioQueue) GetMemoryUsage() int64 {
	return atomic.LoadInt64(&aq.memoryUsage)
}

// memoryManager manages memory usage
func (aq *TTSAudioQueue) memoryManager() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			aq.cleanupMemory()
			
		case <-aq.ctx.Done():
			return
		}
	}
}

// cleanupMemory frees memory from old segments
func (aq *TTSAudioQueue) cleanupMemory() {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	
	memUsage := atomic.LoadInt64(&aq.memoryUsage)
	if memUsage < aq.maxMemory*9/10 {
		return // Below 90% threshold
	}
	
	// Find segments to evict (played and outside retention window)
	evictBoundary := aq.currentIndex - aq.config.RetentionPeriod
	
	for i := 0; i < evictBoundary && i < len(aq.order); i++ {
		segmentID := aq.order[i]
		segment := aq.segments[segmentID]
		
		if segment != nil && segment.Played && segment.Audio != nil {
			// Free audio data
			audioSize := int64(len(segment.Audio) + len(segment.ProcessedAudio))
			segment.Audio = nil
			segment.ProcessedAudio = nil
			atomic.AddInt64(&aq.memoryUsage, -audioSize)
			
			// Return to pool if completely done
			if !segment.Playing {
				delete(aq.segments, segmentID)
				aq.segmentPool.Put(segment)
			}
		}
	}
	
	// Force GC if memory usage is still high
	if atomic.LoadInt64(&aq.memoryUsage) > aq.maxMemory {
		runtime.GC()
	}
}

// setState updates the queue state
func (aq *TTSAudioQueue) setState(state QueueState) {
	aq.stateMu.Lock()
	oldState := aq.state
	aq.state = state
	aq.stateMu.Unlock()
	
	if oldState != state && aq.onStateChange != nil {
		aq.onStateChange(state)
	}
}

// GetState returns the current queue state
func (aq *TTSAudioQueue) GetState() QueueState {
	aq.stateMu.RLock()
	defer aq.stateMu.RUnlock()
	return aq.state
}

// updateMetrics updates queue metrics
func (aq *TTSAudioQueue) updateMetrics(synthesisTime time.Duration) {
	aq.metrics.mu.Lock()
	defer aq.metrics.mu.Unlock()
	
	aq.metrics.synthesisCount++
	aq.metrics.synthesisTime += synthesisTime
	aq.metrics.avgSynthesisTime = aq.metrics.synthesisTime / time.Duration(aq.metrics.synthesisCount)
	aq.metrics.queueDepth = len(aq.order)
	aq.metrics.memoryUsage = atomic.LoadInt64(&aq.memoryUsage)
	aq.metrics.lastUpdate = time.Now()
}

// GetMetrics returns current queue metrics
func (aq *TTSAudioQueue) GetMetrics() map[string]interface{} {
	aq.metrics.mu.RLock()
	synthesisCount := aq.metrics.synthesisCount
	avgSynthesisTime := aq.metrics.avgSynthesisTime
	queueDepth := aq.metrics.queueDepth
	memoryUsage := aq.metrics.memoryUsage
	aq.metrics.mu.RUnlock()
	
	aq.mu.RLock()
	currentIdx := aq.currentIndex
	workerCount := len(aq.workers)
	aq.mu.RUnlock()
	
	return map[string]interface{}{
		"synthesis_count":     synthesisCount,
		"avg_synthesis_time":  avgSynthesisTime,
		"buffer_hits":         atomic.LoadInt64(&aq.metrics.bufferHits),
		"buffer_misses":       atomic.LoadInt64(&aq.metrics.bufferMisses),
		"queue_depth":         queueDepth,
		"memory_usage_mb":     memoryUsage / 1024 / 1024,
		"total_processed":     atomic.LoadInt64(&aq.totalProcessed),
		"total_played":        atomic.LoadInt64(&aq.totalPlayed),
		"current_index":       currentIdx,
		"worker_count":        workerCount,
	}
}

// Stop stops the audio queue
func (aq *TTSAudioQueue) Stop() {
	aq.setState(QueueStateStopped)
	
	// Cancel context
	aq.cancel()
	
	// Wait for workers to finish
	aq.workerWg.Wait()
	
	// Clear queue
	aq.Clear()
}

// Pause pauses the queue
func (aq *TTSAudioQueue) Pause() {
	aq.setState(QueueStatePaused)
}

// Resume resumes the queue
func (aq *TTSAudioQueue) Resume() {
	aq.setState(QueueStatePlaying)
}

// DumpState returns a debug dump of the queue state
func (aq *TTSAudioQueue) DumpState() string {
	aq.mu.RLock()
	defer aq.mu.RUnlock()
	
	dump := fmt.Sprintf("TTSAudioQueue State Dump:\n")
	dump += fmt.Sprintf("  State: %v\n", aq.state)
	dump += fmt.Sprintf("  Current Index: %d\n", aq.currentIndex)
	dump += fmt.Sprintf("  Queue Depth: %d\n", len(aq.order))
	dump += fmt.Sprintf("  Memory Usage: %.2f MB\n", float64(atomic.LoadInt64(&aq.memoryUsage))/1024/1024)
	dump += fmt.Sprintf("  Total Processed: %d\n", atomic.LoadInt64(&aq.totalProcessed))
	dump += fmt.Sprintf("  Total Played: %d\n", atomic.LoadInt64(&aq.totalPlayed))
	dump += fmt.Sprintf("  Workers: %d\n", len(aq.workers))
	
	// Segment details
	dump += fmt.Sprintf("\nSegments:\n")
	for i, segmentID := range aq.order {
		segment := aq.segments[segmentID]
		if segment != nil {
			status := "pending"
			if segment.Audio != nil {
				status = "synthesized"
			}
			if segment.Playing {
				status = "playing"
			} else if segment.Played {
				status = "played"
			}
			
			dump += fmt.Sprintf("  [%d] %s: %s (%.2fs)\n", 
				i, segmentID, status, segment.Duration.Seconds())
		}
	}
	
	return dump
}

// CrossfadeAudio crossfades between two audio segments
func CrossfadeAudio(audio1, audio2 []byte, overlapMs int, sampleRate int) []byte {
	if len(audio1) == 0 {
		return audio2
	}
	if len(audio2) == 0 {
		return audio1
	}
	
	// Calculate overlap in samples
	overlapSamples := (overlapMs * sampleRate) / 1000
	overlapBytes := overlapSamples * 2 // 16-bit samples
	
	// Ensure we don't exceed audio length
	if overlapBytes > len(audio1) || overlapBytes > len(audio2) {
		// Simple concatenation if overlap is too large
		result := make([]byte, len(audio1)+len(audio2))
		copy(result, audio1)
		copy(result[len(audio1):], audio2)
		return result
	}
	
	// Create result buffer
	resultLen := len(audio1) + len(audio2) - overlapBytes
	result := make([]byte, resultLen)
	
	// Copy non-overlapping part of audio1
	copy(result, audio1[:len(audio1)-overlapBytes])
	
	// Crossfade overlapping region
	fadeStart := len(audio1) - overlapBytes
	for i := 0; i < overlapBytes; i += 2 {
		// Get samples
		sample1 := int16(binary.LittleEndian.Uint16(audio1[fadeStart+i : fadeStart+i+2]))
		sample2 := int16(binary.LittleEndian.Uint16(audio2[i : i+2]))
		
		// Calculate fade factors
		fadeOut := float64(overlapBytes-i) / float64(overlapBytes)
		fadeIn := float64(i) / float64(overlapBytes)
		
		// Mix samples
		mixed := int16(float64(sample1)*fadeOut + float64(sample2)*fadeIn)
		binary.LittleEndian.PutUint16(result[fadeStart+i:fadeStart+i+2], uint16(mixed))
	}
	
	// Copy non-overlapping part of audio2
	copy(result[len(audio1):], audio2[overlapBytes:])
	
	return result
}

// WaitForReady waits for the current segment to be ready
func (aq *TTSAudioQueue) WaitForReady(timeout time.Duration) error {
	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	log.Debug("TTS Queue: Waiting for ready", "timeout", timeout)
	
	for {
		select {
		case <-ticker.C:
			aq.mu.RLock()
			hasReady := false
			segmentCount := len(aq.segments)
			readyCount := 0
			
			// Check if the current segment specifically is ready
			if aq.currentIndex >= 0 && aq.currentIndex < len(aq.order) {
				segmentID := aq.order[aq.currentIndex]
				if segment, exists := aq.segments[segmentID]; exists && segment.Audio != nil {
					hasReady = true
					readyCount = 1
					log.Debug("TTS Queue: Current segment is ready", 
						"id", segmentID, 
						"currentIndex", aq.currentIndex,
						"audioSize", len(segment.Audio))
				}
			} else {
				// If no current index set, check for any ready segment
				for id, segment := range aq.segments {
					if segment.Audio != nil {
						hasReady = true
						readyCount++
						log.Debug("TTS Queue: Found ready segment", "id", id, "audioSize", len(segment.Audio))
						break
					}
				}
			}
			aq.mu.RUnlock()
			
			log.Debug("TTS Queue: Check ready status", 
				"hasReady", hasReady, 
				"readyCount", readyCount,
				"totalSegments", segmentCount,
				"currentIndex", aq.currentIndex,
				"elapsed", time.Since(start))
			
			if hasReady {
				log.Debug("TTS Queue: Ready!")
				return nil
			}
			
			if time.Since(start) > timeout {
				log.Error("TTS Queue: Timeout waiting for ready", "timeout", timeout)
				return fmt.Errorf("timeout waiting for audio to be ready")
			}
			
		case <-aq.ctx.Done():
			return fmt.Errorf("queue stopped while waiting")
		}
	}
}

// Preload starts synthesizing segments without playing
func (aq *TTSAudioQueue) Preload() {
	aq.checkLookahead()
}

