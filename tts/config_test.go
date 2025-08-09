package tts

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestDefaultConfig tests that default configuration is valid.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if err := cfg.Validate(); err != nil {
		t.Errorf("Default config should be valid: %v", err)
	}
	
	if cfg.Engine != "mock" {
		t.Errorf("Default engine should be mock, got %s", cfg.Engine)
	}
	
	if cfg.Enabled {
		t.Error("TTS should be disabled by default")
	}
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "invalid engine",
			modify: func(c *Config) {
				c.Engine = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid TTS engine",
		},
		{
			name: "volume too high",
			modify: func(c *Config) {
				c.Volume = 3.0
			},
			wantErr: true,
			errMsg:  "volume must be between",
		},
		{
			name: "volume too low",
			modify: func(c *Config) {
				c.Volume = -1.0
			},
			wantErr: true,
			errMsg:  "volume must be between",
		},
		{
			name: "invalid sample rate",
			modify: func(c *Config) {
				c.SampleRate = 12345
			},
			wantErr: true,
			errMsg:  "invalid sample rate",
		},
		{
			name: "buffer size too small",
			modify: func(c *Config) {
				c.BufferSize = 0
			},
			wantErr: true,
			errMsg:  "buffer size must be between",
		},
		{
			name: "buffer size too large",
			modify: func(c *Config) {
				c.BufferSize = 20
			},
			wantErr: true,
			errMsg:  "buffer size must be between",
		},
		{
			name: "invalid highlight color",
			modify: func(c *Config) {
				c.HighlightColor = "purple"
			},
			wantErr: true,
			errMsg:  "invalid highlight color",
		},
		{
			name: "case insensitive engine",
			modify: func(c *Config) {
				c.Engine = "PIPER"
			},
			wantErr: false,
		},
		{
			name: "case insensitive color",
			modify: func(c *Config) {
				c.HighlightColor = "YELLOW"
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// TestPiperConfigValidation tests Piper configuration validation.
func TestPiperConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  PiperConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultPiperConfig(),
			wantErr: false,
		},
		{
			name: "empty binary",
			config: PiperConfig{
				Binary: "",
				Model:  "test",
			},
			wantErr: true,
		},
		{
			name: "empty model",
			config: PiperConfig{
				Binary: "piper",
				Model:  "",
			},
			wantErr: true,
		},
		{
			name: "invalid length scale",
			config: PiperConfig{
				Binary:      "piper",
				Model:       "test",
				LengthScale: 5.0,
			},
			wantErr: true,
		},
		{
			name: "invalid noise scale",
			config: PiperConfig{
				Binary:      "piper",
				Model:       "test",
				LengthScale: 1.0,
				NoiseScale:  -1.0,
			},
			wantErr: true,
		},
		{
			name: "timeout too short",
			config: PiperConfig{
				Binary:      "piper",
				Model:       "test",
				LengthScale: 1.0,
				NoiseScale:  0.5,
				NoiseW:      0.5,
				Timeout:     500 * time.Millisecond,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGoogleConfigValidation tests Google TTS configuration validation.
func TestGoogleConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  GoogleConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultGoogleConfig(),
			wantErr: false,
		},
		{
			name: "speaking rate too low",
			config: GoogleConfig{
				SpeakingRate: 0.1,
			},
			wantErr: true,
		},
		{
			name: "speaking rate too high",
			config: GoogleConfig{
				SpeakingRate: 5.0,
			},
			wantErr: true,
		},
		{
			name: "pitch too low",
			config: GoogleConfig{
				SpeakingRate: 1.0,
				Pitch:        -25.0,
			},
			wantErr: true,
		},
		{
			name: "pitch too high",
			config: GoogleConfig{
				SpeakingRate: 1.0,
				Pitch:        25.0,
			},
			wantErr: true,
		},
		{
			name: "volume gain too low",
			config: GoogleConfig{
				SpeakingRate: 1.0,
				Pitch:        0.0,
				VolumeGain:   -100.0,
			},
			wantErr: true,
		},
		{
			name: "volume gain too high",
			config: GoogleConfig{
				SpeakingRate: 1.0,
				Pitch:        0.0,
				VolumeGain:   20.0,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMockConfigValidation tests Mock TTS configuration validation.
func TestMockConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  MockConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultMockConfig(),
			wantErr: false,
		},
		{
			name: "wpm too low",
			config: MockConfig{
				WordsPerMinute: 30,
			},
			wantErr: true,
		},
		{
			name: "wpm too high",
			config: MockConfig{
				WordsPerMinute: 600,
			},
			wantErr: true,
		},
		{
			name: "failure rate too high",
			config: MockConfig{
				WordsPerMinute: 150,
				FailureRate:    1.5,
			},
			wantErr: true,
		},
		{
			name: "failure rate negative",
			config: MockConfig{
				WordsPerMinute: 150,
				FailureRate:    -0.1,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestToEngineConfig tests conversion to EngineConfig.
func TestToEngineConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   EngineConfig
	}{
		{
			name: "piper engine",
			config: Config{
				Engine:     "piper",
				SampleRate: 22050,
				Volume:     0.8,
				Piper: PiperConfig{
					Model:       "en_US-lessac-medium",
					LengthScale: 1.2,
				},
			},
			want: EngineConfig{
				Voice:  "en_US-lessac-medium",
				Rate:   1.2,
				Pitch:  0.0,
				Volume: 0.8,
			},
		},
		{
			name: "google engine",
			config: Config{
				Engine:     "google",
				SampleRate: 16000,
				Volume:     1.0,
				Google: GoogleConfig{
					VoiceName:    "en-US-Wavenet-A",
					SpeakingRate: 1.5,
					Pitch:        2.0,
				},
			},
			want: EngineConfig{
				Voice:  "en-US-Wavenet-A",
				Rate:   1.5,
				Pitch:  2.0,
				Volume: 1.0,
			},
		},
		{
			name: "mock engine",
			config: Config{
				Engine:     "mock",
				SampleRate: 44100,
				Volume:     0.5,
			},
			want: EngineConfig{
				Voice:  "mock",
				Rate:   1.0,
				Pitch:  0.0,
				Volume: 0.5,
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ToEngineConfig()
			
			if got.Voice != tt.want.Voice {
				t.Errorf("Voice = %v, want %v", got.Voice, tt.want.Voice)
			}
			if got.Rate != tt.want.Rate {
				t.Errorf("Rate = %v, want %v", got.Rate, tt.want.Rate)
			}
			if got.Pitch != tt.want.Pitch {
				t.Errorf("Pitch = %v, want %v", got.Pitch, tt.want.Pitch)
			}
			if got.Volume != tt.want.Volume {
				t.Errorf("Volume = %v, want %v", got.Volume, tt.want.Volume)
			}
		})
	}
}

// TestToControllerConfig tests conversion to ControllerConfig.
func TestToControllerConfig(t *testing.T) {
	cfg := Config{
		BufferSize:         5,
		BufferAheadEnabled: true,
		AutoPlay:           true,
		PauseOnFocusLoss:   false,
	}
	
	controllerCfg := cfg.ToControllerConfig()
	
	if controllerCfg.BufferSize != 5 {
		t.Errorf("BufferSize = %v, want 5", controllerCfg.BufferSize)
	}
	if !controllerCfg.EnableCaching {
		t.Error("EnableCaching should be true (based on BufferAheadEnabled)")
	}
	if controllerCfg.RetryAttempts != 3 {
		t.Errorf("RetryAttempts = %v, want 3", controllerCfg.RetryAttempts)
	}
	if controllerCfg.RetryDelay != time.Second {
		t.Errorf("RetryDelay = %v, want 1s", controllerCfg.RetryDelay)
	}
	if controllerCfg.GenerationTimeout != 30*time.Second {
		t.Errorf("GenerationTimeout = %v, want 30s", controllerCfg.GenerationTimeout)
	}
}

// TestLoadConfigFromViper tests loading configuration from Viper.
func TestLoadConfigFromViper(t *testing.T) {
	// Save current viper state
	v := viper.New()
	
	// Set test values
	v.Set("tts.enabled", true)
	v.Set("tts.engine", "piper")
	v.Set("tts.volume", 0.9)
	v.Set("tts.buffer_size", 5)
	v.Set("tts.highlight_color", "green")
	v.Set("tts.piper.model", "test-model")
	v.Set("tts.piper.length_scale", 1.5)
	v.Set("tts.google.voice_name", "test-voice")
	v.Set("tts.mock.words_per_minute", 200)
	
	// Replace global viper temporarily
	oldViper := viper.GetViper()
	viper.Reset()
	for key, value := range v.AllSettings() {
		viper.Set(key, value)
	}
	defer func() {
		viper.Reset()
		for key, value := range oldViper.AllSettings() {
			viper.Set(key, value)
		}
	}()
	
	cfg, err := LoadConfigFromViper()
	if err != nil {
		t.Fatalf("LoadConfigFromViper() error = %v", err)
	}
	
	if !cfg.Enabled {
		t.Error("TTS should be enabled")
	}
	if cfg.Engine != "piper" {
		t.Errorf("Engine = %v, want piper", cfg.Engine)
	}
	if cfg.Volume != 0.9 {
		t.Errorf("Volume = %v, want 0.9", cfg.Volume)
	}
	if cfg.BufferSize != 5 {
		t.Errorf("BufferSize = %v, want 5", cfg.BufferSize)
	}
	if cfg.HighlightColor != "green" {
		t.Errorf("HighlightColor = %v, want green", cfg.HighlightColor)
	}
	if cfg.Piper.Model != "test-model" {
		t.Errorf("Piper.Model = %v, want test-model", cfg.Piper.Model)
	}
	if cfg.Piper.LengthScale != 1.5 {
		t.Errorf("Piper.LengthScale = %v, want 1.5", cfg.Piper.LengthScale)
	}
	if cfg.Google.VoiceName != "test-voice" {
		t.Errorf("Google.VoiceName = %v, want test-voice", cfg.Google.VoiceName)
	}
	if cfg.Mock.WordsPerMinute != 200 {
		t.Errorf("Mock.WordsPerMinute = %v, want 200", cfg.Mock.WordsPerMinute)
	}
}

// TestLoadConfigDurationParsing tests duration parsing from Viper.
func TestLoadConfigDurationParsing(t *testing.T) {
	v := viper.New()
	
	// Set duration values
	v.Set("tts.piper.sentence_silence", "500ms")
	v.Set("tts.piper.timeout", "1m")
	v.Set("tts.google.timeout", "5s")
	v.Set("tts.mock.generation_delay", "250ms")
	
	// Replace global viper temporarily
	oldViper := viper.GetViper()
	viper.Reset()
	for key, value := range v.AllSettings() {
		viper.Set(key, value)
	}
	defer func() {
		viper.Reset()
		for key, value := range oldViper.AllSettings() {
			viper.Set(key, value)
		}
	}()
	
	cfg, err := LoadConfigFromViper()
	if err != nil {
		t.Fatalf("LoadConfigFromViper() error = %v", err)
	}
	
	if cfg.Piper.SentenceSilence != 500*time.Millisecond {
		t.Errorf("Piper.SentenceSilence = %v, want 500ms", cfg.Piper.SentenceSilence)
	}
	if cfg.Piper.Timeout != time.Minute {
		t.Errorf("Piper.Timeout = %v, want 1m", cfg.Piper.Timeout)
	}
	if cfg.Google.Timeout != 5*time.Second {
		t.Errorf("Google.Timeout = %v, want 5s", cfg.Google.Timeout)
	}
	if cfg.Mock.GenerationDelay != 250*time.Millisecond {
		t.Errorf("Mock.GenerationDelay = %v, want 250ms", cfg.Mock.GenerationDelay)
	}
}

// TestSetDefaults tests that SetDefaults properly sets Viper defaults.
func TestSetDefaults(t *testing.T) {
	// Replace global viper temporarily
	oldViper := viper.GetViper()
	viper.Reset()
	defer func() {
		viper.Reset()
		for key, value := range oldViper.AllSettings() {
			viper.Set(key, value)
		}
	}()
	
	// Set defaults
	SetDefaults()
	
	// Check some key defaults
	if !viper.IsSet("tts.enabled") {
		t.Error("tts.enabled default not set")
	}
	if viper.GetBool("tts.enabled") {
		t.Error("tts.enabled should default to false")
	}
	
	if !viper.IsSet("tts.engine") {
		t.Error("tts.engine default not set")
	}
	if viper.GetString("tts.engine") != "mock" {
		t.Errorf("tts.engine = %v, want mock", viper.GetString("tts.engine"))
	}
	
	if !viper.IsSet("tts.sample_rate") {
		t.Error("tts.sample_rate default not set")
	}
	if viper.GetInt("tts.sample_rate") != 22050 {
		t.Errorf("tts.sample_rate = %v, want 22050", viper.GetInt("tts.sample_rate"))
	}
	
	if !viper.IsSet("tts.piper.binary") {
		t.Error("tts.piper.binary default not set")
	}
	if viper.GetString("tts.piper.binary") != "piper" {
		t.Errorf("tts.piper.binary = %v, want piper", viper.GetString("tts.piper.binary"))
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}