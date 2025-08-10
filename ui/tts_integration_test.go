package ui

import (
	"testing"
	"github.com/charmbracelet/glow/v2/tts"
)

func TestTTSToggle(t *testing.T) {
	// Create TTS controller
	tc := NewTTSController()
	
	// Initial state should be disabled
	if tc.IsEnabled() {
		t.Error("TTS should be disabled initially")
	}
	
	if status := tc.GetTTSStatus(); status != "" {
		t.Errorf("Expected empty status initially, got: %s", status)
	}
	
	// Simulate pressing 't' to enable
	cmd := tc.HandleTTSKeyPress("t")
	if cmd == nil {
		t.Fatal("Expected command from 't' key press")
	}
	
	// After pressing 't', should be enabled
	if !tc.IsEnabled() {
		t.Error("TTS should be enabled after pressing 't'")
	}
	
	// Status should show something
	status := tc.GetTTSStatus()
	if status == "" {
		t.Error("Expected non-empty status after enabling TTS")
	}
	t.Logf("TTS Status after enabling: %s", status)
	
	// Execute the command to get the message
	msg := cmd()
	if _, ok := msg.(tts.TTSEnabledMsg); !ok {
		t.Errorf("Expected TTSEnabledMsg, got %T", msg)
	}
	
	// Handle the message
	handled, _ := tc.HandleTTSMessage(msg)
	if !handled {
		t.Error("TTSEnabledMsg should be handled")
	}
	
	// Status should still show something
	status = tc.GetTTSStatus()
	if status == "" {
		t.Error("Expected non-empty status after handling message")
	}
	t.Logf("TTS Status after message: %s", status)
}

func TestTTSSpaceKey(t *testing.T) {
	// Create and enable TTS
	tc := NewTTSController()
	
	// Enable TTS
	cmd := tc.HandleTTSKeyPress("t")
	if cmd != nil {
		msg := cmd()
		tc.HandleTTSMessage(msg)
	}
	
	if !tc.IsEnabled() {
		t.Fatal("TTS should be enabled")
	}
	
	// Test space key when enabled
	spaceCmd := tc.HandleTTSKeyPress(" ")
	if spaceCmd == nil {
		t.Error("Expected command from space key when TTS is enabled")
	}
	
	// The command should produce a PlayingMsg
	if spaceCmd != nil {
		msg := spaceCmd()
		t.Logf("Space key produced message type: %T", msg)
	}
}