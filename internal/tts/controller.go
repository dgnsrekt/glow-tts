package tts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/internal/tts/engines"
	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

var (
	// ErrControllerNotStarted is returned when operations are attempted before Start()
	ErrControllerNotStarted = errors.New("controller not started")

	// ErrControllerAlreadyStarted is returned when Start() is called multiple times
	ErrControllerAlreadyStarted = errors.New("controller already started")

	// ErrNoDocumentLoaded is returned when playback operations are attempted without a document
	ErrNoDocumentLoaded = errors.New("no document loaded")
)

// TTSController orchestrates all TTS components and manages the TTS pipeline.
// It coordinates between the UI, TTS engines, audio queue, cache, and audio player.
type TTSController struct {
	// Components
	engine    ttypes.TTSEngine
	player    ttypes.AudioPlayer
	queue     ttypes.SentenceQueue
	cache     ttypes.AudioCache
	parser    ttypes.Parser
	speedCtrl ttypes.SpeedController

	// Configuration
	config     *Config
	engineType ttypes.EngineType

	// State management
	state   ttypes.State
	stateMu sync.RWMutex

	// Document processing
	sentences    []ttypes.Sentence
	currentIndex int
	documentMu   sync.RWMutex

	// Processing control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Error handling
	lastError error
	errorMu   sync.RWMutex

	// Metrics and monitoring
	startTime time.Time
	stats     ControllerStats
	statsMu   sync.RWMutex
}

// ControllerStats tracks controller performance metrics
type ControllerStats struct {
	DocumentsProcessed   int64
	SentencesSynthesized int64
	PlaybackTime         time.Duration
	ErrorCount           int64
	CacheHitRate         float64
	LastActivity         time.Time
}

// NewController creates a new TTS controller with the specified configuration and dependencies.
// The dependencies (player, queue, cache) should be created externally to avoid import cycles.
func NewController(config *Config, player ttypes.AudioPlayer, queue ttypes.SentenceQueue, cache ttypes.AudioCache) (*TTSController, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if player == nil {
		return nil, fmt.Errorf("player cannot be nil")
	}
	if queue == nil {
		return nil, fmt.Errorf("queue cannot be nil")
	}
	if cache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}

	// Create speed controller
	speedCtrl := NewSpeedController()
	speedCtrl.SetSpeed(config.Speed)

	// Create sentence parser
	parser := NewSentenceParser()

	controller := &TTSController{
		config:    config,
		player:    player,
		queue:     queue,
		cache:     cache,
		parser:    parser,
		speedCtrl: speedCtrl,
		state:     ttypes.StateIdle,
		sentences: make([]ttypes.Sentence, 0),
		stats:     ControllerStats{LastActivity: time.Now()},
	}

	return controller, nil
}

// Start initializes the TTS system with the specified engine.
func (c *TTSController) Start(ctx context.Context, engineType ttypes.EngineType) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state != ttypes.StateIdle {
		return ErrControllerAlreadyStarted
	}

	c.setState(ttypes.StateInitializing)
	c.engineType = engineType
	c.startTime = time.Now()

	// Create context for background operations
	c.ctx, c.cancel = context.WithCancel(ctx)

	// Initialize components in order
	if err := c.initializeComponents(); err != nil {
		c.setState(ttypes.StateError)
		c.setError(err)
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// Start background processing
	c.wg.Add(1)
	go c.processingLoop()

	c.setState(ttypes.StateReady)
	c.updateActivity()

	return nil
}

// Stop halts TTS and releases resources.
func (c *TTSController) Stop() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state == ttypes.StateIdle || c.state == ttypes.StateStopping {
		return nil
	}

	c.setState(ttypes.StateStopping)

	// Cancel context to stop all background operations
	if c.cancel != nil {
		c.cancel()
	}

	// Wait for background goroutines to finish
	c.wg.Wait()

	// Stop audio playback
	if c.player != nil {
		if c.player.IsPlaying() {
			c.player.Stop()
		}
		c.player.Close()
	}

	// Close TTS engine
	if c.engine != nil {
		c.engine.Close()
	}

	// Clear queue
	if c.queue != nil {
		c.queue.Clear()
	}

	c.setState(ttypes.StateIdle)
	c.updateActivity()

	return nil
}

// ProcessDocument prepares a document for TTS playback.
func (c *TTSController) ProcessDocument(content string) error {
	if !c.isReady() {
		return ErrControllerNotStarted
	}

	c.setState(ttypes.StateProcessing)
	c.updateActivity()

	// Parse document into sentences
	sentences, err := c.parser.Parse(content)
	if err != nil {
		c.setState(ttypes.StateError)
		c.setError(err)
		return fmt.Errorf("failed to parse document: %w", err)
	}

	// Update document state
	c.documentMu.Lock()
	c.sentences = sentences
	c.currentIndex = 0
	c.documentMu.Unlock()

	// Clear previous queue content
	c.queue.Clear()

	// Enqueue sentences with normal priority
	for _, sentence := range sentences {
		if err := c.queue.Enqueue(sentence, false); err != nil {
			// Log error but continue with other sentences
			c.incrementErrorCount()
		}
	}

	// Update stats
	c.statsMu.Lock()
	c.stats.DocumentsProcessed++
	c.statsMu.Unlock()

	c.setState(ttypes.StateReady)
	return nil
}

// Play starts or resumes playback.
func (c *TTSController) Play() error {
	if !c.isReady() {
		return ErrControllerNotStarted
	}

	c.documentMu.RLock()
	hasContent := len(c.sentences) > 0
	c.documentMu.RUnlock()

	if !hasContent {
		return ErrNoDocumentLoaded
	}

	// If already playing, do nothing
	if c.state == ttypes.StatePlaying {
		return nil
	}

	// If paused, resume
	if c.state == ttypes.StatePaused {
		if err := c.player.Resume(); err != nil {
			c.setError(err)
			return fmt.Errorf("failed to resume playback: %w", err)
		}
		c.setState(ttypes.StatePlaying)
		return nil
	}

	// Start playback from current position
	return c.playCurrentSentence()
}

// Pause pauses playback.
func (c *TTSController) Pause() error {
	if c.state != ttypes.StatePlaying {
		return nil
	}

	if err := c.player.Pause(); err != nil {
		c.setError(err)
		return fmt.Errorf("failed to pause playback: %w", err)
	}

	c.setState(ttypes.StatePaused)
	c.updateActivity()

	return nil
}

// NextSentence navigates to the next sentence.
func (c *TTSController) NextSentence() error {
	if !c.isReady() && c.state != ttypes.StatePlaying && c.state != ttypes.StatePaused {
		return ErrControllerNotStarted
	}

	c.documentMu.Lock()
	defer c.documentMu.Unlock()

	if len(c.sentences) == 0 {
		return ErrNoDocumentLoaded
	}

	// Move to next sentence
	if c.currentIndex < len(c.sentences)-1 {
		c.currentIndex++
	}

	// Enqueue current sentence with high priority
	currentSentence := c.sentences[c.currentIndex]
	if err := c.queue.Enqueue(currentSentence, true); err != nil {
		c.setError(err)
		return fmt.Errorf("failed to enqueue sentence: %w", err)
	}

	c.updateActivity()

	// If currently playing, start playing the new sentence
	if c.state == ttypes.StatePlaying {
		return c.playCurrentSentence()
	}

	return nil
}

// PreviousSentence navigates to the previous sentence.
func (c *TTSController) PreviousSentence() error {
	if !c.isReady() && c.state != ttypes.StatePlaying && c.state != ttypes.StatePaused {
		return ErrControllerNotStarted
	}

	c.documentMu.Lock()
	defer c.documentMu.Unlock()

	if len(c.sentences) == 0 {
		return ErrNoDocumentLoaded
	}

	// Move to previous sentence
	if c.currentIndex > 0 {
		c.currentIndex--
	}

	// Enqueue current sentence with high priority
	currentSentence := c.sentences[c.currentIndex]
	if err := c.queue.Enqueue(currentSentence, true); err != nil {
		c.setError(err)
		return fmt.Errorf("failed to enqueue sentence: %w", err)
	}

	c.updateActivity()

	// If currently playing, start playing the new sentence
	if c.state == ttypes.StatePlaying {
		return c.playCurrentSentence()
	}

	return nil
}

// SetSpeed adjusts the playback speed (0.5 to 2.0).
func (c *TTSController) SetSpeed(speed float64) error {
	if speed < 0.5 || speed > 2.0 {
		return ErrInvalidSpeed
	}

	if err := c.speedCtrl.SetSpeed(speed); err != nil {
		return fmt.Errorf("failed to set speed: %w", err)
	}

	// Update config
	c.config.Speed = speed
	c.updateActivity()

	return nil
}

// GetState returns the current TTS state.
func (c *TTSController) GetState() ttypes.State {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// GetProgress returns the current playback progress.
func (c *TTSController) GetProgress() ttypes.Progress {
	c.documentMu.RLock()
	defer c.documentMu.RUnlock()

	progress := ttypes.Progress{
		TotalSentences: len(c.sentences),
	}

	if len(c.sentences) > 0 {
		progress.CurrentSentence = c.currentIndex

		// Get position from player if playing
		if c.player != nil && c.player.IsPlaying() {
			progress.CurrentPosition = c.player.GetPosition()
		}

		// Calculate processed count (approximation)
		progress.ProcessedCount = c.currentIndex

		// Get cache stats for cached count
		if c.cache != nil {
			stats := c.cache.Stats()
			// Rough estimate based on cache size
			progress.CachedCount = int(stats.Hits)
		}
	}

	return progress
}

// GetError returns the last error that occurred.
func (c *TTSController) GetError() error {
	c.errorMu.RLock()
	defer c.errorMu.RUnlock()
	return c.lastError
}

// GetStats returns controller performance statistics.
func (c *TTSController) GetStats() ControllerStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()

	stats := c.stats

	// Add cache hit rate if available
	if c.cache != nil {
		cacheStats := c.cache.Stats()
		total := cacheStats.Hits + cacheStats.Misses
		if total > 0 {
			stats.CacheHitRate = float64(cacheStats.Hits) / float64(total)
		}
	}

	return stats
}

// initializeComponents initializes the TTS engine based on the selected type.
func (c *TTSController) initializeComponents() error {
	var err error

	// Initialize TTS engine
	switch c.engineType {
	case ttypes.EnginePiper:
		piperConfig := engines.PiperConfig{
			ModelPath:  c.config.Piper.ModelPath,
			ConfigPath: c.config.Piper.ConfigPath,
			Voice:      c.config.Piper.Voice,
			SampleRate: 22050, // Standard sample rate
		}
		c.engine, err = engines.NewPiperEngine(piperConfig)

	case ttypes.EngineGoogle:
		gttsConfig := engines.GTTSConfig{
			Language:          c.config.GTTS.Language,
			Slow:              c.config.GTTS.Slow,
			TempDir:           c.config.GTTS.TempDir,
			RequestsPerMinute: c.config.GTTS.RequestsPerMinute,
		}
		c.engine, err = engines.NewGTTSEngine(gttsConfig)

	default:
		return fmt.Errorf("unsupported engine type: %s", c.engineType)
	}

	if err != nil {
		return fmt.Errorf("failed to create TTS engine: %w", err)
	}

	// Validate engine
	if err := c.engine.Validate(); err != nil {
		return fmt.Errorf("engine validation failed: %w", err)
	}

	return nil
}

// processingLoop runs the main processing loop in a background goroutine.
func (c *TTSController) processingLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-ticker.C:
			if err := c.processQueue(); err != nil {
				c.setError(err)
				c.incrementErrorCount()
			}
		}
	}
}

// processQueue processes sentences from the queue and synthesizes audio.
func (c *TTSController) processQueue() error {
	// Dequeue a sentence if available
	sentence, err := c.queue.Dequeue()
	if err != nil {
		// No sentences available, not an error
		return nil
	}

	// Check if audio is already cached
	if audio, found := c.cache.Get(sentence.CacheKey); found {
		// Audio is cached, we could play it or just mark as processed
		_ = audio // Use audio as needed
		return nil
	}

	// Synthesize audio using the TTS engine
	audio, err := c.engine.Synthesize(c.ctx, sentence.Text, c.speedCtrl.GetSpeed())
	if err != nil {
		return fmt.Errorf("synthesis failed for sentence %s: %w", sentence.ID, err)
	}

	// Cache the audio
	if err := c.cache.Put(sentence.CacheKey, audio); err != nil {
		// Log error but continue - caching failure shouldn't stop synthesis
		c.incrementErrorCount()
	}

	// Update stats
	c.statsMu.Lock()
	c.stats.SentencesSynthesized++
	c.statsMu.Unlock()

	return nil
}

// playCurrentSentence plays the audio for the current sentence.
func (c *TTSController) playCurrentSentence() error {
	c.documentMu.RLock()
	if len(c.sentences) == 0 || c.currentIndex >= len(c.sentences) {
		c.documentMu.RUnlock()
		return ErrNoDocumentLoaded
	}
	currentSentence := c.sentences[c.currentIndex]
	c.documentMu.RUnlock()

	// Get audio from cache or synthesize
	audio, found := c.cache.Get(currentSentence.CacheKey)
	if !found {
		var err error
		audio, err = c.engine.Synthesize(c.ctx, currentSentence.Text, c.speedCtrl.GetSpeed())
		if err != nil {
			c.setError(err)
			return fmt.Errorf("failed to synthesize current sentence: %w", err)
		}

		// Cache for future use
		c.cache.Put(currentSentence.CacheKey, audio)
	}

	// Stop current playback
	if c.player.IsPlaying() {
		c.player.Stop()
	}

	// Start playing new audio
	if err := c.player.Play(audio); err != nil {
		c.setError(err)
		return fmt.Errorf("failed to play audio: %w", err)
	}

	c.setState(ttypes.StatePlaying)
	return nil
}

// Helper methods for state management
func (c *TTSController) setState(newState ttypes.State) {
	c.state = newState
}

func (c *TTSController) isReady() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state == ttypes.StateReady || c.state == ttypes.StatePlaying || c.state == ttypes.StatePaused
}

func (c *TTSController) setError(err error) {
	c.errorMu.Lock()
	c.lastError = err
	c.errorMu.Unlock()
}

func (c *TTSController) updateActivity() {
	c.statsMu.Lock()
	c.stats.LastActivity = time.Now()
	c.statsMu.Unlock()
}

func (c *TTSController) incrementErrorCount() {
	c.statsMu.Lock()
	c.stats.ErrorCount++
	c.statsMu.Unlock()
}
