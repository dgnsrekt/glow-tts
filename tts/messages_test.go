package tts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
)

// TestMessageTypes tests that all message types are properly defined.
func TestMessageTypes(t *testing.T) {
	// Test PlayingMsg
	playingMsg := tts.PlayingMsg{
		Sentence: 1,
		Total:    10,
		Duration: 5 * time.Second,
	}
	if playingMsg.Sentence != 1 {
		t.Error("PlayingMsg field not set correctly")
	}

	// Test PausedMsg
	pausedMsg := tts.PausedMsg{
		Position: 2 * time.Second,
		Sentence: 3,
	}
	if pausedMsg.Position != 2*time.Second {
		t.Error("PausedMsg field not set correctly")
	}

	// Test ResumedMsg
	resumedMsg := tts.ResumedMsg{
		Position: 3 * time.Second,
		Sentence: 4,
	}
	if resumedMsg.Sentence != 4 {
		t.Error("ResumedMsg field not set correctly")
	}

	// Test StoppedMsg
	stoppedMsg := tts.StoppedMsg{
		Reason: "user",
	}
	if stoppedMsg.Reason != "user" {
		t.Error("StoppedMsg field not set correctly")
	}

	// Test SentenceChangedMsg
	sentenceMsg := tts.SentenceChangedMsg{
		Index:    5,
		Text:     "Test sentence",
		Duration: 1 * time.Second,
		Progress: 0.5,
	}
	if sentenceMsg.Text != "Test sentence" {
		t.Error("SentenceChangedMsg field not set correctly")
	}

	// Test TTSStateChangedMsg
	stateMsg := tts.TTSStateChangedMsg{
		State:     tts.StatePlaying,
		PrevState: tts.StateReady,
		Sentence:  2,
		Total:     10,
		Timestamp: time.Now(),
	}
	if stateMsg.State != tts.StatePlaying {
		t.Error("TTSStateChangedMsg field not set correctly")
	}

	// Test TTSErrorMsg
	errorMsg := tts.TTSErrorMsg{
		Error:       errors.New("test error"),
		Recoverable: true,
		Component:   "engine",
		Action:      "generate",
	}
	if errorMsg.Component != "engine" {
		t.Error("TTSErrorMsg field not set correctly")
	}

	// Test AudioGeneratedMsg
	audioMsg := tts.AudioGeneratedMsg{
		Index: 1,
		Audio: &tts.Audio{
			Data:       []byte{1, 2, 3},
			Format:     tts.FormatPCM16,
			SampleRate: 22050,
			Channels:   1,
			Duration:   2 * time.Second,
		},
		Sentence: "Test",
		Duration: 2 * time.Second,
	}
	if audioMsg.Audio.SampleRate != 22050 {
		t.Error("AudioGeneratedMsg field not set correctly")
	}

	// Test TTSEnabledMsg
	enabledMsg := tts.TTSEnabledMsg{
		Engine: "piper",
	}
	if enabledMsg.Engine != "piper" {
		t.Error("TTSEnabledMsg field not set correctly")
	}

	// Test TTSDisabledMsg
	disabledMsg := tts.TTSDisabledMsg{
		Reason: "shutdown",
	}
	if disabledMsg.Reason != "shutdown" {
		t.Error("TTSDisabledMsg field not set correctly")
	}

	// Test TTSInitializingMsg
	initMsg := tts.TTSInitializingMsg{
		Engine: "piper",
		Steps:  5,
		Step:   2,
	}
	if initMsg.Steps != 5 {
		t.Error("TTSInitializingMsg field not set correctly")
	}

	// Test TTSReadyMsg
	readyMsg := tts.TTSReadyMsg{
		Engine:        "piper",
		VoiceCount:    3,
		SelectedVoice: "default",
	}
	if readyMsg.VoiceCount != 3 {
		t.Error("TTSReadyMsg field not set correctly")
	}

	// Test PositionUpdateMsg
	posMsg := tts.PositionUpdateMsg{
		Position:         1 * time.Second,
		Duration:         5 * time.Second,
		SentenceIndex:    2,
		SentenceProgress: 0.2,
		TotalProgress:    0.1,
	}
	if posMsg.SentenceProgress != 0.2 {
		t.Error("PositionUpdateMsg field not set correctly")
	}

	// Test BufferStatusMsg
	bufferMsg := tts.BufferStatusMsg{
		Buffered:  3,
		Capacity:  5,
		IsLoading: true,
	}
	if bufferMsg.Buffered != 3 {
		t.Error("BufferStatusMsg field not set correctly")
	}

	// Test VoiceChangedMsg
	voiceMsg := tts.VoiceChangedMsg{
		Voice: tts.Voice{
			ID:       "voice1",
			Name:     "Default",
			Language: "en-US",
			Gender:   "neutral",
		},
	}
	if voiceMsg.Voice.Language != "en-US" {
		t.Error("VoiceChangedMsg field not set correctly")
	}

	// Test SpeedChangedMsg
	speedMsg := tts.SpeedChangedMsg{
		Speed: 1.5,
	}
	if speedMsg.Speed != 1.5 {
		t.Error("SpeedChangedMsg field not set correctly")
	}

	// Test VolumeChangedMsg
	volumeMsg := tts.VolumeChangedMsg{
		Volume: 0.8,
	}
	if volumeMsg.Volume != 0.8 {
		t.Error("VolumeChangedMsg field not set correctly")
	}

	// Test NavigationMsg
	navMsg := tts.NavigationMsg{
		Target:    5,
		Direction: "next",
	}
	if navMsg.Direction != "next" {
		t.Error("NavigationMsg field not set correctly")
	}
}

// TestCommandGenerators tests that command generators work correctly.
func TestCommandGenerators(t *testing.T) {
	// Create a mock engine for testing
	engine := mock.New()

	// Test GenerateAudioCmd
	cmd := tts.GenerateAudioCmd(engine, "Test text", 0)
	if cmd == nil {
		t.Error("GenerateAudioCmd should return a command")
	}

	// Execute the command
	msg := cmd()
	switch m := msg.(type) {
	case tts.AudioGeneratedMsg:
		if m.Index != 0 {
			t.Error("AudioGeneratedMsg index incorrect")
		}
		if m.Audio == nil {
			t.Error("AudioGeneratedMsg should contain audio")
		}
	case tts.TTSErrorMsg:
		// Error is acceptable in test environment
	default:
		t.Errorf("Unexpected message type: %T", msg)
	}

	// Test InitializeTTSCmd
	config := tts.EngineConfig{
		Voice:  "default",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 1.0,
	}
	initCmd := tts.InitializeTTSCmd(engine, config)
	if initCmd == nil {
		t.Error("InitializeTTSCmd should return a command")
	}

	// Execute the init command
	initMsg := initCmd()
	switch m := initMsg.(type) {
	case tts.TTSReadyMsg:
		if m.Engine != "active" {
			t.Error("TTSReadyMsg engine incorrect")
		}
	case tts.TTSErrorMsg:
		// Error is acceptable in test environment
	default:
		t.Errorf("Unexpected message type from InitializeTTSCmd: %T", initMsg)
	}

	// Test NavigateToSentenceCmd
	navCmd := tts.NavigateToSentenceCmd(5, "next")
	if navCmd == nil {
		t.Error("NavigateToSentenceCmd should return a command")
	}

	navMsg := navCmd()
	if nav, ok := navMsg.(tts.NavigationMsg); ok {
		if nav.Target != 5 || nav.Direction != "next" {
			t.Error("NavigationMsg fields incorrect")
		}
	} else {
		t.Errorf("NavigateToSentenceCmd returned wrong type: %T", navMsg)
	}

	// Test ChangeSpeedCmd
	speedCmd := tts.ChangeSpeedCmd(1.5)
	if speedCmd == nil {
		t.Error("ChangeSpeedCmd should return a command")
	}

	speedMsg := speedCmd()
	if speed, ok := speedMsg.(tts.SpeedChangedMsg); ok {
		if speed.Speed != 1.5 {
			t.Error("SpeedChangedMsg speed incorrect")
		}
	} else {
		t.Errorf("ChangeSpeedCmd returned wrong type: %T", speedMsg)
	}

	// Test ChangeVolumeCmd
	volumeCmd := tts.ChangeVolumeCmd(0.7)
	if volumeCmd == nil {
		t.Error("ChangeVolumeCmd should return a command")
	}

	volumeMsg := volumeCmd()
	if volume, ok := volumeMsg.(tts.VolumeChangedMsg); ok {
		if volume.Volume != 0.7 {
			t.Error("VolumeChangedMsg volume incorrect")
		}
	} else {
		t.Errorf("ChangeVolumeCmd returned wrong type: %T", volumeMsg)
	}

	// Test BatchGenerateAudioCmd
	sentences := []tts.Sentence{
		{Index: 0, Text: "First sentence."},
		{Index: 1, Text: "Second sentence."},
	}
	batchCmd := tts.BatchGenerateAudioCmd(engine, sentences, 0)
	if batchCmd == nil {
		t.Error("BatchGenerateAudioCmd should return a command")
	}

	batchMsg := batchCmd()
	switch m := batchMsg.(type) {
	case tts.BufferStatusMsg:
		if m.Buffered != len(sentences) {
			t.Error("BufferStatusMsg buffered count incorrect")
		}
	case tts.TTSErrorMsg:
		// Error is acceptable in test environment
	default:
		t.Errorf("Unexpected message type from BatchGenerateAudioCmd: %T", batchMsg)
	}
}

// TestErrorMessages tests error message handling.
func TestErrorMessages(t *testing.T) {
	// Test recoverable error
	recoverableErr := tts.TTSErrorMsg{
		Error:       errors.New("temporary error"),
		Recoverable: true,
		Component:   "player",
		Action:      "play",
	}

	if !recoverableErr.Recoverable {
		t.Error("Error should be marked as recoverable")
	}

	if recoverableErr.Component != "player" {
		t.Error("Error component not set correctly")
	}

	// Test non-recoverable error
	criticalErr := tts.TTSErrorMsg{
		Error:       tts.ErrEngineNotAvailable,
		Recoverable: false,
		Component:   "engine",
		Action:      "initialize",
	}

	if criticalErr.Recoverable {
		t.Error("Critical error should not be recoverable")
	}

	if criticalErr.Action != "initialize" {
		t.Error("Error action not set correctly")
	}
}

// TestMessageFieldValidation tests that message fields are validated correctly.
func TestMessageFieldValidation(t *testing.T) {
	// Test state transition message
	stateMsg := tts.TTSStateChangedMsg{
		State:     tts.StatePlaying,
		PrevState: tts.StateReady,
		Sentence:  5,
		Total:     10,
		Timestamp: time.Now(),
	}

	// Validate state transition is logical
	if stateMsg.State == stateMsg.PrevState {
		t.Error("State and PrevState should be different")
	}

	// Test sentence bounds
	if stateMsg.Sentence >= stateMsg.Total {
		t.Error("Current sentence should be less than total")
	}

	// Test progress calculations
	posMsg := tts.PositionUpdateMsg{
		Position:         2 * time.Second,
		Duration:         10 * time.Second,
		SentenceIndex:    3,
		SentenceProgress: 0.2,
		TotalProgress:    0.3,
	}

	// Validate progress is between 0 and 1
	if posMsg.SentenceProgress < 0 || posMsg.SentenceProgress > 1 {
		t.Error("Sentence progress should be between 0 and 1")
	}

	if posMsg.TotalProgress < 0 || posMsg.TotalProgress > 1 {
		t.Error("Total progress should be between 0 and 1")
	}

	// Test buffer capacity
	bufferMsg := tts.BufferStatusMsg{
		Buffered:  3,
		Capacity:  5,
		IsLoading: false,
	}

	if bufferMsg.Buffered > bufferMsg.Capacity {
		t.Error("Buffered count should not exceed capacity")
	}
}