package tts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	
	"github.com/charmbracelet/log"
)

// ControllerState represents the current state of the TTS controller.
type ControllerState int

const (
	// StateUninitialized indicates the controller has not been initialized yet.
	StateUninitialized ControllerState = iota
	// StateReady indicates the controller is initialized and ready to operate.
	StateReady
	// StateRunning indicates the controller is actively processing TTS operations.
	StateRunning
	// StateStopping indicates the controller is in the process of shutting down.
	StateStopping
	// StateStopped indicates the controller has been stopped.
	StateStopped
)

// String returns a string representation of the controller state.
func (s ControllerState) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateReady:
		return "ready"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Controller orchestrates the entire TTS pipeline.
type Controller struct {
	// engine is the TTS engine used for synthesis (Piper, Google TTS, etc.)
	engine TTSEngine

	// queue manages the audio buffer and preprocessing pipeline
	queue *TTSAudioQueue

	// player handles cross-platform audio playback
	player *AudioPlayer

	// parser extracts sentences from markdown documents
	parser TextParser

	// cache manages two-level caching (memory and disk)
	cache CacheManager

	// speedCtrl manages playback speed control
	speedCtrl SpeedController

	// state management
	stateMu sync.RWMutex
	state   ControllerState

	// context for lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// wg tracks active goroutines
	wg sync.WaitGroup

	// config holds controller configuration
	config ControllerConfig
}

// ControllerConfig holds configuration for the TTS controller.
type ControllerConfig struct {
	// Engine specifies which TTS engine to use ("piper" or "gtts")
	Engine string

	// EnableCache enables/disables caching
	EnableCache bool

	// CacheDir specifies the directory for disk cache
	CacheDir string

	// MaxMemoryCacheSize specifies maximum memory cache size in bytes
	MaxMemoryCacheSize int64

	// MaxDiskCacheSize specifies maximum disk cache size in bytes
	MaxDiskCacheSize int64

	// LookaheadSentences specifies how many sentences to preprocess
	LookaheadSentences int

	// DefaultSpeed specifies the default playback speed (1.0 = normal)
	DefaultSpeed float64
}

// NewController creates a new TTS controller with the given configuration.
func NewController(cfg ControllerConfig) (*Controller, error) {
	if cfg.LookaheadSentences <= 0 {
		cfg.LookaheadSentences = 3
	}
	if cfg.DefaultSpeed <= 0 {
		cfg.DefaultSpeed = 1.0
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Controller{
		config: cfg,
		state:  StateUninitialized,
		ctx:    ctx,
		cancel: cancel,
	}

	return c, nil
}

// SetEngine sets the TTS engine for the controller.
func (c *Controller) SetEngine(engine TTSEngine) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state != StateUninitialized && c.state != StateReady {
		return fmt.Errorf("cannot set engine in state %s", c.state)
	}

	c.engine = engine
	return nil
}

// SetParser sets the text parser for the controller.
func (c *Controller) SetParser(parser TextParser) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state != StateUninitialized && c.state != StateReady {
		return fmt.Errorf("cannot set parser in state %s", c.state)
	}

	c.parser = parser
	return nil
}

// SetCache sets the cache manager for the controller.
func (c *Controller) SetCache(cache CacheManager) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state != StateUninitialized && c.state != StateReady {
		return fmt.Errorf("cannot set cache in state %s", c.state)
	}

	c.cache = cache
	return nil
}

// SetSpeedController sets the speed controller.
func (c *Controller) SetSpeedController(speedCtrl SpeedController) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state != StateUninitialized && c.state != StateReady {
		return fmt.Errorf("cannot set speed controller in state %s", c.state)
	}

	c.speedCtrl = speedCtrl
	return nil
}

// SetSpeed sets the playback speed.
func (c *Controller) SetSpeed(speed float64) error {
	if c.speedCtrl == nil {
		return fmt.Errorf("speed controller not initialized")
	}
	return c.speedCtrl.SetSpeed(speed)
}

// GetSpeed returns the current playback speed.
func (c *Controller) GetSpeed() float64 {
	if c.speedCtrl == nil {
		return 1.0
	}
	return c.speedCtrl.GetSpeed()
}

// Play starts or resumes TTS playback.
func (c *Controller) Play(text string) error {
	c.stateMu.Lock()
	state := c.state
	// Set state to running when we start playing
	if state == StateReady {
		c.state = StateRunning
	}
	c.stateMu.Unlock()
	
	if state != StateReady && state != StateRunning {
		return fmt.Errorf("cannot play in state %s", state)
	}
	
	// Check if we have a queue for advanced playback
	if c.queue != nil {
		// Use the queue for buffered playback with lookahead
		err := c.queue.AddText(text)
		if err != nil {
			return fmt.Errorf("failed to add text to queue: %w", err)
		}
		
		// Start preloading in the background
		go c.queue.Preload()
		
		// Wait for the first segment to be ready
		err = c.queue.WaitForReady(5 * time.Second)
		if err != nil {
			return fmt.Errorf("queue not ready: %w", err)
		}
		
		// Get the first segment and play it
		segment, err := c.queue.GetCurrent()
		if err != nil {
			return fmt.Errorf("failed to get current segment: %w", err)
		}
		
		if segment != nil {
			// Use ProcessedAudio if available, otherwise fall back to raw Audio
			audioToPlay := segment.ProcessedAudio
			if len(audioToPlay) == 0 {
				log.Debug("Controller: ProcessedAudio not available, using raw Audio",
					"hasAudio", segment.Audio != nil,
					"audioSize", len(segment.Audio))
				audioToPlay = segment.Audio
			}
			
			if len(audioToPlay) > 0 {
				player := GetGlobalAudioPlayer()
				if player == nil {
					return fmt.Errorf("audio player not initialized")
				}
				return player.PlayPCM(audioToPlay)
			}
		}
		
		return fmt.Errorf("no audio available")
	}
	
	// Fallback to simple playback without queue
	// Parse text into sentences
	if c.parser == nil {
		return fmt.Errorf("text parser not initialized")
	}
	
	sentences, err := c.parser.ParseSentences(text)
	if err != nil {
		return fmt.Errorf("failed to parse text: %w", err)
	}
	
	// Limit the number of sentences to process at once to reduce memory usage
	const maxSentences = 50
	sentencesToProcess := sentences
	if len(sentences) > maxSentences {
		sentencesToProcess = sentences[:maxSentences]
		// Log that we're limiting the sentences
		fmt.Printf("TTS: Processing first %d of %d sentences to reduce memory usage\n", maxSentences, len(sentences))
	}
	
	// Start playback of selected sentences
	if len(sentencesToProcess) > 0 {
		// Concatenate all sentences with brief pauses
		var fullText strings.Builder
		for i, sentence := range sentencesToProcess {
			fullText.WriteString(sentence.Text)
			if i < len(sentencesToProcess)-1 {
				fullText.WriteString(". ") // Add pause between sentences
			}
		}
		
		// Synthesize the full text
		audio, err := c.engine.Synthesize(fullText.String(), c.GetSpeed())
		if err != nil {
			return fmt.Errorf("synthesis failed: %w", err)
		}
		
		// Play the audio using the global player
		player := GetGlobalAudioPlayer()
		if player == nil {
			return fmt.Errorf("audio player not initialized")
		}
		
		return player.PlayPCM(audio)
	}
	
	return nil
}

// Pause pauses TTS playback.
func (c *Controller) Pause() error {
	c.stateMu.RLock()
	state := c.state
	c.stateMu.RUnlock()
	
	if state != StateRunning {
		return fmt.Errorf("cannot pause: not running")
	}
	
	// Pause the audio player
	player := GetGlobalAudioPlayer()
	if player != nil {
		return player.Pause()
	}
	
	return nil
}

// Resume resumes TTS playback.
func (c *Controller) Resume() error {
	c.stateMu.RLock()
	state := c.state
	c.stateMu.RUnlock()
	
	if state != StateRunning {
		return fmt.Errorf("cannot resume: not running")
	}
	
	// Resume the audio player
	player := GetGlobalAudioPlayer()
	if player != nil {
		return player.Resume()
	}
	
	return nil
}

// Next moves to the next sentence/segment in the queue.
func (c *Controller) Next() error {
	if c.queue == nil {
		return fmt.Errorf("queue not initialized")
	}
	
	// Get the next segment from the queue
	segment, err := c.queue.Next()
	if err != nil {
		return fmt.Errorf("failed to get next segment: %w", err)
	}
	
	if segment != nil && segment.ProcessedAudio != nil {
		// Stop current playback
		player := GetGlobalAudioPlayer()
		if player != nil {
			if err := player.Stop(); err != nil {
				// Log error but continue
			}
			// Play the next segment
			return player.PlayPCM(segment.ProcessedAudio)
		}
	}
	
	return fmt.Errorf("no next segment available")
}

// Previous moves to the previous sentence/segment in the queue.
func (c *Controller) Previous() error {
	if c.queue == nil {
		return fmt.Errorf("queue not initialized")
	}
	
	// Get the previous segment from the queue
	segment, err := c.queue.Previous()
	if err != nil {
		return fmt.Errorf("failed to get previous segment: %w", err)
	}
	
	if segment != nil && segment.ProcessedAudio != nil {
		// Stop current playback
		player := GetGlobalAudioPlayer()
		if player != nil {
			if err := player.Stop(); err != nil {
				// Log error but continue
			}
			// Play the previous segment
			return player.PlayPCM(segment.ProcessedAudio)
		}
	}
	
	return fmt.Errorf("no previous segment available")
}


// GetState returns the current controller state.
func (c *Controller) GetState() ControllerState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// setState sets the controller state with proper locking.
func (c *Controller) setState(state ControllerState) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.state = state
}

// TextParser defines the interface for parsing text documents.
type TextParser interface {
	// ParseSentences extracts sentences from the given text.
	ParseSentences(text string) ([]Sentence, error)
}

// Sentence represents a single sentence extracted from text.
type Sentence struct {
	// Text is the cleaned text ready for synthesis
	Text string

	// Position is the sentence index in the document
	Position int

	// Original is the original text with formatting
	Original string
}

// CacheManager defines the interface for managing TTS cache.
type CacheManager interface {
	// Get retrieves cached audio data for the given key.
	Get(key string) ([]byte, bool)

	// Set stores audio data in the cache.
	Set(key string, data []byte) error

	// Clear removes all cached data.
	Clear() error

	// GenerateKey creates a cache key from the given parameters.
	GenerateKey(text string, voice string, speed float64) string
}

// SpeedController defines the interface for managing playback speed.
type SpeedController interface {
	// GetSpeed returns the current speed setting.
	GetSpeed() float64

	// SetSpeed sets the playback speed.
	SetSpeed(speed float64) error

	// IncreaseSpeed increases the speed by one step.
	IncreaseSpeed() error

	// DecreaseSpeed decreases the speed by one step.
	DecreaseSpeed() error

	// GetSpeedSteps returns available speed steps.
	GetSpeedSteps() []float64
}

// AudioPlayer handles cross-platform audio playback.
type AudioPlayer struct {
	// This will be implemented in a separate file
}

// Initialize validates and sets up all controller components.
func (c *Controller) Initialize() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	// Check current state
	if c.state != StateUninitialized {
		return fmt.Errorf("controller already initialized (state: %s)", c.state)
	}

	// Collect initialization errors
	var errors []error

	// Validate engine
	if c.engine == nil {
		errors = append(errors, fmt.Errorf("TTS engine not set"))
	} else if err := c.engine.Validate(); err != nil {
		errors = append(errors, fmt.Errorf("engine validation failed: %w", err))
	}

	// Initialize parser if not set
	if c.parser == nil {
		// Use default parser (to be implemented)
		errors = append(errors, fmt.Errorf("text parser not set"))
	}

	// Initialize cache if enabled
	if c.config.EnableCache && c.cache == nil {
		errors = append(errors, fmt.Errorf("cache enabled but cache manager not set"))
	}

	// Initialize speed controller if not set
	if c.speedCtrl == nil {
		errors = append(errors, fmt.Errorf("speed controller not set"))
	}

	// Initialize audio queue
	if c.queue == nil && c.engine != nil {
		queueConfig := DefaultQueueConfig()
		queueConfig.Engine = c.engine
		queueConfig.Parser = c.parser
		// Note: Cache manager types are incompatible for now
		// TODO: Create adapter or unify cache interfaces
		
		queue, err := NewAudioQueue(queueConfig)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create audio queue: %w", err))
		} else {
			c.queue = queue
		}
	}

	// Initialize audio player
	if c.player == nil {
		c.player = &AudioPlayer{
			// Will be properly initialized when AudioPlayer is implemented
		}
	}

	// Check for errors
	if len(errors) > 0 {
		return fmt.Errorf("initialization failed with %d errors: %v", len(errors), errors)
	}

	// Set state to ready
	c.state = StateReady
	return nil
}

// Start begins TTS operations with proper goroutine management.
func (c *Controller) Start(ctx context.Context) error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	// Check current state
	if c.state != StateReady {
		return fmt.Errorf("cannot start controller in state %s", c.state)
	}

	// Create or update context
	if ctx != nil {
		c.ctx, c.cancel = context.WithCancel(ctx)
	} else {
		c.ctx, c.cancel = context.WithCancel(context.Background())
	}

	// Set state to running
	c.state = StateRunning

	// Start background workers with panic recovery
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.recoverPanic("audio processing")
		c.processAudioQueue()
	}()

	return nil
}

// Stop performs graceful shutdown with resource cleanup.
func (c *Controller) Stop() error {
	c.stateMu.Lock()
	
	// Check if already stopping or stopped
	if c.state == StateStopping || c.state == StateStopped {
		c.stateMu.Unlock()
		return nil
	}

	// Set state to stopping
	c.state = StateStopping
	c.stateMu.Unlock()

	// Cancel context to signal shutdown
	if c.cancel != nil {
		c.cancel()
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		// Timeout waiting for goroutines
		return fmt.Errorf("timeout waiting for goroutines to finish")
	}

	// Clean up resources
	var errors []error

	// Stop audio player
	player := GetGlobalAudioPlayer()
	if player != nil {
		if err := player.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop audio player: %w", err))
		}
	}

	// Clear cache if configured
	if c.cache != nil && c.config.EnableCache {
		if err := c.cache.Clear(); err != nil {
			errors = append(errors, fmt.Errorf("failed to clear cache: %w", err))
		}
	}

	// Set final state
	c.setState(StateStopped)

	if len(errors) > 0 {
		return fmt.Errorf("stop completed with errors: %v", errors)
	}

	return nil
}

// recoverPanic recovers from panics in goroutines.
func (c *Controller) recoverPanic(context string) {
	if r := recover(); r != nil {
		// Log the panic (when logging is implemented)
		fmt.Printf("TTS Controller: panic in %s: %v\n", context, r)
		
		// Attempt graceful shutdown
		c.setState(StateStopping)
		if c.cancel != nil {
			c.cancel()
		}
	}
}

// processAudioQueue manages the audio queue background processing.
func (c *Controller) processAudioQueue() {
	// The queue has its own internal workers and processing
	// This goroutine just monitors the context for shutdown
	<-c.ctx.Done()
	
	// Stop the queue when context is cancelled
	if c.queue != nil {
		c.queue.Stop()
	}
}

// IsRunning returns true if the controller is currently running.
func (c *Controller) IsRunning() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state == StateRunning
}

// WaitForReady waits for the controller to be ready or returns an error.
func (c *Controller) WaitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		state := c.GetState()
		switch state {
		case StateReady, StateRunning:
			return nil
		case StateStopped, StateStopping:
			return fmt.Errorf("controller is shutting down")
		}
		time.Sleep(100 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for controller to be ready")
}

// GetQueue returns the audio queue if available
func (c *Controller) GetQueue() *TTSAudioQueue {
	return c.queue
}

// GetCache returns the cache manager if available
func (c *Controller) GetCache() *Cache {
	// Note: We currently have a type mismatch between CacheManager interface
	// and Cache implementation. For now, return nil.
	// TODO: Unify cache interfaces
	return nil
}