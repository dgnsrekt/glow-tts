package tts

import (
	"context"
	"time"
)

// TTSEngine defines the contract for text-to-speech engines.
// Implementations include Piper (offline) and Google TTS (online).
// Each engine must handle its own caching and resource management.
type TTSEngine interface {
	// Synthesize converts text to audio data.
	// Returns audio in PCM format (16-bit, mono, sample rate per config).
	// The implementation must handle timeout protection internally.
	Synthesize(ctx context.Context, text string, speed float64) ([]byte, error)

	// GetInfo returns engine capabilities and configuration.
	GetInfo() EngineInfo

	// Validate checks if the engine is properly configured and available.
	// This should verify model files (Piper) or API keys (Google).
	Validate() error

	// Close releases any resources held by the engine.
	Close() error
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

// AudioPlayer defines the contract for audio playback.
// Implementations must handle memory management to prevent GC issues.
type AudioPlayer interface {
	// Play starts playback of audio data.
	// The implementation MUST keep the audio data alive during playback.
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

// AudioCache defines the contract for caching synthesized audio.
// Implementations should provide two-level caching (memory + disk).
type AudioCache interface {
	// Get retrieves cached audio for the given key.
	// Returns nil, false if not found.
	Get(key string) ([]byte, bool)

	// Put stores audio data with the given key.
	// The implementation should handle eviction when size limits are reached.
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
	Hits       int64     // Number of cache hits
	Misses     int64     // Number of cache misses
	Evictions  int64     // Number of evictions
	Size       int64     // Current cache size in bytes
	Capacity   int64     // Maximum cache capacity in bytes
	LastCleanup time.Time // Last cleanup time
}

// SentenceQueue defines the contract for managing sentence processing.
// Implementations should support lookahead buffering and priority handling.
type SentenceQueue interface {
	// Enqueue adds a sentence to the queue.
	// Priority sentences (from navigation) should be processed first.
	Enqueue(sentence Sentence, priority bool) error

	// Dequeue removes and returns the next sentence to process.
	// Returns ErrQueueEmpty if the queue is empty.
	Dequeue() (Sentence, error)

	// Peek returns the next sentence without removing it.
	// Returns ErrQueueEmpty if the queue is empty.
	Peek() (Sentence, error)

	// Size returns the current number of sentences in the queue.
	Size() int

	// Clear removes all sentences from the queue.
	Clear()

	// SetLookahead configures the number of sentences to preprocess.
	SetLookahead(count int)
}

// Parser defines the contract for extracting sentences from markdown.
type Parser interface {
	// Parse extracts speakable sentences from markdown content.
	// It should strip markdown formatting and skip code blocks.
	Parse(markdown string) ([]Sentence, error)

	// StripMarkdown removes markdown formatting from text.
	StripMarkdown(text string) string
}

// Controller orchestrates all TTS components.
// This is the main interface for the TTS subsystem.
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