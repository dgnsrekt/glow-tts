package tts

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

// PlatformTestConfig contains platform-specific test configuration
type PlatformTestConfig struct {
	Platform           *PlatformInfo
	SkipAudioTests     bool
	AudioTestTimeout   time.Duration
	BufferTestSize     int
	MaxInitRetries     int
	InitRetryDelay     time.Duration
}

// GetPlatformTestConfig returns platform-specific test configuration
func GetPlatformTestConfig(t *testing.T) *PlatformTestConfig {
	platform := DetectPlatform()
	config := &PlatformTestConfig{
		Platform:         platform,
		AudioTestTimeout: 10 * time.Second,
		BufferTestSize:   1024 * 4, // 4KB test buffer
		MaxInitRetries:   1,
		InitRetryDelay:   100 * time.Millisecond,
	}

	// Skip audio tests in CI or when no audio devices
	if platform.IsCI || !platform.HasAudioDevice {
		config.SkipAudioTests = true
		t.Logf("Skipping audio tests: CI=%v, HasAudioDevice=%v", platform.IsCI, platform.HasAudioDevice)
	}

	// Platform-specific adjustments
	switch platform.OS {
	case PlatformDarwin:
		// macOS needs more retries and longer timeouts
		config.MaxInitRetries = 3
		config.InitRetryDelay = 200 * time.Millisecond
		config.AudioTestTimeout = 15 * time.Second
		
		// macOS can handle larger buffers
		config.BufferTestSize = 1024 * 8 // 8KB
		
		t.Logf("Using macOS test configuration")
		
	case PlatformWindows:
		// Windows needs moderate settings
		config.MaxInitRetries = 2
		config.InitRetryDelay = 150 * time.Millisecond
		
		t.Logf("Using Windows test configuration")
		
	case PlatformLinux:
		// Linux settings depend on audio subsystem
		if platform.AudioSubsystem == AudioSubsystemPulseAudio {
			config.MaxInitRetries = 2
			config.BufferTestSize = 1024 * 6 // 6KB for PulseAudio
			t.Logf("Using Linux PulseAudio test configuration")
		} else {
			// ALSA typically works first try but with smaller buffers
			config.BufferTestSize = 1024 * 2 // 2KB for ALSA
			t.Logf("Using Linux ALSA test configuration")
		}
		
	default:
		t.Logf("Using default test configuration for unknown platform")
	}

	return config
}

// SkipIfNoAudio skips the test if audio is not available
func SkipIfNoAudio(t *testing.T) {
	config := GetPlatformTestConfig(t)
	if config.SkipAudioTests {
		t.Skip("Skipping test: audio not available on this platform")
	}
}

// RunWithPlatformTimeout runs a test function with platform-specific timeout
func RunWithPlatformTimeout(t *testing.T, fn func()) {
	config := GetPlatformTestConfig(t)
	
	done := make(chan bool)
	go func() {
		fn()
		done <- true
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-time.After(config.AudioTestTimeout):
		t.Fatalf("Test timed out after %v", config.AudioTestTimeout)
	}
}

// CreateTestAudioContext creates an audio context appropriate for testing
func CreateTestAudioContext(t *testing.T) (AudioContextInterface, error) {
	config := GetPlatformTestConfig(t)
	
	if config.SkipAudioTests {
		t.Log("Creating mock audio context for testing")
		return NewMockAudioContext()
	}
	
	// Try to create production context with retries
	var lastErr error
	for i := 0; i < config.MaxInitRetries; i++ {
		if i > 0 {
			t.Logf("Retrying audio context creation (attempt %d/%d)", i+1, config.MaxInitRetries)
			time.Sleep(config.InitRetryDelay)
		}
		
		ctx, err := NewProductionAudioContextWithRetry(config.Platform)
		if err == nil {
			t.Log("Successfully created production audio context")
			return ctx, nil
		}
		lastErr = err
	}
	
	// Fall back to mock if production fails
	t.Logf("Failed to create production audio context: %v. Using mock instead.", lastErr)
	return NewMockAudioContext()
}

// GenerateTestAudioData creates test PCM audio data
func GenerateTestAudioData(config *PlatformTestConfig) []byte {
	// Generate sine wave test data
	numSamples := config.BufferTestSize / BytesPerSample
	data := make([]byte, config.BufferTestSize)
	
	// Simple square wave for testing
	for i := 0; i < numSamples; i++ {
		value := int16(0)
		if i%100 < 50 {
			value = 10000
		} else {
			value = -10000
		}
		
		// Little-endian encoding
		data[i*2] = byte(value & 0xFF)
		data[i*2+1] = byte(value >> 8)
	}
	
	return data
}

// AssertAudioContextReady checks if audio context is properly initialized
func AssertAudioContextReady(t *testing.T, ctx AudioContextInterface) {
	if !ctx.IsReady() {
		t.Fatal("Audio context is not ready")
	}
	
	if ctx.SampleRate() != SampleRate {
		t.Errorf("Unexpected sample rate: got %d, want %d", ctx.SampleRate(), SampleRate)
	}
	
	if ctx.ChannelCount() != Channels {
		t.Errorf("Unexpected channel count: got %d, want %d", ctx.ChannelCount(), Channels)
	}
}

// LogPlatformInfo logs detailed platform information for debugging
func LogPlatformInfo(t *testing.T, platform *PlatformInfo) {
	t.Logf("Platform Information:")
	t.Logf("  OS: %s", platform.OS)
	t.Logf("  Audio Subsystem: %s", platform.AudioSubsystem)
	t.Logf("  Has Audio Device: %v", platform.HasAudioDevice)
	t.Logf("  Is CI: %v", platform.IsCI)
	t.Logf("  Should Use Mock: %v", platform.ShouldUseMockAudio())
	
	if len(platform.Details) > 0 {
		t.Logf("  Details:")
		for key, value := range platform.Details {
			t.Logf("    %s: %s", key, value)
		}
	}
}

// SetupPlatformTestEnvironment sets up environment for platform-specific testing
func SetupPlatformTestEnvironment(t *testing.T) func() {
	// Store original environment
	originalCI := os.Getenv("CI")
	originalMockAudio := os.Getenv("MOCK_AUDIO")
	
	// Set up test-specific logging
	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}
	
	// Return cleanup function
	return func() {
		// Restore original environment
		if originalCI != "" {
			os.Setenv("CI", originalCI)
		} else {
			os.Unsetenv("CI")
		}
		
		if originalMockAudio != "" {
			os.Setenv("MOCK_AUDIO", originalMockAudio)
		} else {
			os.Unsetenv("MOCK_AUDIO")
		}
	}
}

// BenchmarkPlatformAudioContext benchmarks audio context creation
func BenchmarkPlatformAudioContext(b *testing.B, platform *PlatformInfo) {
	if platform.ShouldUseMockAudio() {
		b.Skip("Skipping benchmark: using mock audio")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, err := NewProductionAudioContextWithRetry(platform)
		if err != nil {
			b.Fatalf("Failed to create audio context: %v", err)
		}
		ctx.Close()
	}
}

// TestPlatformCompatibility runs platform-specific compatibility tests
func TestPlatformCompatibility(t *testing.T, platform *PlatformInfo) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: fmt.Sprintf("%s_BufferSize", platform.OS),
			test: func(t *testing.T) {
				bufSize := platform.GetPlatformBufferSize()
				if bufSize < 10 || bufSize > 200 {
					t.Errorf("Invalid buffer size for %s: %d", platform.OS, bufSize)
				}
			},
		},
		{
			name: fmt.Sprintf("%s_AudioDetection", platform.OS),
			test: func(t *testing.T) {
				if platform.IsCI {
					t.Skip("Skipping in CI")
				}
				// Platform should detect audio correctly
				if platform.AudioSubsystem == AudioSubsystemNone && platform.HasAudioDevice {
					t.Error("Inconsistent audio detection: has device but no subsystem")
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}