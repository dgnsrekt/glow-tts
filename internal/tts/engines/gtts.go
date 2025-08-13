package engines

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/internal/cache"
	"github.com/charmbracelet/glow/v2/internal/ttypes"
	"golang.org/x/time/rate"
)

// GTTSEngine implements the TTSEngine interface using gTTS (Google Translate TTS).
// It uses gtts-cli to generate MP3 files, then converts to PCM using ffmpeg.
// This provides free TTS without requiring an API key.
type GTTSEngine struct {
	// Configuration
	language   string
	slow       bool
	tempDir    string
	sampleRate int

	// Rate limiting to avoid being blocked by Google
	rateLimiter *rate.Limiter

	// Caching
	cache       *cache.CacheManager
	cacheConfig *cache.CacheConfig

	// Synchronization
	mu sync.RWMutex
}

// GTTSConfig holds configuration for the gTTS engine.
type GTTSConfig struct {
	// Language code (e.g., "en", "es", "fr") - defaults to "en"
	Language string

	// Slow speech (--slow flag) - defaults to false
	Slow bool

	// TempDir for intermediate files - defaults to system temp
	TempDir string

	// Sample rate (optional, defaults to 22050)
	SampleRate int

	// Cache configuration (optional)
	CacheConfig *cache.CacheConfig

	// Rate limit requests per minute to avoid being blocked (defaults to 50)
	RequestsPerMinute int
}

// NewGTTSEngine creates a new gTTS TTS engine.
func NewGTTSEngine(config GTTSConfig) (*GTTSEngine, error) {
	// Set default language
	if config.Language == "" {
		config.Language = "en"
	}

	// Set default temp directory
	if config.TempDir == "" {
		config.TempDir = os.TempDir()
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Set default sample rate (44100 Hz for OTO compatibility)
	if config.SampleRate == 0 {
		config.SampleRate = 44100
	}

	// Set default rate limit
	if config.RequestsPerMinute == 0 {
		config.RequestsPerMinute = 50 // Conservative default
	}

	// Create rate limiter
	rateLimiter := rate.NewLimiter(rate.Every(time.Minute/time.Duration(config.RequestsPerMinute)), 1)

	// Create cache manager if config provided
	var cacheManager *cache.CacheManager
	if config.CacheConfig != nil {
		var err error
		cacheManager, err = cache.NewCacheManager(config.CacheConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache manager: %w", err)
		}
	}

	return &GTTSEngine{
		language:    config.Language,
		slow:        config.Slow,
		tempDir:     config.TempDir,
		sampleRate:  config.SampleRate,
		rateLimiter: rateLimiter,
		cache:       cacheManager,
		cacheConfig: config.CacheConfig,
	}, nil
}

// Synthesize converts text to audio using gTTS.
// Process: text → gtts-cli → MP3 → ffmpeg → PCM
func (e *GTTSEngine) Synthesize(ctx context.Context, text string, speed float64) ([]byte, error) {
	// Check cache first if available
	if e.cache != nil {
		cacheKey := cache.GenerateCacheKey(text, e.language, speed)
		if audio, ok := e.cache.Get(cacheKey); ok {
			return audio, nil
		}
	}

	// Validate text
	if text == "" {
		return nil, errors.New("text cannot be empty")
	}

	// Text size limit (Google has limits on text length)
	const maxTextSize = 5000
	if len(text) > maxTextSize {
		return nil, fmt.Errorf("text too long: %d characters (max %d)", len(text), maxTextSize)
	}

	// Rate limit to avoid being blocked
	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	// Step 1: Generate MP3 using gtts-cli
	mp3Data, err := e.synthesizeToMP3(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("MP3 generation failed: %w", err)
	}

	// Step 2: Convert MP3 to PCM using ffmpeg
	pcmData, err := e.convertMP3ToPCM(ctx, mp3Data, speed)
	if err != nil {
		return nil, fmt.Errorf("MP3 to PCM conversion failed: %w", err)
	}

	// Cache the result if cache is available
	if e.cache != nil {
		cacheKey := cache.GenerateCacheKey(text, e.language, speed)
		// Ignore cache errors as they're non-fatal
		_ = e.cache.Put(cacheKey, pcmData)
	}

	return pcmData, nil
}

// synthesizeToMP3 generates MP3 audio using gtts-cli
func (e *GTTSEngine) synthesizeToMP3(ctx context.Context, text string) ([]byte, error) {
	// Build command arguments
	args := []string{
		text,
		"-l", e.language, // Language
	}

	// Add slow flag if enabled
	if e.slow {
		args = append(args, "--slow")
	}

	// Output to stdout
	args = append(args, "-o", "-")

	// CRITICAL: Create command with timeout context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // Longer timeout for network
	defer cancel()

	cmd := exec.CommandContext(ctx, "gtts-cli", args...)

	// CRITICAL: Pre-configure stdin with the text (though gtts-cli takes text as argument)
	cmd.Stdin = strings.NewReader("")

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
				return nil, fmt.Errorf("gTTS synthesis timeout: %w", ctx.Err())
			}
			// Regular error
			return nil, fmt.Errorf("gtts-cli failed: %w, stderr: %s", err, stderr.String())
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

		return nil, fmt.Errorf("gTTS synthesis timeout after 30s: %w", ctx.Err())
	}

	// Get MP3 data
	mp3Data := stdout.Bytes()

	// Validate output
	if len(mp3Data) == 0 {
		return nil, fmt.Errorf("gtts-cli produced no MP3 output, stderr: %s", stderr.String())
	}

	// Sanity check: MP3 shouldn't be too large
	const maxMP3Size = 50 * 1024 * 1024 // 50MB
	if len(mp3Data) > maxMP3Size {
		return nil, fmt.Errorf("gtts-cli MP3 output too large: %d bytes (max %d)", len(mp3Data), maxMP3Size)
	}

	return mp3Data, nil
}

// convertMP3ToPCM converts MP3 data to PCM using ffmpeg
func (e *GTTSEngine) convertMP3ToPCM(ctx context.Context, mp3Data []byte, speed float64) ([]byte, error) {
	// Create temp file for MP3 input
	mp3File, err := os.CreateTemp(e.tempDir, "gtts-*.mp3")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp MP3 file: %w", err)
	}
	defer os.Remove(mp3File.Name()) // Clean up
	defer mp3File.Close()

	// Write MP3 data to temp file
	if _, err := mp3File.Write(mp3Data); err != nil {
		return nil, fmt.Errorf("failed to write MP3 data: %w", err)
	}
	mp3File.Close() // Close before reading with ffmpeg

	// Build ffmpeg command
	args := []string{
		"-i", mp3File.Name(), // Input MP3 file
		"-f", "s16le", // Output format: signed 16-bit little-endian
		"-ar", "44100", // Force 44100 Hz output for OTO compatibility
		"-ac", "1", // Mono (1 channel)
	}

	// Add speed adjustment using atempo filter if speed != 1.0
	if speed != 1.0 {
		// ffmpeg atempo filter supports 0.5 to 2.0 range
		clampedSpeed := speed
		if clampedSpeed < 0.5 {
			clampedSpeed = 0.5
		} else if clampedSpeed > 2.0 {
			clampedSpeed = 2.0
		}
		args = append(args, "-filter:a", fmt.Sprintf("atempo=%.2f", clampedSpeed))
	}

	// Output to stdout
	args = append(args, "-")

	// CRITICAL: Create command with timeout context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second) // Reasonable timeout for conversion
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// CRITICAL: Pre-configure stdin (not used by ffmpeg in this case)
	cmd.Stdin = strings.NewReader("")

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Create channel for command completion
	done := make(chan error, 1)

	// Run command in goroutine to handle timeout
	go func() {
		done <- cmd.Run()
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		// Command completed
		if err != nil {
			// Check if it's a context error
			if ctx.Err() != nil {
				return nil, fmt.Errorf("ffmpeg conversion timeout: %w", ctx.Err())
			}
			// Regular error
			return nil, fmt.Errorf("ffmpeg failed: %w, stderr: %s", err, stderr.String())
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

		return nil, fmt.Errorf("ffmpeg conversion timeout after 15s: %w", ctx.Err())
	}

	// Get PCM data
	pcmData := stdout.Bytes()

	// Validate output
	if len(pcmData) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no PCM output, stderr: %s", stderr.String())
	}

	// Sanity check: PCM shouldn't be too large
	const maxPCMSize = 20 * 1024 * 1024 // 20MB
	if len(pcmData) > maxPCMSize {
		return nil, fmt.Errorf("ffmpeg PCM output too large: %d bytes (max %d)", len(pcmData), maxPCMSize)
	}

	return pcmData, nil
}

// GetInfo returns engine capabilities and configuration.
func (e *GTTSEngine) GetInfo() ttypes.EngineInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return ttypes.EngineInfo{
		Name:        "gtts",
		Version:     "1.0.0", // Would need to query gtts-cli --version
		SampleRate:  e.sampleRate,
		Channels:    1, // Output mono
		BitDepth:    16,
		MaxTextSize: 5000,
		IsOnline:    true, // Requires internet connection
	}
}

// Validate checks if the engine is properly configured and available.
func (e *GTTSEngine) Validate() error {
	// Check if gtts-cli binary is available
	gttsPath, err := exec.LookPath("gtts-cli")
	if err != nil {
		return fmt.Errorf("gtts-cli not found in PATH: %w\n\nInstall with: pip install gtts", err)
	}

	// Check if ffmpeg binary is available
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w\n\nInstall ffmpeg for audio conversion", err)
	}

	// Check if we can execute gtts-cli
	cmd := exec.Command(gttsPath, "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot execute gtts-cli: %w", err)
	}

	// Check if we can execute ffmpeg
	cmd = exec.Command(ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot execute ffmpeg: %w", err)
	}

	// Try a test synthesis to ensure everything works (including network)
	testCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second) // Longer for network test
	defer cancel()

	_, err = e.Synthesize(testCtx, "Test", 1.0)
	if err != nil {
		return fmt.Errorf("test synthesis failed: %w\n\nCheck internet connection for gTTS", err)
	}

	return nil
}

// Close releases resources held by the engine.
func (e *GTTSEngine) Close() error {
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

// SetLanguage changes the language for synthesis.
func (e *GTTSEngine) SetLanguage(language string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.language = language
}

// GetLanguage returns the current language.
func (e *GTTSEngine) GetLanguage() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.language
}

// SetSlow enables or disables slow speech.
func (e *GTTSEngine) SetSlow(slow bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.slow = slow
}

// GetSlow returns whether slow speech is enabled.
func (e *GTTSEngine) GetSlow() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.slow
}

// GetCacheStats returns cache statistics if caching is enabled.
func (e *GTTSEngine) GetCacheStats() map[string]interface{} {
	if e.cache == nil {
		return nil
	}
	return e.cache.Stats()
}

// ClearCache clears the audio cache if enabled.
func (e *GTTSEngine) ClearCache() error {
	if e.cache == nil {
		return nil
	}
	return e.cache.Clear()
}

// Ensure GTTSEngine implements TTSEngine interface
var _ ttypes.TTSEngine = (*GTTSEngine)(nil)
