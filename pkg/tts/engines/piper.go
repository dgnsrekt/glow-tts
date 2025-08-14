package engines

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Audio format constants for Piper
const (
	// PiperSampleRate is the default sample rate for Piper TTS
	PiperSampleRate = 22050
	// PiperBitsPerSample is the bit depth for PCM audio
	PiperBitsPerSample = 16
	// PiperChannels is the number of audio channels (mono)
	PiperChannels = 1
	// DefaultSpeed is the normal speaking speed
	DefaultSpeed = 1.0
	// MinSpeed is the minimum speaking speed
	MinSpeed = 0.5
	// MaxSpeed is the maximum speaking speed
	MaxSpeed = 2.0
)

// PiperError represents Piper-specific errors
type PiperError struct {
	Type    string
	Message string
	Cause   error
}

func (e *PiperError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("piper %s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("piper %s: %s", e.Type, e.Message)
}

func (e *PiperError) Unwrap() error {
	return e.Cause
}

// PiperEngine implements the TTSEngine interface for Piper TTS
type PiperEngine struct {
	// binaryPath is the path to the piper executable
	binaryPath string
	// modelPath is the path to the ONNX voice model
	modelPath string
	// configPath is the path to the model config JSON (optional)
	configPath string
	// speed is the current speaking speed multiplier
	speed float64
	// voiceName is the name of the selected voice/model
	voiceName string
	// timeout for synthesis operations
	timeout time.Duration
}

// NewPiperEngine creates a new Piper TTS engine instance
func NewPiperEngine() (*PiperEngine, error) {
	engine := &PiperEngine{
		speed:   DefaultSpeed,
		timeout: 30 * time.Second,
	}

	// Try to find the piper binary
	if err := engine.findBinary(); err != nil {
		return nil, err
	}

	// Try to find a default model
	if err := engine.findDefaultModel(); err != nil {
		// Not fatal - user can set model later
		// Just log the warning
	}

	return engine, nil
}

// NewPiperEngineWithModel creates a new Piper engine with a specific model
func NewPiperEngineWithModel(modelPath string) (*PiperEngine, error) {
	engine, err := NewPiperEngine()
	if err != nil {
		return nil, err
	}

	if err := engine.SetModel(modelPath); err != nil {
		return nil, err
	}

	return engine, nil
}

// findBinary locates the piper executable
func (e *PiperEngine) findBinary() error {
	// Check if piper is in PATH
	path, err := exec.LookPath("piper")
	if err == nil {
		e.binaryPath = path
		return nil
	}

	// Check common installation locations
	commonPaths := []string{
		"/usr/local/bin/piper",
		"/usr/bin/piper",
		"/opt/piper/piper",
		filepath.Join(os.Getenv("HOME"), ".local/bin/piper"),
		filepath.Join(os.Getenv("HOME"), "bin/piper"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			e.binaryPath = path
			return nil
		}
	}

	return &PiperError{
		Type:    "dependency",
		Message: "piper binary not found. Please install piper TTS: https://github.com/rhasspy/piper",
		Cause:   err,
	}
}

// findDefaultModel searches for ONNX voice models in standard locations
func (e *PiperEngine) findDefaultModel() error {
	// Common model locations
	modelDirs := []string{
		filepath.Join(os.Getenv("HOME"), ".local/share/piper-voices"),
		"/usr/share/piper-voices",
		"/usr/local/share/piper-voices",
		filepath.Join(os.Getenv("HOME"), ".config/piper/voices"),
		"/opt/piper/voices",
	}

	// Look for any .onnx file
	for _, dir := range modelDirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		// Walk the directory looking for ONNX models
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue walking
			}

			if strings.HasSuffix(path, ".onnx") {
				e.modelPath = path
				// Look for accompanying config file
				configPath := strings.TrimSuffix(path, ".onnx") + ".onnx.json"
				if _, err := os.Stat(configPath); err == nil {
					e.configPath = configPath
				}
				// Extract voice name from path
				e.voiceName = filepath.Base(strings.TrimSuffix(path, ".onnx"))
				return io.EOF // Stop walking, we found a model
			}
			return nil
		})

		if err == io.EOF {
			return nil // Found a model
		}
	}

	return &PiperError{
		Type: "model",
		Message: `No ONNX voice models found. Please download a model from:
https://github.com/rhasspy/piper/releases
and place it in ~/.local/share/piper-voices/`,
	}
}

// SetModel sets the voice model to use
func (e *PiperEngine) SetModel(modelPath string) error {
	// Validate the model file exists
	if _, err := os.Stat(modelPath); err != nil {
		return &PiperError{
			Type:    "model",
			Message: fmt.Sprintf("model file not found: %s", modelPath),
			Cause:   err,
		}
	}

	if !strings.HasSuffix(modelPath, ".onnx") {
		return &PiperError{
			Type:    "model",
			Message: "model file must be an ONNX file (.onnx extension)",
		}
	}

	e.modelPath = modelPath

	// Check for config file
	configPath := strings.TrimSuffix(modelPath, ".onnx") + ".onnx.json"
	if _, err := os.Stat(configPath); err == nil {
		e.configPath = configPath
	} else {
		e.configPath = ""
	}

	// Extract voice name
	e.voiceName = filepath.Base(strings.TrimSuffix(modelPath, ".onnx"))

	return nil
}

// Synthesize converts text to speech audio data
func (e *PiperEngine) Synthesize(text string, speed float64) ([]byte, error) {
	if text == "" {
		return []byte{}, nil
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	// Apply speed if provided, otherwise use engine's default
	if speed <= 0 {
		speed = e.speed
	}

	// Build command arguments
	args := []string{
		"--model", e.modelPath,
		"--output-raw",
	}

	// Add config file if available
	if e.configPath != "" {
		args = append(args, "--config", e.configPath)
	}

	// Add length scale for speed control (inverse relationship)
	// Speed 2.0 = length_scale 0.5 (faster)
	// Speed 0.5 = length_scale 2.0 (slower)
	if speed != 1.0 {
		lengthScale := 1.0 / speed
		args = append(args, "--length-scale", fmt.Sprintf("%.2f", lengthScale))
	}

	// Create command with context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.binaryPath, args...)

	// Set stdin BEFORE starting the process (prevents race condition)
	cmd.Stdin = strings.NewReader(text)

	// Capture stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Set up stdout pipe for audio data
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, &PiperError{
			Type:    "process",
			Message: "failed to create stdout pipe",
			Cause:   err,
		}
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, &PiperError{
			Type:    "process",
			Message: "failed to start piper process",
			Cause:   err,
		}
	}

	// Read audio data from stdout
	var audioData bytes.Buffer
	if _, err := io.Copy(&audioData, stdout); err != nil {
		return nil, &PiperError{
			Type:    "synthesis",
			Message: "failed to read audio data",
			Cause:   err,
		}
	}

	// Wait for process to complete
	if err := cmd.Wait(); err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &PiperError{
				Type:    "timeout",
				Message: fmt.Sprintf("synthesis timed out after %v", e.timeout),
				Cause:   err,
			}
		}

		// Check stderr for error details
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return nil, &PiperError{
				Type:    "synthesis",
				Message: fmt.Sprintf("piper error: %s", strings.TrimSpace(stderrMsg)),
				Cause:   err,
			}
		}

		return nil, &PiperError{
			Type:    "synthesis",
			Message: "synthesis failed",
			Cause:   err,
		}
	}

	audioBytes := audioData.Bytes()

	// Validate we got audio data
	if len(audioBytes) == 0 {
		return nil, &PiperError{
			Type:    "synthesis",
			Message: "no audio data generated",
		}
	}

	// Validate PCM format (should be even number of bytes for 16-bit samples)
	if len(audioBytes)%2 != 0 {
		// Pad with zero byte if odd
		audioBytes = append(audioBytes, 0)
	}

	return audioBytes, nil
}

// SetSpeed sets the speaking speed
func (e *PiperEngine) SetSpeed(speed float64) error {
	if speed < MinSpeed || speed > MaxSpeed {
		return &PiperError{
			Type:    "parameter",
			Message: fmt.Sprintf("speed must be between %.1f and %.1f", MinSpeed, MaxSpeed),
		}
	}
	e.speed = speed
	return nil
}

// Validate checks if the engine is properly configured
func (e *PiperEngine) Validate() error {
	// Check binary
	if e.binaryPath == "" {
		return &PiperError{
			Type:    "dependency",
			Message: "piper binary not configured",
		}
	}

	if _, err := os.Stat(e.binaryPath); err != nil {
		return &PiperError{
			Type:    "dependency",
			Message: fmt.Sprintf("piper binary not found at %s", e.binaryPath),
			Cause:   err,
		}
	}

	// Check model
	if e.modelPath == "" {
		return &PiperError{
			Type:    "model",
			Message: "no voice model configured. Use SetModel() to select a model",
		}
	}

	if _, err := os.Stat(e.modelPath); err != nil {
		return &PiperError{
			Type:    "model",
			Message: fmt.Sprintf("model file not found at %s", e.modelPath),
			Cause:   err,
		}
	}

	return nil
}

// GetName returns the engine name
func (e *PiperEngine) GetName() string {
	if e.voiceName != "" {
		return fmt.Sprintf("Piper (%s)", e.voiceName)
	}
	return "Piper TTS"
}

// IsAvailable checks if the engine can be used
func (e *PiperEngine) IsAvailable() bool {
	return e.Validate() == nil
}

// GetVoice returns the current voice/model name
func (e *PiperEngine) GetVoice() string {
	return e.voiceName
}

// GetModelPath returns the current model path
func (e *PiperEngine) GetModelPath() string {
	return e.modelPath
}

// SetTimeout sets the synthesis timeout duration
func (e *PiperEngine) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}

// GetInfo returns information about the engine configuration
func (e *PiperEngine) GetInfo() map[string]string {
	info := map[string]string{
		"engine":     "piper",
		"binary":     e.binaryPath,
		"model":      e.modelPath,
		"voice":      e.voiceName,
		"speed":      fmt.Sprintf("%.1f", e.speed),
		"sampleRate": fmt.Sprintf("%d", PiperSampleRate),
		"format":     "PCM 16-bit mono",
	}

	if e.configPath != "" {
		info["config"] = e.configPath
	}

	return info
}