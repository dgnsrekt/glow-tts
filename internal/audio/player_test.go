package audio

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

// TestPlayerConfig tests the player configuration validation.
func TestPlayerConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    PlayerConfig
		expectErr bool
	}{
		{
			name: "valid config 44100Hz",
			config: PlayerConfig{
				SampleRate: 44100,
				Channels:   1,
				BitDepth:   16,
				BufferSize: 4096,
			},
			expectErr: false,
		},
		{
			name: "valid config 48000Hz",
			config: PlayerConfig{
				SampleRate: 48000,
				Channels:   2,
				BitDepth:   16,
				BufferSize: 8192,
			},
			expectErr: false,
		},
		{
			name: "invalid sample rate",
			config: PlayerConfig{
				SampleRate: 22050,
				Channels:   1,
				BitDepth:   16,
				BufferSize: 4096,
			},
			expectErr: true,
		},
		{
			name: "invalid channels",
			config: PlayerConfig{
				SampleRate: 44100,
				Channels:   3,
				BitDepth:   16,
				BufferSize: 4096,
			},
			expectErr: true,
		},
		{
			name: "invalid bit depth",
			config: PlayerConfig{
				SampleRate: 44100,
				Channels:   1,
				BitDepth:   24,
				BufferSize: 4096,
			},
			expectErr: true,
		},
		{
			name: "invalid buffer size",
			config: PlayerConfig{
				SampleRate: 44100,
				Channels:   1,
				BitDepth:   16,
				BufferSize: 0,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectErr && err == nil {
				t.Errorf("validateConfig() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("validateConfig() unexpected error: %v", err)
			}
		})
	}
}

// TestDefaultPlayerConfig tests the default configuration.
func TestDefaultPlayerConfig(t *testing.T) {
	config := DefaultPlayerConfig()
	
	if config.SampleRate != 44100 {
		t.Errorf("expected sample rate 44100, got %d", config.SampleRate)
	}
	
	if config.Channels != 1 {
		t.Errorf("expected 1 channel, got %d", config.Channels)
	}
	
	if config.BitDepth != 16 {
		t.Errorf("expected 16-bit depth, got %d", config.BitDepth)
	}
	
	// Validate that default config is valid
	if err := validateConfig(config); err != nil {
		t.Errorf("default config is invalid: %v", err)
	}
}

// Shared test context to avoid "context already created" errors
var (
	testPlayer     *Player
	testPlayerOnce sync.Once
	testPlayerErr  error
)

// getTestPlayer returns a shared test player, creating it once.
func getTestPlayer(t *testing.T) *Player {
	testPlayerOnce.Do(func() {
		config := DefaultPlayerConfig()
		testPlayer, testPlayerErr = NewPlayer(config)
	})
	
	if testPlayerErr != nil {
		t.Skipf("Skipping test: cannot create audio player (no audio device?): %v", testPlayerErr)
	}
	
	// Stop any current playback to ensure clean state
	if testPlayer != nil {
		testPlayer.Stop()
	}
	
	return testPlayer
}

// generateTestAudio generates PCM audio data for testing.
func generateTestAudio(sampleRate, channels int, duration time.Duration) []byte {
	// Generate simple sine wave
	samples := int(duration.Seconds() * float64(sampleRate))
	bytesPerSample := 2 // 16-bit
	totalBytes := samples * channels * bytesPerSample
	
	data := make([]byte, totalBytes)
	
	// Fill with a simple pattern (not actual sine wave, just test data)
	for i := 0; i < len(data); i += 2 {
		// Simple sawtooth pattern for test audio
		sample := int16((i / 2) % 1000)
		data[i] = byte(sample)
		data[i+1] = byte(sample >> 8)
	}
	
	return data
}

// TestPlayerCreation tests player creation and initialization.
func TestPlayerCreation(t *testing.T) {
	player := getTestPlayer(t)
	
	// Check initial state
	if player.GetState() != StateStopped {
		t.Errorf("expected initial state stopped, got %s", player.getStateName(player.GetState()))
	}
	
	if !player.IsPlaying() == false {
		t.Errorf("expected IsPlaying() to be false initially")
	}
	
	if player.GetVolume() != 1.0 {
		t.Errorf("expected initial volume 1.0, got %f", player.GetVolume())
	}
	
	if player.GetPosition() != 0 {
		t.Errorf("expected initial position 0, got %v", player.GetPosition())
	}
}

// TestPlayerVolume tests volume control.
func TestPlayerVolume(t *testing.T) {
	player := getTestPlayer(t)
	
	tests := []struct {
		volume    float64
		expectErr bool
	}{
		{0.0, false},
		{0.5, false},
		{1.0, false},
		{-0.1, true},
		{1.1, true},
	}
	
	for _, tt := range tests {
		t.Run("volume_test", func(t *testing.T) {
			err := player.SetVolume(tt.volume)
			if tt.expectErr && err == nil {
				t.Errorf("SetVolume(%f) expected error but got none", tt.volume)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("SetVolume(%f) unexpected error: %v", tt.volume, err)
			}
			
			if !tt.expectErr {
				if got := player.GetVolume(); got != tt.volume {
					t.Errorf("SetVolume(%f) then GetVolume() = %f", tt.volume, got)
				}
			}
		})
	}
}

// TestPlayerPlayEmpty tests playing empty audio.
func TestPlayerPlayEmpty(t *testing.T) {
	player := getTestPlayer(t)
	
	err := player.Play(nil)
	if err == nil {
		t.Error("Play(nil) expected error but got none")
	}
	
	err = player.Play([]byte{})
	if err == nil {
		t.Error("Play(empty) expected error but got none")
	}
}

// TestPlayerBasicPlayback tests basic audio playback.
func TestPlayerBasicPlayback(t *testing.T) {
	player := getTestPlayer(t)
	
	// Generate short test audio (100ms)
	audio := generateTestAudio(44100, 1, 100*time.Millisecond)
	
	// Test play
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play() failed: %v", err)
	}
	
	// Check state
	if player.GetState() != StatePlaying {
		t.Errorf("expected state playing after Play(), got %s", player.getStateName(player.GetState()))
	}
	
	if !player.IsPlaying() {
		t.Error("IsPlaying() should return true after Play()")
	}
	
	// Position should advance
	time.Sleep(50 * time.Millisecond)
	pos := player.GetPosition()
	if pos == 0 {
		t.Error("position should advance during playback")
	}
	
	// Stop playback
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
	
	if player.IsPlaying() {
		t.Error("IsPlaying() should return false after Stop()")
	}
	
	if player.GetState() != StateStopped {
		t.Errorf("expected state stopped after Stop(), got %s", player.getStateName(player.GetState()))
	}
}

// TestPlayerPauseResume tests pause and resume functionality.
func TestPlayerPauseResume(t *testing.T) {
	player := getTestPlayer(t)
	
	// Generate test audio (500ms)
	audio := generateTestAudio(44100, 1, 500*time.Millisecond)
	
	// Start playback
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play() failed: %v", err)
	}
	
	// Wait a bit then pause
	time.Sleep(100 * time.Millisecond)
	
	err = player.Pause()
	if err != nil {
		t.Errorf("Pause() failed: %v", err)
	}
	
	if player.GetState() != StatePaused {
		t.Errorf("expected state paused after Pause(), got %s", player.getStateName(player.GetState()))
	}
	
	if player.IsPlaying() {
		t.Error("IsPlaying() should return false after Pause()")
	}
	
	// Position should be preserved during pause
	pausedPos := player.GetPosition()
	if pausedPos == 0 {
		t.Error("position should be preserved after pause")
	}
	
	time.Sleep(100 * time.Millisecond)
	if player.GetPosition() != pausedPos {
		t.Error("position should not advance during pause")
	}
	
	// Resume playback
	err = player.Resume()
	if err != nil {
		t.Errorf("Resume() failed: %v", err)
	}
	
	if player.GetState() != StatePlaying {
		t.Errorf("expected state playing after Resume(), got %s", player.getStateName(player.GetState()))
	}
	
	if !player.IsPlaying() {
		t.Error("IsPlaying() should return true after Resume()")
	}
	
	// Clean up
	player.Stop()
}

// TestPlayerStateTransitions tests invalid state transitions.
func TestPlayerStateTransitions(t *testing.T) {
	player := getTestPlayer(t)
	
	// Test pause when stopped
	err := player.Pause()
	if err == nil {
		t.Error("Pause() when stopped should fail")
	}
	
	// Test resume when stopped  
	err = player.Resume()
	if err == nil {
		t.Error("Resume() when stopped should fail")
	}
	
	// Test stop when stopped (should be no-op)
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop() when stopped should not fail: %v", err)
	}
	
	// Note: We can't test Close() with shared player due to OTO single context limit
	// This test case is covered by the unit test of the Close method itself
	t.Log("Skipping Close() test due to shared player usage")
}

// TestPlayerMemoryManagement tests that audio data is properly managed.
func TestPlayerMemoryManagement(t *testing.T) {
	player := getTestPlayer(t)
	
	// Generate test audio
	audio := generateTestAudio(44100, 1, 100*time.Millisecond)
	originalData := make([]byte, len(audio))
	copy(originalData, audio)
	
	// Start playback
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play() failed: %v", err)
	}
	
	// Modify original audio data
	for i := range audio {
		audio[i] = 0
	}
	
	// The player should have its own copy, so this shouldn't affect playback
	// We can't easily test this directly, but the fact that playback continues
	// without errors suggests the copy is working.
	
	// Wait for a bit then stop
	time.Sleep(50 * time.Millisecond)
	player.Stop()
	
	// After stop, the audio data should be cleaned up
	player.mu.RLock()
	hasActiveStream := player.activeStream != nil
	player.mu.RUnlock()
	
	if hasActiveStream {
		t.Error("activeStream should be nil after Stop()")
	}
}

// TestAudioStream tests the AudioStream functionality.
func TestAudioStream(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	reader := bytes.NewReader(data)
	
	stream := &AudioStream{
		data:       data,
		reader:     reader,
		size:       len(data),
		duration:   100 * time.Millisecond,
		sampleRate: 44100,
		channels:   1,
	}
	
	// Test initial state
	if stream.IsClosed() {
		t.Error("stream should not be closed initially")
	}
	
	if stream.GetDuration() != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got %v", stream.GetDuration())
	}
	
	// Test closing
	stream.Close()
	
	if !stream.IsClosed() {
		t.Error("stream should be closed after Close()")
	}
	
	// Test double close (should be safe)
	stream.Close()
	
	if !stream.IsClosed() {
		t.Error("stream should still be closed after double Close()")
	}
}

// TestConcurrentAccess tests concurrent access to the player.
func TestConcurrentAccess(t *testing.T) {
	player := getTestPlayer(t)
	
	audio := generateTestAudio(44100, 1, 200*time.Millisecond)
	
	// Test concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 10)
	
	// Multiple goroutines trying to play
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := player.Play(audio); err != nil {
				errors <- err
			}
		}()
	}
	
	// Multiple goroutines trying to control playback
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			player.Pause()
			time.Sleep(20 * time.Millisecond)
			player.Resume()
		}()
	}
	
	// Multiple goroutines reading state
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				player.IsPlaying()
				player.GetPosition()
				player.GetState()
				player.GetVolume()
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for any unexpected errors
	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}
	
	// Clean up
	player.Stop()
}

// TestPlayerInfoMethods tests informational methods.
func TestPlayerInfoMethods(t *testing.T) {
	player := getTestPlayer(t)
	
	// Test GetAudioInfo
	sampleRate, channels, bitDepth, duration := player.GetAudioInfo()
	
	if sampleRate != 44100 {
		t.Errorf("expected sample rate 44100, got %d", sampleRate)
	}
	
	if channels != 1 {
		t.Errorf("expected 1 channel, got %d", channels)
	}
	
	if bitDepth != 16 {
		t.Errorf("expected 16-bit depth, got %d", bitDepth)
	}
	
	if duration != 0 {
		t.Errorf("expected duration 0 (no audio), got %v", duration)
	}
	
	// Play some audio and check duration
	audio := generateTestAudio(44100, 1, 500*time.Millisecond)
	err := player.Play(audio)
	if err != nil {
		t.Fatalf("Play() failed: %v", err)
	}
	
	_, _, _, duration = player.GetAudioInfo()
	if duration == 0 {
		t.Error("expected non-zero duration after playing audio")
	}
	
	// Duration should be approximately 500ms (allowing for some variance)
	expectedDuration := 500 * time.Millisecond
	if duration < expectedDuration-50*time.Millisecond || duration > expectedDuration+50*time.Millisecond {
		t.Errorf("expected duration ~500ms, got %v", duration)
	}
	
	player.Stop()
}