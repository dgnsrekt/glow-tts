// Package mock provides a mock TTS engine for testing.
package mock

import (
	"fmt"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// MockEngine implements the TTS engine interface for testing.
type MockEngine struct {
	// Configuration
	config      tts.EngineConfig
	delay       time.Duration // Simulated processing delay
	activeVoice tts.Voice

	// Control for testing
	shouldFail   bool
	failureError error

	// State
	available bool
	callCount int
}

// New creates a new mock TTS engine.
func New() *MockEngine {
	return &MockEngine{
		delay:     100 * time.Millisecond, // Default 100ms delay
		available: true,
		activeVoice: tts.Voice{
			ID:       "mock-voice-1",
			Name:     "Mock Voice",
			Language: "en-US",
			Gender:   "neutral",
		},
	}
}

// Initialize prepares the mock engine.
func (e *MockEngine) Initialize(config tts.EngineConfig) error {
	e.config = config
	return nil
}

// GenerateAudio simulates audio generation.
func (e *MockEngine) GenerateAudio(text string) (*tts.Audio, error) {
	e.callCount++

	// Simulate failure if configured
	if e.shouldFail {
		return nil, e.failureError
	}

	// Simulate processing delay
	time.Sleep(e.delay)

	// Generate mock audio (silence)
	duration := e.estimateDuration(text)
	sampleRate := 22050
	samples := int(duration.Seconds() * float64(sampleRate))
	audioData := make([]byte, samples*2) // 16-bit audio

	return &tts.Audio{
		Data:       audioData,
		Format:     tts.FormatPCM16,
		SampleRate: sampleRate,
		Channels:   1,
		Duration:   duration,
	}, nil
}

// GenerateAudioStream simulates streaming audio generation.
func (e *MockEngine) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	e.callCount++

	if e.shouldFail {
		return nil, e.failureError
	}

	ch := make(chan tts.AudioChunk)

	// Simulate streaming in background
	go func() {
		defer close(ch)

		// Simulate processing delay
		time.Sleep(e.delay)

		// Generate mock audio in chunks
		duration := e.estimateDuration(text)
		sampleRate := 22050
		totalSamples := int(duration.Seconds() * float64(sampleRate))
		chunkSize := 4096 // 4KB chunks

		position := 0
		for position < totalSamples*2 { // *2 for 16-bit audio
			remaining := totalSamples*2 - position
			size := chunkSize
			if remaining < chunkSize {
				size = remaining
			}

			chunk := tts.AudioChunk{
				Data:     make([]byte, size),
				Position: position,
				Final:    remaining <= chunkSize,
			}

			ch <- chunk
			position += size

			// Small delay between chunks
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return ch, nil
}

// GetVoices returns available mock voices.
func (e *MockEngine) GetVoices() []tts.Voice {
	return []tts.Voice{
		{
			ID:       "mock-voice-1",
			Name:     "Mock Voice 1",
			Language: "en-US",
			Gender:   "neutral",
		},
		{
			ID:       "mock-voice-2",
			Name:     "Mock Voice 2",
			Language: "en-GB",
			Gender:   "female",
		},
		{
			ID:       "mock-voice-3",
			Name:     "Mock Voice 3",
			Language: "en-US",
			Gender:   "male",
		},
	}
}

// SetVoice sets the active voice.
func (e *MockEngine) SetVoice(voice tts.Voice) error {
	// Validate voice exists
	voices := e.GetVoices()
	found := false
	for _, v := range voices {
		if v.ID == voice.ID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("voice not found: %s", voice.ID)
	}

	e.activeVoice = voice
	return nil
}

// GetCapabilities returns the mock engine's capabilities.
func (e *MockEngine) GetCapabilities() tts.Capabilities {
	return tts.Capabilities{
		SupportsStreaming: true,
		SupportedFormats:  []string{"pcm16", "float32"},
		MaxTextLength:     10000,
		RequiresNetwork:   false,
	}
}

// IsAvailable returns the mock availability state.
func (e *MockEngine) IsAvailable() bool {
	return e.available
}

// Shutdown simulates engine shutdown.
func (e *MockEngine) Shutdown() error {
	e.available = false
	return nil
}

// Test control methods

// SetDelay sets the simulated processing delay.
func (e *MockEngine) SetDelay(delay time.Duration) {
	e.delay = delay
}

// SetFailure configures the engine to fail with the given error.
func (e *MockEngine) SetFailure(err error) {
	e.shouldFail = true
	e.failureError = err
}

// ClearFailure resets the engine to normal operation.
func (e *MockEngine) ClearFailure() {
	e.shouldFail = false
	e.failureError = nil
}

// GetCallCount returns the number of GenerateAudio calls.
func (e *MockEngine) GetCallCount() int {
	return e.callCount
}

// estimateDuration estimates speaking duration for text.
func (e *MockEngine) estimateDuration(text string) time.Duration {
	// Estimate ~150 words per minute
	words := len(text) / 5 // Rough estimate: 5 chars per word
	if words < 1 {
		words = 1
	}
	seconds := float64(words) * 60.0 / 150.0
	return time.Duration(seconds * float64(time.Second))
}