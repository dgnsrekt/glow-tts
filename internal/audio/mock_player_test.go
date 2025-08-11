package audio

import (
	"sync"
	"testing"
	"time"
)

func TestMockPlayer_BasicPlayback(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Test initial state
	if player.GetState() != StateStopped {
		t.Errorf("Initial state should be Stopped, got %v", player.GetState())
	}

	if player.IsPlaying() {
		t.Error("Player should not be playing initially")
	}

	// Test playing audio
	audio := make([]byte, 4410) // 0.05 seconds at 44100Hz, 16-bit mono
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	if !player.IsPlaying() {
		t.Error("Player should be playing after Play()")
	}

	if player.GetState() != StatePlaying {
		t.Errorf("State should be Playing, got %v", player.GetState())
	}

	// Test stop
	err = player.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if player.IsPlaying() {
		t.Error("Player should not be playing after Stop()")
	}

	if player.GetState() != StateStopped {
		t.Errorf("State should be Stopped after Stop(), got %v", player.GetState())
	}
}

func TestMockPlayer_PauseResume(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Create 1 second of audio
	audio := make([]byte, 88200) // 1 second at 44100Hz, 16-bit mono

	// Start playing
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Let it play for a bit
	time.Sleep(200 * time.Millisecond)

	// Pause
	err = player.Pause()
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	if player.GetState() != StatePaused {
		t.Errorf("State should be Paused, got %v", player.GetState())
	}

	pausePosition := player.GetPosition()
	if pausePosition == 0 {
		t.Error("Position should be > 0 after playing for a bit")
	}

	// Wait a bit while paused
	time.Sleep(100 * time.Millisecond)

	// Position shouldn't change while paused
	if player.GetPosition() != pausePosition {
		t.Error("Position should not change while paused")
	}

	// Resume
	err = player.Resume()
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if player.GetState() != StatePlaying {
		t.Errorf("State should be Playing after resume, got %v", player.GetState())
	}

	// Let it play more
	time.Sleep(100 * time.Millisecond)

	// Position should have increased
	resumePosition := player.GetPosition()
	if resumePosition <= pausePosition {
		t.Errorf("Position should increase after resume: paused=%v, resumed=%v", 
			pausePosition, resumePosition)
	}
}

func TestMockPlayer_PositionTracking(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Create 2 seconds of audio
	audio := make([]byte, 176400) // 2 seconds at 44100Hz, 16-bit mono

	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Check position increases over time
	positions := []time.Duration{}
	for i := 0; i < 5; i++ {
		time.Sleep(100 * time.Millisecond)
		pos := player.GetPosition()
		positions = append(positions, pos)
	}

	// Verify positions are increasing
	for i := 1; i < len(positions); i++ {
		if positions[i] <= positions[i-1] {
			t.Errorf("Position should increase: positions[%d]=%v, positions[%d]=%v",
				i-1, positions[i-1], i, positions[i])
		}
	}

	// Position should be approximately 500ms after 500ms of playing
	// (accounting for test overhead)
	finalPos := positions[len(positions)-1]
	if finalPos < 400*time.Millisecond || finalPos > 600*time.Millisecond {
		t.Errorf("Position after ~500ms should be around 500ms, got %v", finalPos)
	}
}

func TestMockPlayer_VolumeControl(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Initial volume should be 1.0
	if player.GetVolume() != 1.0 {
		t.Errorf("Initial volume should be 1.0, got %f", player.GetVolume())
	}

	// Set valid volume
	err := player.SetVolume(0.5)
	if err != nil {
		t.Fatalf("SetVolume(0.5) failed: %v", err)
	}

	if player.GetVolume() != 0.5 {
		t.Errorf("Volume should be 0.5, got %f", player.GetVolume())
	}

	// Test invalid volumes
	err = player.SetVolume(-0.1)
	if err == nil {
		t.Error("SetVolume(-0.1) should fail")
	}

	err = player.SetVolume(1.1)
	if err == nil {
		t.Error("SetVolume(1.1) should fail")
	}

	// Volume should remain unchanged after invalid set
	if player.GetVolume() != 0.5 {
		t.Errorf("Volume should still be 0.5 after invalid set, got %f", player.GetVolume())
	}
}

func TestMockPlayer_StateTransitions(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	audio := make([]byte, 88200) // 1 second

	// Invalid transitions from Stopped
	err := player.Pause()
	if err == nil {
		t.Error("Pause from Stopped should fail")
	}

	err = player.Resume()
	if err == nil {
		t.Error("Resume from Stopped should fail")
	}

	// Start playing
	player.Play(audio)

	// Invalid transition from Playing
	err = player.Resume()
	if err == nil {
		t.Error("Resume from Playing should fail")
	}

	// Pause
	player.Pause()

	// Invalid transition from Paused
	err = player.Pause()
	if err == nil {
		t.Error("Pause from Paused should fail")
	}

	// Close and verify state
	player.Close()
	if player.GetState() != StateClosed {
		t.Errorf("State should be Closed after Close(), got %v", player.GetState())
	}

	// Should not be able to play after close
	err = player.Play(audio)
	if err == nil {
		t.Error("Play after Close should fail")
	}
}

func TestMockPlayer_Callbacks(t *testing.T) {
	playCount := 0
	pauseCount := 0
	resumeCount := 0
	stopCount := 0
	closeCount := 0

	callbacks := MockCallbacks{
		OnPlay: func(audio []byte) {
			playCount++
		},
		OnPause: func() {
			pauseCount++
		},
		OnResume: func() {
			resumeCount++
		},
		OnStop: func() {
			stopCount++
		},
		OnClose: func() {
			closeCount++
		},
	}

	player := NewMockPlayer(callbacks)

	audio := make([]byte, 88200)

	// Test callbacks are called
	player.Play(audio)
	if playCount != 1 {
		t.Errorf("OnPlay should be called once, got %d", playCount)
	}

	player.Pause()
	if pauseCount != 1 {
		t.Errorf("OnPause should be called once, got %d", pauseCount)
	}

	player.Resume()
	if resumeCount != 1 {
		t.Errorf("OnResume should be called once, got %d", resumeCount)
	}

	player.Stop()
	if stopCount != 1 {
		t.Errorf("OnStop should be called once, got %d", stopCount)
	}

	player.Close()
	if closeCount != 1 {
		t.Errorf("OnClose should be called once, got %d", closeCount)
	}
}

func TestMockPlayer_ErrorSimulation(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Enable error simulation with 100% error rate
	player.SetSimulateErrors(true, 1.0)

	audio := make([]byte, 88200)

	// First play should fail (play count starts at 0, 0 % 1 == 0)
	err := player.Play(audio)
	if err == nil {
		t.Error("Play should fail with error simulation enabled")
	}

	// Disable error simulation
	player.SetSimulateErrors(false, 0)

	// Now play should succeed
	err = player.Play(audio)
	if err != nil {
		t.Errorf("Play should succeed with error simulation disabled: %v", err)
	}
}

func TestMockPlayer_Metrics(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	audio := make([]byte, 88200)

	// Perform operations
	player.Play(audio)
	player.Pause()
	player.Resume()
	player.Stop()
	player.Play(audio)
	player.Stop()

	// Check metrics
	metrics := player.GetMetrics()

	if metrics.PlayCount != 2 {
		t.Errorf("PlayCount should be 2, got %d", metrics.PlayCount)
	}

	if metrics.PauseCount != 1 {
		t.Errorf("PauseCount should be 1, got %d", metrics.PauseCount)
	}

	if metrics.ResumeCount != 1 {
		t.Errorf("ResumeCount should be 1, got %d", metrics.ResumeCount)
	}

	if metrics.StopCount != 2 {
		t.Errorf("StopCount should be 2, got %d", metrics.StopCount)
	}
}

func TestMockPlayer_NaturalCompletion(t *testing.T) {
	stopCalled := false
	callbacks := MockCallbacks{
		OnStop: func() {
			stopCalled = true
		},
	}

	player := NewMockPlayer(callbacks)
	defer player.Close()

	// Speed up playback for faster test
	player.SetDelayFactor(0.1) // 10x speed

	// Create short audio (100ms at normal speed, 10ms at 10x speed)
	audio := make([]byte, 8820) // 0.1 seconds at 44100Hz

	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Wait for natural completion
	completed := player.WaitForCompletion(500 * time.Millisecond)
	if !completed {
		t.Error("Playback should complete naturally")
	}

	// State should be Stopped
	if player.GetState() != StateStopped {
		t.Errorf("State should be Stopped after natural completion, got %v", player.GetState())
	}

	// Stop callback should have been called
	if !stopCalled {
		t.Error("OnStop callback should be called on natural completion")
	}

	// Position should be at the end
	pos := player.GetPosition()
	expectedDuration := 100 * time.Millisecond
	if pos < expectedDuration-10*time.Millisecond || pos > expectedDuration+10*time.Millisecond {
		t.Errorf("Final position should be around %v, got %v", expectedDuration, pos)
	}
}

func TestMockPlayer_ConcurrentOperations(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	audio := make([]byte, 176400) // 2 seconds

	// Start playing
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Initial play failed: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Multiple concurrent operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Try various operations
			switch id % 4 {
			case 0:
				player.GetPosition()
			case 1:
				player.GetState()
			case 2:
				player.IsPlaying()
			case 3:
				player.GetVolume()
			}
		}(i)
	}

	// Control operations in separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond)
		if err := player.Pause(); err != nil {
			errors <- err
		}
		time.Sleep(50 * time.Millisecond)
		if err := player.Resume(); err != nil {
			errors <- err
		}
		time.Sleep(50 * time.Millisecond)
		if err := player.Stop(); err != nil {
			errors <- err
		}
	}()

	// Wait for completion
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case err := <-errors:
		t.Fatalf("Concurrent operation failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestMockPlayer_AudioDataStorage(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Create test audio with specific pattern
	audio := make([]byte, 100)
	for i := range audio {
		audio[i] = byte(i % 256)
	}

	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Get stored audio data
	stored := player.GetAudioData()

	// Should be a copy
	if len(stored) != len(audio) {
		t.Errorf("Stored audio length mismatch: got %d, want %d", len(stored), len(audio))
	}

	// Verify content matches
	for i := range audio {
		if stored[i] != audio[i] {
			t.Errorf("Audio data mismatch at index %d: got %d, want %d", i, stored[i], audio[i])
		}
	}

	// Modifying returned data shouldn't affect internal data
	stored[0] = 255
	stored2 := player.GetAudioData()
	if stored2[0] == 255 {
		t.Error("Modifying returned audio data should not affect internal data")
	}

	// After stop, audio data should be nil
	player.Stop()
	stored = player.GetAudioData()
	if stored != nil {
		t.Error("Audio data should be nil after Stop()")
	}
}

func TestMockPlayer_SpeedAdjustment(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Set double speed
	player.SetDelayFactor(0.5) // 2x speed

	// Create 1 second of audio
	audio := make([]byte, 88200)

	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// After 250ms real time, should be at ~500ms playback position
	time.Sleep(250 * time.Millisecond)

	pos := player.GetPosition()
	// Allow some tolerance for test timing
	if pos < 450*time.Millisecond || pos > 550*time.Millisecond {
		t.Errorf("With 2x speed, position after 250ms should be ~500ms, got %v", pos)
	}
}

func TestMockPlayer_AudioDurationCalculation(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	tests := []struct {
		name           string
		audioSize      int
		expectedDuration time.Duration
	}{
		{
			name:           "1 second",
			audioSize:      88200, // 44100 * 2
			expectedDuration: 1 * time.Second,
		},
		{
			name:           "500ms",
			audioSize:      44100,
			expectedDuration: 500 * time.Millisecond,
		},
		{
			name:           "100ms",
			audioSize:      8820,
			expectedDuration: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audio := make([]byte, tt.audioSize)
			
			err := player.Play(audio)
			if err != nil {
				t.Fatalf("Play failed: %v", err)
			}

			// Speed up for test
			player.SetDelayFactor(0.01)

			// Wait for completion
			completed := player.WaitForCompletion(1 * time.Second)
			if !completed {
				t.Error("Playback should complete")
			}

			// Check calculated duration
			pos := player.GetPosition()
			if pos < tt.expectedDuration-10*time.Millisecond || 
			   pos > tt.expectedDuration+10*time.Millisecond {
				t.Errorf("Duration mismatch: expected ~%v, got %v", tt.expectedDuration, pos)
			}

			player.Stop()
		})
	}
}

func TestMockPlayer_SetAudioDuration(t *testing.T) {
	player := DefaultMockPlayer()
	defer player.Close()

	// Play some audio
	audio := make([]byte, 88200) // Would normally be 1 second
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}

	// Override duration to 2 seconds  
	player.SetAudioDuration(2 * time.Second)

	// Don't speed up too much - use 5x speed (0.2 delay factor)
	// So 2 seconds should take about 400ms
	player.SetDelayFactor(0.2)

	// After 200ms, should still be playing
	time.Sleep(200 * time.Millisecond)
	
	if player.GetState() != StatePlaying {
		t.Error("Should still be playing halfway through duration")
	}

	// After another 300ms (total 500ms), should be done
	time.Sleep(300 * time.Millisecond)
	
	if player.GetState() != StateStopped {
		t.Error("Should be stopped after duration completes")
	}
}

// Benchmark tests
func BenchmarkMockPlayer_Play(b *testing.B) {
	player := DefaultMockPlayer()
	defer player.Close()

	audio := make([]byte, 88200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		player.Play(audio)
		player.Stop()
	}
}

func BenchmarkMockPlayer_GetPosition(b *testing.B) {
	player := DefaultMockPlayer()
	defer player.Close()

	audio := make([]byte, 88200)
	player.Play(audio)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.GetPosition()
	}
}

func BenchmarkMockPlayer_StateCheck(b *testing.B) {
	player := DefaultMockPlayer()
	defer player.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.IsPlaying()
	}
}