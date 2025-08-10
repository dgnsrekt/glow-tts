package engines

import (
	"errors"
	"testing"
	
	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
)

// TestFallbackEngine tests the fallback mechanism
func TestFallbackEngine(t *testing.T) {
	// Create a primary mock that always fails
	primaryMock := mock.New()
	primaryMock.SetFailure(errors.New("primary engine failure"))
	
	// Create a fallback mock that always works
	fallbackMock := mock.New()
	
	// Create fallback engine with max 2 failures before switching
	engine := NewFallbackEngine(primaryMock, fallbackMock, 2)
	
	// Initialize
	config := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 1.0,
	}
	
	err := engine.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// First attempt should fail (primary fails, count = 1)
	_, err = engine.GenerateAudio("test 1")
	if err == nil {
		t.Error("Expected first attempt to fail")
	}
	
	// Second attempt should succeed (primary fails, count = 2, switches to fallback)
	audio, err := engine.GenerateAudio("test 2")
	if err != nil {
		t.Errorf("Expected second attempt to succeed with fallback: %v", err)
	}
	if audio == nil {
		t.Error("Expected audio to be generated")
	}
	
	// Status should indicate using fallback
	status := engine.GetStatus()
	if status != "Using fallback engine (primary failed 2 times)" {
		t.Errorf("Unexpected status: %s", status)
	}
	
	// Subsequent calls should use fallback
	audio, err = engine.GenerateAudio("test 3")
	if err != nil {
		t.Errorf("Expected subsequent calls to use fallback: %v", err)
	}
	if audio == nil {
		t.Error("Expected audio to be generated")
	}
}