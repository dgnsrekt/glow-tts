package tts

import (
	"errors"
	"strings"
	"testing"
)

// TestErrorDefinitions tests that all error variables are properly defined.
func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		// Engine errors
		{"ErrEngineNotAvailable", ErrEngineNotAvailable, "TTS engine is not available"},
		{"ErrEngineNotInitialized", ErrEngineNotInitialized, "TTS engine is not initialized"},
		{"ErrEngineAlreadyInitialized", ErrEngineAlreadyInitialized, "TTS engine is already initialized"},
		{"ErrVoiceNotFound", ErrVoiceNotFound, "requested voice not found"},
		{"ErrInvalidVoice", ErrInvalidVoice, "invalid voice configuration"},
		{"ErrGenerationFailed", ErrGenerationFailed, "audio generation failed"},
		{"ErrEngineShutdown", ErrEngineShutdown, "engine has been shut down"},
		
		// Player errors
		{"ErrPlayerNotInitialized", ErrPlayerNotInitialized, "audio player is not initialized"},
		{"ErrPlayerBusy", ErrPlayerBusy, "audio player is busy"},
		{"ErrNothingToPlay", ErrNothingToPlay, "no audio to play"},
		{"ErrAlreadyPlaying", ErrAlreadyPlaying, "audio is already playing"},
		{"ErrNotPlaying", ErrNotPlaying, "no audio is playing"},
		{"ErrAlreadyPaused", ErrAlreadyPaused, "audio is already paused"},
		{"ErrNotPaused", ErrNotPaused, "audio is not paused"},
		{"ErrInvalidAudioFormat", ErrInvalidAudioFormat, "invalid audio format"},
		
		// Parser errors
		{"ErrEmptyContent", ErrEmptyContent, "empty content provided"},
		{"ErrInvalidMarkdown", ErrInvalidMarkdown, "invalid markdown format"},
		{"ErrNoSentencesFound", ErrNoSentencesFound, "no sentences found in content"},
		
		// Synchronization errors
		{"ErrSyncNotStarted", ErrSyncNotStarted, "synchronization not started"},
		{"ErrSyncAlreadyStarted", ErrSyncAlreadyStarted, "synchronization already started"},
		{"ErrInvalidSentenceIndex", ErrInvalidSentenceIndex, "invalid sentence index"},
		
		// Controller errors
		{"ErrControllerNotInitialized", ErrControllerNotInitialized, "TTS controller not initialized"},
		{"ErrControllerShutdown", ErrControllerShutdown, "TTS controller has been shut down"},
		{"ErrInvalidState", ErrInvalidState, "invalid state for operation"},
		{"ErrStateTransition", ErrStateTransition, "invalid state transition"},
		
		// Configuration errors
		{"ErrInvalidConfig", ErrInvalidConfig, "invalid configuration"},
		{"ErrMissingConfig", ErrMissingConfig, "required configuration missing"},
		{"ErrInvalidSampleRate", ErrInvalidSampleRate, "invalid sample rate"},
		{"ErrInvalidChannels", ErrInvalidChannels, "invalid number of channels"},
		
		// Buffer errors
		{"ErrBufferFull", ErrBufferFull, "audio buffer is full"},
		{"ErrBufferEmpty", ErrBufferEmpty, "audio buffer is empty"},
		{"ErrBufferClosed", ErrBufferClosed, "audio buffer is closed"},
		
		// General errors
		{"ErrTimeout", ErrTimeout, "operation timed out"},
		{"ErrCanceled", ErrCanceled, "operation was canceled"},
		{"ErrNotImplemented", ErrNotImplemented, "feature not implemented"},
		{"ErrResourceNotFound", ErrResourceNotFound, "resource not found"},
		{"ErrPermissionDenied", ErrPermissionDenied, "permission denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
				return
			}
			if tt.err.Error() != tt.msg {
				t.Errorf("%s message = %q, want %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

// TestIsRecoverableError tests the IsRecoverableError function.
func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		recoverable bool
	}{
		// Non-recoverable errors
		{"engine not available", ErrEngineNotAvailable, false},
		{"engine shutdown", ErrEngineShutdown, false},
		{"controller shutdown", ErrControllerShutdown, false},
		{"invalid config", ErrInvalidConfig, false},
		{"missing config", ErrMissingConfig, false},
		{"permission denied", ErrPermissionDenied, false},
		
		// Recoverable errors
		{"generation failed", ErrGenerationFailed, true},
		{"buffer full", ErrBufferFull, true},
		{"buffer empty", ErrBufferEmpty, true},
		{"already playing", ErrAlreadyPlaying, true},
		{"not playing", ErrNotPlaying, true},
		{"timeout", ErrTimeout, true},
		{"canceled", ErrCanceled, true},
		
		// Nil error is recoverable
		{"nil error", nil, true},
		
		// Unknown error is recoverable by default
		{"unknown error", errors.New("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRecoverableError(tt.err)
			if result != tt.recoverable {
				t.Errorf("IsRecoverableError(%v) = %v, want %v", tt.err, result, tt.recoverable)
			}
		})
	}
}

// TestTTSError tests the TTSError type.
func TestTTSError(t *testing.T) {
	baseErr := ErrGenerationFailed
	ttsErr := NewTTSError(baseErr, "engine", "generate")
	
	// Test Error() method
	if ttsErr.Error() != baseErr.Error() {
		t.Errorf("TTSError.Error() = %q, want %q", ttsErr.Error(), baseErr.Error())
	}
	
	// Test Unwrap() method
	if ttsErr.Unwrap() != baseErr {
		t.Error("TTSError.Unwrap() should return the base error")
	}
	
	// Test IsRecoverable() method
	if !ttsErr.IsRecoverable() {
		t.Error("TTSError.IsRecoverable() should return true for generation failed")
	}
	
	// Test component and action
	if ttsErr.Component != "engine" {
		t.Errorf("Component = %q, want %q", ttsErr.Component, "engine")
	}
	if ttsErr.Action != "generate" {
		t.Errorf("Action = %q, want %q", ttsErr.Action, "generate")
	}
	
	// Test default severity
	if ttsErr.Severity != SeverityError {
		t.Errorf("Default severity = %v, want %v", ttsErr.Severity, SeverityError)
	}
}

// TestTTSErrorWithSeverity tests severity setting.
func TestTTSErrorWithSeverity(t *testing.T) {
	ttsErr := NewTTSError(ErrTimeout, "controller", "wait")
	ttsErr.WithSeverity(SeverityWarning)
	
	if ttsErr.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", ttsErr.Severity, SeverityWarning)
	}
}

// TestTTSErrorWithContext tests context adding.
func TestTTSErrorWithContext(t *testing.T) {
	ttsErr := NewTTSError(ErrBufferFull, "buffer", "add")
	ttsErr.WithContext("size", 100).WithContext("capacity", 100)
	
	if ttsErr.Context["size"] != 100 {
		t.Errorf("Context[size] = %v, want 100", ttsErr.Context["size"])
	}
	if ttsErr.Context["capacity"] != 100 {
		t.Errorf("Context[capacity] = %v, want 100", ttsErr.Context["capacity"])
	}
}

// TestTTSErrorNilError tests TTSError with nil underlying error.
func TestTTSErrorNilError(t *testing.T) {
	ttsErr := &TTSError{
		Err:       nil,
		Component: "test",
		Action:    "test",
	}
	
	expected := "unknown TTS error"
	if ttsErr.Error() != expected {
		t.Errorf("Error() with nil = %q, want %q", ttsErr.Error(), expected)
	}
}

// TestErrorWrapping tests that errors can be properly wrapped.
func TestErrorWrapping(t *testing.T) {
	baseErr := ErrGenerationFailed
	wrappedErr := errors.Join(baseErr, errors.New("additional context"))
	
	// Check that the wrapped error contains both messages
	errMsg := wrappedErr.Error()
	if !strings.Contains(errMsg, baseErr.Error()) {
		t.Errorf("Wrapped error should contain base error message: %q", errMsg)
	}
	
	// Check that errors.Is works with our errors
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("errors.Is should work with wrapped errors")
	}
}

// TestErrorUniqueness tests that all error messages are unique.
func TestErrorUniqueness(t *testing.T) {
	errorMessages := map[string]string{
		"ErrEngineNotAvailable":        ErrEngineNotAvailable.Error(),
		"ErrEngineNotInitialized":       ErrEngineNotInitialized.Error(),
		"ErrEngineAlreadyInitialized":   ErrEngineAlreadyInitialized.Error(),
		"ErrVoiceNotFound":              ErrVoiceNotFound.Error(),
		"ErrInvalidVoice":               ErrInvalidVoice.Error(),
		"ErrGenerationFailed":           ErrGenerationFailed.Error(),
		"ErrEngineShutdown":             ErrEngineShutdown.Error(),
		"ErrPlayerNotInitialized":       ErrPlayerNotInitialized.Error(),
		"ErrPlayerBusy":                 ErrPlayerBusy.Error(),
		"ErrNothingToPlay":              ErrNothingToPlay.Error(),
		"ErrAlreadyPlaying":             ErrAlreadyPlaying.Error(),
		"ErrNotPlaying":                 ErrNotPlaying.Error(),
		"ErrAlreadyPaused":              ErrAlreadyPaused.Error(),
		"ErrNotPaused":                  ErrNotPaused.Error(),
		"ErrInvalidAudioFormat":         ErrInvalidAudioFormat.Error(),
		"ErrEmptyContent":               ErrEmptyContent.Error(),
		"ErrInvalidMarkdown":            ErrInvalidMarkdown.Error(),
		"ErrNoSentencesFound":           ErrNoSentencesFound.Error(),
		"ErrSyncNotStarted":             ErrSyncNotStarted.Error(),
		"ErrSyncAlreadyStarted":         ErrSyncAlreadyStarted.Error(),
		"ErrInvalidSentenceIndex":       ErrInvalidSentenceIndex.Error(),
		"ErrControllerNotInitialized":   ErrControllerNotInitialized.Error(),
		"ErrControllerShutdown":         ErrControllerShutdown.Error(),
		"ErrInvalidState":               ErrInvalidState.Error(),
		"ErrStateTransition":            ErrStateTransition.Error(),
		"ErrInvalidConfig":              ErrInvalidConfig.Error(),
		"ErrMissingConfig":              ErrMissingConfig.Error(),
		"ErrInvalidSampleRate":          ErrInvalidSampleRate.Error(),
		"ErrInvalidChannels":            ErrInvalidChannels.Error(),
		"ErrBufferFull":                 ErrBufferFull.Error(),
		"ErrBufferEmpty":                ErrBufferEmpty.Error(),
		"ErrBufferClosed":               ErrBufferClosed.Error(),
		"ErrTimeout":                    ErrTimeout.Error(),
		"ErrCanceled":                   ErrCanceled.Error(),
		"ErrNotImplemented":             ErrNotImplemented.Error(),
		"ErrResourceNotFound":           ErrResourceNotFound.Error(),
		"ErrPermissionDenied":           ErrPermissionDenied.Error(),
	}

	// Check for duplicate messages
	seen := make(map[string]string)
	for name, msg := range errorMessages {
		if existing, ok := seen[msg]; ok {
			t.Errorf("Duplicate error message %q used by both %s and %s", msg, existing, name)
		}
		seen[msg] = name
	}
}

// TestErrorSeverity tests ErrorSeverity constants.
func TestErrorSeverity(t *testing.T) {
	if SeverityInfo != 0 {
		t.Errorf("SeverityInfo = %d, want 0", SeverityInfo)
	}
	if SeverityWarning != 1 {
		t.Errorf("SeverityWarning = %d, want 1", SeverityWarning)
	}
	if SeverityError != 2 {
		t.Errorf("SeverityError = %d, want 2", SeverityError)
	}
	if SeverityCritical != 3 {
		t.Errorf("SeverityCritical = %d, want 3", SeverityCritical)
	}
}