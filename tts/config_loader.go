package tts

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// LoadConfigFromViper loads TTS configuration from Viper.
func LoadConfigFromViper() (Config, error) {
	cfg := DefaultConfig()
	
	// Global TTS settings
	if viper.IsSet("tts.enabled") {
		cfg.Enabled = viper.GetBool("tts.enabled")
	}
	if viper.IsSet("tts.engine") {
		cfg.Engine = viper.GetString("tts.engine")
	}
	
	// Audio settings
	if viper.IsSet("tts.sample_rate") {
		cfg.SampleRate = viper.GetInt("tts.sample_rate")
	}
	if viper.IsSet("tts.volume") {
		cfg.Volume = viper.GetFloat64("tts.volume")
	}
	
	// Playback settings
	if viper.IsSet("tts.auto_play") {
		cfg.AutoPlay = viper.GetBool("tts.auto_play")
	}
	if viper.IsSet("tts.pause_on_focus_loss") {
		cfg.PauseOnFocusLoss = viper.GetBool("tts.pause_on_focus_loss")
	}
	if viper.IsSet("tts.buffer_size") {
		cfg.BufferSize = viper.GetInt("tts.buffer_size")
	}
	if viper.IsSet("tts.buffer_ahead") {
		cfg.BufferAheadEnabled = viper.GetBool("tts.buffer_ahead")
	}
	
	// Navigation settings
	if viper.IsSet("tts.wrap_navigation") {
		cfg.WrapNavigation = viper.GetBool("tts.wrap_navigation")
	}
	if viper.IsSet("tts.skip_code_blocks") {
		cfg.SkipCodeBlocks = viper.GetBool("tts.skip_code_blocks")
	}
	if viper.IsSet("tts.skip_urls") {
		cfg.SkipURLs = viper.GetBool("tts.skip_urls")
	}
	
	// Visual settings
	if viper.IsSet("tts.highlight_enabled") {
		cfg.HighlightEnabled = viper.GetBool("tts.highlight_enabled")
	}
	if viper.IsSet("tts.highlight_color") {
		cfg.HighlightColor = viper.GetString("tts.highlight_color")
	}
	if viper.IsSet("tts.show_progress") {
		cfg.ShowProgress = viper.GetBool("tts.show_progress")
	}
	
	// Load Piper config
	cfg.Piper = loadPiperConfig()
	
	// Load Google config
	cfg.Google = loadGoogleConfig()
	
	// Load Mock config
	cfg.Mock = loadMockConfig()
	
	// Validate the loaded configuration
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid TTS configuration: %w", err)
	}
	
	return cfg, nil
}

// loadPiperConfig loads Piper-specific configuration from Viper.
func loadPiperConfig() PiperConfig {
	cfg := DefaultPiperConfig()
	
	if viper.IsSet("tts.piper.binary") {
		cfg.Binary = viper.GetString("tts.piper.binary")
	}
	if viper.IsSet("tts.piper.model") {
		cfg.Model = viper.GetString("tts.piper.model")
	}
	if viper.IsSet("tts.piper.model_path") {
		cfg.ModelPath = viper.GetString("tts.piper.model_path")
	}
	if viper.IsSet("tts.piper.config_path") {
		cfg.ConfigPath = viper.GetString("tts.piper.config_path")
	}
	if viper.IsSet("tts.piper.data_dir") {
		cfg.DataDir = viper.GetString("tts.piper.data_dir")
	}
	if viper.IsSet("tts.piper.output_raw") {
		cfg.OutputRaw = viper.GetBool("tts.piper.output_raw")
	}
	if viper.IsSet("tts.piper.speaker_id") {
		cfg.SpeakerId = viper.GetInt("tts.piper.speaker_id")
	}
	if viper.IsSet("tts.piper.length_scale") {
		cfg.LengthScale = viper.GetFloat64("tts.piper.length_scale")
	}
	if viper.IsSet("tts.piper.noise_scale") {
		cfg.NoiseScale = viper.GetFloat64("tts.piper.noise_scale")
	}
	if viper.IsSet("tts.piper.noise_w") {
		cfg.NoiseW = viper.GetFloat64("tts.piper.noise_w")
	}
	if viper.IsSet("tts.piper.sentence_silence") {
		if d, err := time.ParseDuration(viper.GetString("tts.piper.sentence_silence")); err == nil {
			cfg.SentenceSilence = d
		}
	}
	if viper.IsSet("tts.piper.phoneme_gap") {
		if d, err := time.ParseDuration(viper.GetString("tts.piper.phoneme_gap")); err == nil {
			cfg.PhonemeGap = d
		}
	}
	if viper.IsSet("tts.piper.timeout") {
		if d, err := time.ParseDuration(viper.GetString("tts.piper.timeout")); err == nil {
			cfg.Timeout = d
		}
	}
	
	return cfg
}

// loadGoogleConfig loads Google TTS-specific configuration from Viper.
func loadGoogleConfig() GoogleConfig {
	cfg := DefaultGoogleConfig()
	
	if viper.IsSet("tts.google.api_key") {
		cfg.APIKey = viper.GetString("tts.google.api_key")
	}
	if viper.IsSet("tts.google.language_code") {
		cfg.LanguageCode = viper.GetString("tts.google.language_code")
	}
	if viper.IsSet("tts.google.voice_name") {
		cfg.VoiceName = viper.GetString("tts.google.voice_name")
	}
	if viper.IsSet("tts.google.speaking_rate") {
		cfg.SpeakingRate = viper.GetFloat64("tts.google.speaking_rate")
	}
	if viper.IsSet("tts.google.pitch") {
		cfg.Pitch = viper.GetFloat64("tts.google.pitch")
	}
	if viper.IsSet("tts.google.volume_gain") {
		cfg.VolumeGain = viper.GetFloat64("tts.google.volume_gain")
	}
	if viper.IsSet("tts.google.timeout") {
		if d, err := time.ParseDuration(viper.GetString("tts.google.timeout")); err == nil {
			cfg.Timeout = d
		}
	}
	
	return cfg
}

// loadMockConfig loads Mock TTS-specific configuration from Viper.
func loadMockConfig() MockConfig {
	cfg := DefaultMockConfig()
	
	if viper.IsSet("tts.mock.generation_delay") {
		if d, err := time.ParseDuration(viper.GetString("tts.mock.generation_delay")); err == nil {
			cfg.GenerationDelay = d
		}
	}
	if viper.IsSet("tts.mock.words_per_minute") {
		cfg.WordsPerMinute = viper.GetInt("tts.mock.words_per_minute")
	}
	if viper.IsSet("tts.mock.failure_rate") {
		cfg.FailureRate = viper.GetFloat64("tts.mock.failure_rate")
	}
	if viper.IsSet("tts.mock.simulate_latency") {
		cfg.SimulateLatency = viper.GetBool("tts.mock.simulate_latency")
	}
	
	return cfg
}

// SetDefaults sets default values in Viper for TTS configuration.
func SetDefaults() {
	defaults := DefaultConfig()
	
	// Global TTS settings
	viper.SetDefault("tts.enabled", defaults.Enabled)
	viper.SetDefault("tts.engine", defaults.Engine)
	viper.SetDefault("tts.sample_rate", defaults.SampleRate)
	viper.SetDefault("tts.volume", defaults.Volume)
	
	// Playback settings
	viper.SetDefault("tts.auto_play", defaults.AutoPlay)
	viper.SetDefault("tts.pause_on_focus_loss", defaults.PauseOnFocusLoss)
	viper.SetDefault("tts.buffer_size", defaults.BufferSize)
	viper.SetDefault("tts.buffer_ahead", defaults.BufferAheadEnabled)
	
	// Navigation settings
	viper.SetDefault("tts.wrap_navigation", defaults.WrapNavigation)
	viper.SetDefault("tts.skip_code_blocks", defaults.SkipCodeBlocks)
	viper.SetDefault("tts.skip_urls", defaults.SkipURLs)
	
	// Visual settings
	viper.SetDefault("tts.highlight_enabled", defaults.HighlightEnabled)
	viper.SetDefault("tts.highlight_color", defaults.HighlightColor)
	viper.SetDefault("tts.show_progress", defaults.ShowProgress)
	
	// Piper defaults
	viper.SetDefault("tts.piper.binary", defaults.Piper.Binary)
	viper.SetDefault("tts.piper.model", defaults.Piper.Model)
	viper.SetDefault("tts.piper.output_raw", defaults.Piper.OutputRaw)
	viper.SetDefault("tts.piper.speaker_id", defaults.Piper.SpeakerId)
	viper.SetDefault("tts.piper.length_scale", defaults.Piper.LengthScale)
	viper.SetDefault("tts.piper.noise_scale", defaults.Piper.NoiseScale)
	viper.SetDefault("tts.piper.noise_w", defaults.Piper.NoiseW)
	viper.SetDefault("tts.piper.sentence_silence", defaults.Piper.SentenceSilence.String())
	viper.SetDefault("tts.piper.phoneme_gap", defaults.Piper.PhonemeGap.String())
	viper.SetDefault("tts.piper.timeout", defaults.Piper.Timeout.String())
	
	// Google defaults
	viper.SetDefault("tts.google.language_code", defaults.Google.LanguageCode)
	viper.SetDefault("tts.google.voice_name", defaults.Google.VoiceName)
	viper.SetDefault("tts.google.speaking_rate", defaults.Google.SpeakingRate)
	viper.SetDefault("tts.google.pitch", defaults.Google.Pitch)
	viper.SetDefault("tts.google.volume_gain", defaults.Google.VolumeGain)
	viper.SetDefault("tts.google.timeout", defaults.Google.Timeout.String())
	
	// Mock defaults
	viper.SetDefault("tts.mock.generation_delay", defaults.Mock.GenerationDelay.String())
	viper.SetDefault("tts.mock.words_per_minute", defaults.Mock.WordsPerMinute)
	viper.SetDefault("tts.mock.failure_rate", defaults.Mock.FailureRate)
	viper.SetDefault("tts.mock.simulate_latency", defaults.Mock.SimulateLatency)
}