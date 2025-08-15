package tts

import (
	"os"
	"testing"
)

// setupTestAudioContext sets up an appropriate audio context for testing
// It automatically uses mock context in CI environments
func setupTestAudioContext(t *testing.T) AudioContextInterface {
	t.Helper()
	
	// Reset any existing global context for clean test state
	ResetGlobalAudioContext()
	
	// If we're in CI or explicitly want mock audio, use mock context
	if os.Getenv("CI") == "true" || os.Getenv("MOCK_AUDIO") == "true" {
		t.Log("Using mock audio context for testing")
		ctx, err := NewMockAudioContext()
		if err != nil {
			t.Fatalf("Failed to create mock audio context: %v", err)
		}
		SetGlobalAudioContext(ctx)
		return ctx
	}
	
	// Otherwise try to use real audio, fall back to mock if it fails
	ctx, err := NewAudioContext(AudioContextAuto)
	if err != nil {
		t.Fatalf("Failed to create audio context: %v", err)
	}
	
	SetGlobalAudioContext(ctx)
	return ctx
}

// cleanupTestAudioContext cleans up the audio context after testing
func cleanupTestAudioContext(t *testing.T) {
	t.Helper()
	ResetGlobalAudioContext()
}