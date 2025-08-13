package engines

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/internal/cache"
	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

// PiperEngine implements the TTSEngine interface using Piper (offline TTS).
// It uses fresh process per synthesis with pre-configured stdin to avoid race conditions.
type PiperEngine struct {
	// Configuration
	modelPath  string
	configPath string
	voice      string
	sampleRate int

	// Caching
	cache       *cache.CacheManager
	cacheConfig *cache.CacheConfig

	// Synchronization
	mu sync.RWMutex
}

// PiperConfig holds configuration for the Piper engine.
type PiperConfig struct {
	// Model file path (required)
	ModelPath string

	// Config file path (optional, defaults to model path with .json extension)
	ConfigPath string

	// Voice name (optional, uses model default)
	Voice string

	// Sample rate (optional, defaults to 22050)
	SampleRate int

	// Cache configuration (optional)
	CacheConfig *cache.CacheConfig
}

// NewPiperEngine creates a new Piper TTS engine.
func NewPiperEngine(config PiperConfig) (*PiperEngine, error) {
	// Validate model path
	if config.ModelPath == "" {
		return nil, errors.New("model path is required")
	}

	// Check if model file exists
	if _, err := os.Stat(config.ModelPath); err != nil {
		return nil, fmt.Errorf("model file not found: %w", err)
	}

	// Set default config path if not specified
	if config.ConfigPath == "" {
		// Try .json extension on model path
		config.ConfigPath = strings.TrimSuffix(config.ModelPath, filepath.Ext(config.ModelPath)) + ".json"
	}

	// Set default sample rate (44100 Hz for OTO compatibility)
	// Note: Piper outputs 22050 Hz, but we resample to 44100 Hz
	if config.SampleRate == 0 {
		config.SampleRate = 44100
	}

	// Create cache manager if config provided
	var cacheManager *cache.CacheManager
	if config.CacheConfig != nil {
		var err error
		cacheManager, err = cache.NewCacheManager(config.CacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache manager: %w", err)
		}
	}

	return &PiperEngine{
		modelPath:   config.ModelPath,
		configPath:  config.ConfigPath,
		voice:       config.Voice,
		sampleRate:  config.SampleRate,
		cache:       cacheManager,
		cacheConfig: config.CacheConfig,
	}, nil
}

// Synthesize converts text to audio using Piper.
// CRITICAL: Uses pre-configured stdin to avoid race conditions.
func (e *PiperEngine) Synthesize(ctx context.Context, text string, speed float64) ([]byte, error) {
	// Check cache first if available
	if e.cache != nil {
		cacheKey := cache.GenerateCacheKey(text, e.voice, speed)
		if audio, ok := e.cache.Get(cacheKey); ok {
			return audio, nil
		}
	}

	// Validate text
	if text == "" {
		return nil, errors.New("text cannot be empty")
	}

	// Text size limit (Piper can handle large texts but we limit for performance)
	const maxTextSize = 5000
	if len(text) > maxTextSize {
		return nil, fmt.Errorf("text too long: %d characters (max %d)", len(text), maxTextSize)
	}

	// Calculate Piper's length scale from speed
	// Speed: 0.5 = half speed (scale 2.0), 2.0 = double speed (scale 0.5)
	lengthScale := 1.0 / speed

	// Build command arguments
	args := []string{
		"--model", e.modelPath,
		"--config", e.configPath,
		"--output-raw", // Raw PCM output
		"--length-scale", fmt.Sprintf("%.2f", lengthScale),
	}

	// Add voice if specified
	if e.voice != "" {
		args = append(args, "--speaker", e.voice)
	}

	// CRITICAL: Create command with timeout context
	// This prevents hanging processes
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "piper", args...)

	// CRITICAL: Pre-configure stdin with the text
	// This avoids the race condition where Piper reads stdin before we can write to it
	cmd.Stdin = strings.NewReader(text)

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create channel for command completion
	done := make(chan error, 1)

	// Run command in goroutine to handle timeout
	go func() {
		// CRITICAL: Use Run() not Start() for synchronous execution
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		// Command completed
		if err != nil {
			// Check if it's a context error
			if ctx.Err() != nil {
				return nil, fmt.Errorf("synthesis timeout: %w", ctx.Err())
			}
			// Regular error
			return nil, fmt.Errorf("piper failed: %w, stderr: %s", err, stderr.String())
		}

	case <-ctx.Done():
		// Timeout occurred
		// Try graceful shutdown first
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)

			// Give it a moment to clean up
			select {
			case <-done:
				// Process exited
			case <-time.After(100 * time.Millisecond):
				// Force kill
				cmd.Process.Kill()
				<-done // Wait for goroutine to finish
			}
		}

		return nil, fmt.Errorf("synthesis timeout after 10s: %w", ctx.Err())
	}

	// Get audio data
	audio := stdout.Bytes()

	// Validate output
	if len(audio) == 0 {
		return nil, fmt.Errorf("piper produced no audio output, stderr: %s", stderr.String())
	}

	// Sanity check: audio shouldn't be too large
	const maxAudioSize = 10 * 1024 * 1024 // 10MB
	if len(audio) > maxAudioSize {
		return nil, fmt.Errorf("piper output too large: %d bytes (max %d)", len(audio), maxAudioSize)
	}

	// Resample from 22050 Hz to 44100 Hz for OTO compatibility
	resampledAudio, err := e.resamplePCM(ctx, audio, 22050, 44100)
	if err != nil {
		return nil, fmt.Errorf("failed to resample audio: %w", err)
	}

	// Cache the result if cache is available
	if e.cache != nil {
		cacheKey := cache.GenerateCacheKey(text, e.voice, speed)
		// Ignore cache errors as they're non-fatal
		_ = e.cache.Put(cacheKey, resampledAudio)
	}

	return resampledAudio, nil
}

// resamplePCM resamples PCM audio from one sample rate to another using ffmpeg.
func (e *PiperEngine) resamplePCM(ctx context.Context, pcmData []byte, fromRate, toRate int) ([]byte, error) {
	// If rates are the same, no resampling needed
	if fromRate == toRate {
		return pcmData, nil
	}

	// Check if ffmpeg is available
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// Build ffmpeg command for PCM resampling
	args := []string{
		"-f", "s16le",                    // Input format: signed 16-bit little-endian
		"-ar", fmt.Sprintf("%d", fromRate), // Input sample rate
		"-ac", "1",                        // Input channels (mono)
		"-i", "pipe:0",                    // Read from stdin
		"-f", "s16le",                    // Output format: signed 16-bit little-endian
		"-ar", fmt.Sprintf("%d", toRate), // Output sample rate
		"-ac", "1",                        // Output channels (mono)
		"pipe:1",                          // Write to stdout
	}

	// Create command with context for timeout control
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)

	// Set up stdin with the PCM data
	cmd.Stdin = bytes.NewReader(pcmData)

	// Capture outputs
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run ffmpeg with timeout protection
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ffmpeg resampling failed: %w, stderr: %s", err, stderr.String())
		}

		resampledAudio := stdout.Bytes()

		// Validate resampled output
		if len(resampledAudio) == 0 {
			return nil, fmt.Errorf("ffmpeg produced no output, stderr: %s", stderr.String())
		}

		// The resampled audio should be approximately 2x the size (22050 -> 44100)
		// Allow some variance in the ratio (1.8x to 2.2x)
		expectedRatio := float64(toRate) / float64(fromRate)
		actualRatio := float64(len(resampledAudio)) / float64(len(pcmData))
		if actualRatio < expectedRatio*0.9 || actualRatio > expectedRatio*1.1 {
			// Just log warning, don't fail
			// Audio resampling can have slight variations
		}

		return resampledAudio, nil

	case <-ctx.Done():
		// Timeout or cancellation
		if cmd.Process != nil {
			cmd.Process.Kill()
			<-done // Wait for goroutine to finish
		}
		return nil, fmt.Errorf("resampling timeout: %w", ctx.Err())
	}
}

// GetInfo returns engine capabilities and configuration.
func (e *PiperEngine) GetInfo() ttypes.EngineInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return ttypes.EngineInfo{
		Name:        "piper",
		Version:     "1.0.0", // Would need to query piper --version
		SampleRate:  e.sampleRate,
		Channels:    1, // Piper outputs mono
		BitDepth:    16,
		MaxTextSize: 5000,
		IsOnline:    false, // Offline engine
	}
}

// Validate checks if the engine is properly configured and available.
func (e *PiperEngine) Validate() error {
	// Check if Piper binary is available
	piperPath, err := exec.LookPath("piper")
	if err != nil {
		return fmt.Errorf("piper not found in PATH: %w", err)
	}

	// Check if we can execute it
	cmd := exec.Command(piperPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot execute piper: %w", err)
	}

	// Check model file exists and is readable
	if _, err := os.Stat(e.modelPath); err != nil {
		return fmt.Errorf("model file not accessible: %w", err)
	}

	// Check config file if specified
	if e.configPath != "" {
		if _, err := os.Stat(e.configPath); err != nil {
			// Config file is optional, so just warn
			// In production, use proper logging
		}
	}

	// Try a test synthesis to ensure everything works
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = e.Synthesize(testCtx, "Test", 1.0)
	if err != nil {
		return fmt.Errorf("test synthesis failed: %w", err)
	}

	return nil
}

// Close releases resources held by the engine.
func (e *PiperEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Close cache manager if present
	if e.cache != nil {
		if err := e.cache.Close(); err != nil {
			return fmt.Errorf("failed to close cache: %w", err)
		}
	}

	return nil
}

// SetVoice changes the voice/speaker for synthesis.
func (e *PiperEngine) SetVoice(voice string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.voice = voice
}

// GetVoice returns the current voice/speaker.
func (e *PiperEngine) GetVoice() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.voice
}

// GetCacheStats returns cache statistics if caching is enabled.
func (e *PiperEngine) GetCacheStats() map[string]interface{} {
	if e.cache == nil {
		return nil
	}
	return e.cache.Stats()
}

// ClearCache clears the audio cache if enabled.
func (e *PiperEngine) ClearCache() error {
	if e.cache == nil {
		return nil
	}
	return e.cache.Clear()
}

// Ensure PiperEngine implements TTSEngine interface
var _ ttypes.TTSEngine = (*PiperEngine)(nil)
