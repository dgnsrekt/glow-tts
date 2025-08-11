package tts

import (
	"time"

	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

// AudioData represents synthesized audio with metadata
type AudioData struct {
	// Data is the raw audio bytes (PCM format)
	Data []byte

	// SampleRate is the audio sample rate in Hz
	SampleRate int

	// Channels is the number of audio channels (1=mono, 2=stereo)
	Channels int

	// BitDepth is the bits per sample (typically 16)
	BitDepth int

	// Duration is the audio duration
	Duration time.Duration

	// SentenceID links this audio to a specific sentence
	SentenceID string
}

// Config represents TTS configuration
type Config struct {
	// Engine is the selected TTS engine
	Engine ttypes.EngineType

	// CacheDir is the directory for cached audio
	CacheDir string

	// MaxCacheSize is the maximum cache size in bytes
	MaxCacheSize int64

	// Lookahead is the number of sentences to preprocess
	Lookahead int

	// Speed is the playback speed multiplier (0.5 to 2.0)
	Speed float64

	// Piper contains Piper-specific configuration
	Piper PiperConfig

	// GTTS contains gTTS-specific configuration
	GTTS GTTSConfigSection
}

// PiperConfig contains Piper engine configuration
type PiperConfig struct {
	// ModelPath is the path to the Piper model file
	ModelPath string

	// ConfigPath is the path to the model config file
	ConfigPath string

	// Voice is the selected voice name
	Voice string

	// SpeakerID is the speaker ID for multi-speaker models
	SpeakerID int
}

// GTTSConfigSection contains gTTS configuration for the TTS config
type GTTSConfigSection struct {
	// Language is the language code (e.g., "en", "es", "fr")
	Language string

	// Slow enables slower speech pace
	Slow bool

	// TempDir is the directory for temporary files
	TempDir string

	// RequestsPerMinute is the rate limit for requests
	RequestsPerMinute int
}

// QueueStats provides queue performance metrics
type QueueStats struct {
	// TotalEnqueued is the total number of items enqueued
	TotalEnqueued int

	// TotalProcessed is the total number of items processed
	TotalProcessed int

	// CurrentSize is the current queue size
	CurrentSize int

	// MaxSize is the maximum queue size reached
	MaxSize int

	// AverageWaitTime is the average time items spend in queue
	AverageWaitTime time.Duration

	// ProcessingRate is the items per second processing rate
	ProcessingRate float64
}
