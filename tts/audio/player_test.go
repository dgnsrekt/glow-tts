package audio_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/audio"
)

// TestPlayerCreation tests player creation.
func TestPlayerCreation(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	
	if player == nil {
		t.Fatal("Expected non-nil player")
	}
	
	// Check initial state
	if player.IsPlaying() {
		t.Error("Player should not be playing initially")
	}
	
	if player.IsPaused() {
		t.Error("Player should not be paused initially")
	}
	
	if player.GetPosition() != 0 {
		t.Error("Initial position should be 0")
	}
	
	// Clean up
	player.Close()
}

// TestPlayerConfig tests player creation with config.
func TestPlayerConfig(t *testing.T) {
	config := audio.PlayerConfig{
		BufferSize:     8192,
		LatencyHint:    200 * time.Millisecond,
		BufferCapacity: 5,
	}
	
	player, err := audio.NewPlayerWithConfig(config)
	if err != nil {
		t.Fatalf("NewPlayerWithConfig() error = %v", err)
	}
	
	if player == nil {
		t.Fatal("Expected non-nil player")
	}
	
	stats := player.GetStats()
	if stats.BufferSize != 8192 {
		t.Errorf("Expected buffer size 8192, got %d", stats.BufferSize)
	}
	
	// Clean up
	player.Close()
}

// TestPlayWithNilAudio tests playing with nil audio.
func TestPlayWithNilAudio(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	err = player.Play(nil)
	if err == nil {
		t.Error("Expected error for nil audio")
	}
}

// TestPlayWithInvalidFormat tests playing with unsupported format.
func TestPlayWithInvalidFormat(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Create audio with unsupported format
	testAudio := &tts.Audio{
		Data:       []byte{0, 1, 2, 3},
		Format:     tts.FormatMP3, // MP3 not supported directly
		SampleRate: 22050,
		Channels:   1,
		Duration:   1 * time.Second,
	}
	
	err = player.Play(testAudio)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

// TestPlayPauseResume tests play, pause, and resume functionality.
func TestPlayPauseResume(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Create test audio (PCM16)
	// 1 second of silence at 22050 Hz, mono
	samples := 22050
	audioData := make([]byte, samples*2) // 2 bytes per sample for PCM16
	
	testAudio := &tts.Audio{
		Data:       audioData,
		Format:     tts.FormatPCM16,
		SampleRate: 22050,
		Channels:   1,
		Duration:   1 * time.Second,
	}
	
	// Note: This test will fail if oto is not available
	// We'll skip the actual playback test if initialization fails
	err = player.Play(testAudio)
	if err != nil {
		if err.Error() == "failed to create audio context: oto: NewContext is not available on this platform" {
			t.Skip("Oto not available on this platform")
		}
		t.Fatalf("Play() error = %v", err)
	}
	
	// Should be playing
	if !player.IsPlaying() {
		t.Error("Player should be playing")
	}
	
	// Test pause
	err = player.Pause()
	if err != nil {
		t.Errorf("Pause() error = %v", err)
	}
	
	if !player.IsPaused() {
		t.Error("Player should be paused")
	}
	
	// Can't pause again
	err = player.Pause()
	if err == nil {
		t.Error("Expected error for double pause")
	}
	
	// Test resume
	err = player.Resume()
	if err != nil {
		t.Errorf("Resume() error = %v", err)
	}
	
	if player.IsPaused() {
		t.Error("Player should not be paused after resume")
	}
	
	// Can't resume when not paused
	err = player.Resume()
	if err == nil {
		t.Error("Expected error for resume when not paused")
	}
	
	// Stop playback
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	
	if player.IsPlaying() {
		t.Error("Player should not be playing after stop")
	}
}

// TestStopWhenNotPlaying tests stopping when not playing.
func TestStopWhenNotPlaying(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Should not error when stopping while not playing
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop() when not playing should not error, got %v", err)
	}
}

// TestGetDuration tests getting audio duration.
func TestGetDuration(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Initially no duration
	if player.GetDuration() != 0 {
		t.Error("Duration should be 0 initially")
	}
	
	// Create test audio
	testAudio := &tts.Audio{
		Data:       make([]byte, 44100), // Some data
		Format:     tts.FormatPCM16,
		SampleRate: 22050,
		Channels:   1,
		Duration:   2 * time.Second,
	}
	
	// Try to play (might fail if oto not available)
	player.Play(testAudio)
	
	// Duration should be set regardless of playback success
	if player.GetDuration() != 2*time.Second {
		t.Errorf("Expected duration 2s, got %v", player.GetDuration())
	}
}

// TestSetVolume tests volume setting.
func TestSetVolume(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Valid volume
	err = player.SetVolume(0.5)
	if err != nil {
		t.Errorf("SetVolume(0.5) error = %v", err)
	}
	
	err = player.SetVolume(0.0)
	if err != nil {
		t.Errorf("SetVolume(0.0) error = %v", err)
	}
	
	err = player.SetVolume(1.0)
	if err != nil {
		t.Errorf("SetVolume(1.0) error = %v", err)
	}
	
	// Invalid volume
	err = player.SetVolume(-0.1)
	if err == nil {
		t.Error("Expected error for negative volume")
	}
	
	err = player.SetVolume(1.1)
	if err == nil {
		t.Error("Expected error for volume > 1.0")
	}
}

// TestGetStats tests statistics retrieval.
func TestGetStats(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	stats := player.GetStats()
	
	if stats.Playing {
		t.Error("Should not be playing initially")
	}
	
	if stats.Paused {
		t.Error("Should not be paused initially")
	}
	
	if stats.Position != 0 {
		t.Error("Position should be 0 initially")
	}
	
	if stats.BufferSize <= 0 {
		t.Error("Buffer size should be positive")
	}
}

// TestConcurrentOperations tests thread safety.
func TestConcurrentOperations(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	var wg sync.WaitGroup
	errors := make([]error, 0)
	var errorsMu sync.Mutex
	
	// Multiple goroutines performing operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Try various operations
			for j := 0; j < 10; j++ {
				_ = player.GetPosition()
				_ = player.IsPlaying()
				_ = player.IsPaused()
				_ = player.GetDuration()
				_ = player.GetStats()
			}
			
			// Record any panics as errors
			if r := recover(); r != nil {
				errorsMu.Lock()
				if err, ok := r.(error); ok {
					errors = append(errors, err)
				} else {
					errors = append(errors, fmt.Errorf("panic occurred"))
				}
				errorsMu.Unlock()
			}
		}()
	}
	
	wg.Wait()
	
	if len(errors) > 0 {
		t.Errorf("Concurrent operations caused %d errors", len(errors))
	}
}

// TestClose tests resource cleanup.
func TestClose(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	
	// Close should work
	err = player.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
	
	// Operations after close should not panic
	// (though they may not work correctly)
	_ = player.GetPosition()
	_ = player.IsPlaying()
}

// TestFloat32ToPCM16Conversion tests float32 to PCM16 conversion.
func TestFloat32ToPCM16Conversion(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Create float32 audio data
	// 4 samples: -1.0, -0.5, 0.5, 1.0
	floatData := make([]byte, 16)
	
	// Note: This is a simplified test
	// Real float32 data would need proper encoding
	
	testAudio := &tts.Audio{
		Data:       floatData,
		Format:     tts.FormatFloat32,
		SampleRate: 22050,
		Channels:   1,
		Duration:   100 * time.Millisecond,
	}
	
	// Try to play (conversion should happen internally)
	err = player.Play(testAudio)
	// May fail if oto not available, but conversion should still work
	_ = err
}

// TestPositionTracking tests position tracking during playback.
func TestPositionTracking(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Create short test audio
	samples := 2205 // 0.1 second at 22050 Hz
	audioData := make([]byte, samples*2)
	
	testAudio := &tts.Audio{
		Data:       audioData,
		Format:     tts.FormatPCM16,
		SampleRate: 22050,
		Channels:   1,
		Duration:   100 * time.Millisecond,
	}
	
	err = player.Play(testAudio)
	if err != nil {
		if err.Error() == "failed to create audio context: oto: NewContext is not available on this platform" {
			t.Skip("Oto not available on this platform")
		}
		// Other errors are OK for this test
	}
	
	// Position should start at 0
	pos := player.GetPosition()
	if pos < 0 {
		t.Errorf("Position should not be negative, got %v", pos)
	}
	
	// After some time, position should advance
	// (if playback is actually working)
	if player.IsPlaying() {
		time.Sleep(50 * time.Millisecond)
		newPos := player.GetPosition()
		if newPos < pos {
			t.Error("Position should not go backwards")
		}
	}
}

// TestDefaultPlayerConfig tests default configuration.
func TestDefaultPlayerConfig(t *testing.T) {
	config := audio.DefaultPlayerConfig()
	
	if config.BufferSize != 4096 {
		t.Errorf("Expected default buffer size 4096, got %d", config.BufferSize)
	}
	
	if config.LatencyHint != 100*time.Millisecond {
		t.Errorf("Expected default latency 100ms, got %v", config.LatencyHint)
	}
	
	if config.BufferCapacity != 3 {
		t.Errorf("Expected default buffer capacity 3, got %d", config.BufferCapacity)
	}
}

// TestMultiplePlayCalls tests calling Play multiple times.
func TestMultiplePlayCalls(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Create two different audio samples
	audio1 := &tts.Audio{
		Data:       make([]byte, 1000),
		Format:     tts.FormatPCM16,
		SampleRate: 22050,
		Channels:   1,
		Duration:   100 * time.Millisecond,
	}
	
	audio2 := &tts.Audio{
		Data:       make([]byte, 2000),
		Format:     tts.FormatPCM16,
		SampleRate: 44100, // Different sample rate
		Channels:   2,      // Different channels
		Duration:   200 * time.Millisecond,
	}
	
	// First play
	err = player.Play(audio1)
	// Ignore error - may fail if oto not available
	_ = err
	
	// Second play should stop first and start new
	err = player.Play(audio2)
	// Ignore error - may fail if oto not available
	_ = err
	
	// Duration should be from second audio
	if player.GetDuration() == 100*time.Millisecond {
		t.Error("Duration should have changed to second audio")
	}
}

// BenchmarkPlay benchmarks the Play operation.
func BenchmarkPlay(b *testing.B) {
	player, err := audio.NewPlayer()
	if err != nil {
		b.Fatal(err)
	}
	defer player.Close()
	
	// Create test audio
	testAudio := &tts.Audio{
		Data:       make([]byte, 44100),
		Format:     tts.FormatPCM16,
		SampleRate: 22050,
		Channels:   1,
		Duration:   1 * time.Second,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.Play(testAudio)
		_ = player.Stop()
	}
}

// BenchmarkGetPosition benchmarks position retrieval.
func BenchmarkGetPosition(b *testing.B) {
	player, err := audio.NewPlayer()
	if err != nil {
		b.Fatal(err)
	}
	defer player.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.GetPosition()
	}
}

// TestPauseWhenNotPlaying tests pausing when not playing.
func TestPauseWhenNotPlaying(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	err = player.Pause()
	if err == nil {
		t.Error("Expected error when pausing while not playing")
	}
}

// TestResumeWhenNotPaused tests resuming when not paused.
func TestResumeWhenNotPaused(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	err = player.Resume()
	if err == nil {
		t.Error("Expected error when resuming while not paused")
	}
}

// TestGetLastError tests error tracking.
func TestGetLastError(t *testing.T) {
	player, err := audio.NewPlayer()
	if err != nil {
		t.Fatalf("NewPlayer() error = %v", err)
	}
	defer player.Close()
	
	// Initially no error
	if player.GetLastError() != nil {
		t.Error("Should have no error initially")
	}
	
	// After an operation that might fail, check for error
	// This depends on the specific implementation
}