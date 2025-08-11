package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Player implements tts.AudioPlayer for cross-platform audio playback using oto.
// It manages memory carefully to prevent GC issues that cause audio static.
type Player struct {
	// OTO context - initialized once and reused
	context *oto.Context

	// Current playback state
	player *oto.Player

	// CRITICAL: Keep audio data alive during playback
	activeStream *AudioStream

	// State management
	state    atomic.Int32  // PlayerState from mock_player.go
	volume   atomic.Uint64 // float64 as uint64 bits
	position atomic.Int64  // nanoseconds

	// Timing and position tracking
	startTime  time.Time
	pausedAt   time.Duration
	totalPause time.Duration

	// Synchronization
	mu      sync.RWMutex
	stateMu sync.Mutex // Separate mutex for state changes

	// Configuration
	sampleRate int
	channels   int
	bitDepth   int
	bufferSize int

	// Error handling
	lastError error
	errorMu   sync.RWMutex
}

// AudioStream represents an active audio stream with data kept alive.
// CRITICAL: This prevents GC of audio data during playback.
type AudioStream struct {
	data       []byte        // Must stay alive during playback!
	reader     io.ReadSeeker // Created from data
	size       int
	duration   time.Duration
	sampleRate int
	channels   int

	// Synchronization for cleanup
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// PlayerConfig contains configuration for the audio player.
type PlayerConfig struct {
	SampleRate int // 44100 or 48000 Hz only
	Channels   int // 1 = mono, 2 = stereo
	BitDepth   int // 16 bits per sample
	BufferSize int // Buffer size for streaming
}

// DefaultPlayerConfig returns the default player configuration.
func DefaultPlayerConfig() PlayerConfig {
	return PlayerConfig{
		SampleRate: 44100, // CD quality
		Channels:   1,     // Mono for TTS
		BitDepth:   16,    // Standard bit depth
		BufferSize: 4096,  // 4KB buffer
	}
}

// NewPlayer creates a new audio player with the specified configuration.
func NewPlayer(config PlayerConfig) (*Player, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize OTO context
	op := &oto.NewContextOptions{
		SampleRate:   config.SampleRate,
		ChannelCount: config.Channels,
		Format:       oto.FormatSignedInt16LE, // 16-bit little endian
		BufferSize:   time.Duration(config.BufferSize) * time.Second / time.Duration(config.SampleRate*config.Channels*2),
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("failed to create oto context: %w", err)
	}

	// Wait for context to be ready
	<-readyChan

	player := &Player{
		context:    ctx,
		sampleRate: config.SampleRate,
		channels:   config.Channels,
		bitDepth:   config.BitDepth,
		bufferSize: config.BufferSize,
	}

	// Initialize state
	player.state.Store(int32(StateStopped))
	player.SetVolume(1.0) // Full volume by default

	return player, nil
}

// validateConfig validates the player configuration.
func validateConfig(config PlayerConfig) error {
	// OTO only supports specific sample rates reliably
	if config.SampleRate != 44100 && config.SampleRate != 48000 {
		return fmt.Errorf("sample rate must be 44100 or 48000 Hz, got %d", config.SampleRate)
	}

	if config.Channels != 1 && config.Channels != 2 {
		return fmt.Errorf("channels must be 1 (mono) or 2 (stereo), got %d", config.Channels)
	}

	if config.BitDepth != 16 {
		return fmt.Errorf("bit depth must be 16, got %d", config.BitDepth)
	}

	if config.BufferSize <= 0 {
		return errors.New("buffer size must be positive")
	}

	return nil
}

// Play starts playback of audio data.
// CRITICAL: This implementation keeps the audio data alive during playback.
func (p *Player) Play(audio []byte) error {
	if len(audio) == 0 {
		return errors.New("audio data is empty")
	}

	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	// Check if player is closed
	if PlayerState(p.state.Load()) == StateClosed {
		return errors.New("player is closed")
	}

	// Stop any current playback
	if err := p.stopInternal(); err != nil {
		return fmt.Errorf("failed to stop current playback: %w", err)
	}

	// Create audio stream with data kept alive
	stream, err := p.createAudioStream(audio)
	if err != nil {
		return fmt.Errorf("failed to create audio stream: %w", err)
	}

	// Create OTO player from stream
	player := p.context.NewPlayer(stream.reader)
	if player == nil {
		return errors.New("failed to create oto player")
	}

	// Apply current volume
	volume := p.getVolume()
	player.SetVolume(volume)

	// Store references to prevent GC
	p.mu.Lock()
	p.player = player
	p.activeStream = stream
	p.startTime = time.Now()
	p.pausedAt = 0
	p.totalPause = 0
	p.position.Store(0)
	p.mu.Unlock()

	// Start playback
	player.Play()

	// Update state
	p.state.Store(int32(StatePlaying))

	return nil
}

// createAudioStream creates an AudioStream with data kept alive.
func (p *Player) createAudioStream(audio []byte) (*AudioStream, error) {
	// Make a copy to ensure we own the data
	data := make([]byte, len(audio))
	copy(data, audio)

	// Calculate duration based on audio format
	// Formula: samples = bytes / (channels * bytes_per_sample)
	// Duration = samples / sample_rate
	bytesPerSample := p.bitDepth / 8
	samples := len(data) / (p.channels * bytesPerSample)
	duration := time.Duration(samples) * time.Second / time.Duration(p.sampleRate)

	stream := &AudioStream{
		data:       data, // CRITICAL: Keep alive!
		reader:     bytes.NewReader(data),
		size:       len(data),
		duration:   duration,
		sampleRate: p.sampleRate,
		channels:   p.channels,
	}

	return stream, nil
}

// Pause pauses the current playback.
func (p *Player) Pause() error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	currentState := PlayerState(p.state.Load())
	if currentState != StatePlaying {
		return fmt.Errorf("cannot pause: player is %s", p.getStateName(currentState))
	}

	// Pause the OTO player
	p.mu.Lock()
	if p.player != nil {
		p.player.Pause()
	}

	// Record pause position and time
	p.pausedAt = p.getPositionInternal()
	p.mu.Unlock()

	// Update state
	p.state.Store(int32(StatePaused))

	return nil
}

// Resume resumes paused playback.
func (p *Player) Resume() error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	currentState := PlayerState(p.state.Load())
	if currentState != StatePaused {
		return fmt.Errorf("cannot resume: player is %s", p.getStateName(currentState))
	}

	// Resume the OTO player
	p.mu.Lock()
	if p.player != nil {
		p.player.Play()
	}

	// Update timing - add paused duration to total pause time
	now := time.Now()
	pauseDuration := now.Sub(p.startTime.Add(p.pausedAt))
	p.totalPause += pauseDuration
	p.mu.Unlock()

	// Update state
	p.state.Store(int32(StatePlaying))

	return nil
}

// Stop stops playback and releases resources.
func (p *Player) Stop() error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	return p.stopInternal()
}

// stopInternal stops playback without locking.
func (p *Player) stopInternal() error {
	currentState := PlayerState(p.state.Load())
	if currentState == StateStopped || currentState == StateClosed {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop and close OTO player
	if p.player != nil {
		p.player.Pause() // Pause first
		p.player.Close() // Then close
		p.player = nil
	}

	// Clean up audio stream (allows GC)
	if p.activeStream != nil {
		p.activeStream.Close()
		p.activeStream = nil
	}

	// Reset position and timing
	p.position.Store(0)
	p.pausedAt = 0
	p.totalPause = 0

	// Update state
	p.state.Store(int32(StateStopped))

	return nil
}

// IsPlaying returns whether audio is currently playing.
func (p *Player) IsPlaying() bool {
	return PlayerState(p.state.Load()) == StatePlaying
}

// GetPosition returns the current playback position.
func (p *Player) GetPosition() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.getPositionInternal()
}

// getPositionInternal calculates position without locking.
func (p *Player) getPositionInternal() time.Duration {
	currentState := PlayerState(p.state.Load())

	switch currentState {
	case StatePlaying:
		// Calculate position based on elapsed time
		elapsed := time.Since(p.startTime) - p.totalPause

		// Clamp to stream duration if we have one
		if p.activeStream != nil && elapsed > p.activeStream.duration {
			elapsed = p.activeStream.duration
			// Check if playback should be considered finished
			go p.checkPlaybackComplete()
		}
		return elapsed

	case StatePaused:
		return p.pausedAt

	case StateStopped, StateClosed:
		return 0

	default:
		return time.Duration(p.position.Load())
	}
}

// checkPlaybackComplete checks if playback has naturally completed.
func (p *Player) checkPlaybackComplete() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeStream == nil {
		return
	}

	// If we've reached the end of the stream, stop playback
	currentPos := p.getPositionInternal()
	if currentPos >= p.activeStream.duration {
		p.state.Store(int32(StateStopped))

		// Clean up player
		if p.player != nil {
			p.player.Close()
			p.player = nil
		}

		// Clean up stream
		if p.activeStream != nil {
			p.activeStream.Close()
			p.activeStream = nil
		}
	}
}

// SetVolume sets the playback volume (0.0 to 1.0).
func (p *Player) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0, got %f", volume)
	}

	// Store volume atomically
	p.volume.Store(uint64(volume * 1000000)) // Store as integer for atomic ops

	// Apply to current player if active
	p.mu.RLock()
	if p.player != nil {
		p.player.SetVolume(volume)
	}
	p.mu.RUnlock()

	return nil
}

// getVolume gets the current volume.
func (p *Player) getVolume() float64 {
	return float64(p.volume.Load()) / 1000000.0
}

// Close releases audio device and resources.
func (p *Player) Close() error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	// Stop playback
	if err := p.stopInternal(); err != nil {
		// Log error but continue cleanup
		p.setError(fmt.Errorf("error stopping playback during close: %w", err))
	}

	// Close OTO context
	p.mu.Lock()
	if p.context != nil {
		// Note: oto.Context doesn't have a Close method in v3
		// The context will be garbage collected when no longer referenced
		p.context = nil
	}
	p.mu.Unlock()

	// Update state
	p.state.Store(int32(StateClosed))

	return p.getError()
}

// GetState returns the current player state.
func (p *Player) GetState() PlayerState {
	return PlayerState(p.state.Load())
}

// GetVolume returns the current volume.
func (p *Player) GetVolume() float64 {
	return p.getVolume()
}

// GetAudioInfo returns information about the current audio stream.
func (p *Player) GetAudioInfo() (sampleRate, channels, bitDepth int, duration time.Duration) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sampleRate = p.sampleRate
	channels = p.channels
	bitDepth = p.bitDepth

	if p.activeStream != nil {
		duration = p.activeStream.duration
	}

	return
}

// setError stores an error atomically.
func (p *Player) setError(err error) {
	p.errorMu.Lock()
	defer p.errorMu.Unlock()
	p.lastError = err
}

// getError retrieves the last error.
func (p *Player) getError() error {
	p.errorMu.RLock()
	defer p.errorMu.RUnlock()
	return p.lastError
}

// getStateName returns a string representation of the player state.
func (p *Player) getStateName(state PlayerState) string {
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

// Close closes the audio stream and allows GC of data.
func (s *AudioStream) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.closed = true
		// Allow GC of audio data
		s.data = nil
		s.reader = nil
	})
}

// IsClosed returns whether the stream is closed.
func (s *AudioStream) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// GetDuration returns the stream duration.
func (s *AudioStream) GetDuration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.duration
}

// Player implements the AudioPlayer interface for cross-platform audio playback
