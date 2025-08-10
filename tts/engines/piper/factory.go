// Package piper provides the Piper TTS engine integration.
package piper

import (
	"log"
	"os"
	"time"
	
	"github.com/charmbracelet/glow/v2/tts"
)

// NewEngine creates a new Piper engine instance.
// This factory function decides whether to use V1, V2, or SimpleEngine based on configuration.
func NewEngine(config Config) (tts.Engine, error) {
	// Check for fresh mode - use SimpleEngine for maximum stability
	envMode := os.Getenv("PIPER_FRESH_MODE")
	log.Printf("[DEBUG Piper Factory] PIPER_FRESH_MODE=%s", envMode)
	
	if envMode == "true" || envMode == "1" {
		log.Printf("[INFO Piper] Using SimpleEngine (fresh mode)")
		return NewSimpleEngine(config), nil
	}
	
	// Check if V2 is explicitly requested or if we're in an environment
	// that needs better stability
	useV2 := false
	
	// Check environment variable
	if v2Env := os.Getenv("PIPER_USE_V2"); v2Env == "true" || v2Env == "1" {
		useV2 = true
	}
	
	// Auto-detect if we should use V2 based on issues
	// If max restarts is set high, use V2 for better stability
	if config.MaxRestarts > 3 {
		useV2 = true
	}
	
	// Default to V2 for better stability (unless already handled above)
	useV2 = true
	
	if useV2 {
		log.Printf("[INFO Piper] Using V2 engine")
		return NewEngineV2(config)
	}
	
	// Fall back to V1 (original implementation)
	return newEngineV1(config)
}

// newEngineV1 creates the original Piper engine.
func newEngineV1(config Config) (tts.Engine, error) {
	// Create the V1 engine directly
	return newEngineOriginal(config)
}

// DefaultConfig returns a default configuration for Piper.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	
	return Config{
		BinaryPath:          findPiperBinary(),
		ModelPath:           "",
		ConfigPath:          "",
		WorkDir:             home,
		OutputRaw:           false,
		SampleRate:          22050,
		StartupTimeout:      10 * time.Second,
		RequestTimeout:      30 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		MaxRestarts:         5, // Higher for V2 to trigger it
		RestartDelay:        time.Second,
	}
}