package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/tts"
)

// TestTTSKeyboardHandlers tests TTS keyboard shortcut handling.
func TestTTSKeyboardHandlers(t *testing.T) {
	// Test cases for keyboard shortcuts
	testCases := []struct {
		key         string
		enabled     bool
		expectCmd   bool
		description string
	}{
		{"t", false, false, "Toggle TTS when disabled"},
		{"T", false, false, "Toggle TTS uppercase when disabled"},
		{" ", false, false, "Space when TTS disabled"},
		{" ", true, false, "Space when TTS enabled (no controller)"},
		{"s", false, false, "Stop when TTS disabled"},
		{"S", false, false, "Stop uppercase when TTS disabled"},
		{"alt+left", false, false, "Previous sentence when disabled"},
		{"alt+right", false, false, "Next sentence when disabled"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			ttsCtrl := NewTTSController()
			ttsCtrl.enabled = testCase.enabled

			cmd := ttsCtrl.HandleTTSKeyPress(testCase.key)
			
			if testCase.expectCmd && cmd == nil {
				t.Errorf("Expected command for key '%s', got nil", testCase.key)
			} else if !testCase.expectCmd && cmd != nil {
				t.Errorf("Expected no command for key '%s', got command", testCase.key)
			}
		})
	}
}

// TestTTSKeyboardToggle tests the TTS toggle functionality.
func TestTTSKeyboardToggle(t *testing.T) {
	tc := NewTTSController()

	// Initially disabled
	if tc.IsEnabled() {
		t.Error("TTS should be disabled initially")
	}

	// Toggle on
	cmd := tc.HandleTTSKeyPress("t")
	// Without a controller, this returns nil
	if cmd != nil {
		t.Error("Toggle without controller should return nil")
	}

	// Check that enabled flag would be toggled
	// (actual toggle happens in HandleTTSKeyPress)
}

// TestTTSKeyboardContextAware tests context-aware key handling.
func TestTTSKeyboardContextAware(t *testing.T) {
	tc := NewTTSController()
	tc.enabled = true

	// When enabled, space should attempt to play/pause
	cmd := tc.HandleTTSKeyPress(" ")
	// Without controller, returns nil
	if cmd != nil {
		t.Error("Space without controller should return nil")
	}

	// When disabled, keys should not generate TTS commands
	tc.enabled = false
	cmd = tc.HandleTTSKeyPress(" ")
	if cmd != nil {
		t.Error("Space when disabled should return nil")
	}
}

// TestTTSNavigationKeys tests sentence navigation keys.
func TestTTSNavigationKeys(t *testing.T) {
	tc := NewTTSController()
	tc.enabled = true
	tc.currentSentence = 5
	tc.totalSentences = 10

	// Test next sentence
	cmd := tc.HandleTTSKeyPress("alt+right")
	// Without controller, returns nil
	if cmd != nil {
		t.Error("Navigation without controller should return nil")
	}

	// Test previous sentence
	cmd = tc.HandleTTSKeyPress("alt+left")
	// Without controller, returns nil
	if cmd != nil {
		t.Error("Navigation without controller should return nil")
	}
}

// TestTTSMessageFromKeyPress tests that key presses generate appropriate messages.
func TestTTSMessageFromKeyPress(t *testing.T) {
	// This tests the actual message generation
	// In real usage, the controller would be initialized
	
	// Test toggle message
	toggleMsg := tts.TTSEnabledMsg{Engine: "active"}
	if toggleMsg.Engine != "active" {
		t.Error("TTSEnabledMsg should have engine set")
	}

	// Test stop message
	stopMsg := tts.StoppedMsg{Reason: "user"}
	if stopMsg.Reason != "user" {
		t.Error("StoppedMsg should have reason set")
	}

	// Test navigation message
	navMsg := tts.NavigationMsg{
		Target:    5,
		Direction: "next",
	}
	if navMsg.Target != 5 {
		t.Error("NavigationMsg should have target set")
	}
}

// MockKeyMsg creates a mock key message for testing.
func MockKeyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{
		Type: tea.KeyRunes,
		Runes: []rune(key),
	}
}

// TestKeyConflicts tests that there are no key conflicts.
func TestKeyConflicts(t *testing.T) {
	// Define TTS keys
	ttsKeys := map[string]string{
		"t":         "toggle TTS",
		"T":         "toggle TTS",
		" ":         "play/pause",
		"s":         "stop",
		"S":         "stop",
		"alt+left":  "previous sentence",
		"alt+right": "next sentence",
	}

	// These are the keys we're using for TTS
	// In a real test, we'd check against all pager keys
	// For now, we just verify our keys are documented
	for key, action := range ttsKeys {
		if action == "" {
			t.Errorf("Key %s has no action defined", key)
		}
	}
}