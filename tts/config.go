package tts

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config contains all TTS configuration options.
type Config struct {
	// Global TTS settings
	Enabled bool   `yaml:"enabled" env:"GLOW_TTS_ENABLED" envDefault:"false"`
	Engine  string `yaml:"engine" env:"GLOW_TTS_ENGINE" envDefault:"mock"`
	
	// Audio settings
	SampleRate int     `yaml:"sample_rate" env:"GLOW_TTS_SAMPLE_RATE" envDefault:"22050"`
	Volume     float64 `yaml:"volume" env:"GLOW_TTS_VOLUME" envDefault:"1.0"`
	
	// Playback settings
	AutoPlay           bool `yaml:"auto_play" env:"GLOW_TTS_AUTO_PLAY" envDefault:"false"`
	PauseOnFocusLoss   bool `yaml:"pause_on_focus_loss" env:"GLOW_TTS_PAUSE_ON_FOCUS_LOSS" envDefault:"true"`
	BufferSize         int  `yaml:"buffer_size" env:"GLOW_TTS_BUFFER_SIZE" envDefault:"3"`
	BufferAheadEnabled bool `yaml:"buffer_ahead" env:"GLOW_TTS_BUFFER_AHEAD" envDefault:"true"`
	
	// Navigation settings
	WrapNavigation bool `yaml:"wrap_navigation" env:"GLOW_TTS_WRAP_NAVIGATION" envDefault:"true"`
	SkipCodeBlocks bool `yaml:"skip_code_blocks" env:"GLOW_TTS_SKIP_CODE_BLOCKS" envDefault:"false"`
	SkipURLs       bool `yaml:"skip_urls" env:"GLOW_TTS_SKIP_URLS" envDefault:"false"`
	
	// Visual settings
	HighlightEnabled bool   `yaml:"highlight_enabled" env:"GLOW_TTS_HIGHLIGHT_ENABLED" envDefault:"true"`
	HighlightColor   string `yaml:"highlight_color" env:"GLOW_TTS_HIGHLIGHT_COLOR" envDefault:"yellow"`
	ShowProgress     bool   `yaml:"show_progress" env:"GLOW_TTS_SHOW_PROGRESS" envDefault:"true"`
	
	// Engine-specific configurations
	Piper  PiperConfig  `yaml:"piper"`
	Google GoogleConfig `yaml:"google"`
	Mock   MockConfig   `yaml:"mock"`
}

// PiperConfig contains Piper TTS engine specific settings.
type PiperConfig struct {
	Binary     string            `yaml:"binary" env:"GLOW_TTS_PIPER_BINARY" envDefault:"piper"`
	Model      string            `yaml:"model" env:"GLOW_TTS_PIPER_MODEL" envDefault:"en_US-lessac-medium"`
	ModelPath  string            `yaml:"model_path" env:"GLOW_TTS_PIPER_MODEL_PATH"`
	ConfigPath string            `yaml:"config_path" env:"GLOW_TTS_PIPER_CONFIG_PATH"`
	DataDir    string            `yaml:"data_dir" env:"GLOW_TTS_PIPER_DATA_DIR"`
	OutputRaw  bool              `yaml:"output_raw" env:"GLOW_TTS_PIPER_OUTPUT_RAW" envDefault:"true"`
	SpeakerId  int               `yaml:"speaker_id" env:"GLOW_TTS_PIPER_SPEAKER_ID" envDefault:"0"`
	LengthScale float64          `yaml:"length_scale" env:"GLOW_TTS_PIPER_LENGTH_SCALE" envDefault:"1.0"`
	NoiseScale float64           `yaml:"noise_scale" env:"GLOW_TTS_PIPER_NOISE_SCALE" envDefault:"0.667"`
	NoiseW     float64           `yaml:"noise_w" env:"GLOW_TTS_PIPER_NOISE_W" envDefault:"0.8"`
	SentenceSilence time.Duration `yaml:"sentence_silence" env:"GLOW_TTS_PIPER_SENTENCE_SILENCE" envDefault:"200ms"`
	PhonemeGap time.Duration     `yaml:"phoneme_gap" env:"GLOW_TTS_PIPER_PHONEME_GAP" envDefault:"0ms"`
	Timeout    time.Duration     `yaml:"timeout" env:"GLOW_TTS_PIPER_TIMEOUT" envDefault:"30s"`
}

// GoogleConfig contains Google TTS engine specific settings.
type GoogleConfig struct {
	APIKey      string  `yaml:"api_key" env:"GLOW_TTS_GOOGLE_API_KEY"`
	LanguageCode string  `yaml:"language_code" env:"GLOW_TTS_GOOGLE_LANGUAGE_CODE" envDefault:"en-US"`
	VoiceName    string  `yaml:"voice_name" env:"GLOW_TTS_GOOGLE_VOICE_NAME" envDefault:"en-US-Standard-A"`
	SpeakingRate float64 `yaml:"speaking_rate" env:"GLOW_TTS_GOOGLE_SPEAKING_RATE" envDefault:"1.0"`
	Pitch        float64 `yaml:"pitch" env:"GLOW_TTS_GOOGLE_PITCH" envDefault:"0.0"`
	VolumeGain   float64 `yaml:"volume_gain" env:"GLOW_TTS_GOOGLE_VOLUME_GAIN" envDefault:"0.0"`
	Timeout      time.Duration `yaml:"timeout" env:"GLOW_TTS_GOOGLE_TIMEOUT" envDefault:"10s"`
}

// MockConfig contains Mock TTS engine specific settings for testing.
type MockConfig struct {
	GenerationDelay time.Duration `yaml:"generation_delay" env:"GLOW_TTS_MOCK_GENERATION_DELAY" envDefault:"100ms"`
	WordsPerMinute  int           `yaml:"words_per_minute" env:"GLOW_TTS_MOCK_WORDS_PER_MINUTE" envDefault:"150"`
	FailureRate     float64       `yaml:"failure_rate" env:"GLOW_TTS_MOCK_FAILURE_RATE" envDefault:"0.0"`
	SimulateLatency bool          `yaml:"simulate_latency" env:"GLOW_TTS_MOCK_SIMULATE_LATENCY" envDefault:"true"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:    false,
		Engine:     "mock",
		SampleRate: 22050,
		Volume:     1.0,
		
		AutoPlay:           false,
		PauseOnFocusLoss:   true,
		BufferSize:         3,
		BufferAheadEnabled: true,
		
		WrapNavigation: true,
		SkipCodeBlocks: false,
		SkipURLs:       false,
		
		HighlightEnabled: true,
		HighlightColor:   "yellow",
		ShowProgress:     true,
		
		Piper:  DefaultPiperConfig(),
		Google: DefaultGoogleConfig(),
		Mock:   DefaultMockConfig(),
	}
}

// DefaultPiperConfig returns default Piper configuration.
func DefaultPiperConfig() PiperConfig {
	cfg := PiperConfig{
		Binary:          "piper",
		Model:           "en_US-lessac-medium",
		OutputRaw:       true,
		SpeakerId:       0,
		LengthScale:     1.0,
		NoiseScale:      0.667,
		NoiseW:          0.8,
		SentenceSilence: 200 * time.Millisecond,
		PhonemeGap:      0,
		Timeout:         30 * time.Second,
	}
	
	// Try to detect common Piper installation paths
	if runtime.GOOS == "linux" {
		cfg.DataDir = filepath.Join("/usr", "share", "piper")
	} else if runtime.GOOS == "darwin" {
		cfg.DataDir = filepath.Join("/usr", "local", "share", "piper")
	}
	
	return cfg
}

// DefaultGoogleConfig returns default Google TTS configuration.
func DefaultGoogleConfig() GoogleConfig {
	return GoogleConfig{
		LanguageCode: "en-US",
		VoiceName:    "en-US-Standard-A",
		SpeakingRate: 1.0,
		Pitch:        0.0,
		VolumeGain:   0.0,
		Timeout:      10 * time.Second,
	}
}

// DefaultMockConfig returns default Mock TTS configuration.
func DefaultMockConfig() MockConfig {
	return MockConfig{
		GenerationDelay: 100 * time.Millisecond,
		WordsPerMinute:  150,
		FailureRate:     0.0,
		SimulateLatency: true,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	// Validate engine selection
	validEngines := []string{"mock", "piper", "google"}
	engineValid := false
	for _, e := range validEngines {
		if strings.EqualFold(c.Engine, e) {
			engineValid = true
			c.Engine = strings.ToLower(c.Engine)
			break
		}
	}
	if !engineValid {
		return fmt.Errorf("invalid TTS engine '%s': must be one of %v", c.Engine, validEngines)
	}
	
	// Validate volume
	if c.Volume < 0.0 || c.Volume > 2.0 {
		return fmt.Errorf("volume must be between 0.0 and 2.0, got %f", c.Volume)
	}
	
	// Validate sample rate
	validSampleRates := []int{8000, 16000, 22050, 24000, 44100, 48000}
	sampleRateValid := false
	for _, sr := range validSampleRates {
		if c.SampleRate == sr {
			sampleRateValid = true
			break
		}
	}
	if !sampleRateValid {
		return fmt.Errorf("invalid sample rate %d: must be one of %v", c.SampleRate, validSampleRates)
	}
	
	// Validate buffer size
	if c.BufferSize < 1 || c.BufferSize > 10 {
		return fmt.Errorf("buffer size must be between 1 and 10, got %d", c.BufferSize)
	}
	
	// Validate highlight color
	validColors := []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white", "none"}
	colorValid := false
	for _, color := range validColors {
		if strings.EqualFold(c.HighlightColor, color) {
			colorValid = true
			c.HighlightColor = strings.ToLower(c.HighlightColor)
			break
		}
	}
	if !colorValid {
		return fmt.Errorf("invalid highlight color '%s': must be one of %v", c.HighlightColor, validColors)
	}
	
	// Validate engine-specific config
	switch c.Engine {
	case "piper":
		if err := c.Piper.Validate(); err != nil {
			return fmt.Errorf("piper config: %w", err)
		}
	case "google":
		if err := c.Google.Validate(); err != nil {
			return fmt.Errorf("google config: %w", err)
		}
	case "mock":
		if err := c.Mock.Validate(); err != nil {
			return fmt.Errorf("mock config: %w", err)
		}
	}
	
	return nil
}

// Validate checks if the Piper configuration is valid.
func (c *PiperConfig) Validate() error {
	if c.Binary == "" {
		return fmt.Errorf("piper binary path cannot be empty")
	}
	
	if c.Model == "" {
		return fmt.Errorf("piper model cannot be empty")
	}
	
	if c.LengthScale <= 0 || c.LengthScale > 3.0 {
		return fmt.Errorf("length_scale must be between 0.1 and 3.0, got %f", c.LengthScale)
	}
	
	if c.NoiseScale < 0 || c.NoiseScale > 2.0 {
		return fmt.Errorf("noise_scale must be between 0.0 and 2.0, got %f", c.NoiseScale)
	}
	
	if c.NoiseW < 0 || c.NoiseW > 2.0 {
		return fmt.Errorf("noise_w must be between 0.0 and 2.0, got %f", c.NoiseW)
	}
	
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", c.Timeout)
	}
	
	return nil
}

// Validate checks if the Google TTS configuration is valid.
func (c *GoogleConfig) Validate() error {
	if c.SpeakingRate < 0.25 || c.SpeakingRate > 4.0 {
		return fmt.Errorf("speaking_rate must be between 0.25 and 4.0, got %f", c.SpeakingRate)
	}
	
	if c.Pitch < -20.0 || c.Pitch > 20.0 {
		return fmt.Errorf("pitch must be between -20.0 and 20.0, got %f", c.Pitch)
	}
	
	if c.VolumeGain < -96.0 || c.VolumeGain > 16.0 {
		return fmt.Errorf("volume_gain must be between -96.0 and 16.0, got %f", c.VolumeGain)
	}
	
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", c.Timeout)
	}
	
	return nil
}

// Validate checks if the Mock configuration is valid.
func (c *MockConfig) Validate() error {
	if c.WordsPerMinute < 50 || c.WordsPerMinute > 500 {
		return fmt.Errorf("words_per_minute must be between 50 and 500, got %d", c.WordsPerMinute)
	}
	
	if c.FailureRate < 0.0 || c.FailureRate > 1.0 {
		return fmt.Errorf("failure_rate must be between 0.0 and 1.0, got %f", c.FailureRate)
	}
	
	return nil
}

// ToEngineConfig converts TTS config to engine config based on selected engine.
func (c *Config) ToEngineConfig() EngineConfig {
	ec := EngineConfig{}
	
	switch c.Engine {
	case "piper":
		ec.Voice = c.Piper.Model
		ec.Rate = float32(c.Piper.LengthScale)
		ec.Pitch = 0.0 // Piper doesn't have pitch control
		ec.Volume = float32(c.Volume)
	case "google":
		ec.Voice = c.Google.VoiceName
		ec.Rate = float32(c.Google.SpeakingRate)
		ec.Pitch = float32(c.Google.Pitch)
		ec.Volume = float32(c.Volume)
	case "mock":
		// Mock engine uses defaults
		ec.Voice = "mock"
		ec.Rate = 1.0
		ec.Pitch = 0.0
		ec.Volume = float32(c.Volume)
	}
	
	return ec
}

// ToControllerConfig converts TTS config to controller config.
func (c *Config) ToControllerConfig() ControllerConfig {
	return ControllerConfig{
		BufferSize:        c.BufferSize,
		RetryAttempts:     3, // Default retry attempts
		RetryDelay:        time.Second,
		GenerationTimeout: 30 * time.Second,
		EnableCaching:     c.BufferAheadEnabled, // Use buffer ahead as caching indicator
	}
}