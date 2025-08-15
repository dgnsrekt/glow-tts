package tts

import (
	"os"
	"testing"
	"time"
)

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