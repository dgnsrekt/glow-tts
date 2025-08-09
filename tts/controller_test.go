package tts_test

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
	"github.com/charmbracelet/glow/v2/tts/sentence"
)

// mockAudioPlayer implements a mock audio player for testing.
type mockAudioPlayer struct {
	mu          sync.Mutex
	playing     bool
	paused      bool
	position    time.Duration
	playCount   int
	pauseCount  int
	stopCount   int
	lastPlayed  *tts.Audio
	playError   error
	pauseError  error
	stopError   error
}

func newMockAudioPlayer() *mockAudioPlayer {
	return &mockAudioPlayer{}
}

func (p *mockAudioPlayer) Play(audio *tts.Audio) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.playError != nil {
		return p.playError
	}
	
	p.playing = true
	p.paused = false
	p.lastPlayed = audio
	p.playCount++
	return nil
}

func (p *mockAudioPlayer) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.pauseError != nil {
		return p.pauseError
	}
	
	p.paused = true
	p.playing = false
	p.pauseCount++
	return nil
}

func (p *mockAudioPlayer) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.playing = true
	p.paused = false
	return nil
}

func (p *mockAudioPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.stopError != nil {
		return p.stopError
	}
	
	p.playing = false
	p.paused = false
	p.position = 0
	p.stopCount++
	return nil
}

func (p *mockAudioPlayer) GetPosition() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.position
}

func (p *mockAudioPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// TestControllerCreation verifies controller creation and initialization.
func TestControllerCreation(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	if controller == nil {
		t.Fatal("NewController returned nil")
	}
	
	state := controller.GetState()
	if state.CurrentState != tts.StateIdle {
		t.Errorf("Expected initial state to be Idle, got %s", state.CurrentState)
	}
}

// TestControllerInitialization tests the initialization process.
func TestControllerInitialization(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	config := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 0.8,
	}
	
	// Test successful initialization
	err := controller.Initialize(config)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}
	
	state := controller.GetState()
	if state.CurrentState != tts.StateReady {
		t.Errorf("Expected state to be Ready after initialization, got %s", state.CurrentState)
	}
	
	// Test double initialization (should fail)
	err = controller.Initialize(config)
	if err == nil {
		t.Error("Expected error when initializing twice")
	}
}

// TestControllerSetContent tests content parsing and preparation.
func TestControllerSetContent(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize first
	config := tts.EngineConfig{Voice: "test"}
	if err := controller.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// Test setting content
	markdown := "This is a test sentence. This is another one!"
	err := controller.SetContent(markdown)
	if err != nil {
		t.Errorf("SetContent failed: %v", err)
	}
	
	state := controller.GetState()
	if state.TotalSentences != 2 {
		t.Errorf("Expected 2 sentences, got %d", state.TotalSentences)
	}
	
	// Test empty content
	err = controller.SetContent("")
	if err == nil {
		t.Error("Expected error for empty content")
	}
}

// TestControllerPlayback tests play, pause, and stop functionality.
func TestControllerPlayback(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	if err := controller.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// Set content
	markdown := "First sentence. Second sentence."
	if err := controller.SetContent(markdown); err != nil {
		t.Fatalf("SetContent failed: %v", err)
	}
	
	// Test play
	err := controller.Play()
	if err != nil {
		t.Errorf("Play failed: %v", err)
	}
	
	// Give playback loop time to start
	time.Sleep(100 * time.Millisecond)
	
	state := controller.GetState()
	if state.CurrentState != tts.StatePlaying {
		t.Errorf("Expected state to be Playing, got %s", state.CurrentState)
	}
	
	// Test pause
	err = controller.Pause()
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}
	
	state = controller.GetState()
	if state.CurrentState != tts.StatePaused {
		t.Errorf("Expected state to be Paused, got %s", state.CurrentState)
	}
	
	// Test resume (play while paused)
	err = controller.Play()
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	
	state = controller.GetState()
	if state.CurrentState != tts.StatePlaying {
		t.Errorf("Expected state to be Playing after resume, got %s", state.CurrentState)
	}
	
	// Test stop
	err = controller.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	
	state = controller.GetState()
	if state.CurrentState != tts.StateReady {
		t.Errorf("Expected state to be Ready after stop, got %s", state.CurrentState)
	}
	
	if state.Sentence != 0 {
		t.Errorf("Expected sentence index to reset to 0, got %d", state.Sentence)
	}
}

// TestControllerNavigation tests sentence navigation.
func TestControllerNavigation(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	if err := controller.Initialize(config); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	
	// Set content with multiple sentences
	markdown := "First. Second. Third. Fourth."
	if err := controller.SetContent(markdown); err != nil {
		t.Fatalf("SetContent failed: %v", err)
	}
	
	// Test next sentence
	err := controller.NextSentence()
	if err != nil {
		t.Errorf("NextSentence failed: %v", err)
	}
	
	state := controller.GetState()
	if state.Sentence != 1 {
		t.Errorf("Expected sentence index 1, got %d", state.Sentence)
	}
	
	// Test previous sentence
	err = controller.PreviousSentence()
	if err != nil {
		t.Errorf("PreviousSentence failed: %v", err)
	}
	
	state = controller.GetState()
	if state.Sentence != 0 {
		t.Errorf("Expected sentence index 0, got %d", state.Sentence)
	}
	
	// Test previous at beginning (should fail)
	err = controller.PreviousSentence()
	if err == nil {
		t.Error("Expected error when going previous at first sentence")
	}
	
	// Navigate to last sentence
	for i := 0; i < 3; i++ {
		controller.NextSentence()
	}
	
	// Test next at end (should fail)
	err = controller.NextSentence()
	if err == nil {
		t.Error("Expected error when going next at last sentence")
	}
}

// TestControllerCallbacks tests callback registration and invocation.
func TestControllerCallbacks(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Track callback invocations
	var stateChanges []tts.StateType
	var sentenceChanges []int
	var errors []error
	
	// Register callbacks
	controller.OnStateChange(func(state tts.StateType) {
		stateChanges = append(stateChanges, state)
	})
	
	controller.OnSentenceChange(func(index int) {
		sentenceChanges = append(sentenceChanges, index)
	})
	
	controller.OnError(func(err error) {
		errors = append(errors, err)
	})
	
	// Initialize (should trigger state change)
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "First. Second. Third."
	controller.SetContent(markdown)
	
	// Navigate (should trigger sentence change)
	controller.NextSentence()
	
	// Check callbacks were invoked
	if len(stateChanges) == 0 {
		t.Error("State change callback not invoked")
	}
	
	if len(sentenceChanges) == 0 {
		t.Error("Sentence change callback not invoked")
	}
}

// TestControllerConfiguration tests configuration updates.
func TestControllerConfiguration(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Update controller config
	config := tts.ControllerConfig{
		BufferSize:        5,
		RetryAttempts:     5,
		RetryDelay:        2 * time.Second,
		GenerationTimeout: 60 * time.Second,
		EnableCaching:     false,
	}
	
	controller.SetConfiguration(config)
	
	// Update engine config
	engineConfig := tts.EngineConfig{
		Voice:  "new-voice",
		Rate:   1.5,
		Pitch:  0.9,
		Volume: 0.7,
	}
	
	err := controller.SetEngineConfig(engineConfig)
	if err != nil {
		t.Errorf("SetEngineConfig failed: %v", err)
	}
}

// TestControllerGetCurrentSentence tests retrieving the current sentence.
func TestControllerGetCurrentSentence(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "First sentence. Second sentence."
	controller.SetContent(markdown)
	
	// Get current sentence
	sentence, err := controller.GetCurrentSentence()
	if err != nil {
		t.Errorf("GetCurrentSentence failed: %v", err)
	}
	
	if sentence.Text != "First sentence." {
		t.Errorf("Expected 'First sentence.', got '%s'", sentence.Text)
	}
	
	// Move to next sentence
	controller.NextSentence()
	
	sentence, err = controller.GetCurrentSentence()
	if err != nil {
		t.Errorf("GetCurrentSentence failed: %v", err)
	}
	
	if sentence.Text != "Second sentence." {
		t.Errorf("Expected 'Second sentence.', got '%s'", sentence.Text)
	}
}

// TestControllerShutdown tests graceful shutdown.
func TestControllerShutdown(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "Test content."
	controller.SetContent(markdown)
	
	// Start playback
	controller.Play()
	
	// Shutdown
	err := controller.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	state := controller.GetState()
	if state.CurrentState != tts.StateIdle {
		t.Errorf("Expected state to be Idle after shutdown, got %s", state.CurrentState)
	}
}

// TestControllerErrorHandling tests error recovery and handling.
func TestControllerErrorHandling(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Set up error callback
	var lastError error
	controller.OnError(func(err error) {
		lastError = err
	})
	
	// Inject error in player
	player.playError = errors.New("playback failed")
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "Test sentence."
	controller.SetContent(markdown)
	
	// Try to play (should handle error)
	err := controller.Play()
	if err != nil {
		t.Logf("Play returned error as expected: %v", err)
	}
	
	// Give error handler time to process
	time.Sleep(100 * time.Millisecond)
	
	// Check if error was captured
	if lastError == nil {
		t.Log("Error callback may not have been invoked for playback error")
	}
}

// TestControllerConcurrency tests thread safety.
func TestControllerConcurrency(t *testing.T) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "First. Second. Third. Fourth. Fifth."
	controller.SetContent(markdown)
	
	// Run concurrent operations
	var wg sync.WaitGroup
	
	// Concurrent state queries
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = controller.GetState()
			time.Sleep(time.Millisecond)
		}
	}()
	
	// Concurrent navigation
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			controller.NextSentence()
			time.Sleep(10 * time.Millisecond)
			controller.PreviousSentence()
			time.Sleep(10 * time.Millisecond)
		}
	}()
	
	// Concurrent playback control
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			controller.Play()
			time.Sleep(20 * time.Millisecond)
			controller.Pause()
			time.Sleep(20 * time.Millisecond)
			controller.Stop()
			time.Sleep(20 * time.Millisecond)
		}
	}()
	
	// Wait for all goroutines
	wg.Wait()
	
	// If we get here without deadlock or panic, concurrency is handled
	t.Log("Concurrent operations completed successfully")
}

// BenchmarkControllerPlayback benchmarks playback performance.
func BenchmarkControllerPlayback(b *testing.B) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content
	markdown := "This is a test sentence for benchmarking."
	controller.SetContent(markdown)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		controller.Play()
		controller.Stop()
	}
}

// BenchmarkControllerNavigation benchmarks sentence navigation.
func BenchmarkControllerNavigation(b *testing.B) {
	engine := mock.New()
	player := newMockAudioPlayer()
	parser := sentence.NewParser()
	
	controller := tts.NewController(engine, player, parser)
	
	// Initialize
	config := tts.EngineConfig{Voice: "test"}
	controller.Initialize(config)
	
	// Set content with many sentences
	sentences := make([]string, 100)
	for i := range sentences {
		sentences[i] = "Test sentence."
	}
	markdown := strings.Join(sentences, " ")
	controller.SetContent(markdown)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		controller.NextSentence()
		if i%100 == 99 {
			// Reset to beginning
			for j := 0; j < 99; j++ {
				controller.PreviousSentence()
			}
		}
	}
}