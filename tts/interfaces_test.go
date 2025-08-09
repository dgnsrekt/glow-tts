package tts_test

import (
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
)

// TestEngineInterface verifies the Engine interface is properly implemented.
func TestEngineInterface(t *testing.T) {
	// Create mock engine
	var engine tts.Engine = mock.New()

	// Test initialization
	err := engine.Initialize(tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 0.8,
	})
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	// Test availability
	if !engine.IsAvailable() {
		t.Error("Engine should be available after initialization")
	}

	// Test voice operations
	voices := engine.GetVoices()
	if len(voices) == 0 {
		t.Error("Engine should have at least one voice")
	}

	err = engine.SetVoice(voices[0])
	if err != nil {
		t.Errorf("SetVoice failed: %v", err)
	}

	// Test capabilities
	caps := engine.GetCapabilities()
	if caps.MaxTextLength <= 0 {
		t.Error("MaxTextLength should be positive")
	}

	// Test audio generation
	audio, err := engine.GenerateAudio("Test text")
	if err != nil {
		t.Errorf("GenerateAudio failed: %v", err)
	}
	if audio == nil || len(audio.Data) == 0 {
		t.Error("Generated audio should have data")
	}

	// Test streaming
	ch, err := engine.GenerateAudioStream("Test streaming")
	if err != nil {
		t.Errorf("GenerateAudioStream failed: %v", err)
	}

	// Read at least one chunk
	select {
	case chunk := <-ch:
		if chunk.Data == nil {
			t.Error("Audio chunk should have data")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for audio chunk")
	}

	// Test shutdown
	err = engine.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}

// TestAudioPlayerInterface verifies the AudioPlayer interface contracts.
func TestAudioPlayerInterface(t *testing.T) {
	// This test verifies the interface can be implemented
	// Actual implementation testing will be in audio package tests
	var _ tts.AudioPlayer = (*mockPlayer)(nil)
}

// mockPlayer is a minimal AudioPlayer implementation for testing
type mockPlayer struct{}

func (p *mockPlayer) Play(audio *tts.Audio) error          { return nil }
func (p *mockPlayer) Pause() error                         { return nil }
func (p *mockPlayer) Resume() error                        { return nil }
func (p *mockPlayer) Stop() error                          { return nil }
func (p *mockPlayer) GetPosition() time.Duration           { return 0 }
func (p *mockPlayer) IsPlaying() bool                      { return false }

// TestSentenceParserInterface verifies the SentenceParser interface contracts.
func TestSentenceParserInterface(t *testing.T) {
	var _ tts.SentenceParser = (*mockParser)(nil)
}

// mockParser is a minimal SentenceParser implementation for testing
type mockParser struct{}

func (p *mockParser) Parse(markdown string) []tts.Sentence {
	return []tts.Sentence{{Text: "Test"}}
}
func (p *mockParser) EstimateDuration(text string) time.Duration {
	return time.Second
}

// TestSynchronizerInterface verifies the Synchronizer interface contracts.
func TestSynchronizerInterface(t *testing.T) {
	var _ tts.Synchronizer = (*mockSync)(nil)
}

// mockSync is a minimal Synchronizer implementation for testing
type mockSync struct{}

func (s *mockSync) Start(sentences []tts.Sentence, player tts.AudioPlayer) {}
func (s *mockSync) Stop()                                                  {}
func (s *mockSync) GetCurrentSentence() int                                { return 0 }
func (s *mockSync) OnSentenceChange(callback func(int))                    {}