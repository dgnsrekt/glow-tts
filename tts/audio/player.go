// Package audio provides real audio playback functionality for TTS.
package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/ebitengine/oto/v3"
)

// Player implements the AudioPlayer interface with real audio output.
type Player struct {
	// Audio context and player
	otoContext *oto.Context
	otoPlayer  *oto.Player
	
	// State management
	mu       sync.RWMutex
	playing  int32 // atomic: 1 if playing, 0 if not
	paused   int32 // atomic: 1 if paused, 0 if not
	stopping int32 // atomic: 1 if stopping, 0 if not
	
	// Current audio
	currentAudio *tts.Audio
	audioBuffer  *bytes.Reader
	totalSamples int64
	
	// Position tracking
	position      time.Duration
	startTime     time.Time
	pauseTime     time.Time
	pausedDuration time.Duration
	sampleRate    int
	channels      int
	bytesPerSample int
	
	// Playback control
	ctx           context.Context
	cancel        context.CancelFunc
	playbackDone  chan struct{}
	positionTicker *time.Ticker
	
	// Buffer management
	bufferSize    int
	readBuffer    []byte
	audioDataBuffer *Buffer // Use existing Buffer for sentence buffering
	
	// Error tracking
	lastError     error
	errorLock     sync.RWMutex
}

// PlayerConfig holds configuration for the audio player.
type PlayerConfig struct {
	// BufferSize is the size of the audio buffer in bytes
	BufferSize int
	
	// LatencyHint provides a hint for audio latency
	LatencyHint time.Duration
	
	// BufferCapacity is the number of sentences to buffer
	BufferCapacity int
}

// DefaultPlayerConfig returns the default player configuration.
func DefaultPlayerConfig() PlayerConfig {
	return PlayerConfig{
		BufferSize:     4096,
		LatencyHint:    100 * time.Millisecond, // 100ms latency for smooth playback
		BufferCapacity: 3, // Buffer 3 sentences ahead
	}
}

// NewPlayer creates a new audio player with real audio output.
func NewPlayer() (*Player, error) {
	return NewPlayerWithConfig(DefaultPlayerConfig())
}

// NewPlayerWithConfig creates a new audio player with the given configuration.
func NewPlayerWithConfig(config PlayerConfig) (*Player, error) {
	// Set defaults
	if config.BufferSize <= 0 {
		config.BufferSize = 4096
	}
	if config.LatencyHint <= 0 {
		config.LatencyHint = 100 * time.Millisecond
	}
	if config.BufferCapacity <= 0 {
		config.BufferCapacity = 3
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create audio data buffer
	bufferConfig := DefaultBufferConfig()
	bufferConfig.Capacity = config.BufferCapacity
	audioBuffer := NewBuffer(bufferConfig)
	
	player := &Player{
		bufferSize:      config.BufferSize,
		readBuffer:      make([]byte, config.BufferSize),
		audioDataBuffer: audioBuffer,
		ctx:             ctx,
		cancel:          cancel,
		playbackDone:    make(chan struct{}),
	}
	
	return player, nil
}

// initializeOtoContext initializes the oto audio context with the given format.
func (p *Player) initializeOtoContext(sampleRate, channels int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Don't reinitialize if format hasn't changed
	if p.otoContext != nil && p.sampleRate == sampleRate && p.channels == channels {
		return nil
	}
	
	// Close existing context if any
	if p.otoContext != nil {
		p.otoContext.Suspend()
		p.otoContext = nil
	}
	
	// Create new oto context
	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channels,
		Format:       oto.FormatSignedInt16LE,
	}
	
	context, ready, err := oto.NewContext(op)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}
	
	// Wait for context to be ready
	<-ready
	
	p.otoContext = context
	p.sampleRate = sampleRate
	p.channels = channels
	p.bytesPerSample = 2 // 16-bit samples
	
	return nil
}

// Play starts playing the given audio.
func (p *Player) Play(audio *tts.Audio) error {
	if audio == nil {
		return errors.New("audio is nil")
	}
	
	// If already playing, just replace the buffer instead of stopping
	// This prevents oto context recreation issues
	if atomic.LoadInt32(&p.playing) == 1 {
		// Pause current playback
		if p.otoPlayer != nil {
			p.otoPlayer.Pause()
		}
		// We'll replace the buffer below
	}
	
	// Initialize audio context if needed or if format changed
	needInit := p.otoContext == nil || 
		p.sampleRate != audio.SampleRate || 
		p.channels != audio.Channels
	
	if needInit {
		if err := p.initializeOtoContext(audio.SampleRate, audio.Channels); err != nil {
			return err
		}
	}
	
	p.mu.Lock()
	
	// Convert audio format if needed
	audioData := audio.Data
	if audio.Format == tts.FormatFloat32 {
		// Convert Float32 to PCM16
		audioData = convertFloat32ToPCM16(audio.Data)
	} else if audio.Format != tts.FormatPCM16 {
		p.mu.Unlock()
		return fmt.Errorf("unsupported audio format: %v", audio.Format)
	}
	
	// Set up audio buffer
	p.currentAudio = audio
	p.audioBuffer = bytes.NewReader(audioData)
	p.totalSamples = int64(len(audioData) / (p.bytesPerSample * p.channels))
	
	// Reset position tracking
	p.position = 0
	p.pausedDuration = 0
	p.startTime = time.Now()
	
	// Close old player if exists before creating new one
	if p.otoPlayer != nil {
		p.otoPlayer.Close()
		p.otoPlayer = nil
	}
	
	// Create new oto player
	p.otoPlayer = p.otoContext.NewPlayer(p.audioBuffer)
	
	// Set state
	atomic.StoreInt32(&p.playing, 1)
	atomic.StoreInt32(&p.paused, 0)
	atomic.StoreInt32(&p.stopping, 0)
	
	// Create new done channel only if nil
	if p.playbackDone == nil {
		p.playbackDone = make(chan struct{})
	}
	
	p.mu.Unlock()
	
	// Start playback goroutine only if not already running
	if atomic.LoadInt32(&p.playing) == 1 {
		go p.playbackLoop()
	}
	
	// Start position tracking
	go p.trackPosition()
	
	// Start the actual playback
	p.otoPlayer.Play()
	
	return nil
}

// playbackLoop manages the audio playback.
func (p *Player) playbackLoop() {
	defer func() {
		// Safely close channel with recovery
		defer func() {
			if r := recover(); r != nil {
				// Channel was already closed, ignore
			}
		}()
		
		select {
		case <-p.playbackDone:
			// Already closed
		default:
			close(p.playbackDone)
		}
		
		// Auto-stop when playback completes
		if atomic.LoadInt32(&p.stopping) == 0 {
			atomic.StoreInt32(&p.playing, 0)
			atomic.StoreInt32(&p.paused, 0)
		}
	}()
	
	// Monitor for completion or stop signal
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			// Check if playback is complete
			if atomic.LoadInt32(&p.playing) == 0 || atomic.LoadInt32(&p.stopping) == 1 {
				return
			}
			
			// Check if we've reached the end
			p.mu.RLock()
			if p.audioBuffer != nil && p.audioBuffer.Len() == 0 {
				p.mu.RUnlock()
				return
			}
			p.mu.RUnlock()
		}
	}
}

// trackPosition tracks the current playback position.
func (p *Player) trackPosition() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-p.playbackDone:
			return
		case <-ticker.C:
			p.updatePosition()
		}
	}
}

// updatePosition updates the current playback position.
func (p *Player) updatePosition() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if atomic.LoadInt32(&p.playing) == 0 {
		return
	}
	
	if atomic.LoadInt32(&p.paused) == 1 {
		// Don't update position while paused
		return
	}
	
	// Calculate position based on bytes read
	if p.audioBuffer != nil && p.totalSamples > 0 {
		bytesRead := int64(p.audioBuffer.Size()) - int64(p.audioBuffer.Len())
		samplesRead := bytesRead / int64(p.bytesPerSample*p.channels)
		
		// Calculate duration based on samples
		if p.sampleRate > 0 {
			seconds := float64(samplesRead) / float64(p.sampleRate)
			p.position = time.Duration(seconds * float64(time.Second))
		}
	}
}

// Pause temporarily stops playback.
func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if atomic.LoadInt32(&p.playing) == 0 {
		return errors.New("not playing")
	}
	
	if atomic.LoadInt32(&p.paused) == 1 {
		return errors.New("already paused")
	}
	
	// Pause oto player
	if p.otoPlayer != nil {
		p.otoPlayer.Pause()
	}
	
	atomic.StoreInt32(&p.paused, 1)
	p.pauseTime = time.Now()
	
	return nil
}

// Resume continues playback from paused position.
func (p *Player) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if atomic.LoadInt32(&p.playing) == 0 {
		return errors.New("not playing")
	}
	
	if atomic.LoadInt32(&p.paused) == 0 {
		return errors.New("not paused")
	}
	
	// Calculate paused duration
	if !p.pauseTime.IsZero() {
		p.pausedDuration += time.Since(p.pauseTime)
		p.pauseTime = time.Time{}
	}
	
	// Resume oto player
	if p.otoPlayer != nil {
		p.otoPlayer.Play()
	}
	
	atomic.StoreInt32(&p.paused, 0)
	
	return nil
}

// Stop halts playback and resets position.
func (p *Player) Stop() error {
	// Mark as stopping
	atomic.StoreInt32(&p.stopping, 1)
	defer atomic.StoreInt32(&p.stopping, 0)
	
	p.mu.Lock()
	
	if atomic.LoadInt32(&p.playing) == 0 {
		p.mu.Unlock()
		return nil
	}
	
	// Close oto player
	if p.otoPlayer != nil {
		p.otoPlayer.Pause()
		err := p.otoPlayer.Close()
		if err != nil {
			p.recordError(fmt.Errorf("failed to close player: %w", err))
		}
		p.otoPlayer = nil
	}
	
	// Reset state
	atomic.StoreInt32(&p.playing, 0)
	atomic.StoreInt32(&p.paused, 0)
	p.position = 0
	p.pausedDuration = 0
	p.currentAudio = nil
	p.audioBuffer = nil
	
	p.mu.Unlock()
	
	// Wait for playback loop to finish
	select {
	case <-p.playbackDone:
		// Playback stopped
	case <-time.After(1 * time.Second):
		// Timeout
		return errors.New("timeout waiting for playback to stop")
	}
	
	return nil
}

// GetPosition returns the current playback position.
func (p *Player) GetPosition() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.position
}

// IsPlaying returns true if audio is currently playing.
func (p *Player) IsPlaying() bool {
	return atomic.LoadInt32(&p.playing) == 1 && atomic.LoadInt32(&p.paused) == 0
}

// IsPaused returns true if playback is paused.
func (p *Player) IsPaused() bool {
	return atomic.LoadInt32(&p.playing) == 1 && atomic.LoadInt32(&p.paused) == 1
}

// GetDuration returns the duration of the current audio.
func (p *Player) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.currentAudio != nil {
		return p.currentAudio.Duration
	}
	return 0
}

// SetVolume sets the playback volume (0.0 to 1.0).
func (p *Player) SetVolume(volume float32) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0, got %f", volume)
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Note: oto v3 doesn't directly support volume control
	// This would need to be implemented by modifying the audio data
	// For now, we just validate the input
	
	return nil
}

// Close releases all resources.
func (p *Player) Close() error {
	// Stop playback
	if err := p.Stop(); err != nil {
		return err
	}
	
	// Cancel context
	p.cancel()
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Suspend oto context
	if p.otoContext != nil {
		p.otoContext.Suspend()
		p.otoContext = nil
	}
	
	return nil
}

// recordError records an error for tracking.
func (p *Player) recordError(err error) {
	p.errorLock.Lock()
	defer p.errorLock.Unlock()
	p.lastError = err
}

// GetLastError returns the last recorded error.
func (p *Player) GetLastError() error {
	p.errorLock.RLock()
	defer p.errorLock.RUnlock()
	return p.lastError
}

// GetStats returns player statistics.
func (p *Player) GetStats() PlayerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	stats := PlayerStats{
		Playing:        atomic.LoadInt32(&p.playing) == 1,
		Paused:         atomic.LoadInt32(&p.paused) == 1,
		Position:       p.position,
		Duration:       0,
		SampleRate:     p.sampleRate,
		Channels:       p.channels,
		BufferSize:     p.bufferSize,
	}
	
	if p.currentAudio != nil {
		stats.Duration = p.currentAudio.Duration
		stats.Format = p.currentAudio.Format
	}
	
	return stats
}

// PlayerStats holds player statistics.
type PlayerStats struct {
	Playing    bool
	Paused     bool
	Position   time.Duration
	Duration   time.Duration
	SampleRate int
	Channels   int
	Format     tts.AudioFormat
	BufferSize int
}

// convertFloat32ToPCM16 converts float32 audio to PCM16.
func convertFloat32ToPCM16(data []byte) []byte {
	if len(data)%4 != 0 {
		return nil
	}
	
	floatSamples := len(data) / 4
	pcm16Data := make([]byte, floatSamples*2)
	
	for i := 0; i < floatSamples; i++ {
		// Read float32 sample
		bits := binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
		floatVal := *(*float32)(unsafe.Pointer(&bits))
		
		// Convert to int16 (-32768 to 32767)
		var intVal int16
		if floatVal >= 1.0 {
			intVal = 32767
		} else if floatVal <= -1.0 {
			intVal = -32768
		} else {
			intVal = int16(floatVal * 32767)
		}
		
		// Write PCM16 sample
		binary.LittleEndian.PutUint16(pcm16Data[i*2:(i+1)*2], uint16(intVal))
	}
	
	return pcm16Data
}