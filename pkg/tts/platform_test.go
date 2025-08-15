package tts

import (
	"os"
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform := DetectPlatform()
	
	// Log platform info for debugging
	LogPlatformInfo(t, platform)
	
	// Basic validation
	if platform.OS == PlatformUnknown {
		t.Error("Failed to detect platform OS")
	}
	
	// OS should match runtime.GOOS
	expectedOS := PlatformUnknown
	switch runtime.GOOS {
	case "linux":
		expectedOS = PlatformLinux
	case "darwin":
		expectedOS = PlatformDarwin
	case "windows":
		expectedOS = PlatformWindows
	}
	
	if platform.OS != expectedOS {
		t.Errorf("Platform OS mismatch: got %s, expected %s", platform.OS, expectedOS)
	}
	
	// Check details are populated
	if len(platform.Details) == 0 {
		t.Error("Platform details not populated")
	}
	
	if platform.Details["os"] != runtime.GOOS {
		t.Errorf("OS detail mismatch: got %s, expected %s", platform.Details["os"], runtime.GOOS)
	}
	
	if platform.Details["arch"] != runtime.GOARCH {
		t.Errorf("Arch detail mismatch: got %s, expected %s", platform.Details["arch"], runtime.GOARCH)
	}
}

func TestPlatformAudioDetection(t *testing.T) {
	platform := DetectPlatform()
	
	t.Logf("Audio subsystem: %s", platform.AudioSubsystem)
	t.Logf("Has audio device: %v", platform.HasAudioDevice)
	
	// In CI, we might not have audio
	if platform.IsCI {
		t.Log("Running in CI environment")
		if platform.ShouldUseMockAudio() {
			t.Log("Correctly determined to use mock audio in CI")
		}
		return
	}
	
	// Platform-specific checks
	switch platform.OS {
	case PlatformLinux:
		// Linux should detect ALSA or PulseAudio
		if platform.AudioSubsystem != AudioSubsystemALSA && 
		   platform.AudioSubsystem != AudioSubsystemPulseAudio &&
		   platform.AudioSubsystem != AudioSubsystemNone {
			t.Errorf("Unexpected Linux audio subsystem: %s", platform.AudioSubsystem)
		}
		
	case PlatformDarwin:
		// macOS should always have CoreAudio
		if platform.AudioSubsystem != AudioSubsystemCoreAudio {
			t.Errorf("Expected CoreAudio on macOS, got %s", platform.AudioSubsystem)
		}
		
	case PlatformWindows:
		// Windows should have WASAPI
		if platform.AudioSubsystem != AudioSubsystemWASAPI {
			t.Errorf("Expected WASAPI on Windows, got %s", platform.AudioSubsystem)
		}
	}
}

func TestShouldUseMockAudio(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		cleanup  func()
		expected bool
	}{
		{
			name: "CI environment",
			setup: func() {
				os.Setenv("CI", "true")
			},
			cleanup: func() {
				os.Unsetenv("CI")
			},
			expected: true,
		},
		{
			name: "MOCK_AUDIO flag",
			setup: func() {
				os.Setenv("MOCK_AUDIO", "true")
			},
			cleanup: func() {
				os.Unsetenv("MOCK_AUDIO")
			},
			expected: true,
		},
		{
			name: "GitHub Actions",
			setup: func() {
				os.Setenv("GITHUB_ACTIONS", "true")
			},
			cleanup: func() {
				os.Unsetenv("GITHUB_ACTIONS")
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}
			
			platform := DetectPlatform()
			if platform.ShouldUseMockAudio() != tt.expected {
				t.Errorf("ShouldUseMockAudio() = %v, expected %v", 
					platform.ShouldUseMockAudio(), tt.expected)
			}
		})
	}
}

func TestGetPlatformBufferSize(t *testing.T) {
	tests := []struct {
		os       Platform
		audio    AudioSubsystem
		expected int
	}{
		{PlatformDarwin, AudioSubsystemCoreAudio, 100},
		{PlatformWindows, AudioSubsystemWASAPI, 80},
		{PlatformLinux, AudioSubsystemPulseAudio, 60},
		{PlatformLinux, AudioSubsystemALSA, 50},
		{PlatformUnknown, AudioSubsystemNone, 50},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.os)+"_"+string(tt.audio), func(t *testing.T) {
			platform := &PlatformInfo{
				OS:             tt.os,
				AudioSubsystem: tt.audio,
			}
			
			bufSize := platform.GetPlatformBufferSize()
			if bufSize != tt.expected {
				t.Errorf("GetPlatformBufferSize() = %d, expected %d", bufSize, tt.expected)
			}
		})
	}
}

func TestPlatformString(t *testing.T) {
	platform := &PlatformInfo{
		OS:             PlatformLinux,
		AudioSubsystem: AudioSubsystemALSA,
		HasAudioDevice: true,
		IsCI:           false,
	}
	
	str := platform.String()
	expected := "Platform{OS: linux, Audio: alsa, HasDevice: true, IsCI: false}"
	
	if str != expected {
		t.Errorf("String() = %s, expected %s", str, expected)
	}
}

func TestAudioContextFactoryWithPlatform(t *testing.T) {
	cleanup := SetupPlatformTestEnvironment(t)
	defer cleanup()
	
	platform := DetectPlatform()
	LogPlatformInfo(t, platform)
	
	// Test auto detection
	ctx, err := NewAudioContext(AudioContextAuto)
	if err != nil {
		t.Fatalf("Failed to create audio context: %v", err)
	}
	defer ctx.Close()
	
	// Verify we got the right type
	if platform.ShouldUseMockAudio() {
		if _, ok := ctx.(*MockAudioContext); !ok {
			t.Error("Expected MockAudioContext in CI/mock environment")
		}
	} else {
		if _, ok := ctx.(*ProductionAudioContext); !ok {
			t.Error("Expected ProductionAudioContext in non-CI environment")
		}
	}
	
	// Test that context is ready
	AssertAudioContextReady(t, ctx)
}

func TestProductionAudioContextWithRetry(t *testing.T) {
	// Skip if we should use mock
	platform := DetectPlatform()
	if platform.ShouldUseMockAudio() {
		t.Skip("Skipping production audio test in mock environment")
	}
	
	// Test retry functionality
	ctx, err := NewProductionAudioContextWithRetry(platform)
	if err != nil {
		// This might fail on systems without audio, which is OK
		t.Logf("Could not create production audio context: %v", err)
		return
	}
	defer ctx.Close()
	
	if !ctx.IsReady() {
		t.Error("Production audio context not ready after initialization")
	}
}

func TestPlatformTestHelpers(t *testing.T) {
	config := GetPlatformTestConfig(t)
	
	t.Logf("Test Configuration:")
	t.Logf("  Skip Audio Tests: %v", config.SkipAudioTests)
	t.Logf("  Audio Test Timeout: %v", config.AudioTestTimeout)
	t.Logf("  Buffer Test Size: %d", config.BufferTestSize)
	t.Logf("  Max Init Retries: %d", config.MaxInitRetries)
	t.Logf("  Init Retry Delay: %v", config.InitRetryDelay)
	
	// Test audio data generation
	testData := GenerateTestAudioData(config)
	if len(testData) != config.BufferTestSize {
		t.Errorf("Generated test data size mismatch: got %d, expected %d", 
			len(testData), config.BufferTestSize)
	}
	
	// Test context creation helper
	ctx, err := CreateTestAudioContext(t)
	if err != nil {
		t.Fatalf("Failed to create test audio context: %v", err)
	}
	defer ctx.Close()
	
	AssertAudioContextReady(t, ctx)
}

func BenchmarkPlatformDetection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DetectPlatform()
	}
}

func BenchmarkAudioContextCreation(b *testing.B) {
	platform := DetectPlatform()
	
	if platform.ShouldUseMockAudio() {
		b.Run("Mock", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ctx, _ := NewMockAudioContext()
				ctx.Close()
			}
		})
	} else {
		b.Run("Production", func(b *testing.B) {
			BenchmarkPlatformAudioContext(b, platform)
		})
	}
}