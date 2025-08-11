package tts

import (
	"context"
	"testing"
	"time"
)

// Compile-time interface compliance checks
// These ensure our future implementations will satisfy the interfaces

type mockEngine struct{}

func (m *mockEngine) Synthesize(ctx context.Context, text string, speed float64) ([]byte, error) {
	return nil, nil
}

func (m *mockEngine) GetInfo() EngineInfo {
	return EngineInfo{
		Name:       "mock",
		Version:    "1.0.0",
		SampleRate: 44100,
		Channels:   1,
		BitDepth:   16,
	}
}

func (m *mockEngine) Validate() error {
	return nil
}

func (m *mockEngine) Close() error {
	return nil
}

type mockPlayer struct{}

func (m *mockPlayer) Play(audio []byte) error              { return nil }
func (m *mockPlayer) Pause() error                          { return nil }
func (m *mockPlayer) Resume() error                         { return nil }
func (m *mockPlayer) Stop() error                           { return nil }
func (m *mockPlayer) IsPlaying() bool                       { return false }
func (m *mockPlayer) GetPosition() time.Duration            { return 0 }
func (m *mockPlayer) SetVolume(volume float64) error        { return nil }
func (m *mockPlayer) Close() error                          { return nil }

type mockCache struct{}

func (m *mockCache) Get(key string) ([]byte, bool)     { return nil, false }
func (m *mockCache) Put(key string, audio []byte) error { return nil }
func (m *mockCache) Delete(key string) error            { return nil }
func (m *mockCache) Clear() error                       { return nil }
func (m *mockCache) Size() int64                        { return 0 }
func (m *mockCache) Stats() CacheStats                  { return CacheStats{} }

type mockQueue struct{}

func (m *mockQueue) Enqueue(sentence Sentence, priority bool) error { return nil }
func (m *mockQueue) Dequeue() (Sentence, error)                     { return Sentence{}, nil }
func (m *mockQueue) Peek() (Sentence, error)                        { return Sentence{}, nil }
func (m *mockQueue) Size() int                                      { return 0 }
func (m *mockQueue) Clear()                                         {}
func (m *mockQueue) SetLookahead(count int)                         {}

type mockParser struct{}

func (m *mockParser) Parse(markdown string) ([]Sentence, error) { return nil, nil }
func (m *mockParser) StripMarkdown(text string) string          { return text }

type mockController struct{}

func (m *mockController) Start(ctx context.Context, engineType EngineType) error { return nil }
func (m *mockController) Stop() error                                            { return nil }
func (m *mockController) ProcessDocument(content string) error                   { return nil }
func (m *mockController) Play() error                                            { return nil }
func (m *mockController) Pause() error                                           { return nil }
func (m *mockController) NextSentence() error                                    { return nil }
func (m *mockController) PreviousSentence() error                                { return nil }
func (m *mockController) SetSpeed(speed float64) error                           { return nil }
func (m *mockController) GetState() State                                        { return StateIdle }
func (m *mockController) GetProgress() Progress                                  { return Progress{} }

type mockSpeedController struct{}

func (m *mockSpeedController) GetSpeed() float64        { return 1.0 }
func (m *mockSpeedController) SetSpeed(speed float64) error { return nil }
func (m *mockSpeedController) Increase() float64        { return 1.25 }
func (m *mockSpeedController) Decrease() float64        { return 0.75 }
func (m *mockSpeedController) ToPiperScale() string     { return "1.0" }
func (m *mockSpeedController) ToGoogleRate() float64    { return 1.0 }

// Ensure all mock implementations satisfy their interfaces
var (
	_ TTSEngine        = (*mockEngine)(nil)
	_ AudioPlayer      = (*mockPlayer)(nil)
	_ AudioCache       = (*mockCache)(nil)
	_ SentenceQueue    = (*mockQueue)(nil)
	_ Parser           = (*mockParser)(nil)
	_ Controller       = (*mockController)(nil)
	_ SpeedController  = (*mockSpeedController)(nil)
)

// TestInterfaceCompilation verifies that all interfaces compile
func TestInterfaceCompilation(t *testing.T) {
	// This test just ensures everything compiles
	t.Log("All interfaces compile successfully")
}

// TestErrorTypes verifies error types work correctly
func TestErrorTypes(t *testing.T) {
	// Test TTSError creation
	err := NewTTSError(ErrorCodeEngineFailure, "test error", nil)
	if err == nil {
		t.Fatal("Expected error to be created")
	}
	
	// Test error with context
	err = err.WithContext("engine", "piper")
	if err.Context["engine"] != "piper" {
		t.Errorf("Expected context to contain engine=piper")
	}
	
	// Test fatal error detection
	fatalErr := NewTTSError(ErrorCodeAudioDevice, "device error", nil)
	if !fatalErr.IsFatal() {
		t.Errorf("Expected audio device error to be fatal")
	}
	
	// Test retryable error detection
	retryErr := NewTTSError(ErrorCodeTimeout, "timeout", nil)
	if !retryErr.IsRetryable() {
		t.Errorf("Expected timeout error to be retryable")
	}
}

// TestStateString verifies state string conversion
func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "idle"},
		{StateInitializing, "initializing"},
		{StateReady, "ready"},
		{StateProcessing, "processing"},
		{StatePlaying, "playing"},
		{StatePaused, "paused"},
		{StateStopping, "stopping"},
		{StateError, "error"},
	}
	
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State.String() = %v, want %v", got, tt.expected)
		}
	}
}

// TestProgressCalculation verifies progress percentage calculation
func TestProgressCalculation(t *testing.T) {
	p := Progress{
		CurrentSentence: 5,
		TotalSentences:  10,
		CurrentPosition: 0,
		TotalDuration:   0,
	}
	
	// Should be 50% complete (5 out of 10 sentences)
	if percent := p.PercentComplete(); percent != 50.0 {
		t.Errorf("Expected 50%% complete, got %.2f%%", percent)
	}
	
	// Test with no sentences
	p2 := Progress{TotalSentences: 0}
	if percent := p2.PercentComplete(); percent != 0 {
		t.Errorf("Expected 0%% for no sentences, got %.2f%%", percent)
	}
}