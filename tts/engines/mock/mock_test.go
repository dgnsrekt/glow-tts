package mock

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// TestNewMockEngine tests mock engine creation.
func TestNewMockEngine(t *testing.T) {
	engine := New()
	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
	
	if !engine.IsAvailable() {
		t.Error("Mock engine should be available by default")
	}
}

// TestInitialize tests mock engine initialization.
func TestInitialize(t *testing.T) {
	engine := New()
	
	config := tts.EngineConfig{
		Voice:  "test-voice",
		Rate:   1.5,
		Pitch:  0.5,
		Volume: 0.8,
	}
	
	err := engine.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

// TestGenerateAudio tests audio generation.
func TestGenerateAudio(t *testing.T) {
	engine := New()
	engine.SetDelay(10 * time.Millisecond) // Short delay for testing
	
	err := engine.Initialize(tts.EngineConfig{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	text := "Hello, world!"
	audio, err := engine.GenerateAudio(text)
	if err != nil {
		t.Fatalf("GenerateAudio failed: %v", err)
	}
	
	if audio == nil {
		t.Fatal("Expected non-nil audio")
	}
	
	if len(audio.Data) == 0 {
		t.Error("Expected non-empty audio data")
	}
	
	// Check audio format
	if audio.Format != tts.FormatPCM16 {
		t.Errorf("Expected FormatPCM16, got %v", audio.Format)
	}
	
	if audio.SampleRate != 22050 {
		t.Errorf("Expected 22050 sample rate, got %d", audio.SampleRate)
	}
	
	if audio.Channels != 1 {
		t.Errorf("Expected 1 channel, got %d", audio.Channels)
	}
	
	// Check duration is reasonable
	if audio.Duration <= 0 {
		t.Error("Expected positive duration")
	}
}

// TestGenerateAudioWithFailure tests error injection during audio generation.
func TestGenerateAudioWithFailure(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	
	testError := errors.New("test error")
	engine.SetFailure(testError)
	
	_, err := engine.GenerateAudio("test")
	if err != testError {
		t.Errorf("Expected injected error, got %v", err)
	}
	
	// Clear failure and try again
	engine.ClearFailure()
	_, err = engine.GenerateAudio("test")
	if err != nil {
		t.Errorf("Unexpected error after clearing failure: %v", err)
	}
}

// TestGenerateAudioStream tests streaming audio generation.
func TestGenerateAudioStream(t *testing.T) {
	engine := New()
	engine.SetDelay(5 * time.Millisecond)
	engine.Initialize(tts.EngineConfig{})
	
	text := "Test streaming audio"
	ch, err := engine.GenerateAudioStream(text)
	if err != nil {
		t.Fatalf("GenerateAudioStream failed: %v", err)
	}
	
	if ch == nil {
		t.Fatal("Expected non-nil channel")
	}
	
	// Read chunks from channel
	chunks := []tts.AudioChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}
	
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
	
	// Last chunk should be marked as final
	lastChunk := chunks[len(chunks)-1]
	if !lastChunk.Final {
		t.Error("Last chunk should be marked as final")
	}
}

// TestGetVoices tests voice listing.
func TestGetVoices(t *testing.T) {
	engine := New()
	
	voices := engine.GetVoices()
	if len(voices) != 3 {
		t.Errorf("Expected 3 voices, got %d", len(voices))
	}
	
	// Check first voice details
	if len(voices) > 0 {
		voice := voices[0]
		if voice.ID == "" {
			t.Error("Voice ID should not be empty")
		}
		if voice.Name == "" {
			t.Error("Voice name should not be empty")
		}
		if voice.Language == "" {
			t.Error("Voice language should not be empty")
		}
	}
}

// TestSetVoice tests voice changing.
func TestSetVoice(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	
	// Get available voices
	voices := engine.GetVoices()
	if len(voices) == 0 {
		t.Skip("No voices available")
	}
	
	// Set valid voice
	err := engine.SetVoice(voices[0])
	if err != nil {
		t.Errorf("SetVoice failed for valid voice: %v", err)
	}
	
	// Try invalid voice
	invalidVoice := tts.Voice{ID: "invalid-voice"}
	err = engine.SetVoice(invalidVoice)
	if err == nil {
		t.Error("Expected error for invalid voice")
	}
}

// TestGetCapabilities tests capability reporting.
func TestGetCapabilities(t *testing.T) {
	engine := New()
	
	caps := engine.GetCapabilities()
	
	if !caps.SupportsStreaming {
		t.Error("Mock engine should support streaming")
	}
	
	if len(caps.SupportedFormats) == 0 {
		t.Error("Should have at least one supported format")
	}
	
	if caps.MaxTextLength <= 0 {
		t.Error("MaxTextLength should be positive")
	}
	
	if caps.RequiresNetwork {
		t.Error("Mock engine should not require network")
	}
}

// TestShutdown tests engine shutdown.
func TestShutdown(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	
	// Should be available before shutdown
	if !engine.IsAvailable() {
		t.Error("Engine should be available before shutdown")
	}
	
	err := engine.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	// Should not be available after shutdown
	if engine.IsAvailable() {
		t.Error("Engine should not be available after shutdown")
	}
}

// TestCallCount tests call counting.
func TestCallCount(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	engine.SetDelay(0) // No delay for speed
	
	initialCount := engine.GetCallCount()
	
	// Make some calls
	engine.GenerateAudio("test 1")
	engine.GenerateAudio("test 2")
	engine.GenerateAudioStream("test 3")
	
	finalCount := engine.GetCallCount()
	expectedIncrease := 3
	
	if finalCount-initialCount != expectedIncrease {
		t.Errorf("Expected call count to increase by %d, got %d", 
			expectedIncrease, finalCount-initialCount)
	}
}

// TestConcurrentGeneration tests thread safety.
func TestConcurrentGeneration(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	engine.SetDelay(5 * time.Millisecond)
	
	done := make(chan bool, 10)
	errorChan := make(chan error, 10)
	
	for i := 0; i < 10; i++ {
		go func(n int) {
			defer func() { done <- true }()
			
			text := "test text"
			audio, err := engine.GenerateAudio(text)
			if err != nil {
				errorChan <- err
				return
			}
			
			if audio == nil || len(audio.Data) == 0 {
				errorChan <- errors.New("invalid audio generated")
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Check for errors
	select {
	case err := <-errorChan:
		t.Errorf("Concurrent generation error: %v", err)
	default:
		// No errors
	}
}

// TestEstimateDuration tests duration estimation.
func TestEstimateDuration(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	engine.SetDelay(0) // No delay for accurate timing
	
	tests := []struct {
		text        string
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{"short", 200 * time.Millisecond, 2 * time.Second},
		{"This is a longer text with multiple words", 2 * time.Second, 5 * time.Second},
		{"", 200 * time.Millisecond, 2 * time.Second},
	}
	
	for _, tt := range tests {
		audio, err := engine.GenerateAudio(tt.text)
		if err != nil {
			t.Errorf("GenerateAudio failed for '%s': %v", tt.text, err)
			continue
		}
		
		if audio.Duration < tt.minDuration || audio.Duration > tt.maxDuration {
			t.Errorf("Text '%s': duration %v not in range [%v, %v]", 
				tt.text, audio.Duration, tt.minDuration, tt.maxDuration)
		}
	}
}

// TestStreamWithFailure tests streaming with error injection.
func TestStreamWithFailure(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	
	testError := errors.New("stream error")
	engine.SetFailure(testError)
	
	_, err := engine.GenerateAudioStream("test")
	if err != testError {
		t.Errorf("Expected injected error, got %v", err)
	}
}

// TestDelayConfiguration tests delay configuration.
func TestDelayConfiguration(t *testing.T) {
	engine := New()
	engine.Initialize(tts.EngineConfig{})
	
	// Test with no delay
	engine.SetDelay(0)
	start := time.Now()
	engine.GenerateAudio("test")
	duration := time.Since(start)
	
	if duration > 50*time.Millisecond {
		t.Errorf("Generation took too long with no delay: %v", duration)
	}
	
	// Test with delay
	delay := 100 * time.Millisecond
	engine.SetDelay(delay)
	start = time.Now()
	engine.GenerateAudio("test")
	duration = time.Since(start)
	
	if duration < delay {
		t.Errorf("Generation should take at least %v, took %v", delay, duration)
	}
}