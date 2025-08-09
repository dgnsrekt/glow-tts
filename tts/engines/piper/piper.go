// Package piper implements the Piper TTS engine integration.
package piper

import (
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/charmbracelet/glow/v2/tts"
)

// PiperEngine implements the TTS engine interface using Piper.
type PiperEngine struct {
	// Process management
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	// Configuration
	binary string
	model  string
	config tts.EngineConfig

	// State
	mu      sync.Mutex
	running bool

	// Error channel
	errorCh chan error
}

// New creates a new Piper TTS engine.
func New(binary, model string) (*PiperEngine, error) {
	engine := &PiperEngine{
		binary:  binary,
		model:   model,
		errorCh: make(chan error, 1),
	}

	// TODO: Start Piper process
	return engine, nil
}

// Initialize prepares the engine for use.
func (e *PiperEngine) Initialize(config tts.EngineConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config = config
	// TODO: Start Piper process if not running
	return fmt.Errorf("not yet implemented")
}

// GenerateAudio converts text to audio using Piper.
func (e *PiperEngine) GenerateAudio(text string) (*tts.Audio, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil, fmt.Errorf("engine not running")
	}

	// TODO: Send text to Piper and read audio response
	return nil, fmt.Errorf("not yet implemented")
}

// GenerateAudioStream converts text to audio using streaming.
func (e *PiperEngine) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil, fmt.Errorf("engine not running")
	}

	// TODO: Implement streaming audio generation
	return nil, fmt.Errorf("not yet implemented")
}

// GetVoices returns available Piper voices.
func (e *PiperEngine) GetVoices() []tts.Voice {
	// TODO: Query available Piper models
	return []tts.Voice{
		{
			ID:       "en_US-lessac-medium",
			Name:     "Lessac (US English)",
			Language: "en-US",
			Gender:   "neutral",
		},
	}
}

// SetVoice sets the active Piper voice/model.
func (e *PiperEngine) SetVoice(voice tts.Voice) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// TODO: Validate and switch Piper model
	e.model = voice.ID
	return nil
}

// GetCapabilities returns Piper's capabilities.
func (e *PiperEngine) GetCapabilities() tts.Capabilities {
	return tts.Capabilities{
		SupportsStreaming: false, // Piper doesn't stream natively
		SupportedFormats:  []string{"pcm16", "wav"},
		MaxTextLength:     50000, // Reasonable limit
		RequiresNetwork:   false, // Piper runs locally
	}
}

// IsAvailable checks if Piper is ready for use.
func (e *PiperEngine) IsAvailable() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// Shutdown stops the Piper process and cleans up resources.
func (e *PiperEngine) Shutdown() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	// TODO: Stop Piper process
	e.running = false
	return nil
}