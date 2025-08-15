package tts

import (
	"bytes"
	"os"
	"testing"
)

func TestMockAudioContext(t *testing.T) {
	// Create mock context
	mockCtx, err := NewMockAudioContext()
	if err != nil {
		t.Fatalf("Failed to create mock audio context: %v", err)
	}
	defer mockCtx.Close()
	
	// Use as interface
	var ctx AudioContextInterface = mockCtx

	// Verify context is ready
	if !ctx.IsReady() {
		t.Error("Mock context should be ready immediately")
	}

	// Verify sample rate and channels
	if ctx.SampleRate() != SampleRate {
		t.Errorf("Expected sample rate %d, got %d", SampleRate, ctx.SampleRate())
	}
	if ctx.ChannelCount() != Channels {
		t.Errorf("Expected %d channels, got %d", Channels, ctx.ChannelCount())
	}

	// Create a player with test data
	testData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	reader := bytes.NewReader(testData)
	
	player, err := ctx.NewPlayer(reader)
	if err != nil {
		t.Fatalf("Failed to create player: %v", err)
	}
	defer player.Close()

	// Test player operations
	player.Play()
	if !player.IsPlaying() {
		t.Error("Player should be playing after Play()")
	}

	player.Pause()
	if player.IsPlaying() {
		t.Error("Player should not be playing after Pause()")
	}

	// Test volume
	player.SetVolume(0.5)
	if player.Volume() != 0.5 {
		t.Errorf("Expected volume 0.5, got %f", player.Volume())
	}

	// Test reset
	if err := player.Reset(); err != nil {
		t.Errorf("Failed to reset player: %v", err)
	}

	// Verify test helpers
	if mockCtx.PlayersCreated != 1 {
		t.Errorf("Expected 1 player created, got %d", mockCtx.PlayersCreated)
	}
}

func TestAudioContextFactory(t *testing.T) {
	tests := []struct {
		name         string
		contextType  AudioContextType
		expectMock   bool
	}{
		{
			name:        "Mock context requested",
			contextType: AudioContextMock,
			expectMock:  true,
		},
		{
			name:        "Auto context in test",
			contextType: AudioContextAuto,
			expectMock:  false, // May be production or mock depending on environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewAudioContext(tt.contextType)
			if err != nil {
				t.Fatalf("Failed to create audio context: %v", err)
			}
			defer ctx.Close()

			if !ctx.IsReady() {
				t.Error("Context should be ready")
			}

			// Check if we got the expected type (if we can determine it)
			if tt.expectMock {
				if _, ok := ctx.(*MockAudioContext); !ok {
					t.Error("Expected mock audio context")
				}
			}
		})
	}
}

func TestCIDetection(t *testing.T) {
	// Save original env
	originalCI := getEnv("CI")
	originalMockAudio := getEnv("MOCK_AUDIO")
	
	// Restore env after test
	defer func() {
		setEnv("CI", originalCI)
		setEnv("MOCK_AUDIO", originalMockAudio)
	}()

	tests := []struct {
		name      string
		envVars   map[string]string
		expectCI  bool
	}{
		{
			name:      "No CI environment",
			envVars:   map[string]string{},
			expectCI:  false,
		},
		{
			name:      "CI environment variable set",
			envVars:   map[string]string{"CI": "true"},
			expectCI:  true,
		},
		{
			name:      "GitHub Actions environment",
			envVars:   map[string]string{"GITHUB_ACTIONS": "true"},
			expectCI:  true,
		},
		{
			name:      "Mock audio requested",
			envVars:   map[string]string{"MOCK_AUDIO": "true"},
			expectCI:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI-related environment variables
			ciVars := []string{
				"CI",
				"CONTINUOUS_INTEGRATION",
				"GITHUB_ACTIONS",
				"GITLAB_CI",
				"JENKINS_URL",
				"TRAVIS",
				"CIRCLECI",
				"BUILDKITE",
				"DRONE",
				"TEAMCITY_VERSION",
				"MOCK_AUDIO",
				"GLOW_TTS_MOCK_AUDIO",
			}
			
			// Store original values to restore later
			originalEnv := make(map[string]string)
			for _, key := range ciVars {
				if val, exists := os.LookupEnv(key); exists {
					originalEnv[key] = val
				}
				clearEnv(key)
			}
			
			// Restore original environment after test
			defer func() {
				for key, val := range originalEnv {
					os.Setenv(key, val)
				}
			}()
			
			// Set test environment
			for k, v := range tt.envVars {
				setEnv(k, v)
			}

			if IsCI() != tt.expectCI {
				t.Errorf("Expected IsCI() = %v, got %v", tt.expectCI, IsCI())
			}
		})
	}
}

// Helper functions for environment manipulation
func getEnv(key string) string {
	return os.Getenv(key)
}

func setEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

func clearEnv(key string) {
	os.Unsetenv(key)
}