package ui

// Config contains TUI-specific configuration.
type Config struct {
	ShowAllFiles     bool
	ShowLineNumbers  bool
	Gopath           string `env:"GOPATH"`
	HomeDir          string `env:"HOME"`
	GlamourMaxWidth  uint
	GlamourStyle     string `env:"GLAMOUR_STYLE"`
	EnableMouse      bool
	PreserveNewLines bool

	// Working directory or file path
	Path string

	// TTS configuration
	TTSEngine  string // TTS engine to use (piper/gtts), empty means TTS disabled
	TTSEnabled bool   // Whether TTS is enabled via --tts flag

	// TTS cache configuration
	TTSCacheDir     string // Cache directory for TTS audio files
	TTSMaxCacheSize int    // Maximum cache size in MB

	// TTS engine-specific configuration
	TTSPiperModel string  // Piper model path
	TTSPiperSpeed float64 // Piper speech speed
	TTSGTTSLang   string  // gTTS language code
	TTSGTTSSlow   bool    // gTTS slow speech flag

	// For debugging the UI
	HighPerformancePager bool `env:"GLOW_HIGH_PERFORMANCE_PAGER" envDefault:"true"`
	GlamourEnabled       bool `env:"GLOW_ENABLE_GLAMOUR"         envDefault:"true"`
}
