// Package piper provides the Piper TTS engine integration.
package piper

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"time"
	
	"github.com/charmbracelet/glow/v2/tts"
)

// SimpleEngine uses a fresh process for each request.
type SimpleEngine struct {
	config Config
}

// NewSimpleEngine creates a simple Piper engine.
func NewSimpleEngine(config Config) *SimpleEngine {
	return &SimpleEngine{
		config: config,
	}
}

// Initialize prepares the engine for use.
func (e *SimpleEngine) Initialize(config tts.EngineConfig) error {
	// Nothing to initialize for simple engine
	return nil
}

// GenerateAudio converts text to audio using a fresh Piper process.
func (e *SimpleEngine) GenerateAudio(text string) (*tts.Audio, error) {
	log.Printf("[DEBUG SimpleEngine] GenerateAudio called with text: %.50s...", text)
	
	// Build command
	args := []string{
		"--model", e.config.ModelPath,
		"--output-raw",
	}
	
	log.Printf("[DEBUG SimpleEngine] Running: %s %v", e.config.BinaryPath, args)
	cmd := exec.Command(e.config.BinaryPath, args...)
	
	// Set up stdin
	cmd.Stdin = bytes.NewBufferString(text + "\n")
	
	// Run and get output
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("piper failed: %w", err)
	}
	
	if len(output) == 0 {
		return nil, fmt.Errorf("no audio generated")
	}
	
	log.Printf("[DEBUG SimpleEngine] Generated %d bytes of audio", len(output))
	
	// Calculate duration
	samples := len(output) / 2 // 16-bit samples
	duration := time.Duration(float64(samples) / float64(e.config.SampleRate) * float64(time.Second))
	
	return &tts.Audio{
		Data:       output,
		Format:     tts.FormatPCM16,
		SampleRate: e.config.SampleRate,
		Channels:   1,
		Duration:   duration,
	}, nil
}

// Shutdown cleans up resources.
func (e *SimpleEngine) Shutdown() error {
	return nil
}

// IsAvailable checks if the engine is available.
func (e *SimpleEngine) IsAvailable() bool {
	// Check if binary exists
	cmd := exec.Command(e.config.BinaryPath, "--version")
	return cmd.Run() == nil
}

// GetVoices returns available voices.
func (e *SimpleEngine) GetVoices() []tts.Voice {
	return []tts.Voice{
		{ID: "default", Name: "Default", Language: "en-US"},
	}
}

// SetVoice sets the current voice.
func (e *SimpleEngine) SetVoice(voice tts.Voice) error {
	return nil
}

// GetCapabilities returns engine capabilities.
func (e *SimpleEngine) GetCapabilities() tts.Capabilities {
	return tts.Capabilities{
		SupportsStreaming: false,
		SupportedFormats:  []string{"pcm16"},
		MaxTextLength:     10000,
		RequiresNetwork:   false,
	}
}

// GenerateAudioStream is not supported.
func (e *SimpleEngine) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	return nil, fmt.Errorf("streaming not supported")
}