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
	
	if tts.speedController == nil {
		t.Error("Expected speed controller to be initialized")
	} else if tts.speedController.GetSpeed() != 1.0 {
		t.Errorf("Expected default speed 1.0, got %f", tts.speedController.GetSpeed())
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
	if tts.speedController == nil {
		t.Error("Expected speed controller to be initialized")
	} else if tts.speedController.GetSpeed() != 1.0 {
		t.Errorf("Expected initial speed 1.0, got %f", tts.speedController.GetSpeed())
	}
	
	// Test speed controller exists and can change speeds
	if tts.speedController != nil {
		// Test increase speed
		newSpeed, err := tts.speedController.NextSpeed()
		if err != nil {
			t.Errorf("Failed to increase speed: %v", err)
		} else if newSpeed <= 1.0 {
			t.Errorf("Expected speed to increase from 1.0, got %f", newSpeed)
		}
		
		// Test decrease speed back
		prevSpeed, err := tts.speedController.PreviousSpeed()
		if err != nil {
			t.Errorf("Failed to decrease speed: %v", err)
		} else if prevSpeed != 1.0 {
			t.Errorf("Expected speed to return to 1.0, got %f", prevSpeed)
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
	
	// Log the actual status for debugging
	t.Logf("Actual status: %q", status)
	
	// Should contain engine name (could be styled differently)
	if !contains(status, "PIPER") && !contains(status, "Piper") && !contains(status, "piper") {
		t.Error("Expected status to contain engine name")
	}
	
	// The status might be empty when not initialized yet
	// Just check that we get a status string
	if len(status) < 5 {
		t.Error("Expected longer status string")
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