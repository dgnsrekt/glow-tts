package tts

import (
	"os"
	"testing"
	"time"
)

// setupTestAudioContext sets up an appropriate audio context for testing
// It automatically uses mock context in CI environments
func setupTestAudioContext(t *testing.T) AudioContextInterface {
	t.Helper()
	
	// Reset any existing global context for clean test state
	ResetGlobalAudioContext()
	
	// If we're in CI or explicitly want mock audio, use mock context
	if os.Getenv("CI") == "true" || os.Getenv("MOCK_AUDIO") == "true" {
		t.Log("Using mock audio context for testing")
		ctx, err := NewMockAudioContext()
		if err != nil {
			t.Fatalf("Failed to create mock audio context: %v", err)
		}
		SetGlobalAudioContext(ctx)
		return ctx
	}
	
	// Otherwise try to use real audio, fall back to mock if it fails
	ctx, err := NewAudioContext(AudioContextAuto)
	if err != nil {
		t.Fatalf("Failed to create audio context: %v", err)
	}
	
	SetGlobalAudioContext(ctx)
	return ctx
}

// cleanupTestAudioContext cleans up the audio context after testing
func cleanupTestAudioContext(t *testing.T) {
	t.Helper()
	ResetGlobalAudioContext()
}

// waitForState waits for a specific state with retries and timeout
// This is more robust than checking state once after a fixed sleep
func waitForState(t *testing.T, getState func() PlaybackState, expected PlaybackState, timeout time.Duration, description string) bool {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	checkInterval := 10 * time.Millisecond
	
	// For CI environments, use longer intervals to reduce CPU usage
	if os.Getenv("CI") == "true" {
		checkInterval = 25 * time.Millisecond
	}
	
	for time.Now().Before(deadline) {
		if getState() == expected {
			return true
		}
		time.Sleep(checkInterval)
	}
	
	// Final check after timeout
	actual := getState()
	if actual != expected {
		t.Logf("%s: Expected state %v, got %v after %v timeout", description, expected, actual, timeout)
	}
	return actual == expected
}

// waitForQueueState waits for a queue to reach a specific state
func waitForQueueState(t *testing.T, queue *TTSAudioQueue, expected QueueState, timeout time.Duration) bool {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	checkInterval := 10 * time.Millisecond
	
	// For CI environments, use longer intervals
	if os.Getenv("CI") == "true" {
		checkInterval = 25 * time.Millisecond
	}
	
	for time.Now().Before(deadline) {
		if queue.GetState() == expected {
			return true
		}
		time.Sleep(checkInterval)
	}
	
	actual := queue.GetState()
	if actual != expected {
		t.Logf("Expected queue state %v, got %v after %v timeout", expected, actual, timeout)
	}
	return actual == expected
}

// waitForCondition waits for a condition to become true
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, description string) bool {
	t.Helper()
	
	deadline := time.Now().Add(timeout)
	checkInterval := 10 * time.Millisecond
	
	// For CI environments, use longer intervals
	if os.Getenv("CI") == "true" {
		checkInterval = 25 * time.Millisecond
	}
	
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	
	t.Logf("%s: Condition not met after %v timeout", description, timeout)
	return false
}