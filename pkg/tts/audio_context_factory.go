package tts

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

var (
	// globalAudioContextInterface is the global audio context instance
	globalAudioContextInterface AudioContextInterface
	audioContextInterfaceOnce   sync.Once
	audioContextInterfaceErr    error
)

// IsCI detects if we're running in a CI environment
func IsCI() bool {
	// Check common CI environment variables
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
	}

	for _, envVar := range ciVars {
		if val := os.Getenv(envVar); val != "" && val != "false" {
			log.Debug("CI environment detected", "variable", envVar, "value", val)
			return true
		}
	}

	// Check for mock audio environment variable
	if os.Getenv("MOCK_AUDIO") == "true" || os.Getenv("GLOW_TTS_MOCK_AUDIO") == "true" {
		log.Debug("Mock audio requested via environment variable")
		return true
	}

	return false
}

// IsTesting detects if we're running in test mode
func IsTesting() bool {
	// Check if we're running under `go test`
	if os.Getenv("GO_TEST") == "1" {
		return true
	}
	
	// Check for test-specific environment variable
	if os.Getenv("TESTING") == "true" {
		return true
	}
	
	return false
}

// NewAudioContext creates an appropriate audio context based on the environment
func NewAudioContext(contextType AudioContextType) (AudioContextInterface, error) {
	switch contextType {
	case AudioContextProduction:
		log.Debug("Creating production audio context")
		return NewProductionAudioContext()
		
	case AudioContextMock:
		log.Debug("Creating mock audio context")
		return NewMockAudioContext()
		
	case AudioContextAuto:
		// Detect platform capabilities
		platform := DetectPlatform()
		log.Debug("Platform detection complete", "info", platform.String())
		
		// Use mock if platform suggests it
		if platform.ShouldUseMockAudio() {
			reason := "unknown"
			if platform.IsCI {
				reason = "CI environment"
			} else if !platform.HasAudioDevice {
				reason = "no audio devices"
			} else if platform.AudioSubsystem == AudioSubsystemNone {
				reason = "no audio subsystem"
			}
			log.Info("Using mock audio context", "reason", reason)
			return NewMockAudioContext()
		}
		
		// Try to create production context with platform-specific handling
		log.Debug("Attempting to create production audio context",
			"platform", platform.OS,
			"audio", platform.AudioSubsystem)
		
		prodCtx, err := NewProductionAudioContextWithRetry(platform)
		if err != nil {
			log.Warn("Failed to create production audio context, falling back to mock",
				"error", err,
				"platform", platform.OS)
			return NewMockAudioContext()
		}
		return prodCtx, nil
		
	default:
		return nil, fmt.Errorf("unknown audio context type: %v", contextType)
	}
}

// GetGlobalAudioContext returns the global audio context instance
// It creates the context on first call and reuses it for subsequent calls
func GetGlobalAudioContext() (AudioContextInterface, error) {
	audioContextInterfaceOnce.Do(func() {
		globalAudioContextInterface, audioContextInterfaceErr = NewAudioContext(AudioContextAuto)
	})
	
	if audioContextInterfaceErr != nil {
		return nil, audioContextInterfaceErr
	}
	
	return globalAudioContextInterface, nil
}

// SetGlobalAudioContext allows setting a specific audio context (useful for testing)
func SetGlobalAudioContext(ctx AudioContextInterface) {
	globalAudioContextInterface = ctx
}

// ResetGlobalAudioContext resets the global audio context
// This is mainly useful for testing
func ResetGlobalAudioContext() {
	if globalAudioContextInterface != nil {
		_ = globalAudioContextInterface.Close()
	}
	globalAudioContextInterface = nil
	audioContextInterfaceOnce = sync.Once{}
	audioContextInterfaceErr = nil
}