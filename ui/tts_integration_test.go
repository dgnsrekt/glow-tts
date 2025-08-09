package ui

import (
	"testing"

	"github.com/charmbracelet/glow/v2/tts"
)

// TestTTSControllerCreation tests TTS controller creation.
func TestTTSControllerCreation(t *testing.T) {
	tc := NewTTSController()
	if tc == nil {
		t.Fatal("Expected non-nil TTS controller")
	}

	if tc.IsEnabled() {
		t.Error("TTS should not be enabled initially")
	}

	if tc.GetCurrentSentence() != -1 {
		t.Error("Current sentence should be -1 initially")
	}
}

// TestTTSMessageHandling tests TTS message handling.
func TestTTSMessageHandling(t *testing.T) {
	tc := NewTTSController()
	tc.enabled = true // Enable TTS for message handling

	// Test SentenceChangedMsg
	msg := tts.SentenceChangedMsg{
		Index:    5,
		Text:     "Test sentence",
		Duration: 0,
		Progress: 0.5,
	}

	handled, cmd := tc.HandleTTSMessage(msg)
	if !handled {
		t.Error("SentenceChangedMsg should be handled")
	}
	if cmd != nil {
		t.Error("SentenceChangedMsg should not return a command")
	}

	// Test PlayingMsg
	playMsg := tts.PlayingMsg{
		Sentence: 2,
		Total:    10,
	}

	handled, _ = tc.HandleTTSMessage(playMsg)
	if !handled {
		t.Error("PlayingMsg should be handled")
	}

	// Test StoppedMsg
	stopMsg := tts.StoppedMsg{
		Reason: "user",
	}

	handled, _ = tc.HandleTTSMessage(stopMsg)
	if !handled {
		t.Error("StoppedMsg should be handled")
	}
}

// TestTTSKeyHandling tests TTS keyboard shortcut handling.
func TestTTSKeyHandling(t *testing.T) {
	tc := NewTTSController()

	// Test toggle key - even without controller, toggle returns a command
	// to indicate enabling/disabling
	cmd := tc.HandleTTSKeyPress("t")
	// This returns nil because controller is nil
	if cmd != nil {
		t.Error("Toggle without controller should return nil")
	}

	// Test space key (play/pause)
	cmd = tc.HandleTTSKeyPress(" ")
	// Without controller, this should return nil
	if cmd != nil {
		t.Error("Space without controller should return nil")
	}

	// Test stop key
	cmd = tc.HandleTTSKeyPress("s")
	// Without controller, this should return nil
	if cmd != nil {
		t.Error("Stop without controller should return nil")
	}
}

// TestTTSStatus tests TTS status display.
func TestTTSStatus(t *testing.T) {
	tc := NewTTSController()

	// Initially disabled, should return empty string
	status := tc.GetTTSStatus()
	if status != "" {
		t.Error("Status should be empty when disabled")
	}

	// Enable TTS and set initializing state in status display
	tc.enabled = true
	if tc.statusDisplay != nil {
		tc.statusDisplay.state = tts.StateInitializing
	}

	// Should show initializing status
	status = tc.GetTTSStatus()
	if status == "" {
		t.Error("Expected non-empty status when initializing")
	}
}

// TestSentenceHighlight tests sentence highlighting.
func TestSentenceHighlight(t *testing.T) {
	tc := NewTTSController()

	content := "This is test content."

	// When disabled, content should be unchanged
	result := tc.ApplySentenceHighlight(content)
	if result != content {
		t.Error("Content should be unchanged when TTS is disabled")
	}

	// Enable TTS
	tc.enabled = true

	// With no current sentence, content should still be unchanged
	result = tc.ApplySentenceHighlight(content)
	if result != content {
		t.Error("Content should be unchanged when no sentence is selected")
	}
}