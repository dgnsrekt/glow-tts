package engines

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/pkg/tts"
	"github.com/charmbracelet/log"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GTTSEngine implements the TTSEngine interface using Google Text-to-Speech via gtts-cli
type GTTSEngine struct {
	// Configuration
	language string
	speed    float64
	
	// Dependencies
	gttsBinary   string
	ffmpegBinary string
	
	// State
	mu            sync.RWMutex
	initialized   bool
	lastError     error
	
	// Temp directory for intermediate files
	tempDir string
}

// NewGTTSEngine creates a new Google TTS engine instance
func NewGTTSEngine() (*GTTSEngine, error) {
	log.Debug("GTTS: NewGTTSEngine called", "HOME", os.Getenv("HOME"))
	
	// Create temp directory for intermediate files
	tempDir := filepath.Join(os.TempDir(), "glow-gtts")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	
	engine := &GTTSEngine{
		language: "en",     // Default to English
		speed:    1.0,      // Normal speed
		tempDir:  tempDir,
	}
	
	// Detect dependencies
	if err := engine.detectDependencies(); err != nil {
		log.Error("GTTS: Dependency detection failed", "error", err)
		return nil, err
	}
	
	engine.initialized = true
	log.Info("Google TTS engine initialized", 
		"language", engine.language,
		"gtts", engine.gttsBinary,
		"ffmpeg", engine.ffmpegBinary)
	
	return engine, nil
}

// detectDependencies checks for required external tools
func (e *GTTSEngine) detectDependencies() error {
	// Check for gtts-cli
	gttsPaths := []string{
		"gtts-cli",
		"/usr/local/bin/gtts-cli",
		"/usr/bin/gtts-cli",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "gtts-cli"),
	}
	
	log.Debug("GTTS: Searching for gtts-cli", "paths", gttsPaths, "HOME", os.Getenv("HOME"))
	
	for _, path := range gttsPaths {
		log.Debug("GTTS: Checking path", "path", path)
		
		// First check if file exists (for absolute paths)
		if _, err := os.Stat(path); err == nil {
			log.Debug("GTTS: File exists", "path", path)
			// Verify it's actually gtts-cli
			cmd := exec.Command(path, "--help")
			if output, err := cmd.CombinedOutput(); err == nil {
				outputStr := string(output)
				log.Debug("GTTS: Help output", "path", path, "output", outputStr[:min(100, len(outputStr))])
				if strings.Contains(outputStr, "Google Translate") ||
				   strings.Contains(outputStr, "Text-to-Speech") ||
				   strings.Contains(outputStr, "gtts-cli") ||
				   strings.Contains(outputStr, "mp3 format") {
					e.gttsBinary = path
					log.Info("GTTS: Found gtts-cli", "path", path)
					break
				} else {
					log.Debug("GTTS: Help output doesn't match expected patterns", "path", path)
				}
			} else {
				log.Debug("GTTS: Failed to run --help", "path", path, "error", err)
			}
		} else if _, err := exec.LookPath(path); err == nil {
			log.Debug("GTTS: Found in PATH", "path", path)
			// Try to find in PATH
			// Verify it's actually gtts-cli
			cmd := exec.Command(path, "--help")
			if output, err := cmd.CombinedOutput(); err == nil {
				outputStr := string(output)
				log.Debug("GTTS: Help output", "path", path, "output", outputStr[:min(100, len(outputStr))])
				if strings.Contains(outputStr, "Google Translate") ||
				   strings.Contains(outputStr, "Text-to-Speech") ||
				   strings.Contains(outputStr, "gtts-cli") ||
				   strings.Contains(outputStr, "mp3 format") {
					e.gttsBinary = path
					log.Info("GTTS: Found gtts-cli", "path", path)
					break
				} else {
					log.Debug("GTTS: Help output doesn't match expected patterns", "path", path)
				}
			} else {
				log.Debug("GTTS: Failed to run --help", "path", path, "error", err)
			}
		} else {
			log.Debug("GTTS: Path not found", "path", path, "error", err)
		}
	}
	
	if e.gttsBinary == "" {
		return fmt.Errorf("gtts-cli not found. Install with: pip install gtts")
	}
	
	// Check for ffmpeg
	ffmpegPaths := []string{
		"ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/usr/bin/ffmpeg",
		"/opt/homebrew/bin/ffmpeg", // macOS ARM
	}
	
	for _, path := range ffmpegPaths {
		if _, err := exec.LookPath(path); err == nil {
			// Verify it's actually ffmpeg
			cmd := exec.Command(path, "-version")
			if output, err := cmd.CombinedOutput(); err == nil {
				if strings.Contains(string(output), "ffmpeg") {
					e.ffmpegBinary = path
					log.Debug("Found ffmpeg", "path", path)
					break
				}
			}
		}
	}
	
	if e.ffmpegBinary == "" {
		return fmt.Errorf("ffmpeg not found. Install with your package manager (apt/brew/etc)")
	}
	
	return nil
}

// Synthesize converts text to speech audio using Google TTS
func (e *GTTSEngine) Synthesize(text string, speed float64) ([]byte, error) {
	e.mu.RLock()
	if !e.initialized {
		e.mu.RUnlock()
		return nil, fmt.Errorf("engine not initialized")
	}
	e.mu.RUnlock()
	
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}
	
	// Start metrics tracking (commented out for now - needs refactoring)
	// metrics := tts.StartSynthesis("gtts", text)
	
	// Create temporary files for the pipeline
	timestamp := time.Now().UnixNano()
	mp3File := filepath.Join(e.tempDir, fmt.Sprintf("gtts_%d.mp3", timestamp))
	defer os.Remove(mp3File) // Clean up temp file
	
	// Step 1: Generate MP3 using gtts-cli
	log.Debug("GTTS: Generating MP3", "textLen", len(text))
	
	// Build gtts-cli command
	// gtts-cli "text" --output file.mp3 --lang en
	args := []string{
		text,
		"--output", mp3File,
		"--lang", e.language,
	}
	
	cmd := exec.Command(e.gttsBinary, args...)
	
	// Capture output for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	// Run with timeout protection using TimeoutExecutor
	timeoutConfig := tts.DefaultTimeoutConfig()
	timeoutConfig.Timeout = 10 * time.Second
	executor := tts.NewTimeoutExecutor(timeoutConfig)
	
	if err := executor.RunWithTimeout(cmd); err != nil {
		log.Error("GTTS: Failed to generate MP3", 
			"error", err,
			"stderr", stderr.String())
		if strings.Contains(err.Error(), "timed out") {
			return nil, fmt.Errorf("gtts-cli timed out (network issue?)")
		}
		return nil, fmt.Errorf("gtts-cli failed: %w\nstderr: %s", err, stderr.String())
	}
	
	// Verify MP3 was created
	if _, err := os.Stat(mp3File); err != nil {
		return nil, fmt.Errorf("MP3 file not created: %w", err)
	}
	
	// Step 2: Convert MP3 to PCM using ffmpeg
	log.Debug("GTTS: Converting MP3 to PCM", "speed", speed)
	
	// Build ffmpeg command to convert MP3 to raw PCM
	// We need: 16-bit, mono, 22050Hz PCM data
	ffmpegArgs := []string{
		"-i", mp3File,           // Input MP3 file
		"-f", "s16le",           // 16-bit signed little-endian
		"-ar", "22050",          // Sample rate 22050Hz
		"-ac", "1",              // Mono (1 channel)
	}
	
	// Add speed control using atempo filter if needed
	if speed != 1.0 {
		// ffmpeg atempo filter accepts values between 0.5 and 2.0
		// For values outside this range, we need to chain multiple filters
		tempoValue := speed
		
		if tempoValue < 0.5 {
			tempoValue = 0.5
		} else if tempoValue > 2.0 {
			tempoValue = 2.0
		}
		
		ffmpegArgs = append(ffmpegArgs, "-filter:a", fmt.Sprintf("atempo=%.2f", tempoValue))
	}
	
	// Output to pipe (stdout)
	ffmpegArgs = append(ffmpegArgs, "-")
	
	ffmpegCmd := exec.Command(e.ffmpegBinary, ffmpegArgs...)
	
	// Capture the PCM output
	var pcmBuffer bytes.Buffer
	var ffmpegStderr bytes.Buffer
	ffmpegCmd.Stdout = &pcmBuffer
	ffmpegCmd.Stderr = &ffmpegStderr
	
	// Run ffmpeg with timeout protection
	if err := executor.RunWithTimeout(ffmpegCmd); err != nil {
		log.Error("GTTS: Failed to convert MP3 to PCM",
			"error", err,
			"stderr", ffmpegStderr.String())
		if strings.Contains(err.Error(), "timed out") {
			return nil, fmt.Errorf("ffmpeg conversion timed out")
		}
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}
	
	pcmData := pcmBuffer.Bytes()
	if len(pcmData) == 0 {
		// metrics.EndSynthesis(0, false, fmt.Errorf("no audio data generated"))
		return nil, fmt.Errorf("no audio data generated")
	}
	
	// End metrics tracking (commented out for now - needs refactoring)
	// metrics.EndSynthesis(len(pcmData), false, nil)
	
	log.Debug("GTTS: Synthesis complete", "pcmSize", len(pcmData))
	return pcmData, nil
}

// SetSpeed sets the playback speed
func (e *GTTSEngine) SetSpeed(speed float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if speed < 0.25 || speed > 4.0 {
		return fmt.Errorf("speed must be between 0.25 and 4.0")
	}
	
	e.speed = speed
	return nil
}

// GetSpeed returns the current playback speed
func (e *GTTSEngine) GetSpeed() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.speed
}

// SetLanguage sets the language for TTS
func (e *GTTSEngine) SetLanguage(lang string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Common language codes supported by gTTS
	supportedLangs := map[string]bool{
		"en": true, "es": true, "fr": true, "de": true,
		"it": true, "pt": true, "ru": true, "ja": true,
		"ko": true, "zh": true, "ar": true, "hi": true,
		"nl": true, "pl": true, "tr": true, "sv": true,
	}
	
	if !supportedLangs[lang] {
		return fmt.Errorf("unsupported language: %s", lang)
	}
	
	e.language = lang
	return nil
}

// GetName returns the engine name
func (e *GTTSEngine) GetName() string {
	return "Google TTS"
}

// Validate checks if the engine is properly configured
func (e *GTTSEngine) Validate() error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if !e.initialized {
		return fmt.Errorf("engine not initialized")
	}
	
	if e.gttsBinary == "" {
		return fmt.Errorf("gtts-cli not found")
	}
	
	if e.ffmpegBinary == "" {
		return fmt.Errorf("ffmpeg not found")
	}
	
	// Test internet connectivity (gtts requires internet)
	cmd := exec.Command("ping", "-c", "1", "-W", "1", "google.com")
	if err := cmd.Run(); err != nil {
		log.Warn("GTTS: No internet connectivity detected")
		// Don't fail validation, just warn
	}
	
	return nil
}

// IsAvailable checks if the engine is available and ready to use
func (e *GTTSEngine) IsAvailable() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if !e.initialized {
		return false
	}
	
	// Check that binaries still exist
	if _, err := os.Stat(e.gttsBinary); err != nil {
		return false
	}
	if _, err := os.Stat(e.ffmpegBinary); err != nil {
		return false
	}
	
	return true
}

// Cleanup removes temporary files
func (e *GTTSEngine) Cleanup() error {
	if e.tempDir != "" {
		return os.RemoveAll(e.tempDir)
	}
	return nil
}