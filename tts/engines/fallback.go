// Package engines provides text-to-speech engine implementations.
package engines

import (
	"fmt"
	"log"
	"sync"
	
	"github.com/charmbracelet/glow/v2/tts"
)

// FallbackEngine wraps a primary engine with automatic fallback to a secondary engine
// when the primary fails consistently.
type FallbackEngine struct {
	primary     tts.Engine
	fallback    tts.Engine
	failures    int
	maxFailures int
	usingFallback bool
	mu          sync.RWMutex
}

// NewFallbackEngine creates a new engine with automatic fallback capability.
func NewFallbackEngine(primary, fallback tts.Engine, maxFailures int) *FallbackEngine {
	return &FallbackEngine{
		primary:     primary,
		fallback:    fallback,
		maxFailures: maxFailures,
		failures:    0,
		usingFallback: false,
	}
}

// Initialize initializes both engines.
func (f *FallbackEngine) Initialize(config tts.EngineConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Try to initialize primary
	primaryErr := f.primary.Initialize(config)
	if primaryErr != nil {
		log.Printf("[WARNING TTS] Primary engine initialization failed: %v", primaryErr)
	}
	
	// Always initialize fallback
	fallbackErr := f.fallback.Initialize(config)
	if fallbackErr != nil {
		log.Printf("[ERROR TTS] Fallback engine initialization failed: %v", fallbackErr)
		// If both fail, return primary error as it's more important
		if primaryErr != nil {
			return fmt.Errorf("both engines failed: primary=%v, fallback=%v", primaryErr, fallbackErr)
		}
	}
	
	// If primary failed but fallback succeeded, switch to fallback
	if primaryErr != nil && fallbackErr == nil {
		f.usingFallback = true
		log.Printf("[WARNING TTS] Using fallback engine due to primary initialization failure")
	}
	
	return nil
}

// GenerateAudio generates audio using the active engine, with automatic fallback.
func (f *FallbackEngine) GenerateAudio(text string) (*tts.Audio, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// If already using fallback, use it directly
	if f.usingFallback {
		log.Printf("[DEBUG TTS] Using fallback engine for audio generation")
		return f.fallback.GenerateAudio(text)
	}
	
	// Try primary engine
	audio, err := f.primary.GenerateAudio(text)
	if err == nil {
		// Success - reset failure counter
		if f.failures > 0 {
			log.Printf("[INFO TTS] Primary engine recovered after %d failures", f.failures)
			f.failures = 0
		}
		return audio, nil
	}
	
	// Primary failed
	f.failures++
	log.Printf("[WARNING TTS] Primary engine failed (attempt %d/%d): %v", f.failures, f.maxFailures, err)
	
	// Check if we should switch to fallback
	if f.failures >= f.maxFailures {
		log.Printf("[WARNING TTS] Primary engine failed %d times, switching to fallback engine", f.failures)
		f.usingFallback = true
		
		// Try fallback
		audio, err := f.fallback.GenerateAudio(text)
		if err != nil {
			return nil, fmt.Errorf("both engines failed: %v", err)
		}
		return audio, nil
	}
	
	// Haven't reached max failures yet, return error
	return nil, err
}

// IsAvailable checks if either engine is available.
func (f *FallbackEngine) IsAvailable() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.usingFallback {
		return f.fallback.IsAvailable()
	}
	
	// Check primary first
	if f.primary.IsAvailable() {
		return true
	}
	
	// Primary not available, check fallback
	if f.fallback.IsAvailable() {
		// Switch to fallback
		f.mu.RUnlock()
		f.mu.Lock()
		f.usingFallback = true
		log.Printf("[WARNING TTS] Primary engine not available, switching to fallback")
		f.mu.Unlock()
		f.mu.RLock()
		return true
	}
	
	return false
}

// GetVoices returns voices from the active engine.
func (f *FallbackEngine) GetVoices() []tts.Voice {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.usingFallback {
		return f.fallback.GetVoices()
	}
	return f.primary.GetVoices()
}

// SetVoice sets the voice on both engines.
func (f *FallbackEngine) SetVoice(voice tts.Voice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	// Set on both engines
	primaryErr := f.primary.SetVoice(voice)
	fallbackErr := f.fallback.SetVoice(voice)
	
	// Return primary error if using primary, fallback error if using fallback
	if f.usingFallback {
		return fallbackErr
	}
	return primaryErr
}

// GetCapabilities returns capabilities from the active engine.
func (f *FallbackEngine) GetCapabilities() tts.Capabilities {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.usingFallback {
		return f.fallback.GetCapabilities()
	}
	return f.primary.GetCapabilities()
}

// GenerateAudioStream generates audio stream using the active engine.
func (f *FallbackEngine) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.usingFallback {
		return f.fallback.GenerateAudioStream(text)
	}
	
	// Try primary first
	stream, err := f.primary.GenerateAudioStream(text)
	if err != nil {
		// Log and try fallback
		log.Printf("[WARNING TTS] Primary engine stream failed: %v, trying fallback", err)
		return f.fallback.GenerateAudioStream(text)
	}
	return stream, nil
}

// Shutdown shuts down both engines.
func (f *FallbackEngine) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	var errors []error
	
	if err := f.primary.Shutdown(); err != nil {
		errors = append(errors, fmt.Errorf("primary shutdown: %w", err))
	}
	
	if err := f.fallback.Shutdown(); err != nil {
		errors = append(errors, fmt.Errorf("fallback shutdown: %w", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}
	
	return nil
}

// Reset attempts to reset to primary engine.
func (f *FallbackEngine) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.failures = 0
	f.usingFallback = false
	log.Printf("[INFO TTS] Reset to primary engine")
}

// GetStatus returns the current engine status.
func (f *FallbackEngine) GetStatus() string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	if f.usingFallback {
		return fmt.Sprintf("Using fallback engine (primary failed %d times)", f.failures)
	}
	return fmt.Sprintf("Using primary engine (failures: %d/%d)", f.failures, f.maxFailures)
}