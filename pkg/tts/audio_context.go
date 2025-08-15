package tts

import (
	"io"
	"time"
)

// AudioContextInterface defines the interface for audio operations
// This allows for both real (oto-based) and mock implementations
type AudioContextInterface interface {
	// NewPlayer creates a new audio player
	NewPlayer(r io.Reader) (AudioPlayerInterface, error)
	
	// Close closes the audio context and releases resources
	Close() error
	
	// IsReady returns whether the context is ready for use
	IsReady() bool
	
	// SampleRate returns the sample rate of the audio context
	SampleRate() int
	
	// ChannelCount returns the number of channels
	ChannelCount() int
}

// AudioPlayerInterface defines the interface for audio players
type AudioPlayerInterface interface {
	// Play starts or resumes playback
	Play()
	
	// Pause pauses playback
	Pause()
	
	// IsPlaying returns whether audio is currently playing
	IsPlaying() bool
	
	// Reset resets the player to the beginning
	Reset() error
	
	// Close closes the player and releases resources
	Close() error
	
	// SetVolume sets the playback volume (0.0 to 1.0)
	SetVolume(volume float64)
	
	// Volume returns the current volume
	Volume() float64
	
	// Seek seeks to a specific position
	Seek(offset int64, whence int) (int64, error)
	
	// BufferedDuration returns the duration of buffered audio
	BufferedDuration() time.Duration
}

// AudioContextType represents the type of audio context to create
type AudioContextType int

const (
	// AudioContextProduction uses real audio hardware via oto
	AudioContextProduction AudioContextType = iota
	// AudioContextMock uses a mock implementation for testing
	AudioContextMock
	// AudioContextAuto automatically detects the appropriate type
	AudioContextAuto
)