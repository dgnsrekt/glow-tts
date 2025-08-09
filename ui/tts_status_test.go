package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// TestTTSStatusDisplayCreation tests status display creation.
func TestTTSStatusDisplayCreation(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	if display == nil {
		t.Fatal("Expected non-nil status display")
	}
	
	if display.IsActive() {
		t.Error("Display should not be active initially")
	}
	
	status := display.CompactStatus()
	if status != "" {
		t.Error("Initial compact status should be empty")
	}
}

// TestTTSStatusDisplayUpdate tests updating from state.
func TestTTSStatusDisplayUpdate(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Update with playing state
	state := tts.State{
		CurrentState:   tts.StatePlaying,
		Sentence:       2,
		TotalSentences: 10,
		Position:       5 * time.Second,
		Duration:       30 * time.Second,
	}
	
	display.Update(state)
	
	if !display.IsActive() {
		t.Error("Display should be active when playing")
	}
	
	status := display.CompactStatus()
	if status == "" {
		t.Error("Compact status should not be empty when playing")
	}
	
	// Check that status contains play icon
	if !strings.Contains(status, "▶") {
		t.Error("Playing status should contain play icon")
	}
	
	// Check sentence counter
	if !strings.Contains(status, "3/10") {
		t.Error("Status should show sentence 3 of 10")
	}
}

// TestTTSStatusDisplayMessages tests updating from messages.
func TestTTSStatusDisplayMessages(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Test PlayingMsg
	playMsg := tts.PlayingMsg{
		Sentence: 4,
		Total:    20,
		Duration: 10 * time.Second,
	}
	
	display.UpdateFromMessage(playMsg)
	
	if display.state != tts.StatePlaying {
		t.Error("State should be playing after PlayingMsg")
	}
	
	if display.currentSentence != 4 {
		t.Error("Current sentence should be 4")
	}
	
	// Test PausedMsg
	pauseMsg := tts.PausedMsg{
		Position: 3 * time.Second,
		Sentence: 4,
	}
	
	display.UpdateFromMessage(pauseMsg)
	
	if display.state != tts.StatePaused {
		t.Error("State should be paused after PausedMsg")
	}
	
	status := display.CompactStatus()
	if !strings.Contains(status, "⏸") {
		t.Error("Paused status should contain pause icon")
	}
	
	// Test StoppedMsg
	stopMsg := tts.StoppedMsg{
		Reason: "user",
	}
	
	display.UpdateFromMessage(stopMsg)
	
	if display.state != tts.StateIdle {
		t.Error("State should be idle after StoppedMsg")
	}
	
	if display.currentSentence != -1 {
		t.Error("Current sentence should be -1 after stop")
	}
}

// TestTTSStatusProgressBar tests progress bar rendering.
func TestTTSStatusProgressBar(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Set up state with progress
	display.state = tts.StatePlaying
	display.currentSentence = 5
	display.totalSentences = 10
	display.progress = 0.5
	
	// Test progress bar with sufficient width
	bar := display.ProgressBar(20)
	if bar == "" {
		t.Error("Progress bar should not be empty")
	}
	
	// Should have both filled and empty parts
	if !strings.Contains(bar, "█") {
		t.Error("Progress bar should contain filled blocks")
	}
	
	if !strings.Contains(bar, "░") {
		t.Error("Progress bar should contain empty blocks")
	}
	
	// Test with insufficient width
	bar = display.ProgressBar(5)
	if bar != "" {
		t.Error("Progress bar should be empty for width < 10")
	}
}

// TestTTSStatusDetailedDisplay tests detailed status display.
func TestTTSStatusDetailedDisplay(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Set up state
	display.state = tts.StatePlaying
	display.currentSentence = 7
	display.totalSentences = 15
	display.position = 10 * time.Second
	display.duration = 45 * time.Second
	display.isBuffering = true
	display.bufferCount = 3
	
	// Get detailed status
	detailed := display.DetailedStatus(40)
	
	if detailed == "" {
		t.Error("Detailed status should not be empty")
	}
	
	// Check for expected content
	expectedContent := []string{
		"TTS Status",
		"State:",
		"Sentence: 8 of 15",
		"Position:",
		"Buffering: 3 sentences ready",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(detailed, expected) {
			t.Errorf("Detailed status should contain '%s'", expected)
		}
	}
}

// TestTTSStatusError tests error state display.
func TestTTSStatusError(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Update with error message
	errorMsg := tts.TTSErrorMsg{
		Error:       tts.ErrEngineNotAvailable,
		Recoverable: false,
		Component:   "engine",
		Action:      "initialize",
	}
	
	display.UpdateFromMessage(errorMsg)
	
	if display.state != tts.StateError {
		t.Error("State should be error after TTSErrorMsg")
	}
	
	status := display.CompactStatus()
	if !strings.Contains(status, "✗") {
		t.Error("Error status should contain error icon")
	}
	
	// Check detailed status includes error
	detailed := display.DetailedStatus(40)
	if !strings.Contains(detailed, "Error:") {
		t.Error("Detailed status should show error message")
	}
}

// TestTTSStatusColors tests state colors.
func TestTTSStatusColors(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	testCases := []struct {
		state tts.StateType
		icon  string
	}{
		{tts.StatePlaying, "▶"},
		{tts.StatePaused, "⏸"},
		{tts.StateReady, "■"},
		{tts.StateInitializing, "⟳"},
		{tts.StateError, "✗"},
		{tts.StateStopping, "◼"},
	}
	
	for _, tc := range testCases {
		display.state = tc.state
		icon := display.getStateIcon()
		if icon != tc.icon {
			t.Errorf("State %v should have icon %s, got %s", tc.state, tc.icon, icon)
		}
	}
}

// TestTTSStatusReset tests resetting the display.
func TestTTSStatusReset(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Set up some state
	display.state = tts.StatePlaying
	display.currentSentence = 5
	display.totalSentences = 10
	display.isBuffering = true
	display.errorMessage = "test error"
	
	// Reset
	display.Reset()
	
	if display.state != tts.StateIdle {
		t.Error("State should be idle after reset")
	}
	
	if display.currentSentence != -1 {
		t.Error("Current sentence should be -1 after reset")
	}
	
	if display.totalSentences != 0 {
		t.Error("Total sentences should be 0 after reset")
	}
	
	if display.isBuffering {
		t.Error("Should not be buffering after reset")
	}
	
	if display.errorMessage != "" {
		t.Error("Error message should be empty after reset")
	}
}

// TestTTSStatusClone tests cloning the display.
func TestTTSStatusClone(t *testing.T) {
	display := NewTTSStatusDisplay()
	
	// Set up state
	display.state = tts.StatePlaying
	display.currentSentence = 3
	display.totalSentences = 10
	display.progress = 0.3
	
	// Clone
	clone := display.Clone()
	
	if clone == display {
		t.Error("Clone should be a different instance")
	}
	
	if clone.state != display.state {
		t.Error("Clone should have same state")
	}
	
	if clone.currentSentence != display.currentSentence {
		t.Error("Clone should have same current sentence")
	}
	
	// Modify original
	display.currentSentence = 5
	
	// Clone should be unchanged
	if clone.currentSentence != 3 {
		t.Error("Clone should not be affected by changes to original")
	}
}

// TestFormatDuration tests duration formatting.
func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0:00"},
		{30 * time.Second, "0:30"},
		{90 * time.Second, "1:30"},
		{125 * time.Second, "2:05"},
		{-5 * time.Second, "0:00"}, // Negative should show as 0:00
	}
	
	for _, tc := range testCases {
		result := formatDuration(tc.duration)
		if result != tc.expected {
			t.Errorf("formatDuration(%v) = %s, want %s", tc.duration, result, tc.expected)
		}
	}
}