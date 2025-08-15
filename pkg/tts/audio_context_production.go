//go:build !nocgo
// +build !nocgo

package tts

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/ebitengine/oto/v3"
)

// ProductionAudioContext implements AudioContextInterface using real oto audio
type ProductionAudioContext struct {
	context *oto.Context
	mu      sync.Mutex
	ready   bool
}

// NewProductionAudioContext creates a new production audio context
func NewProductionAudioContext() (*ProductionAudioContext, error) {
	pac := &ProductionAudioContext{}
	if err := pac.initialize(); err != nil {
		return nil, err
	}
	return pac, nil
}

// NewProductionAudioContextWithRetry creates a production audio context with platform-specific retry logic
func NewProductionAudioContextWithRetry(platform *PlatformInfo) (*ProductionAudioContext, error) {
	// Configure retry based on platform
	maxRetries := 1
	retryDelay := time.Millisecond * 100
	
	switch platform.OS {
	case PlatformDarwin:
		// macOS CoreAudio can have race conditions during initialization
		maxRetries = 3
		retryDelay = time.Millisecond * 200
		log.Debug("Using macOS retry strategy", "retries", maxRetries, "delay", retryDelay)
	case PlatformWindows:
		// Windows WASAPI might need retry if exclusive mode conflicts
		maxRetries = 2
		retryDelay = time.Millisecond * 150
		log.Debug("Using Windows retry strategy", "retries", maxRetries, "delay", retryDelay)
	case PlatformLinux:
		// Linux ALSA/PulseAudio generally works first try
		if platform.AudioSubsystem == AudioSubsystemPulseAudio {
			// PulseAudio might need retry if daemon is starting
			maxRetries = 2
			retryDelay = time.Millisecond * 100
			log.Debug("Using Linux PulseAudio retry strategy", "retries", maxRetries, "delay", retryDelay)
		} else {
			log.Debug("Using Linux ALSA strategy (no retry)")
		}
	default:
		log.Debug("Using default retry strategy", "retries", maxRetries, "delay", retryDelay)
	}
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			log.Debug("Retrying audio context initialization", "attempt", i+1, "of", maxRetries)
			time.Sleep(retryDelay)
		}
		
		pac := &ProductionAudioContext{}
		if err := pac.initializeWithPlatform(platform); err != nil {
			lastErr = err
			log.Debug("Audio context initialization failed", "attempt", i+1, "error", err)
			
			// On macOS, specifically handle CoreAudio errors
			if platform.OS == PlatformDarwin && i < maxRetries-1 {
				// Give CoreAudio more time to stabilize
				time.Sleep(time.Millisecond * 100)
			}
			continue
		}
		
		// Success
		log.Info("Production audio context initialized successfully", "attempt", i+1)
		return pac, nil
	}
	
	return nil, fmt.Errorf("failed to initialize audio context after %d attempts: %w", maxRetries, lastErr)
}

// initializeWithPlatform creates the OTO audio context with platform-specific settings
func (pac *ProductionAudioContext) initializeWithPlatform(platform *PlatformInfo) error {
	pac.mu.Lock()
	defer pac.mu.Unlock()

	if pac.ready {
		return nil
	}

	// Create OTO context with our TTS audio format
	options := &oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: Channels,
		Format:       oto.FormatSignedInt16LE,
	}

	// Use platform-specific buffer sizes
	options.BufferSize = time.Millisecond * time.Duration(platform.GetPlatformBufferSize())
	
	log.Debug("Initializing production audio context with platform settings",
		"platform", platform.OS,
		"audio_subsystem", platform.AudioSubsystem,
		"sample_rate", options.SampleRate,
		"channels", options.ChannelCount,
		"buffer_size", options.BufferSize)

	// Create context with ready channel
	context, readyChan, err := oto.NewContext(options)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}

	// Platform-specific ready timeout
	readyTimeout := 5 * time.Second
	if platform.OS == PlatformDarwin {
		// Give macOS more time for CoreAudio initialization
		readyTimeout = 10 * time.Second
	}

	// Wait for context to be ready
	select {
	case <-readyChan:
		pac.context = context
		pac.ready = true
		log.Debug("Production audio context initialized successfully",
			"platform", platform.OS,
			"subsystem", platform.AudioSubsystem)
	case <-time.After(readyTimeout):
		// Context doesn't have Close in oto v3, it will be garbage collected
		return fmt.Errorf("audio context initialization timeout after %v", readyTimeout)
	}

	return nil
}

// initialize creates the OTO audio context
func (pac *ProductionAudioContext) initialize() error {
	pac.mu.Lock()
	defer pac.mu.Unlock()

	if pac.ready {
		return nil
	}

	// Create OTO context with our TTS audio format
	options := &oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: Channels,
		Format:       oto.FormatSignedInt16LE,
	}

	// Platform-specific buffer size adjustments
	switch runtime.GOOS {
	case "darwin":
		// macOS benefits from larger buffers
		options.BufferSize = time.Millisecond * 100
	case "windows":
		// Windows WASAPI works well with moderate buffers
		options.BufferSize = time.Millisecond * 80
	default:
		// Linux ALSA and others
		options.BufferSize = time.Millisecond * 50
	}

	log.Debug("Initializing production audio context",
		"sample_rate", options.SampleRate,
		"channels", options.ChannelCount,
		"buffer_size", options.BufferSize)

	// Create context with ready channel
	context, readyChan, err := oto.NewContext(options)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}

	// Wait for context to be ready
	select {
	case <-readyChan:
		pac.context = context
		pac.ready = true
		log.Debug("Production audio context initialized successfully")
	case <-time.After(5 * time.Second):
		// Context doesn't have Close in oto v3, it will be garbage collected
		return fmt.Errorf("audio context initialization timeout")
	}

	return nil
}

// NewPlayer creates a new audio player
func (pac *ProductionAudioContext) NewPlayer(r io.Reader) (AudioPlayerInterface, error) {
	pac.mu.Lock()
	defer pac.mu.Unlock()

	if !pac.ready || pac.context == nil {
		return nil, fmt.Errorf("audio context not ready")
	}

	// Create oto player
	player := pac.context.NewPlayer(r)
	
	// Wrap in our interface implementation
	return &ProductionAudioPlayer{
		player: player,
		reader: r,
	}, nil
}

// Close closes the audio context
func (pac *ProductionAudioContext) Close() error {
	pac.mu.Lock()
	defer pac.mu.Unlock()

	// In oto v3, context doesn't have Close method
	// It will be cleaned up when garbage collected
	pac.ready = false
	pac.context = nil
	return nil
}

// IsReady returns whether the context is ready
func (pac *ProductionAudioContext) IsReady() bool {
	pac.mu.Lock()
	defer pac.mu.Unlock()
	return pac.ready
}

// SampleRate returns the sample rate
func (pac *ProductionAudioContext) SampleRate() int {
	return SampleRate
}

// ChannelCount returns the number of channels
func (pac *ProductionAudioContext) ChannelCount() int {
	return Channels
}

// ProductionAudioPlayer wraps an oto.Player to implement AudioPlayerInterface
type ProductionAudioPlayer struct {
	player *oto.Player
	reader io.Reader
	mu     sync.Mutex
	volume float64
}

// Play starts or resumes playback
func (pap *ProductionAudioPlayer) Play() {
	pap.player.Play()
}

// Pause pauses playback
func (pap *ProductionAudioPlayer) Pause() {
	pap.player.Pause()
}

// IsPlaying returns whether audio is currently playing
func (pap *ProductionAudioPlayer) IsPlaying() bool {
	return pap.player.IsPlaying()
}

// Reset resets the player to the beginning
func (pap *ProductionAudioPlayer) Reset() error {
	if seeker, ok := pap.reader.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		return err
	}
	return fmt.Errorf("reader does not support seeking")
}

// Close closes the player
func (pap *ProductionAudioPlayer) Close() error {
	return pap.player.Close()
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (pap *ProductionAudioPlayer) SetVolume(volume float64) {
	pap.mu.Lock()
	defer pap.mu.Unlock()
	pap.volume = volume
	pap.player.SetVolume(volume)
}

// Volume returns the current volume
func (pap *ProductionAudioPlayer) Volume() float64 {
	pap.mu.Lock()
	defer pap.mu.Unlock()
	if pap.volume == 0 {
		return 1.0 // Default volume
	}
	return pap.volume
}

// Seek seeks to a specific position
func (pap *ProductionAudioPlayer) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := pap.reader.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fmt.Errorf("reader does not support seeking")
}

// BufferedDuration returns the duration of buffered audio
func (pap *ProductionAudioPlayer) BufferedDuration() time.Duration {
	// oto v3 doesn't provide BufferedDuration, return a default
	return 100 * time.Millisecond
}