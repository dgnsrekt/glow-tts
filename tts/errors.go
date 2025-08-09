package tts

import "errors"

// Common errors for the TTS system.
var (
	// Engine errors
	ErrEngineNotAvailable = errors.New("TTS engine is not available")
	ErrEngineNotInitialized = errors.New("TTS engine is not initialized")
	ErrEngineAlreadyInitialized = errors.New("TTS engine is already initialized")
	ErrVoiceNotFound = errors.New("requested voice not found")
	ErrInvalidVoice = errors.New("invalid voice configuration")
	ErrGenerationFailed = errors.New("audio generation failed")
	ErrEngineShutdown = errors.New("engine has been shut down")
	
	// Player errors
	ErrPlayerNotInitialized = errors.New("audio player is not initialized")
	ErrPlayerBusy = errors.New("audio player is busy")
	ErrNothingToPlay = errors.New("no audio to play")
	ErrAlreadyPlaying = errors.New("audio is already playing")
	ErrNotPlaying = errors.New("no audio is playing")
	ErrAlreadyPaused = errors.New("audio is already paused")
	ErrNotPaused = errors.New("audio is not paused")
	ErrInvalidAudioFormat = errors.New("invalid audio format")
	
	// Parser errors
	ErrEmptyContent = errors.New("empty content provided")
	ErrInvalidMarkdown = errors.New("invalid markdown format")
	ErrNoSentencesFound = errors.New("no sentences found in content")
	
	// Synchronization errors
	ErrSyncNotStarted = errors.New("synchronization not started")
	ErrSyncAlreadyStarted = errors.New("synchronization already started")
	ErrInvalidSentenceIndex = errors.New("invalid sentence index")
	
	// Controller errors
	ErrControllerNotInitialized = errors.New("TTS controller not initialized")
	ErrControllerShutdown = errors.New("TTS controller has been shut down")
	ErrInvalidState = errors.New("invalid state for operation")
	ErrStateTransition = errors.New("invalid state transition")
	
	// Configuration errors
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrMissingConfig = errors.New("required configuration missing")
	ErrInvalidSampleRate = errors.New("invalid sample rate")
	ErrInvalidChannels = errors.New("invalid number of channels")
	
	// Buffer errors
	ErrBufferFull = errors.New("audio buffer is full")
	ErrBufferEmpty = errors.New("audio buffer is empty")
	ErrBufferClosed = errors.New("audio buffer is closed")
	
	// General errors
	ErrTimeout = errors.New("operation timed out")
	ErrCanceled = errors.New("operation was canceled")
	ErrNotImplemented = errors.New("feature not implemented")
	ErrResourceNotFound = errors.New("resource not found")
	ErrPermissionDenied = errors.New("permission denied")
)

// IsRecoverableError checks if an error is recoverable.
func IsRecoverableError(err error) bool {
	if err == nil {
		return true
	}
	
	// Non-recoverable errors
	switch err {
	case ErrEngineNotAvailable,
		ErrEngineShutdown,
		ErrControllerShutdown,
		ErrInvalidConfig,
		ErrMissingConfig,
		ErrPermissionDenied:
		return false
	}
	
	// Most errors are recoverable
	return true
}

// ErrorSeverity represents the severity of an error.
type ErrorSeverity int

const (
	// SeverityInfo is for informational messages.
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning is for warnings that don't prevent operation.
	SeverityWarning
	// SeverityError is for errors that prevent normal operation.
	SeverityError
	// SeverityCritical is for errors that require immediate attention.
	SeverityCritical
)

// TTSError provides detailed error information.
type TTSError struct {
	Err       error         // The underlying error
	Component string        // Component that generated the error
	Action    string        // Action being performed when error occurred
	Severity  ErrorSeverity // Severity of the error
	Timestamp int64         // Unix timestamp when error occurred
	Context   map[string]interface{} // Additional context
}

// Error implements the error interface.
func (e *TTSError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown TTS error"
}

// Unwrap returns the underlying error.
func (e *TTSError) Unwrap() error {
	return e.Err
}

// IsRecoverable checks if the error is recoverable.
func (e *TTSError) IsRecoverable() bool {
	return IsRecoverableError(e.Err)
}

// NewTTSError creates a new TTS error with context.
func NewTTSError(err error, component, action string) *TTSError {
	return &TTSError{
		Err:       err,
		Component: component,
		Action:    action,
		Severity:  SeverityError,
		Context:   make(map[string]interface{}),
	}
}

// WithSeverity sets the error severity.
func (e *TTSError) WithSeverity(severity ErrorSeverity) *TTSError {
	e.Severity = severity
	return e
}

// WithContext adds context to the error.
func (e *TTSError) WithContext(key string, value interface{}) *TTSError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}