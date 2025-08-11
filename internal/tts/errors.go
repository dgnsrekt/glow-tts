package tts

import (
	"errors"
	"fmt"
)

// Common TTS errors
var (
	// ErrNoEngineConfigured indicates no TTS engine has been selected
	ErrNoEngineConfigured = errors.New("no TTS engine configured - specify --tts piper or --tts gtts")

	// ErrEngineNotAvailable indicates the selected engine is not available
	ErrEngineNotAvailable = errors.New("selected TTS engine is not available")

	// ErrInvalidEngine indicates an unknown engine was specified
	ErrInvalidEngine = errors.New("invalid TTS engine specified")

	// ErrSynthesisFailed indicates synthesis operation failed
	ErrSynthesisFailed = errors.New("text synthesis failed")

	// ErrAudioDeviceUnavailable indicates audio device cannot be accessed
	ErrAudioDeviceUnavailable = errors.New("audio device unavailable")

	// ErrQueueFull indicates the sentence queue is at capacity
	ErrQueueFull = errors.New("sentence queue is full")

	// ErrQueueEmpty indicates the queue has no items
	ErrQueueEmpty = errors.New("sentence queue is empty")

	// ErrCacheFull indicates cache has reached size limit
	ErrCacheFull = errors.New("cache size limit reached")

	// ErrInvalidSpeed indicates speed value is out of range
	ErrInvalidSpeed = errors.New("speed must be between 0.5 and 2.0")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrCanceled indicates an operation was canceled
	ErrCanceled = errors.New("operation canceled")
)

// TTSError represents a TTS-specific error with additional context
type TTSError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *TTSError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *TTSError) Unwrap() error {
	return e.Cause
}

// ErrorCode identifies specific error types
type ErrorCode string

const (
	// Engine errors
	ErrorCodeEngineFailure     ErrorCode = "ENGINE_FAILURE"
	ErrorCodeEngineUnavailable ErrorCode = "ENGINE_UNAVAILABLE"
	ErrorCodeEngineTimeout     ErrorCode = "ENGINE_TIMEOUT"

	// Audio errors
	ErrorCodeAudioFailure ErrorCode = "AUDIO_FAILURE"
	ErrorCodeAudioDevice  ErrorCode = "AUDIO_DEVICE"
	ErrorCodeAudioFormat  ErrorCode = "AUDIO_FORMAT"

	// Queue errors
	ErrorCodeQueueFull  ErrorCode = "QUEUE_FULL"
	ErrorCodeQueueEmpty ErrorCode = "QUEUE_EMPTY"

	// Cache errors
	ErrorCodeCacheFull      ErrorCode = "CACHE_FULL"
	ErrorCodeCacheCorrupted ErrorCode = "CACHE_CORRUPTED"

	// Input errors
	ErrorCodeInvalidInput ErrorCode = "INVALID_INPUT"
	ErrorCodeTextTooLong  ErrorCode = "TEXT_TOO_LONG"

	// System errors
	ErrorCodeTimeout           ErrorCode = "TIMEOUT"
	ErrorCodeCanceled          ErrorCode = "CANCELED"
	ErrorCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
)

// NewTTSError creates a new TTS error with context
func NewTTSError(code ErrorCode, message string, cause error) *TTSError {
	return &TTSError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *TTSError) WithContext(key string, value interface{}) *TTSError {
	e.Context[key] = value
	return e
}

// IsFatal returns true if the error should stop TTS operation
func (e *TTSError) IsFatal() bool {
	switch e.Code {
	case ErrorCodeEngineUnavailable,
		ErrorCodeAudioDevice,
		ErrorCodeResourceExhausted:
		return true
	default:
		return false
	}
}

// IsRetryable returns true if the operation can be retried
func (e *TTSError) IsRetryable() bool {
	switch e.Code {
	case ErrorCodeTimeout,
		ErrorCodeEngineTimeout,
		ErrorCodeCacheFull:
		return true
	default:
		return false
	}
}
