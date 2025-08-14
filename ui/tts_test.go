package ui

import (
	"testing"
)

func TestTTSState(t *testing.T) {
	// Test TTS state creation
	tts := NewTTSState("piper")
	
	if tts == nil {
		t.Fatal("Expected TTS state to be created")
	}
	
	if !tts.IsEnabled() {
		t.Error("Expected TTS to be enabled")
	}
	
	if tts.engine != "piper" {
		t.Errorf("Expected engine 'piper', got '%s'", tts.engine)
	}
	
	if tts.currentSpeed != 1.0 {
		t.Errorf("Expected default speed 1.0, got %f", tts.currentSpeed)
	}
	
	if !tts.isStopped {
		t.Error("Expected initial state to be stopped")
	}
}

func TestTTSStateDisabled(t *testing.T) {
	// Test nil TTS state
	var tts *TTSState
	
	if tts.IsEnabled() {
		t.Error("Expected nil TTS state to be disabled")
	}
	
	// Test empty engine
	tts = NewTTSState("")
	if tts.IsEnabled() {
		t.Error("Expected empty engine TTS state to be disabled")
	}
}

func TestTTSSpeedControl(t *testing.T) {
	tts := NewTTSState("gtts")
	
	// Test initial speed
	if tts.currentSpeed != 1.0 {
		t.Errorf("Expected initial speed 1.0, got %f", tts.currentSpeed)
	}
	
	// Test speed steps
	expectedSteps := []float64{0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0}
	if len(tts.speedSteps) != len(expectedSteps) {
		t.Errorf("Expected %d speed steps, got %d", len(expectedSteps), len(tts.speedSteps))
	}
	
	for i, expected := range expectedSteps {
		if tts.speedSteps[i] != expected {
			t.Errorf("Speed step %d: expected %f, got %f", i, expected, tts.speedSteps[i])
		}
	}
}

func TestTTSStatusRender(t *testing.T) {
	tts := NewTTSState("piper")
	
	// Test status rendering
	status := tts.RenderStatus()
	if status == "" {
		t.Error("Expected non-empty status")
	}
	
	// Should contain engine name
	if !contains(status, "PIPER") {
		t.Error("Expected status to contain engine name")
	}
	
	// Should show stopped state
	if !contains(status, "■") {
		t.Error("Expected status to show stopped icon")
	}
	
	// Should show speed
	if !contains(status, "1.0x") {
		t.Error("Expected status to show speed")
	}
}

func TestTTSKeyboardHelp(t *testing.T) {
	tts := NewTTSState("gtts")
	
	help := tts.GetKeyboardHelp()
	if help == "" {
		t.Error("Expected non-empty help text")
	}
	
	// Check for key shortcuts
	expectedKeys := []string{"Space", "←/→", "+/-", "S"}
	for _, key := range expectedKeys {
		if !contains(help, key) {
			t.Errorf("Expected help to contain '%s'", key)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}