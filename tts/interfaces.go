package tts

import (
	"time"
)

// Engine defines the interface for text-to-speech engines.
type Engine interface {
	// Initialize prepares the engine for use with the given configuration.
	Initialize(config EngineConfig) error

	// GenerateAudio converts text to audio data synchronously.
	GenerateAudio(text string) (*Audio, error)

	// GenerateAudioStream converts text to audio data asynchronously.
	// Returns a channel that receives audio chunks as they are generated.
	GenerateAudioStream(text string) (<-chan AudioChunk, error)

	// IsAvailable checks if the engine is ready for use.
	IsAvailable() bool

	// GetVoices returns the list of available voices.
	GetVoices() []Voice

	// SetVoice sets the active voice for audio generation.
	SetVoice(voice Voice) error

	// GetCapabilities returns the engine's capabilities.
	GetCapabilities() Capabilities

	// Shutdown cleanly stops the engine and releases resources.
	Shutdown() error
}

// AudioPlayer defines the interface for audio playback.
type AudioPlayer interface {
	// Play starts playing the given audio.
	Play(audio *Audio) error

	// Pause temporarily stops playback.
	Pause() error

	// Resume continues playback from paused position.
	Resume() error

	// Stop halts playback and resets position.
	Stop() error

	// GetPosition returns the current playback position.
	GetPosition() time.Duration

	// IsPlaying returns true if audio is currently playing.
	IsPlaying() bool
}

// SentenceParser defines the interface for extracting sentences from markdown.
type SentenceParser interface {
	// Parse extracts sentences from markdown content.
	Parse(markdown string) []Sentence

	// EstimateDuration estimates the speaking duration for text.
	EstimateDuration(text string) time.Duration
}

// Synchronizer defines the interface for audio-visual synchronization.
type Synchronizer interface {
	// Start begins synchronization tracking.
	Start(sentences []Sentence, player AudioPlayer)

	// Stop halts synchronization tracking.
	Stop()

	// GetCurrentSentence returns the index of the current sentence.
	GetCurrentSentence() int

	// OnSentenceChange registers a callback for sentence changes.
	OnSentenceChange(callback func(index int))
}

// EngineConfig holds configuration for TTS engines.
type EngineConfig struct {
	Voice      string  // Voice identifier
	Rate       float32 // Speech rate multiplier (1.0 = normal)
	Pitch      float32 // Pitch adjustment
	Volume     float32 // Volume level (0.0 to 1.0)
}

// Audio represents generated audio data.
type Audio struct {
	Data       []byte        // Raw audio data
	Format     AudioFormat   // Audio format (PCM16, Float32, etc.)
	SampleRate int           // Sample rate in Hz
	Channels   int           // Number of audio channels
	Duration   time.Duration // Duration of the audio
}

// AudioFormat represents the format of audio data.
type AudioFormat int

const (
	// FormatPCM16 represents 16-bit PCM audio.
	FormatPCM16 AudioFormat = iota
	// FormatFloat32 represents 32-bit float audio.
	FormatFloat32
	// FormatMP3 represents MP3 compressed audio.
	FormatMP3
)

// Sentence represents a parsed sentence with metadata.
type Sentence struct {
	Index    int           // Index in the sentence array
	Text     string        // Plain text content
	Markdown string        // Original markdown
	Start    int           // Start position in original content
	End      int           // End position in original content
	Duration time.Duration // Estimated speaking duration
}

// Voice represents a TTS voice configuration.
type Voice struct {
	ID       string // Voice identifier
	Name     string // Human-readable name
	Language string // Language code (e.g., "en-US")
	Gender   string // Voice gender
}

// Capabilities describes what an engine can do.
type Capabilities struct {
	SupportsStreaming bool     // Can stream audio generation
	SupportedFormats  []string // Audio formats the engine can produce
	MaxTextLength     int      // Maximum text length per request
	RequiresNetwork   bool     // Needs internet connection
}

// AudioChunk represents a piece of streaming audio.
type AudioChunk struct {
	Data     []byte // Audio data chunk
	Final    bool   // True if this is the last chunk
	Position int    // Position in the stream
}