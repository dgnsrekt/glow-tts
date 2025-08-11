package audio

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/internal/tts"
)

// MockPlayer implements tts.AudioPlayer for testing purposes.
// It simulates audio playback without actually producing sound.
type MockPlayer struct {
	// State management
	state      atomic.Int32 // PlayerState
	startTime  time.Time
	pausedAt   time.Duration
	totalPause time.Duration

	// Audio data
	audioData     []byte
	audioDuration time.Duration
	volume        float64

	// Position tracking
	position atomic.Int64 // in nanoseconds

	// Test callbacks
	callbacks MockCallbacks

	// Synchronization
	mu         sync.RWMutex
	playbackWg sync.WaitGroup
	stopCh     chan struct{}

	// Test configuration
	simulateErrors bool
	errorRate      float64
	delayFactor    float64 // Speed up/slow down simulated playback

	// Metrics for testing
	playCount    atomic.Int64
	pauseCount   atomic.Int64
	resumeCount  atomic.Int64
	stopCount    atomic.Int64
}

// PlayerState represents the current state of the player.
type PlayerState int32

const (
	StateStopped PlayerState = iota
	StatePlaying
	StatePaused
	StateClosed
)

// MockCallbacks provides hooks for testing.
type MockCallbacks struct {
	OnPlay   func(audio []byte)
	OnPause  func()
	OnResume func()
	OnStop   func()
	OnClose  func()
}

// DefaultMockPlayer creates a new mock player with default settings.
func DefaultMockPlayer() *MockPlayer {
	mp := &MockPlayer{
		volume:      1.0,
		delayFactor: 1.0,
		stopCh:      make(chan struct{}),
	}
	mp.state.Store(int32(StateStopped))
	return mp
}

// NewMockPlayer creates a new mock player with custom callbacks.
func NewMockPlayer(callbacks MockCallbacks) *MockPlayer {
	mp := DefaultMockPlayer()
	mp.callbacks = callbacks
	return mp
}

// Play starts playback of audio data.
func (mp *MockPlayer) Play(audio []byte) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Check if closed
	if PlayerState(mp.state.Load()) == StateClosed {
		return errors.New("player is closed")
	}

	// Stop any current playback
	if PlayerState(mp.state.Load()) == StatePlaying {
		mp.stopInternal()
	}

	// Simulate error if configured
	if mp.simulateErrors && mp.shouldError() {
		return errors.New("simulated playback error")
	}

	// Store audio data (simulate keeping it alive)
	mp.audioData = make([]byte, len(audio))
	copy(mp.audioData, audio)

	// Calculate simulated duration based on audio size
	// Assume 44100Hz, 16-bit mono audio (2 bytes per sample)
	samplesPerSecond := 44100
	bytesPerSample := 2
	numSamples := len(audio) / bytesPerSample
	mp.audioDuration = time.Duration(numSamples) * time.Second / time.Duration(samplesPerSecond)

	// Reset position and timing
	mp.position.Store(0)
	mp.pausedAt = 0
	mp.totalPause = 0
	mp.startTime = time.Now()

	// Update state
	mp.state.Store(int32(StatePlaying))
	mp.playCount.Add(1)

	// Start playback simulation goroutine
	mp.stopCh = make(chan struct{})
	mp.playbackWg.Add(1)
	go mp.simulatePlayback()

	// Call test callback
	if mp.callbacks.OnPlay != nil {
		mp.callbacks.OnPlay(audio)
	}

	return nil
}

// Pause pauses the current playback.
func (mp *MockPlayer) Pause() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	currentState := PlayerState(mp.state.Load())
	if currentState != StatePlaying {
		return fmt.Errorf("cannot pause: player is %s", mp.getStateName(currentState))
	}

	// Record pause position
	mp.pausedAt = mp.getPositionInternal()
	mp.state.Store(int32(StatePaused))
	mp.pauseCount.Add(1)

	// Stop the playback simulation
	close(mp.stopCh)

	// Call test callback
	if mp.callbacks.OnPause != nil {
		mp.callbacks.OnPause()
	}

	return nil
}

// Resume resumes paused playback.
func (mp *MockPlayer) Resume() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	currentState := PlayerState(mp.state.Load())
	if currentState != StatePaused {
		return fmt.Errorf("cannot resume: player is %s", mp.getStateName(currentState))
	}

	// Store the paused position
	mp.position.Store(int64(mp.pausedAt))

	// Calculate pause duration and add to total
	pauseTime := time.Now()
	pauseDuration := pauseTime.Sub(mp.startTime.Add(mp.pausedAt))
	mp.totalPause += pauseDuration

	// Update state
	mp.state.Store(int32(StatePlaying))
	mp.resumeCount.Add(1)

	// Restart playback simulation from paused position
	mp.stopCh = make(chan struct{})
	mp.playbackWg.Add(1)
	go mp.simulatePlayback()

	// Call test callback
	if mp.callbacks.OnResume != nil {
		mp.callbacks.OnResume()
	}

	return nil
}

// Stop stops playback and releases resources.
func (mp *MockPlayer) Stop() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	return mp.stopInternal()
}

// stopInternal is the internal stop implementation without locking.
func (mp *MockPlayer) stopInternal() error {
	currentState := PlayerState(mp.state.Load())
	if currentState == StateStopped || currentState == StateClosed {
		return nil
	}

	// Stop playback simulation
	if currentState == StatePlaying || currentState == StatePaused {
		if currentState == StatePlaying {
			close(mp.stopCh)
			mp.playbackWg.Wait()
		}
	}

	// Clear audio data
	mp.audioData = nil
	mp.position.Store(0)
	mp.state.Store(int32(StateStopped))
	mp.stopCount.Add(1)

	// Call test callback
	if mp.callbacks.OnStop != nil {
		mp.callbacks.OnStop()
	}

	return nil
}

// IsPlaying returns whether audio is currently playing.
func (mp *MockPlayer) IsPlaying() bool {
	return PlayerState(mp.state.Load()) == StatePlaying
}

// GetPosition returns the current playback position.
func (mp *MockPlayer) GetPosition() time.Duration {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	return mp.getPositionInternal()
}

// getPositionInternal returns position without locking.
func (mp *MockPlayer) getPositionInternal() time.Duration {
	currentState := PlayerState(mp.state.Load())

	switch currentState {
	case StatePlaying:
		// Calculate position based on elapsed time
		elapsed := time.Since(mp.startTime) - mp.totalPause
		// Apply delay factor for testing speed adjustments
		// delayFactor < 1.0 means faster playback
		elapsed = time.Duration(float64(elapsed) / mp.delayFactor)
		if elapsed > mp.audioDuration {
			elapsed = mp.audioDuration
		}
		return elapsed

	case StatePaused:
		return mp.pausedAt

	default:
		return time.Duration(mp.position.Load())
	}
}

// SetVolume sets the playback volume (0.0 to 1.0).
func (mp *MockPlayer) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0, got %f", volume)
	}

	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.volume = volume
	return nil
}

// Close releases audio device and resources.
func (mp *MockPlayer) Close() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Stop any playback
	if PlayerState(mp.state.Load()) == StatePlaying {
		mp.stopInternal()
	}

	mp.state.Store(int32(StateClosed))

	// Call test callback
	if mp.callbacks.OnClose != nil {
		mp.callbacks.OnClose()
	}

	return nil
}

// simulatePlayback simulates audio playback progression.
func (mp *MockPlayer) simulatePlayback() {
	defer mp.playbackWg.Done()

	// Adjust ticker rate based on delay factor for more accurate simulation
	tickInterval := time.Duration(float64(100*time.Millisecond) * mp.delayFactor)
	if tickInterval < 10*time.Millisecond {
		tickInterval = 10 * time.Millisecond // Minimum tick interval
	}
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	startPos := time.Duration(mp.position.Load())
	simulationStart := time.Now()

	for {
		select {
		case <-mp.stopCh:
			return

		case <-ticker.C:
			// Update position
			elapsed := time.Since(simulationStart)
			// Apply delay factor (delayFactor < 1.0 means faster playback)
			elapsed = time.Duration(float64(elapsed) / mp.delayFactor)
			newPos := startPos + elapsed

			// Check if we've reached the end
			if newPos >= mp.audioDuration {
				mp.position.Store(int64(mp.audioDuration))
				mp.mu.Lock()
				mp.state.Store(int32(StateStopped))
				mp.mu.Unlock()

				// Call stop callback for natural end
				if mp.callbacks.OnStop != nil {
					mp.callbacks.OnStop()
				}
				return
			}

			mp.position.Store(int64(newPos))
		}
	}
}

// Test helper methods

// GetState returns the current player state for testing.
func (mp *MockPlayer) GetState() PlayerState {
	return PlayerState(mp.state.Load())
}

// GetVolume returns the current volume for testing.
func (mp *MockPlayer) GetVolume() float64 {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.volume
}

// SetDelayFactor sets the playback speed factor for testing.
// 1.0 is normal speed, 0.5 is half speed, 2.0 is double speed.
func (mp *MockPlayer) SetDelayFactor(factor float64) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.delayFactor = factor
}

// SetSimulateErrors enables error simulation for testing.
func (mp *MockPlayer) SetSimulateErrors(enabled bool, rate float64) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.simulateErrors = enabled
	mp.errorRate = rate
}

// shouldError determines if an error should be simulated.
func (mp *MockPlayer) shouldError() bool {
	// Simple deterministic error simulation based on play count
	return mp.playCount.Load()%int64(1/mp.errorRate) == 0
}

// GetMetrics returns playback metrics for testing.
func (mp *MockPlayer) GetMetrics() MockPlayerMetrics {
	return MockPlayerMetrics{
		PlayCount:   mp.playCount.Load(),
		PauseCount:  mp.pauseCount.Load(),
		ResumeCount: mp.resumeCount.Load(),
		StopCount:   mp.stopCount.Load(),
	}
}

// MockPlayerMetrics contains playback metrics for testing.
type MockPlayerMetrics struct {
	PlayCount   int64
	PauseCount  int64
	ResumeCount int64
	StopCount   int64
}

// WaitForCompletion waits for the current playback to complete naturally.
// Returns false if playback was stopped or an error occurred.
func (mp *MockPlayer) WaitForCompletion(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return false
		case <-ticker.C:
			if PlayerState(mp.state.Load()) == StateStopped {
				// Check if we reached the end
				pos := mp.GetPosition()
				return pos >= mp.audioDuration
			}
		}
	}
}

// SetAudioDuration allows tests to override the calculated duration.
func (mp *MockPlayer) SetAudioDuration(duration time.Duration) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.audioDuration = duration
}

// GetAudioData returns the currently loaded audio data for testing.
func (mp *MockPlayer) GetAudioData() []byte {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	if mp.audioData == nil {
		return nil
	}
	// Return a copy to prevent modification
	data := make([]byte, len(mp.audioData))
	copy(data, mp.audioData)
	return data
}

// getStateName returns a string representation of the state.
func (mp *MockPlayer) getStateName(state PlayerState) string {
	switch state {
	case StateStopped:
		return "stopped"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Ensure MockPlayer implements AudioPlayer interface
var _ tts.AudioPlayer = (*MockPlayer)(nil)