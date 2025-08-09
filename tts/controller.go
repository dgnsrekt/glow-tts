// Package tts provides text-to-speech functionality for Glow.
package tts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Controller orchestrates all TTS components and manages state.
type Controller struct {
	// Core components
	engine     Engine
	player     AudioPlayer
	parser     SentenceParser
	sync       Synchronizer

	// State management
	state    *State
	machine  *StateMachine
	mu       sync.RWMutex

	// Audio buffering
	audioBuffer map[int]*Audio // Pre-generated audio by sentence index
	bufferMu    sync.RWMutex

	// Sentences and content
	sentences    []Sentence
	currentIndex int

	// Configuration
	config       ControllerConfig
	engineConfig EngineConfig

	// Control channels
	stopCh      chan struct{}
	pauseCh     chan struct{}
	resumeCh    chan struct{}
	errorCh     chan error
	sentenceCh  chan int

	// Callbacks
	onStateChange    func(StateType)
	onSentenceChange func(int)
	onError          func(error)

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Flags
	isShuttingDown bool
}

// ControllerConfig holds configuration for the TTS controller.
type ControllerConfig struct {
	BufferSize        int           // Number of sentences to pre-generate
	RetryAttempts     int           // Number of retry attempts on error
	RetryDelay        time.Duration // Delay between retry attempts
	GenerationTimeout time.Duration // Timeout for audio generation
	EnableCaching     bool          // Cache generated audio
}

// DefaultControllerConfig returns a sensible default configuration.
func DefaultControllerConfig() ControllerConfig {
	return ControllerConfig{
		BufferSize:        3,
		RetryAttempts:     3,
		RetryDelay:        time.Second,
		GenerationTimeout: 30 * time.Second,
		EnableCaching:     true,
	}
}

// NewController creates a new TTS controller with the given components.
func NewController(engine Engine, player AudioPlayer, parser SentenceParser) *Controller {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Controller{
		engine:      engine,
		player:      player,
		parser:      parser,
		state:       &State{CurrentState: StateIdle},
		machine:     NewStateMachine(),
		audioBuffer: make(map[int]*Audio),
		config:      DefaultControllerConfig(),
		stopCh:      make(chan struct{}, 1),
		pauseCh:     make(chan struct{}, 1),
		resumeCh:    make(chan struct{}, 1),
		errorCh:     make(chan error, 10),
		sentenceCh:  make(chan int, 10),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Setup state machine callbacks
	c.setupStateMachine()

	return c
}

// Initialize prepares the TTS system for use.
func (c *Controller) Initialize(config EngineConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already initialized
	if c.state.CurrentState != StateIdle {
		return errors.New("TTS already initialized")
	}

	// Transition to initializing
	if !c.machine.Transition(StateInitializing) {
		return errors.New("failed to start initialization")
	}

	// Initialize the engine
	c.engineConfig = config
	if err := c.engine.Initialize(config); err != nil {
		c.machine.Transition(StateError)
		c.state.LastError = err
		return fmt.Errorf("engine initialization failed: %w", err)
	}

	// Check engine availability
	if !c.engine.IsAvailable() {
		c.machine.Transition(StateError)
		err := errors.New("engine not available after initialization")
		c.state.LastError = err
		return err
	}

	// Transition to ready
	if !c.machine.Transition(StateReady) {
		return errors.New("failed to transition to ready state")
	}

	c.state.CurrentState = StateReady
	c.notifyStateChange(StateReady)

	// Start error monitoring goroutine
	go c.monitorErrors()

	return nil
}

// SetContent parses and prepares content for TTS playback.
func (c *Controller) SetContent(markdown string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Parse sentences
	c.sentences = c.parser.Parse(markdown)
	if len(c.sentences) == 0 {
		return errors.New("no sentences found in content")
	}

	// Update state
	c.state.TotalSentences = len(c.sentences)
	c.state.Sentence = 0
	c.currentIndex = 0

	// Clear audio buffer
	c.clearAudioBuffer()

	// Pre-generate audio for first few sentences if caching enabled
	if c.config.EnableCaching && c.state.CurrentState == StateReady {
		go c.preGenerateAudio(0, min(c.config.BufferSize, len(c.sentences)))
	}

	return nil
}

// Play starts or resumes TTS playback.
func (c *Controller) Play() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we can play
	if !c.state.CanPlay() {
		return fmt.Errorf("cannot play in state: %s", c.state.CurrentState)
	}

	// Check if we have content
	if len(c.sentences) == 0 {
		return errors.New("no content to play")
	}

	// Handle resume from pause
	if c.state.CurrentState == StatePaused {
		select {
		case c.resumeCh <- struct{}{}:
		default:
		}
		c.machine.Transition(StatePlaying)
		c.state.CurrentState = StatePlaying
		c.notifyStateChange(StatePlaying)
		return nil
	}

	// Start playback
	if !c.machine.Transition(StatePlaying) {
		return errors.New("failed to start playback")
	}

	c.state.CurrentState = StatePlaying
	c.notifyStateChange(StatePlaying)

	// Start playback goroutine
	go c.playbackLoop()

	return nil
}

// Pause temporarily stops TTS playback.
func (c *Controller) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.state.CanPause() {
		return fmt.Errorf("cannot pause in state: %s", c.state.CurrentState)
	}

	select {
	case c.pauseCh <- struct{}{}:
	default:
	}

	c.machine.Transition(StatePaused)
	c.state.CurrentState = StatePaused
	c.notifyStateChange(StatePaused)

	if c.player.IsPlaying() {
		return c.player.Pause()
	}

	return nil
}

// Stop halts TTS playback and resets to the beginning.
func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.state.CanStop() {
		return fmt.Errorf("cannot stop in state: %s", c.state.CurrentState)
	}

	// Send stop signal
	select {
	case c.stopCh <- struct{}{}:
	default:
	}

	// Transition to stopping
	c.machine.Transition(StateStopping)
	c.state.CurrentState = StateStopping
	c.notifyStateChange(StateStopping)

	// Stop player
	if err := c.player.Stop(); err != nil {
		c.handleError(fmt.Errorf("failed to stop player: %w", err))
	}

	// Stop synchronizer if active
	if c.sync != nil {
		c.sync.Stop()
	}

	// Reset state
	c.state.Sentence = 0
	c.state.Position = 0
	c.currentIndex = 0

	// Transition to ready
	c.machine.Transition(StateReady)
	c.state.CurrentState = StateReady
	c.notifyStateChange(StateReady)

	return nil
}

// NextSentence moves to the next sentence.
func (c *Controller) NextSentence() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentIndex >= len(c.sentences)-1 {
		return errors.New("already at last sentence")
	}

	c.currentIndex++
	c.state.Sentence = c.currentIndex

	// Notify about sentence change
	if c.onSentenceChange != nil {
		c.onSentenceChange(c.currentIndex)
	}

	// Pre-generate upcoming sentences
	if c.config.EnableCaching {
		go c.preGenerateAudio(c.currentIndex+1, min(c.currentIndex+c.config.BufferSize, len(c.sentences)))
	}

	return nil
}

// PreviousSentence moves to the previous sentence.
func (c *Controller) PreviousSentence() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentIndex <= 0 {
		return errors.New("already at first sentence")
	}

	c.currentIndex--
	c.state.Sentence = c.currentIndex

	// Notify about sentence change
	if c.onSentenceChange != nil {
		c.onSentenceChange(c.currentIndex)
	}

	return nil
}

// GetState returns a copy of the current state.
func (c *Controller) GetState() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c.state
}

// GetCurrentSentence returns the current sentence being played.
func (c *Controller) GetCurrentSentence() (Sentence, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentIndex < 0 || c.currentIndex >= len(c.sentences) {
		return Sentence{}, errors.New("invalid sentence index")
	}

	return c.sentences[c.currentIndex], nil
}

// SetConfiguration updates the controller configuration.
func (c *Controller) SetConfiguration(config ControllerConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = config
}

// SetEngineConfig updates the engine configuration.
func (c *Controller) SetEngineConfig(config EngineConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.engineConfig = config

	// Re-initialize engine with new config if already initialized
	if c.state.CurrentState != StateIdle {
		return c.engine.Initialize(config)
	}

	return nil
}

// OnStateChange registers a callback for state changes.
func (c *Controller) OnStateChange(fn func(StateType)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = fn
}

// OnSentenceChange registers a callback for sentence changes.
func (c *Controller) OnSentenceChange(fn func(int)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onSentenceChange = fn
}

// OnError registers a callback for errors.
func (c *Controller) OnError(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = fn
}

// Shutdown gracefully stops the TTS system and releases resources.
func (c *Controller) Shutdown() error {
	c.mu.Lock()
	c.isShuttingDown = true
	c.mu.Unlock()

	// Cancel context to stop all goroutines
	c.cancel()

	// Stop playback if active
	if c.state.IsActive() {
		c.Stop()
	}

	// Transition to idle
	c.machine.Transition(StateIdle)
	c.state.CurrentState = StateIdle

	// Shutdown engine
	if err := c.engine.Shutdown(); err != nil {
		return fmt.Errorf("engine shutdown failed: %w", err)
	}

	// Clear resources
	c.clearAudioBuffer()

	// Close channels
	close(c.stopCh)
	close(c.pauseCh)
	close(c.resumeCh)
	close(c.errorCh)
	close(c.sentenceCh)

	return nil
}

// CreateBubbleTeaCommand creates a Bubble Tea command for async operations.
func (c *Controller) CreateBubbleTeaCommand() tea.Cmd {
	return func() tea.Msg {
		// This would integrate with the Bubble Tea event loop
		// Returning state changes as messages
		return TTSStateChangedMsg{
			State:    c.state.CurrentState,
			Sentence: c.state.Sentence,
			Total:    c.state.TotalSentences,
		}
	}
}

// Private helper methods

func (c *Controller) setupStateMachine() {
	// Setup enter callbacks
	c.machine.OnEnter(StateInitializing, func() {
		c.state.CurrentState = StateInitializing
	})

	c.machine.OnEnter(StateReady, func() {
		c.state.CurrentState = StateReady
	})

	c.machine.OnEnter(StatePlaying, func() {
		c.state.CurrentState = StatePlaying
	})

	c.machine.OnEnter(StateError, func() {
		c.state.CurrentState = StateError
	})
}

func (c *Controller) playbackLoop() {
	for c.currentIndex < len(c.sentences) {
		select {
		case <-c.ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-c.pauseCh:
			// Wait for resume
			select {
			case <-c.ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-c.resumeCh:
				// Continue playback
			}
		default:
			// Play current sentence
			if err := c.playSentence(c.currentIndex); err != nil {
				c.handleError(fmt.Errorf("playback error: %w", err))
				return
			}

			// Move to next sentence
			c.currentIndex++
			c.state.Sentence = c.currentIndex

			// Notify about sentence change
			if c.onSentenceChange != nil {
				c.onSentenceChange(c.currentIndex)
			}
		}
	}

	// Playback complete
	c.Stop()
}

func (c *Controller) playSentence(index int) error {
	if index >= len(c.sentences) {
		return errors.New("sentence index out of range")
	}

	sentence := c.sentences[index]

	// Get or generate audio
	audio, err := c.getOrGenerateAudio(index)
	if err != nil {
		return fmt.Errorf("failed to get audio for sentence %d: %w", index, err)
	}

	// Play the audio
	if err := c.player.Play(audio); err != nil {
		return fmt.Errorf("failed to play audio: %w", err)
	}

	// Wait for playback to complete
	// This is simplified - real implementation would monitor player state
	time.Sleep(sentence.Duration)

	return nil
}

func (c *Controller) getOrGenerateAudio(index int) (*Audio, error) {
	// Check buffer first
	c.bufferMu.RLock()
	if audio, ok := c.audioBuffer[index]; ok {
		c.bufferMu.RUnlock()
		return audio, nil
	}
	c.bufferMu.RUnlock()

	// Generate audio
	sentence := c.sentences[index]
	ctx, cancel := context.WithTimeout(c.ctx, c.config.GenerationTimeout)
	defer cancel()

	// Generate with retry logic
	var audio *Audio
	var err error
	for attempt := 0; attempt < c.config.RetryAttempts; attempt++ {
		audio, err = c.generateAudioWithContext(ctx, sentence.Text)
		if err == nil {
			break
		}

		if attempt < c.config.RetryAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.config.RetryDelay):
				// Retry after delay
			}
		}
	}

	if err != nil {
		return nil, err
	}

	// Cache if enabled
	if c.config.EnableCaching {
		c.bufferMu.Lock()
		c.audioBuffer[index] = audio
		c.bufferMu.Unlock()
	}

	return audio, nil
}

func (c *Controller) generateAudioWithContext(ctx context.Context, text string) (*Audio, error) {
	// This would integrate with the engine's async generation
	// For now, use synchronous generation
	return c.engine.GenerateAudio(text)
}

func (c *Controller) preGenerateAudio(start, end int) {
	for i := start; i < end && i < len(c.sentences); i++ {
		// Skip if already cached
		c.bufferMu.RLock()
		_, exists := c.audioBuffer[i]
		c.bufferMu.RUnlock()
		if exists {
			continue
		}

		// Generate and cache
		if _, err := c.getOrGenerateAudio(i); err != nil {
			// Log error but don't stop pre-generation
			c.handleError(fmt.Errorf("pre-generation failed for sentence %d: %w", i, err))
		}
	}
}

func (c *Controller) clearAudioBuffer() {
	c.bufferMu.Lock()
	defer c.bufferMu.Unlock()
	c.audioBuffer = make(map[int]*Audio)
}

func (c *Controller) monitorErrors() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case err := <-c.errorCh:
			c.handleError(err)
		}
	}
}

func (c *Controller) handleError(err error) {
	c.mu.Lock()
	c.state.LastError = err
	c.mu.Unlock()

	if c.onError != nil {
		c.onError(err)
	}
}

func (c *Controller) notifyStateChange(state StateType) {
	if c.onStateChange != nil {
		c.onStateChange(state)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}