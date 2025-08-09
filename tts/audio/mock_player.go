// Package audio provides audio playback functionality for TTS.
package audio

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// MockPlayer implements AudioPlayer interface for testing.
// It simulates audio playback with accurate timing without actual audio output.
type MockPlayer struct {
	// State management
	mu       sync.RWMutex
	playing  int32 // atomic: 1 if playing, 0 if not
	paused   int32 // atomic: 1 if paused, 0 if not
	position time.Duration
	duration time.Duration

	// Current audio
	currentAudio *tts.Audio

	// Timing simulation
	ticker       *time.Ticker
	tickInterval time.Duration
	startTime    time.Time
	pauseTime    time.Time
	pausedDur    time.Duration

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}

	// Test control
	speedMultiplier float64 // Allows tests to speed up/slow down playback
	callbacks       MockCallbacks
	history         []PlaybackEvent

	// Error injection for testing
	playError   error
	pauseError  error
	resumeError error
	stopError   error
}

// MockCallbacks holds callback functions for testing.
type MockCallbacks struct {
	OnPlay   func(audio *tts.Audio)
	OnPause  func()
	OnResume func()
	OnStop   func()
	OnTick   func(position time.Duration)
}

// PlaybackEvent records an event for testing verification.
type PlaybackEvent struct {
	Type      string
	Timestamp time.Time
	Position  time.Duration
	Audio     *tts.Audio
}

// NewMockPlayer creates a new mock audio player for testing.
func NewMockPlayer() *MockPlayer {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockPlayer{
		tickInterval:    10 * time.Millisecond,
		speedMultiplier: 1.0,
		ctx:             ctx,
		cancel:          cancel,
		stopCh:          make(chan struct{}, 1), // Buffered to prevent blocking
		history:         make([]PlaybackEvent, 0),
	}
}

// Play starts playing the given audio.
func (mp *MockPlayer) Play(audio *tts.Audio) error {
	// Check for injected error
	if mp.playError != nil {
		return mp.playError
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if already playing
	if atomic.LoadInt32(&mp.playing) == 1 && atomic.LoadInt32(&mp.paused) == 0 {
		return errors.New("already playing")
	}

	// Reset state for new playback
	mp.currentAudio = audio
	mp.position = 0
	mp.pausedDur = 0
	
	if audio != nil {
		mp.duration = audio.Duration
	} else {
		mp.duration = 0
	}

	// Set playing state
	atomic.StoreInt32(&mp.playing, 1)
	atomic.StoreInt32(&mp.paused, 0)

	// Record event
	mp.recordEvent("play", audio)

	// Start timing simulation
	mp.startTiming()

	// Trigger callback
	if mp.callbacks.OnPlay != nil {
		mp.callbacks.OnPlay(audio)
	}

	return nil
}

// Pause temporarily stops playback.
func (mp *MockPlayer) Pause() error {
	// Check for injected error
	if mp.pauseError != nil {
		return mp.pauseError
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if playing and not already paused
	if atomic.LoadInt32(&mp.playing) == 0 {
		return errors.New("not playing")
	}
	if atomic.LoadInt32(&mp.paused) == 1 {
		return errors.New("already paused")
	}

	// Set paused state
	atomic.StoreInt32(&mp.paused, 1)
	mp.pauseTime = time.Now()

	// Stop ticker
	if mp.ticker != nil {
		mp.ticker.Stop()
		mp.ticker = nil
	}

	// Record event
	mp.recordEvent("pause", nil)

	// Trigger callback
	if mp.callbacks.OnPause != nil {
		mp.callbacks.OnPause()
	}

	return nil
}

// Resume continues playback from paused position.
func (mp *MockPlayer) Resume() error {
	// Check for injected error
	if mp.resumeError != nil {
		return mp.resumeError
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if paused
	if atomic.LoadInt32(&mp.paused) == 0 {
		return errors.New("not paused")
	}

	// Calculate paused duration
	if !mp.pauseTime.IsZero() {
		mp.pausedDur += time.Since(mp.pauseTime)
		mp.pauseTime = time.Time{}
	}

	// Resume state
	atomic.StoreInt32(&mp.paused, 0)

	// Restart timing
	mp.startTiming()

	// Record event
	mp.recordEvent("resume", nil)

	// Trigger callback
	if mp.callbacks.OnResume != nil {
		mp.callbacks.OnResume()
	}

	return nil
}

// Stop halts playback and resets position.
func (mp *MockPlayer) Stop() error {
	// Check for injected error
	if mp.stopError != nil {
		return mp.stopError
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if playing
	if atomic.LoadInt32(&mp.playing) == 0 {
		return nil // Already stopped
	}

	// Stop ticker
	if mp.ticker != nil {
		mp.ticker.Stop()
		mp.ticker = nil
	}

	// Reset state
	atomic.StoreInt32(&mp.playing, 0)
	atomic.StoreInt32(&mp.paused, 0)
	mp.position = 0
	mp.pausedDur = 0
	mp.currentAudio = nil
	mp.startTime = time.Time{}
	mp.pauseTime = time.Time{}

	// Signal stop (non-blocking)
	select {
	case mp.stopCh <- struct{}{}:
	default:
		// Channel might be full, drain it first
		select {
		case <-mp.stopCh:
		default:
		}
		// Try again
		select {
		case mp.stopCh <- struct{}{}:
		default:
		}
	}

	// Record event
	mp.recordEvent("stop", nil)

	// Trigger callback
	if mp.callbacks.OnStop != nil {
		mp.callbacks.OnStop()
	}

	return nil
}

// GetPosition returns the current playback position.
func (mp *MockPlayer) GetPosition() time.Duration {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.position
}

// IsPlaying returns true if audio is currently playing.
func (mp *MockPlayer) IsPlaying() bool {
	return atomic.LoadInt32(&mp.playing) == 1 && atomic.LoadInt32(&mp.paused) == 0
}

// startTiming starts the timing simulation.
func (mp *MockPlayer) startTiming() {
	// Stop existing ticker
	if mp.ticker != nil {
		mp.ticker.Stop()
		mp.ticker = nil
	}

	// Set start time if new playback
	if mp.position == 0 {
		mp.startTime = time.Now()
		mp.pausedDur = 0
	}

	// Create ticker with speed multiplier
	interval := time.Duration(float64(mp.tickInterval) / mp.speedMultiplier)
	if interval <= 0 {
		interval = mp.tickInterval
	}
	mp.ticker = time.NewTicker(interval)

	// Start position update goroutine
	go mp.updatePosition()
}

// updatePosition updates the playback position based on elapsed time.
func (mp *MockPlayer) updatePosition() {
	// Get a local reference to the ticker channel
	mp.mu.RLock()
	ticker := mp.ticker
	mp.mu.RUnlock()
	
	if ticker == nil {
		return
	}
	
	for {
		select {
		case <-mp.ctx.Done():
			return
		case <-mp.stopCh:
			return
		case <-ticker.C:
			mp.mu.Lock()
			
			// Check if still playing and not paused
			if atomic.LoadInt32(&mp.playing) == 0 || atomic.LoadInt32(&mp.paused) == 1 {
				mp.mu.Unlock()
				return
			}

			// Calculate position based on elapsed time
			if !mp.startTime.IsZero() {
				elapsed := time.Since(mp.startTime) - mp.pausedDur
				// Apply speed multiplier
				mp.position = time.Duration(float64(elapsed) * mp.speedMultiplier)

				// Check if reached end
				if mp.duration > 0 && mp.position >= mp.duration {
					mp.position = mp.duration
					
					// Stop ticker
					if mp.ticker != nil {
						mp.ticker.Stop()
						mp.ticker = nil
					}
					
					// Update state to stopped but keep position at duration
					atomic.StoreInt32(&mp.playing, 0)
					atomic.StoreInt32(&mp.paused, 0)
					
					// Record event
					mp.recordEvent("stop", nil)
					
					// Trigger callback
					if mp.callbacks.OnStop != nil {
						mp.mu.Unlock()
						mp.callbacks.OnStop()
					} else {
						mp.mu.Unlock()
					}
					return
				}

				// Trigger tick callback
				if mp.callbacks.OnTick != nil {
					pos := mp.position
					mp.mu.Unlock()
					mp.callbacks.OnTick(pos)
				} else {
					mp.mu.Unlock()
				}
			} else {
				mp.mu.Unlock()
			}
		}
	}
}

// recordEvent records a playback event for testing.
func (mp *MockPlayer) recordEvent(eventType string, audio *tts.Audio) {
	event := PlaybackEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Position:  mp.position,
		Audio:     audio,
	}
	mp.history = append(mp.history, event)
}

// Test Control Methods

// SetSpeedMultiplier sets the playback speed multiplier for testing.
// 1.0 = normal speed, 2.0 = double speed, 0.5 = half speed.
func (mp *MockPlayer) SetSpeedMultiplier(multiplier float64) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	if multiplier <= 0 {
		multiplier = 1.0
	}
	mp.speedMultiplier = multiplier
	
	// Restart ticker if playing
	if mp.IsPlaying() {
		mp.startTiming()
	}
}

// SetCallbacks sets the test callbacks.
func (mp *MockPlayer) SetCallbacks(callbacks MockCallbacks) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.callbacks = callbacks
}

// GetHistory returns the playback event history.
func (mp *MockPlayer) GetHistory() []PlaybackEvent {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	
	// Return a copy to prevent external modification
	history := make([]PlaybackEvent, len(mp.history))
	copy(history, mp.history)
	return history
}

// ClearHistory clears the playback event history.
func (mp *MockPlayer) ClearHistory() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.history = mp.history[:0]
}

// GetCurrentAudio returns the currently loaded audio.
func (mp *MockPlayer) GetCurrentAudio() *tts.Audio {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.currentAudio
}

// GetState returns the current player state for testing.
func (mp *MockPlayer) GetState() (playing, paused bool, position, duration time.Duration) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	
	return atomic.LoadInt32(&mp.playing) == 1,
		atomic.LoadInt32(&mp.paused) == 1,
		mp.position,
		mp.duration
}

// SetPosition sets the playback position for testing.
func (mp *MockPlayer) SetPosition(position time.Duration) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	if position < 0 {
		position = 0
	}
	if mp.duration > 0 && position > mp.duration {
		position = mp.duration
	}
	
	mp.position = position
	
	// Adjust start time to match new position
	if atomic.LoadInt32(&mp.playing) == 1 {
		mp.startTime = time.Now().Add(-time.Duration(float64(position) / mp.speedMultiplier))
	}
}

// InjectError injects an error for testing specific error conditions.
func (mp *MockPlayer) InjectError(method string, err error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	switch method {
	case "play":
		mp.playError = err
	case "pause":
		mp.pauseError = err
	case "resume":
		mp.resumeError = err
	case "stop":
		mp.stopError = err
	}
}

// ClearErrors clears all injected errors.
func (mp *MockPlayer) ClearErrors() {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	
	mp.playError = nil
	mp.pauseError = nil
	mp.resumeError = nil
	mp.stopError = nil
}

// WaitForPosition waits until the specified position is reached or timeout occurs.
func (mp *MockPlayer) WaitForPosition(target time.Duration, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-deadline:
			return errors.New("timeout waiting for position")
		case <-ticker.C:
			if mp.GetPosition() >= target {
				return nil
			}
			if !mp.IsPlaying() {
				return errors.New("playback stopped before reaching position")
			}
		}
	}
}

// SimulateCompletion simulates the completion of current audio playback.
func (mp *MockPlayer) SimulateCompletion() {
	mp.mu.Lock()
	
	if mp.duration > 0 && atomic.LoadInt32(&mp.playing) == 1 {
		// Set position to duration
		mp.position = mp.duration
		
		// Stop ticker
		if mp.ticker != nil {
			mp.ticker.Stop()
			mp.ticker = nil
		}
		
		// Update state to stopped but keep position
		atomic.StoreInt32(&mp.playing, 0)
		atomic.StoreInt32(&mp.paused, 0)
		
		// Record event
		mp.recordEvent("stop", nil)
		
		// Trigger callback
		if mp.callbacks.OnStop != nil {
			mp.mu.Unlock()
			mp.callbacks.OnStop()
		} else {
			mp.mu.Unlock()
		}
	} else {
		mp.mu.Unlock()
	}
}

// Close cleans up the mock player resources.
func (mp *MockPlayer) Close() {
	// Stop playback first
	mp.Stop()
	
	// Cancel context
	mp.cancel()
	
	// Stop ticker if exists
	mp.mu.Lock()
	if mp.ticker != nil {
		mp.ticker.Stop()
		mp.ticker = nil
	}
	mp.mu.Unlock()
}