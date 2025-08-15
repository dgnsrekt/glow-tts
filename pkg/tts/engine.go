package tts

// TTSEngine defines the interface that all TTS engine implementations must satisfy.
// Engines are responsible for converting text to speech audio data.
type TTSEngine interface {
	// Synthesize converts text to PCM audio data.
	// The output must be in 16-bit mono 22050Hz PCM format.
	// Speed parameter should be between 0.5 and 2.0 (1.0 = normal speed).
	Synthesize(text string, speed float64) ([]byte, error)

	// SetSpeed validates and prepares the engine for the given speed setting.
	// Speed should be between 0.5 and 2.0 (1.0 = normal speed).
	// Returns an error if the speed is out of range or not supported.
	SetSpeed(speed float64) error

	// Validate checks if all required dependencies for the engine are available.
	// This includes checking for required binaries, models, or network connectivity.
	// Returns an error with detailed information about missing dependencies.
	Validate() error

	// GetName returns the human-readable name of the TTS engine.
	// For example: "Piper" or "Google TTS".
	GetName() string

	// IsAvailable performs a runtime check to determine if the engine can be used.
	// This is a lightweight check compared to Validate().
	IsAvailable() bool
}

// EngineConfig holds common configuration for TTS engines.
type EngineConfig struct {
	// Voice specifies the voice/model to use for synthesis
	Voice string

	// Language specifies the language code (e.g., "en-US")
	Language string

	// CachePath specifies where to store cached audio
	CachePath string

	// Timeout specifies the maximum time for synthesis operations
	Timeout int // in seconds
}