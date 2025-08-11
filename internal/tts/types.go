package tts

import (
	"time"
)

// EngineType represents the TTS engine selection
type EngineType string

const (
	// EnginePiper represents the Piper offline TTS engine
	EnginePiper EngineType = "piper"
	
	// EngineGoogle represents the Google Cloud TTS engine
	EngineGoogle EngineType = "gtts"
	
	// EngineNone represents no engine selected
	EngineNone EngineType = ""
)

// State represents the current TTS system state
type State int

const (
	// StateIdle indicates TTS is not active
	StateIdle State = iota
	
	// StateInitializing indicates TTS is starting up
	StateInitializing
	
	// StateReady indicates TTS is ready to process
	StateReady
	
	// StateProcessing indicates TTS is processing document
	StateProcessing
	
	// StatePlaying indicates audio is playing
	StatePlaying
	
	// StatePaused indicates playback is paused
	StatePaused
	
	// StateStopping indicates TTS is shutting down
	StateStopping
	
	// StateError indicates an error occurred
	StateError
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateProcessing:
		return "processing"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Sentence represents a text sentence for TTS processing
type Sentence struct {
	// ID uniquely identifies the sentence
	ID string
	
	// Text is the sentence content to synthesize
	Text string
	
	// Position is the sentence's position in the document
	Position int
	
	// StartOffset is the character offset in the original document
	StartOffset int
	
	// EndOffset is the ending character offset in the original document
	EndOffset int
	
	// Priority indicates processing priority
	Priority Priority
	
	// CacheKey is the computed cache key for this sentence
	CacheKey string
}

// Priority defines processing priority for sentences
type Priority int

const (
	// PriorityLow for background preprocessing
	PriorityLow Priority = iota
	
	// PriorityNormal for standard sequential processing
	PriorityNormal
	
	// PriorityHigh for user-initiated navigation
	PriorityHigh
	
	// PriorityImmediate for current playback
	PriorityImmediate
)

// Progress represents the current playback progress
type Progress struct {
	// CurrentSentence is the index of the current sentence
	CurrentSentence int
	
	// TotalSentences is the total number of sentences
	TotalSentences int
	
	// CurrentPosition is the playback position in the current sentence
	CurrentPosition time.Duration
	
	// TotalDuration is the total duration of the current sentence
	TotalDuration time.Duration
	
	// ProcessedCount is the number of sentences already synthesized
	ProcessedCount int
	
	// CachedCount is the number of sentences in cache
	CachedCount int
}

// PercentComplete returns the overall completion percentage
func (p Progress) PercentComplete() float64 {
	if p.TotalSentences == 0 {
		return 0
	}
	
	sentenceProgress := float64(p.CurrentSentence) / float64(p.TotalSentences)
	
	// Add current sentence progress if duration is known
	if p.TotalDuration > 0 {
		currentProgress := float64(p.CurrentPosition) / float64(p.TotalDuration)
		sentenceProgress += currentProgress / float64(p.TotalSentences)
	}
	
	return sentenceProgress * 100
}

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
	Engine EngineType
	
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

// Direction represents navigation direction
type Direction int

const (
	// DirectionForward navigates to next sentence
	DirectionForward Direction = iota
	
	// DirectionBackward navigates to previous sentence
	DirectionBackward
)

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