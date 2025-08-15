package tts

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// TTSConfig represents the TTS configuration
type TTSConfig struct {
	// Default engine to use when --tts flag is provided without engine name
	DefaultEngine string `yaml:"default_engine" mapstructure:"default_engine"`
	
	// Engine-specific settings
	Engines EngineConfigs `yaml:"engines" mapstructure:"engines"`
	
	// Cache settings
	Cache TTSCacheConfig `yaml:"cache" mapstructure:"cache"`
	
	// Playback settings
	Playback PlaybackConfig `yaml:"playback" mapstructure:"playback"`
	
	// Advanced settings
	Advanced AdvancedConfig `yaml:"advanced" mapstructure:"advanced"`
}

// EngineConfigs holds configuration for each engine
type EngineConfigs struct {
	Piper PiperConfig `yaml:"piper" mapstructure:"piper"`
	GTTS  GTTSConfig  `yaml:"gtts" mapstructure:"gtts"`
}

// PiperConfig holds Piper-specific configuration
type PiperConfig struct {
	// Path to the ONNX model file
	ModelPath string `yaml:"model_path" mapstructure:"model_path"`
	
	// Voice name to use
	Voice string `yaml:"voice" mapstructure:"voice"`
	
	// Default speed (0.5 to 2.0)
	DefaultSpeed float64 `yaml:"default_speed" mapstructure:"default_speed"`
	
	// Enable cache
	EnableCache bool `yaml:"enable_cache" mapstructure:"enable_cache"`
}

// GTTSConfig holds Google TTS-specific configuration
type GTTSConfig struct {
	// Language code (e.g., "en", "es", "fr")
	Language string `yaml:"language" mapstructure:"language"`
	
	// Default speed (0.5 to 2.0)
	DefaultSpeed float64 `yaml:"default_speed" mapstructure:"default_speed"`
	
	// TLD for regional accents (e.g., "com", "co.uk", "com.au")
	TLD string `yaml:"tld" mapstructure:"tld"`
	
	// Enable slow mode
	Slow bool `yaml:"slow" mapstructure:"slow"`
}

// TTSCacheConfig holds cache-related settings
type TTSCacheConfig struct {
	// Enable caching
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
	
	// Cache directory (defaults to ~/.cache/glow/tts)
	Directory string `yaml:"directory" mapstructure:"directory"`
	
	// Maximum cache size in MB
	MaxSizeMB int `yaml:"max_size_mb" mapstructure:"max_size_mb"`
	
	// Cache expiration in hours
	ExpirationHours int `yaml:"expiration_hours" mapstructure:"expiration_hours"`
}

// PlaybackConfig holds playback-related settings
type PlaybackConfig struct {
	// Default playback speed
	DefaultSpeed float64 `yaml:"default_speed" mapstructure:"default_speed"`
	
	// Speed increment for +/- keys
	SpeedIncrement float64 `yaml:"speed_increment" mapstructure:"speed_increment"`
	
	// Lookahead sentences for preprocessing
	LookaheadSentences int `yaml:"lookahead_sentences" mapstructure:"lookahead_sentences"`
	
	// Auto-play on document open
	AutoPlay bool `yaml:"auto_play" mapstructure:"auto_play"`
}

// AdvancedConfig holds advanced settings
type AdvancedConfig struct {
	// Synthesis timeout in seconds
	SynthesisTimeout int `yaml:"synthesis_timeout" mapstructure:"synthesis_timeout"`
	
	// Worker threads for parallel synthesis
	WorkerThreads int `yaml:"worker_threads" mapstructure:"worker_threads"`
	
	// Debug logging
	DebugLogging bool `yaml:"debug_logging" mapstructure:"debug_logging"`
	
	// Audio buffer size in KB
	AudioBufferKB int `yaml:"audio_buffer_kb" mapstructure:"audio_buffer_kb"`
}

// DefaultTTSConfig returns the default TTS configuration
func DefaultTTSConfig() *TTSConfig {
	return &TTSConfig{
		DefaultEngine: "piper",
		Engines: EngineConfigs{
			Piper: PiperConfig{
				ModelPath:    "",
				Voice:        "",
				DefaultSpeed: 1.0,
				EnableCache:  true,
			},
			GTTS: GTTSConfig{
				Language:     "en",
				DefaultSpeed: 1.0,
				TLD:          "com",
				Slow:         false,
			},
		},
		Cache: TTSCacheConfig{
			Enabled:         true,
			Directory:       "",
			MaxSizeMB:       100,
			ExpirationHours: 24 * 7, // 1 week
		},
		Playback: PlaybackConfig{
			DefaultSpeed:       1.0,
			SpeedIncrement:     0.25,
			LookaheadSentences: 3,
			AutoPlay:           false,
		},
		Advanced: AdvancedConfig{
			SynthesisTimeout: 30,
			WorkerThreads:    2,
			DebugLogging:     false,
			AudioBufferKB:    256,
		},
	}
}

// configPaths returns the paths to check for config files
func configPaths() []string {
	paths := []string{}
	
	// Current directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".glow", "glow-tts.yml"))
	}
	
	// User config directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "glow", "glow-tts.yml"))
	}
	
	return paths
}

// LoadTTSConfig loads the TTS configuration from file
func LoadTTSConfig() (*TTSConfig, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	
	// Set defaults
	config := DefaultTTSConfig()
	
	// Check each config path
	var configFound bool
	for _, path := range configPaths() {
		if _, err := os.Stat(path); err == nil {
			v.SetConfigFile(path)
			if err := v.ReadInConfig(); err != nil {
				log.Warn("Failed to read TTS config", "path", path, "error", err)
				continue
			}
			
			// Unmarshal into our config struct
			if err := v.Unmarshal(config); err != nil {
				log.Warn("Failed to parse TTS config", "path", path, "error", err)
				continue
			}
			
			log.Info("Loaded TTS configuration", "path", path)
			configFound = true
			break
		}
	}
	
	if !configFound {
		log.Debug("No TTS config file found, using defaults")
	}
	
	// Validate and set defaults for cache directory
	if config.Cache.Directory == "" {
		if home, err := os.UserHomeDir(); err == nil {
			config.Cache.Directory = filepath.Join(home, ".cache", "glow", "tts")
		}
	}
	
	return config, nil
}

// SaveTTSConfig saves the TTS configuration to file
func SaveTTSConfig(config *TTSConfig, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	log.Info("Saved TTS configuration", "path", path)
	return nil
}

// GenerateExampleConfig generates an example configuration file
func GenerateExampleConfig() string {
	config := DefaultTTSConfig()
	
	// Add some example values
	config.DefaultEngine = "piper"
	config.Engines.Piper.ModelPath = "~/.local/share/piper-voices/en_US-amy-medium.onnx"
	config.Engines.Piper.Voice = "amy"
	config.Engines.GTTS.Language = "en"
	config.Cache.Directory = "~/.cache/glow/tts"
	
	data, _ := yaml.Marshal(config)
	
	header := `# Glow TTS Configuration File
# 
# This file configures Text-to-Speech settings for Glow.
# Place this file at:
#   - ./.glow/glow-tts.yml (project-specific)
#   - ~/.config/glow/glow-tts.yml (user-wide)
#
# The project-specific config takes precedence over user config.

`
	
	return header + string(data)
}

// ApplyConfig applies the configuration to a controller config
func (c *TTSConfig) ApplyToController(cfg *ControllerConfig) {
	// Apply cache settings
	cfg.EnableCache = c.Cache.Enabled
	cfg.CacheDir = c.Cache.Directory
	// TODO: Add these fields to ControllerConfig when implementing full cache support
	// cfg.CacheMaxSize = int64(c.Cache.MaxSizeMB) * 1024 * 1024
	// cfg.CacheExpiration = time.Duration(c.Cache.ExpirationHours) * time.Hour
	
	// Apply playback settings
	cfg.DefaultSpeed = c.Playback.DefaultSpeed
	cfg.LookaheadSentences = c.Playback.LookaheadSentences
	
	// TODO: Add these fields to ControllerConfig when implementing advanced features
	// cfg.SynthesisTimeout = time.Duration(c.Advanced.SynthesisTimeout) * time.Second
	// cfg.WorkerThreads = c.Advanced.WorkerThreads
}

// GetEngineOrDefault returns the specified engine or the default if empty
func (c *TTSConfig) GetEngineOrDefault(engine string) string {
	if engine == "" {
		return c.DefaultEngine
	}
	return engine
}