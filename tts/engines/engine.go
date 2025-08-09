// Package engines provides text-to-speech engine implementations.
package engines

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
	Data     []byte
	Final    bool // True if this is the last chunk
	Position int  // Position in the stream
}