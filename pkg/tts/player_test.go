package tts

import (
	"bytes"
	"encoding/binary"
	"math"
	"runtime"
	"sync"
	"testing"
	"time"
)

// generateTestPCM generates test PCM audio data
func generateTestPCM(durationMs int, frequency float64) []byte {
	numSamples := (SampleRate * durationMs) / 1000
	data := make([]byte, numSamples*BytesPerSample)
	
	buf := bytes.NewBuffer(data[:0])
	for i := 0; i < numSamples; i++ {
		// Generate sine wave
		t := float64(i) / float64(SampleRate)
		value := math.Sin(2 * math.Pi * frequency * t)
		// Convert to 16-bit signed integer
		sample := int16(value * 32767)
		binary.Write(buf, binary.LittleEndian, sample)
	}
	
	return buf.Bytes()
}

// generateSilence generates silent PCM data
func generateSilence(durationMs int) []byte {
	numSamples := (SampleRate * durationMs) / 1000
	return make([]byte, numSamples*BytesPerSample)
}

func TestAudioContextInitialization(t *testing.T) {
	// Test singleton pattern with new interface
	ctx1, err1 := GetGlobalAudioContext()
	if err1 != nil {
		t.Fatalf("Failed to get audio context: %v", err1)
	}
	
	ctx2, err2 := GetGlobalAudioContext()
	if err2 != nil {
		t.Fatalf("Failed to get audio context second time: %v", err2)
	}
	
	if ctx1 != ctx2 {
		t.Error("Audio context is not a singleton")
	}
	
	if !ctx1.IsReady() {
		t.Error("Audio context not ready")
	}
}

func TestNewAudioStream(t *testing.T) {
	tests := []struct {
		name        string
		pcmData     []byte
		expectError bool
	}{
		{
			name:        "valid PCM data",
			pcmData:     generateTestPCM(100, 440), // 100ms of 440Hz tone
			expectError: false,
		},
		{
			name:        "empty data",
			pcmData:     []byte{},
			expectError: true,
		},
		{
			name:        "odd length data",
			pcmData:     []byte{0x00, 0x01, 0x02}, // 3 bytes (not aligned to 16-bit)
			expectError: true,
		},
		{
			name:        "silent audio",
			pcmData:     generateSilence(50),
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := NewAudioStream(tt.pcmData)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			
			defer stream.Close()
			
			// Verify initial state
			if stream.GetState() != PlaybackStopped {
				t.Errorf("Expected initial state Stopped, got %v", stream.GetState())
			}
			
			// Verify duration calculation
			if len(tt.pcmData) > 0 {
				expectedSamples := len(tt.pcmData) / BytesPerSample
				expectedDuration := time.Duration(expectedSamples) * time.Second / SampleRate
				if stream.GetDuration() != expectedDuration {
					t.Errorf("Expected duration %v, got %v", expectedDuration, stream.GetDuration())
				}
			}
			
			// Verify memory is pinned
			if !stream.pinned {
				t.Error("Audio data should be pinned in memory")
			}
		})
	}
}

func TestPlaybackControls(t *testing.T) {
	// Generate 200ms of test audio
	pcmData := generateTestPCM(200, 440)
	stream, err := NewAudioStream(pcmData)
	if err != nil {
		t.Fatalf("Failed to create audio stream: %v", err)
	}
	defer stream.Close()
	
	// Test Play
	err = stream.Play()
	if err != nil {
		t.Errorf("Play failed: %v", err)
	}
	
	if stream.GetState() != PlaybackPlaying {
		t.Errorf("Expected state Playing, got %v", stream.GetState())
	}
	
	// Wait a bit for playback to start
	time.Sleep(50 * time.Millisecond)
	
	// Test Pause
	err = stream.Pause()
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}
	
	if stream.GetState() != PlaybackPaused {
		t.Errorf("Expected state Paused, got %v", stream.GetState())
	}
	
	pausedPosition := stream.GetPosition()
	t.Logf("Paused at position: %v", pausedPosition)
	
	// Test Resume (Play while paused)
	err = stream.Play()
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	
	if stream.GetState() != PlaybackPlaying {
		t.Errorf("Expected state Playing after resume, got %v", stream.GetState())
	}
	
	// Position should continue from where it was paused
	time.Sleep(100 * time.Millisecond)
	resumedPosition := stream.GetPosition()
	t.Logf("Resumed position after 100ms: %v", resumedPosition)
	if resumedPosition <= pausedPosition {
		t.Errorf("Position should advance after resume. Paused: %v, Resumed: %v", pausedPosition, resumedPosition)
	}
	
	// Test Stop
	err = stream.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	
	if stream.GetState() != PlaybackStopped {
		t.Errorf("Expected state Stopped, got %v", stream.GetState())
	}
	
	// Position should reset to 0
	if stream.GetPosition() != 0 {
		t.Errorf("Expected position 0 after stop, got %v", stream.GetPosition())
	}
}

func TestPlaybackCompletion(t *testing.T) {
	// Generate very short audio (50ms)
	pcmData := generateTestPCM(50, 440)
	stream, err := NewAudioStream(pcmData)
	if err != nil {
		t.Fatalf("Failed to create audio stream: %v", err)
	}
	defer stream.Close()
	
	// Start playback
	err = stream.Play()
	if err != nil {
		t.Fatalf("Play failed: %v", err)
	}
	
	// Wait for playback to complete (with some buffer)
	time.Sleep(150 * time.Millisecond)
	
	// State should return to Stopped
	if stream.GetState() != PlaybackStopped {
		t.Errorf("Expected state Stopped after completion, got %v", stream.GetState())
	}
	
	// Position should be reset
	if stream.GetPosition() != 0 {
		t.Errorf("Expected position 0 after completion, got %v", stream.GetPosition())
	}
}

func TestMemoryManagement(t *testing.T) {
	// Track memory allocations
	var m1, m2 runtime.MemStats
	
	// Force GC and get baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Create and destroy multiple streams
	for i := 0; i < 10; i++ {
		pcmData := generateTestPCM(100, 440)
		stream, err := NewAudioStream(pcmData)
		if err != nil {
			t.Fatalf("Failed to create stream %d: %v", i, err)
		}
		
		// Play briefly
		stream.Play()
		time.Sleep(10 * time.Millisecond)
		stream.Stop()
		stream.Close()
	}
	
	// Force GC and check memory
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	// Memory should not grow significantly (allow 10MB tolerance)
	memGrowth := int64(m2.Alloc) - int64(m1.Alloc)
	if memGrowth > 10*1024*1024 {
		t.Errorf("Memory grew by %d bytes, possible leak", memGrowth)
	}
}

func TestConcurrentPlayback(t *testing.T) {
	// Test multiple streams playing concurrently
	numStreams := 3
	streams := make([]*AudioStream, numStreams)
	
	for i := 0; i < numStreams; i++ {
		// Different frequencies for each stream
		frequency := 440.0 * float64(i+1)
		pcmData := generateTestPCM(100, frequency)
		
		stream, err := NewAudioStream(pcmData)
		if err != nil {
			t.Fatalf("Failed to create stream %d: %v", i, err)
		}
		streams[i] = stream
		defer stream.Close()
	}
	
	// Start all streams concurrently
	var wg sync.WaitGroup
	for i, stream := range streams {
		wg.Add(1)
		go func(idx int, s *AudioStream) {
			defer wg.Done()
			
			err := s.Play()
			if err != nil {
				t.Errorf("Stream %d play failed: %v", idx, err)
			}
		}(i, stream)
	}
	
	wg.Wait()
	
	// Verify all are playing
	for i, stream := range streams {
		if stream.GetState() != PlaybackPlaying {
			t.Errorf("Stream %d not playing", i)
		}
	}
	
	// Stop all streams
	for _, stream := range streams {
		stream.Stop()
	}
}

func TestAudioPlayer(t *testing.T) {
	player := NewTTSAudioPlayer()
	defer player.Close()
	
	// Test playing PCM data
	pcmData := generateTestPCM(100, 440)
	err := player.PlayPCM(pcmData)
	if err != nil {
		t.Errorf("PlayPCM failed: %v", err)
	}
	
	if player.GetState() != PlaybackPlaying {
		t.Errorf("Expected state Playing, got %v", player.GetState())
	}
	
	// Test pause
	err = player.Pause()
	if err != nil {
		t.Errorf("Pause failed: %v", err)
	}
	
	if player.GetState() != PlaybackPaused {
		t.Errorf("Expected state Paused, got %v", player.GetState())
	}
	
	// Test resume
	err = player.Resume()
	if err != nil {
		t.Errorf("Resume failed: %v", err)
	}
	
	if player.GetState() != PlaybackPlaying {
		t.Errorf("Expected state Playing after resume, got %v", player.GetState())
	}
	
	// Test stop
	err = player.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
	
	if player.GetState() != PlaybackStopped {
		t.Errorf("Expected state Stopped, got %v", player.GetState())
	}
	
	// Test playing new audio replaces old
	newPCMData := generateTestPCM(50, 880)
	err = player.PlayPCM(newPCMData)
	if err != nil {
		t.Errorf("Second PlayPCM failed: %v", err)
	}
	
	// Should be playing the new audio
	if player.GetState() != PlaybackPlaying {
		t.Errorf("Expected state Playing for new audio, got %v", player.GetState())
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("double play", func(t *testing.T) {
		pcmData := generateTestPCM(100, 440)
		stream, _ := NewAudioStream(pcmData)
		defer stream.Close()
		
		stream.Play()
		err := stream.Play() // Should be idempotent
		if err != nil {
			t.Errorf("Double play should not error: %v", err)
		}
	})
	
	t.Run("pause when not playing", func(t *testing.T) {
		pcmData := generateTestPCM(100, 440)
		stream, _ := NewAudioStream(pcmData)
		defer stream.Close()
		
		err := stream.Pause()
		if err == nil {
			t.Error("Pause should error when not playing")
		}
	})
	
	t.Run("multiple stops", func(t *testing.T) {
		pcmData := generateTestPCM(100, 440)
		stream, _ := NewAudioStream(pcmData)
		defer stream.Close()
		
		stream.Play()
		stream.Stop()
		err := stream.Stop() // Should be idempotent
		if err != nil {
			t.Errorf("Multiple stops should not error: %v", err)
		}
	})
	
	t.Run("close multiple times", func(t *testing.T) {
		pcmData := generateTestPCM(100, 440)
		stream, _ := NewAudioStream(pcmData)
		
		err1 := stream.Close()
		err2 := stream.Close() // Should be safe
		
		if err1 != nil {
			t.Errorf("First close failed: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second close failed: %v", err2)
		}
	})
}

func TestFinalizer(t *testing.T) {
	// Test that finalizer cleans up if Close() is not called
	pcmData := generateTestPCM(100, 440)
	
	// Create stream in a scope
	func() {
		stream, err := NewAudioStream(pcmData)
		if err != nil {
			t.Fatalf("Failed to create stream: %v", err)
		}
		
		// Start playing but don't close
		stream.Play()
		// Stream goes out of scope here
	}()
	
	// Force garbage collection
	runtime.GC()
	runtime.Gosched()
	time.Sleep(100 * time.Millisecond)
	
	// If we get here without crashing, finalizer worked
	// (Can't easily test internal state after GC)
}

func BenchmarkAudioStreamCreation(b *testing.B) {
	pcmData := generateTestPCM(100, 440)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream, err := NewAudioStream(pcmData)
		if err != nil {
			b.Fatal(err)
		}
		stream.Close()
	}
}

func BenchmarkPlaybackStartStop(b *testing.B) {
	pcmData := generateTestPCM(100, 440)
	stream, err := NewAudioStream(pcmData)
	if err != nil {
		b.Fatal(err)
	}
	defer stream.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stream.Play()
		stream.Stop()
	}
}

func TestPlatformSpecific(t *testing.T) {
	// Test platform-specific buffer sizes are set correctly
	ctx, err := GetGlobalAudioContext()
	if err != nil {
		t.Fatalf("Failed to get audio context: %v", err)
	}
	
	// Verify context is initialized
	if !ctx.IsReady() {
		t.Error("Audio context should be ready")
	}
	
	// Platform-specific tests could be added here
	switch runtime.GOOS {
	case "darwin":
		// macOS specific tests
		t.Log("Running on macOS")
	case "windows":
		// Windows specific tests
		t.Log("Running on Windows")
	case "linux":
		// Linux specific tests
		t.Log("Running on Linux")
	default:
		t.Logf("Running on %s", runtime.GOOS)
	}
}