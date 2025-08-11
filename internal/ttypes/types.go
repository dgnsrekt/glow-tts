// Package ttypes contains shared types and interfaces for the TTS system.
// This package is used to break import cycles between tts, engines, audio, and queue packages.
package ttypes

import (
	"context"
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

// EngineInfo describes engine capabilities and configuration.
type EngineInfo struct {
	Name        string // Engine name (e.g., "piper", "google")
	Version     string // Engine version
	SampleRate  int    // Audio sample rate in Hz
	Channels    int    // Number of audio channels (1=mono, 2=stereo)
	BitDepth    int    // Bits per sample (typically 16)
	MaxTextSize int    // Maximum text size in characters
	IsOnline    bool   // Whether the engine requires internet
}

// TTSEngine defines the contract for text-to-speech engines.
type TTSEngine interface {
	// Synthesize converts text to audio data.
	// Returns audio in PCM format (16-bit, mono, sample rate per config).
	Synthesize(ctx context.Context, text string, speed float64) ([]byte, error)

	// GetInfo returns engine capabilities and configuration.
	GetInfo() EngineInfo

	// Validate checks if the engine is properly configured and available.
	Validate() error

	// Close releases any resources held by the engine.
	Close() error
}

// AudioPlayer defines the contract for audio playback.
type AudioPlayer interface {
	// Play starts playback of audio data.
	Play(audio []byte) error

	// Pause pauses the current playback.
	Pause() error

	// Resume resumes paused playback.
	Resume() error

	// Stop stops playback and releases resources.
	Stop() error

	// IsPlaying returns whether audio is currently playing.
	IsPlaying() bool

	// GetPosition returns the current playback position.
	GetPosition() time.Duration

	// SetVolume sets the playback volume (0.0 to 1.0).
	SetVolume(volume float64) error

	// Close releases audio device and resources.
	Close() error
}

// Direction represents navigation direction
type Direction int

const (
	// DirectionForward navigates to next sentence
	DirectionForward Direction = iota

	// DirectionBackward navigates to previous sentence
	DirectionBackward
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

// SentenceQueue defines the contract for managing sentence processing.
type SentenceQueue interface {
	// Enqueue adds a sentence to the queue.
	Enqueue(sentence Sentence, priority bool) error

	// Dequeue removes and returns the next sentence to process.
	Dequeue() (Sentence, error)

	// Peek returns the next sentence without removing it.
	Peek() (Sentence, error)

	// Size returns the current number of sentences in the queue.
	Size() int

	// Clear removes all sentences from the queue.
	Clear()

	// SetLookahead configures the number of sentences to preprocess.
	SetLookahead(count int)
}

// AudioCache defines the contract for caching synthesized audio.
type AudioCache interface {
	// Get retrieves cached audio for the given key.
	Get(key string) ([]byte, bool)

	// Put stores audio data with the given key.
	Put(key string, audio []byte) error

	// Delete removes the cached entry for the given key.
	Delete(key string) error

	// Clear removes all cached entries.
	Clear() error

	// Size returns the current cache size in bytes.
	Size() int64

	// Stats returns cache statistics.
	Stats() CacheStats
}

// CacheStats provides cache performance metrics.
type CacheStats struct {
	Hits        int64     // Number of cache hits
	Misses      int64     // Number of cache misses
	Evictions   int64     // Number of evictions
	Size        int64     // Current cache size in bytes
	Capacity    int64     // Maximum cache capacity in bytes
	LastCleanup time.Time // Last cleanup time
}

// Parser defines the contract for extracting sentences from markdown.
type Parser interface {
	// Parse extracts speakable sentences from markdown content.
	Parse(markdown string) ([]Sentence, error)

	// StripMarkdown removes markdown formatting from text.
	StripMarkdown(text string) string
}

// SpeedController manages playback speed adjustments.
type SpeedController interface {
	// GetSpeed returns the current speed multiplier.
	GetSpeed() float64

	// SetSpeed sets the speed multiplier (0.5 to 2.0).
	SetSpeed(speed float64) error

	// Increase increments to the next speed step.
	Increase() float64

	// Decrease decrements to the previous speed step.
	Decrease() float64

	// ToPiperScale converts speed to Piper's length-scale parameter.
	ToPiperScale() string

	// ToGoogleRate converts speed to Google's speaking_rate parameter.
	ToGoogleRate() float64
}

// Controller orchestrates all TTS components.
type Controller interface {
	// Start initializes the TTS system with the specified engine.
	Start(ctx context.Context, engineType EngineType) error

	// Stop halts TTS and releases resources.
	Stop() error

	// ProcessDocument prepares a document for TTS playback.
	ProcessDocument(content string) error

	// Play starts or resumes playback.
	Play() error

	// Pause pauses playback.
	Pause() error

	// NextSentence navigates to the next sentence.
	NextSentence() error

	// PreviousSentence navigates to the previous sentence.
	PreviousSentence() error

	// SetSpeed adjusts the playback speed (0.5 to 2.0).
	SetSpeed(speed float64) error

	// GetState returns the current TTS state.
	GetState() State

	// GetProgress returns the current playback progress.
	GetProgress() Progress
}
