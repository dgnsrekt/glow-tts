package tts

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
)

// Platform represents the current operating system platform
type Platform string

const (
	PlatformLinux   Platform = "linux"
	PlatformDarwin  Platform = "darwin"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

// AudioSubsystem represents the available audio subsystem
type AudioSubsystem string

const (
	AudioSubsystemALSA      AudioSubsystem = "alsa"
	AudioSubsystemPulseAudio AudioSubsystem = "pulseaudio"
	AudioSubsystemCoreAudio AudioSubsystem = "coreaudio"
	AudioSubsystemWASAPI    AudioSubsystem = "wasapi"
	AudioSubsystemNone      AudioSubsystem = "none"
)

// PlatformInfo contains information about the current platform
type PlatformInfo struct {
	OS             Platform
	AudioSubsystem AudioSubsystem
	HasAudioDevice bool
	IsCI           bool
	Details        map[string]string
}

// DetectPlatform detects the current platform and audio capabilities
func DetectPlatform() *PlatformInfo {
	info := &PlatformInfo{
		OS:      getPlatform(),
		IsCI:    IsCI(),
		Details: make(map[string]string),
	}

	// Detect audio subsystem based on platform
	switch info.OS {
	case PlatformLinux:
		info.AudioSubsystem = detectLinuxAudio()
		info.HasAudioDevice = checkLinuxAudioDevices()
	case PlatformDarwin:
		info.AudioSubsystem = AudioSubsystemCoreAudio
		info.HasAudioDevice = checkDarwinAudioDevices()
	case PlatformWindows:
		info.AudioSubsystem = AudioSubsystemWASAPI
		info.HasAudioDevice = checkWindowsAudioDevices()
	default:
		info.AudioSubsystem = AudioSubsystemNone
		info.HasAudioDevice = false
	}

	// Add platform details
	info.Details["os"] = runtime.GOOS
	info.Details["arch"] = runtime.GOARCH
	info.Details["goversion"] = runtime.Version()
	
	log.Debug("Platform detected",
		"os", info.OS,
		"audio", info.AudioSubsystem,
		"has_device", info.HasAudioDevice,
		"is_ci", info.IsCI)

	return info
}

// getPlatform returns the current platform
func getPlatform() Platform {
	switch runtime.GOOS {
	case "linux":
		return PlatformLinux
	case "darwin":
		return PlatformDarwin
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

// detectLinuxAudio detects the audio subsystem on Linux
func detectLinuxAudio() AudioSubsystem {
	// Check for PulseAudio first (more common on desktop Linux)
	if isCommandAvailable("pactl") {
		if output, err := exec.Command("pactl", "info").Output(); err == nil {
			if strings.Contains(string(output), "Server Name") {
				log.Debug("PulseAudio detected")
				return AudioSubsystemPulseAudio
			}
		}
	}

	// Check for ALSA
	if _, err := os.Stat("/proc/asound"); err == nil {
		log.Debug("ALSA detected via /proc/asound")
		return AudioSubsystemALSA
	}

	// Check for ALSA commands
	if isCommandAvailable("aplay") {
		log.Debug("ALSA detected via aplay command")
		return AudioSubsystemALSA
	}

	return AudioSubsystemNone
}

// checkLinuxAudioDevices checks if audio devices are available on Linux
func checkLinuxAudioDevices() bool {
	// Check /dev/snd for ALSA devices
	if _, err := os.Stat("/dev/snd"); err == nil {
		entries, err := os.ReadDir("/dev/snd")
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "pcm") {
					log.Debug("Linux audio devices found in /dev/snd")
					return true
				}
			}
		}
	}

	// Check ALSA cards
	if _, err := os.Stat("/proc/asound/cards"); err == nil {
		content, err := os.ReadFile("/proc/asound/cards")
		if err == nil && len(content) > 0 && !strings.Contains(string(content), "no soundcards") {
			log.Debug("Linux audio cards found in /proc/asound/cards")
			return true
		}
	}

	// Try PulseAudio
	if isCommandAvailable("pactl") {
		if output, err := exec.Command("pactl", "list", "short", "sinks").Output(); err == nil {
			if len(output) > 0 {
				log.Debug("PulseAudio sinks found")
				return true
			}
		}
	}

	log.Debug("No Linux audio devices found")
	return false
}

// checkDarwinAudioDevices checks if audio devices are available on macOS
func checkDarwinAudioDevices() bool {
	// On macOS, CoreAudio is almost always available
	// We could use system_profiler SPAudioDataType for more detailed check
	if isCommandAvailable("system_profiler") {
		if output, err := exec.Command("system_profiler", "SPAudioDataType").Output(); err == nil {
			if strings.Contains(string(output), "Device") {
				log.Debug("macOS audio devices found")
				return true
			}
		}
	}

	// Assume audio is available on macOS unless proven otherwise
	log.Debug("Assuming macOS has audio devices")
	return true
}

// checkWindowsAudioDevices checks if audio devices are available on Windows
func checkWindowsAudioDevices() bool {
	// On Windows, we can check for the Windows Audio service
	// This is a simplified check - a more thorough check would use Windows APIs
	if isCommandAvailable("sc") {
		if output, err := exec.Command("sc", "query", "AudioSrv").Output(); err == nil {
			if strings.Contains(string(output), "RUNNING") {
				log.Debug("Windows Audio Service is running")
				return true
			}
		}
	}

	// Assume audio is available on Windows
	log.Debug("Assuming Windows has audio devices")
	return true
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// ShouldUseMockAudio determines if mock audio should be used based on platform info
func (p *PlatformInfo) ShouldUseMockAudio() bool {
	// Use mock in CI
	if p.IsCI {
		return true
	}

	// Use mock if no audio subsystem
	if p.AudioSubsystem == AudioSubsystemNone {
		return true
	}

	// Use mock if no audio devices
	if !p.HasAudioDevice {
		return true
	}

	return false
}

// GetPlatformBufferSize returns the recommended buffer size for the platform
func (p *PlatformInfo) GetPlatformBufferSize() int {
	switch p.OS {
	case PlatformDarwin:
		// macOS benefits from larger buffers
		return 100
	case PlatformWindows:
		// Windows WASAPI works well with moderate buffers
		return 80
	case PlatformLinux:
		// Linux ALSA needs smaller buffers to avoid underruns
		if p.AudioSubsystem == AudioSubsystemPulseAudio {
			return 60
		}
		return 50
	default:
		return 50
	}
}

// String returns a string representation of the platform info
func (p *PlatformInfo) String() string {
	return fmt.Sprintf("Platform{OS: %s, Audio: %s, HasDevice: %v, IsCI: %v}",
		p.OS, p.AudioSubsystem, p.HasAudioDevice, p.IsCI)
}