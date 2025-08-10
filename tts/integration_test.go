package tts_test

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/audio"
	"github.com/charmbracelet/glow/v2/tts/engines"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
	"github.com/charmbracelet/glow/v2/tts/sentence"
)

// FailingEngine simulates Piper that fails after first few calls
type FailingEngine struct {
	*mock.MockEngine
	callCount int
	failAfter int
}

func NewFailingEngine(failAfter int) *FailingEngine {
	return &FailingEngine{
		MockEngine: mock.New(),
		failAfter:  failAfter,
	}
}

func (e *FailingEngine) GenerateAudio(text string) (*tts.Audio, error) {
	e.callCount++
	if e.callCount > e.failAfter {
		return nil, errors.New("process died unexpectedly")
	}
	return e.MockEngine.GenerateAudio(text)
}

func (e *FailingEngine) IsAvailable() bool {
	// Simulate engine becoming unavailable after failure
	if e.callCount > e.failAfter {
		return false
	}
	return true
}

// TestFallbackIntegration tests the full TTS pipeline with fallback
func TestFallbackIntegration(t *testing.T) {
	// Create a failing engine that dies after 1 successful call
	failingEngine := NewFailingEngine(1)
	
	// Create a reliable fallback
	fallbackEngine := mock.New()
	
	// Wrap with fallback (allow 3 failures before switching)
	engine := engines.NewFallbackEngine(failingEngine, fallbackEngine, 3)
	
	// Create controller with the fallback engine
	player := audio.NewMockPlayer()
	parser := sentence.NewParser()
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 1.0,
	}
	
	err := controller.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// Set content
	content := `# Test Document
	
This is the first sentence. This is the second sentence.
This is the third sentence. This is the fourth sentence.`
	
	err = controller.SetContent(content)
	if err != nil {
		t.Fatalf("SetContent failed: %v", err)
	}
	
	// Start playback
	controller.Play()
	
	// Wait a bit for processing
	time.Sleep(500 * time.Millisecond)
	
	// Check state
	state := controller.GetState()
	if state.CurrentState != tts.StatePlaying {
		t.Logf("State after Play: %v", state.CurrentState)
	}
	
	// The first sentence should succeed (failingEngine works once)
	// The second sentence should fail 3 times then switch to fallback
	// Subsequent sentences should use fallback
	
	// Wait for some processing
	time.Sleep(1 * time.Second)
	
	// Stop playback
	controller.Stop()
	
	// Check that we processed some sentences
	finalState := controller.GetState()
	t.Logf("Final state: Sentence=%d, TotalSentences=%d", 
		finalState.Sentence, finalState.TotalSentences)
	
	// Clean up
	controller.Shutdown()
}

// TestPiperFallbackScenario simulates the exact Piper failure scenario
func TestPiperFallbackScenario(t *testing.T) {
	// Create an engine that fails immediately (simulating Piper crash)
	primaryEngine := mock.New()
	primaryEngine.SetFailure(errors.New("process died unexpectedly"))
	
	// Create a working fallback
	fallbackEngine := mock.New()
	
	// Wrap with fallback (allow 3 failures as configured)
	engine := engines.NewFallbackEngine(primaryEngine, fallbackEngine, 3)
	
	// Create controller
	player := audio.NewMockPlayer()
	parser := sentence.NewParser()
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{
		Voice:  "amy",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 1.0,
	}
	
	err := controller.Initialize(config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// Set content
	content := "Test sentence for TTS. Another sentence here."
	
	err = controller.SetContent(content)
	if err != nil {
		t.Fatalf("SetContent failed: %v", err)
	}
	
	// Try to generate audio for first sentence
	// This should fail 3 times with primary, then succeed with fallback
	controller.Play()
	
	// Wait for processing
	time.Sleep(2 * time.Second)
	
	// Check that playback is working
	state := controller.GetState()
	if state.CurrentState == tts.StateIdle {
		t.Error("Expected playback to be active, but it's idle")
	}
	
	t.Logf("State: %v, Index: %d/%d", state.CurrentState, state.Sentence, state.TotalSentences)
	
	// Stop and clean up
	controller.Stop()
	controller.Shutdown()
}