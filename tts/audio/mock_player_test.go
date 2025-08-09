package audio_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/audio"
)

// TestMockPlayerBasicPlayback tests basic play, pause, resume, stop operations.
func TestMockPlayerBasicPlayback(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	// Create test audio
	testAudio := &tts.Audio{
		Data:       []byte("test"),
		Format:     tts.FormatPCM16,
		SampleRate: 44100,
		Channels:   2,
		Duration:   2 * time.Second,
	}

	// Test initial state
	if player.IsPlaying() {
		t.Error("Player should not be playing initially")
	}
	if pos := player.GetPosition(); pos != 0 {
		t.Errorf("Initial position should be 0, got %v", pos)
	}

	// Test Play
	err := player.Play(testAudio)
	if err != nil {
		t.Errorf("Play failed: %v", err)
	}
	if !player.IsPlaying() {
		t.Error("Player should be playing after Play")
	}

	// Wait a bit for position to advance
	time.Sleep(50 * time.Millisecond)
	pos1 := player.GetPosition()
	if pos1 == 0 {
		t.Error("Position should advance while playing")
	}

	// Test Pause
	err = player.Pause()
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}
	if player.IsPlaying() {
		t.Error("Player should not be playing after Pause")
	}

	// Position should not advance while paused
	pausedPos := player.GetPosition()
	time.Sleep(50 * time.Millisecond)
	if player.GetPosition() != pausedPos {
		t.Error("Position should not advance while paused")
	}

	// Test Resume
	err = player.Resume()
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	if !player.IsPlaying() {
		t.Error("Player should be playing after Resume")
	}

	// Position should advance after resume
	time.Sleep(50 * time.Millisecond)
	pos2 := player.GetPosition()
	if pos2 <= pausedPos {
		t.Error("Position should advance after Resume")
	}

	// Test Stop
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	if player.IsPlaying() {
		t.Error("Player should not be playing after Stop")
	}
	if player.GetPosition() != 0 {
		t.Error("Position should reset to 0 after Stop")
	}
}

// TestMockPlayerTimingAccuracy tests the accuracy of timing simulation.
func TestMockPlayerTimingAccuracy(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 2 * time.Second,
	}

	// Start playing first
	err := player.Play(testAudio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Test normal speed first
	time.Sleep(100 * time.Millisecond)
	pos1 := player.GetPosition()
	t.Logf("Position after 100ms at 1x speed: %v", pos1)
	
	// Position should be around 100ms
	if pos1 < 50*time.Millisecond || pos1 > 150*time.Millisecond {
		t.Errorf("Normal speed timing inaccurate: expected ~100ms, got %v", pos1)
	}
	
	// Now test with speed multiplier
	player.Stop()
	player.SetSpeedMultiplier(10.0)
	
	err = player.Play(testAudio)
	if err != nil {
		t.Fatalf("Play failed after speed change: %v", err)
	}
	
	// Wait for 100ms real time = 1s simulated time at 10x speed
	time.Sleep(100 * time.Millisecond)

	pos2 := player.GetPosition()
	t.Logf("Position after 100ms at 10x speed: %v", pos2)
	
	// Allow 30% tolerance for timing at high speed
	expectedMin := 700 * time.Millisecond
	expectedMax := 1300 * time.Millisecond

	if pos2 < expectedMin || pos2 > expectedMax {
		t.Errorf("High speed timing inaccurate: expected ~1s, got %v", pos2)
	}
}

// TestMockPlayerAutoStop tests automatic stop at end of duration.
func TestMockPlayerAutoStop(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 200 * time.Millisecond,
	}

	err := player.Play(testAudio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Wait for playback to complete at normal speed
	time.Sleep(250 * time.Millisecond)

	if player.IsPlaying() {
		t.Error("Player should auto-stop at end of duration")
	}

	pos := player.GetPosition()
	// Allow small tolerance for final position
	if pos < 199*time.Millisecond || pos > 201*time.Millisecond {
		t.Errorf("Final position should equal duration: got %v, want ~%v", pos, testAudio.Duration)
	}
}

// TestMockPlayerCallbacks tests callback functionality.
func TestMockPlayerCallbacks(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	var (
		playCount   int
		pauseCount  int
		resumeCount int
		stopCount   int
		tickCount   int
		mu          sync.Mutex
	)

	callbacks := audio.MockCallbacks{
		OnPlay: func(audio *tts.Audio) {
			mu.Lock()
			playCount++
			mu.Unlock()
		},
		OnPause: func() {
			mu.Lock()
			pauseCount++
			mu.Unlock()
		},
		OnResume: func() {
			mu.Lock()
			resumeCount++
			mu.Unlock()
		},
		OnStop: func() {
			mu.Lock()
			stopCount++
			mu.Unlock()
		},
		OnTick: func(position time.Duration) {
			mu.Lock()
			tickCount++
			mu.Unlock()
		},
	}

	player.SetCallbacks(callbacks)

	testAudio := &tts.Audio{
		Duration: 1 * time.Second,
	}

	// Test callbacks are triggered
	player.Play(testAudio)
	time.Sleep(30 * time.Millisecond) // Allow some ticks
	player.Pause()
	player.Resume()
	player.Stop()

	// Give callbacks time to execute
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if playCount != 1 {
		t.Errorf("OnPlay should be called once, got %d", playCount)
	}
	if pauseCount != 1 {
		t.Errorf("OnPause should be called once, got %d", pauseCount)
	}
	if resumeCount != 1 {
		t.Errorf("OnResume should be called once, got %d", resumeCount)
	}
	if stopCount != 1 {
		t.Errorf("OnStop should be called once, got %d", stopCount)
	}
	if tickCount == 0 {
		t.Error("OnTick should be called at least once")
	}
}

// TestMockPlayerHistory tests event history recording.
func TestMockPlayerHistory(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 1 * time.Second,
	}

	// Perform operations
	player.Play(testAudio)
	player.Pause()
	player.Resume()
	player.Stop()

	// Check history
	history := player.GetHistory()
	if len(history) != 4 {
		t.Errorf("Expected 4 events in history, got %d", len(history))
	}

	expectedEvents := []string{"play", "pause", "resume", "stop"}
	for i, event := range history {
		if event.Type != expectedEvents[i] {
			t.Errorf("Event %d: expected %s, got %s", i, expectedEvents[i], event.Type)
		}
	}

	// Test clear history
	player.ClearHistory()
	history = player.GetHistory()
	if len(history) != 0 {
		t.Errorf("History should be empty after clear, got %d events", len(history))
	}
}

// TestMockPlayerErrorInjection tests error injection for testing.
func TestMockPlayerErrorInjection(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 1 * time.Second,
	}

	// Inject play error
	expectedErr := errors.New("test play error")
	player.InjectError("play", expectedErr)
	
	err := player.Play(testAudio)
	if err != expectedErr {
		t.Errorf("Expected injected error, got %v", err)
	}

	// Clear errors and try again
	player.ClearErrors()
	err = player.Play(testAudio)
	if err != nil {
		t.Errorf("Play should succeed after clearing errors: %v", err)
	}

	// Inject pause error
	player.InjectError("pause", errors.New("test pause error"))
	err = player.Pause()
	if err == nil {
		t.Error("Expected pause error")
	}
}

// TestMockPlayerSetPosition tests manual position setting.
func TestMockPlayerSetPosition(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 10 * time.Second,
	}

	player.Play(testAudio)

	// Set position to 5 seconds
	player.SetPosition(5 * time.Second)
	pos := player.GetPosition()
	
	// Allow small tolerance
	if pos < 4900*time.Millisecond || pos > 5100*time.Millisecond {
		t.Errorf("SetPosition failed: expected ~5s, got %v", pos)
	}

	// Test boundary conditions
	player.SetPosition(-1 * time.Second)
	if player.GetPosition() != 0 {
		t.Error("Negative position should clamp to 0")
	}

	player.SetPosition(20 * time.Second)
	if player.GetPosition() != testAudio.Duration {
		t.Error("Position beyond duration should clamp to duration")
	}
}

// TestMockPlayerWaitForPosition tests waiting for specific position.
func TestMockPlayerWaitForPosition(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	// Use high speed for faster test
	player.SetSpeedMultiplier(10.0)

	testAudio := &tts.Audio{
		Duration: 5 * time.Second,
	}

	player.Play(testAudio)

	// Wait for 1 second position (100ms real time at 10x)
	err := player.WaitForPosition(1*time.Second, 500*time.Millisecond)
	if err != nil {
		t.Errorf("WaitForPosition failed: %v", err)
	}

	pos := player.GetPosition()
	if pos < 1*time.Second {
		t.Errorf("Position should be at least 1s, got %v", pos)
	}

	// Test timeout
	err = player.WaitForPosition(10*time.Second, 100*time.Millisecond)
	if err == nil {
		t.Error("WaitForPosition should timeout")
	}
}

// TestMockPlayerSimulateCompletion tests simulating playback completion.
func TestMockPlayerSimulateCompletion(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 10 * time.Second,
	}

	player.Play(testAudio)
	time.Sleep(20 * time.Millisecond) // Let it play a bit

	// Simulate completion
	player.SimulateCompletion()

	if player.IsPlaying() {
		t.Error("Player should stop after simulated completion")
	}

	pos := player.GetPosition()
	if pos != testAudio.Duration {
		t.Errorf("Position should be at duration after completion: got %v, want %v", 
			pos, testAudio.Duration)
	}
}

// TestMockPlayerConcurrentAccess tests thread safety.
func TestMockPlayerConcurrentAccess(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 5 * time.Second,
	}

	var wg sync.WaitGroup
	errs := make([]error, 0)
	var errorsMu sync.Mutex

	// Concurrent operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Try various operations
			player.Play(testAudio)
			player.GetPosition()
			player.IsPlaying()
			player.Pause()
			player.Resume()
			player.GetState()
			player.Stop()
			
			// Record any panics as errors
			if r := recover(); r != nil {
				errorsMu.Lock()
				errs = append(errs, errors.New("panic in concurrent operation"))
				errorsMu.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		t.Errorf("Concurrent access caused %d errors", len(errs))
	}
}

// TestMockPlayerStateTransitions tests valid state transitions.
func TestMockPlayerStateTransitions(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 1 * time.Second,
	}

	// Test invalid transitions
	err := player.Pause() // Can't pause when not playing
	if err == nil {
		t.Error("Pause should fail when not playing")
	}

	err = player.Resume() // Can't resume when not paused
	if err == nil {
		t.Error("Resume should fail when not paused")
	}

	// Start playing
	player.Play(testAudio)

	// Can't play again while playing
	err = player.Play(testAudio)
	if err == nil {
		t.Error("Play should fail when already playing")
	}

	// Pause
	player.Pause()

	// Can't pause again
	err = player.Pause()
	if err == nil {
		t.Error("Pause should fail when already paused")
	}

	// Can resume
	err = player.Resume()
	if err != nil {
		t.Errorf("Resume should succeed when paused: %v", err)
	}

	// Can't resume when not paused
	err = player.Resume()
	if err == nil {
		t.Error("Resume should fail when not paused")
	}

	// Stop always succeeds
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop should always succeed: %v", err)
	}

	// Stop again should still succeed (idempotent)
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop should be idempotent: %v", err)
	}
}

// TestMockPlayerGetState tests state retrieval.
func TestMockPlayerGetState(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 1 * time.Second,
	}

	// Initial state
	playing, paused, pos, dur := player.GetState()
	if playing || paused || pos != 0 || dur != 0 {
		t.Error("Initial state incorrect")
	}

	// Playing state
	player.Play(testAudio)
	playing, paused, pos, dur = player.GetState()
	if !playing || paused || dur != 1*time.Second {
		t.Error("Playing state incorrect")
	}

	// Paused state
	player.Pause()
	playing, paused, _, _ = player.GetState()
	if !playing || !paused {
		t.Error("Paused state incorrect")
	}

	// Stopped state
	player.Stop()
	playing, paused, pos, _ = player.GetState()
	if playing || paused || pos != 0 {
		t.Error("Stopped state incorrect")
	}
}

// TestMockPlayerNilAudio tests handling of nil audio.
func TestMockPlayerNilAudio(t *testing.T) {
	player := audio.NewMockPlayer()
	defer player.Close()

	// Playing nil audio should not panic
	err := player.Play(nil)
	if err != nil {
		t.Errorf("Play with nil audio failed: %v", err)
	}

	// Should handle operations gracefully
	player.Pause()
	player.Resume()
	player.Stop()
	
	// No panic = success
}

// BenchmarkMockPlayerPositionUpdate benchmarks position update performance.
func BenchmarkMockPlayerPositionUpdate(b *testing.B) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 10 * time.Second,
	}

	player.Play(testAudio)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.GetPosition()
	}
}

// BenchmarkMockPlayerStateChange benchmarks state change operations.
func BenchmarkMockPlayerStateChange(b *testing.B) {
	player := audio.NewMockPlayer()
	defer player.Close()

	testAudio := &tts.Audio{
		Duration: 10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player.Play(testAudio)
		player.Pause()
		player.Resume()
		player.Stop()
	}
}