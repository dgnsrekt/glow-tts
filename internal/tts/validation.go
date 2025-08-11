package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

// ValidationResult contains the result of engine validation
type ValidationResult struct {
	// Engine is the validated engine type
	Engine ttypes.EngineType

	// Available indicates if the engine is available and configured
	Available bool

	// Error contains any validation error
	Error error

	// Guidance provides setup instructions if validation failed
	Guidance string

	// Details contains additional validation information
	Details map[string]string
}

// ValidateEngineSelection validates that a TTS engine has been explicitly chosen.
// It checks the CLI argument first, then config, and requires explicit selection.
// Returns ErrNoEngineConfigured if no engine is selected.
func ValidateEngineSelection(cliArg string, config Config) (ttypes.EngineType, error) {
	// 1. CLI argument takes precedence
	engineType := cliArg

	// 2. Use config if no CLI arg
	if engineType == "" {
		engineType = string(config.Engine)
	}

	// 3. Require explicit selection - no defaults, no fallback
	if engineType == "" {
		return ttypes.EngineNone, fmt.Errorf("%w\n\nPlease specify an engine:\n  glow --tts piper document.md    # Use Piper (offline)\n  glow --tts gtts document.md     # Use Google TTS (online)\n\nOr set a default in ~/.config/glow/config.yml:\n  tts:\n    engine: piper  # or \"gtts\"", ErrNoEngineConfigured)
	}

	// 4. Validate engine type (normalize aliases)
	switch engineType {
	case "piper":
		return ttypes.EnginePiper, nil
	case "gtts", "google":
		return ttypes.EngineGoogle, nil
	default:
		return ttypes.EngineNone, fmt.Errorf("%w: %s\n\nSupported engines:\n  - piper (offline TTS)\n  - gtts (Google TTS)", ErrInvalidEngine, engineType)
	}
}

// ValidateEngine performs comprehensive validation of the specified engine.
// It checks availability, configuration, and performs a test synthesis.
func ValidateEngine(engineType ttypes.EngineType, config Config) *ValidationResult {
	result := &ValidationResult{
		Engine:  engineType,
		Details: make(map[string]string),
	}

	switch engineType {
	case ttypes.EnginePiper:
		result = validatePiperEngine(config.Piper, result)
	case ttypes.EngineGoogle:
		result = validateGoogleEngine(config.GTTS, result)
	case ttypes.EngineNone:
		result.Error = ErrNoEngineConfigured
		result.Guidance = "Please specify a TTS engine with --tts flag or in config file"
	default:
		result.Error = fmt.Errorf("%w: %s", ErrInvalidEngine, engineType)
		result.Guidance = "Supported engines: piper, gtts"
	}

	return result
}

// validatePiperEngine validates the Piper TTS engine configuration and availability
func validatePiperEngine(config PiperConfig, result *ValidationResult) *ValidationResult {
	result.Details["engine"] = "Piper (Offline TTS)"

	// Check if Piper binary is available
	piperPath, err := exec.LookPath("piper")
	if err != nil {
		result.Error = fmt.Errorf("Piper TTS not found in PATH: %w", err)
		result.Guidance = buildPiperInstallGuidance()
		return result
	}
	result.Details["binary_path"] = piperPath

	// Check if we can execute Piper
	cmd := exec.Command(piperPath, "--version")
	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("cannot execute Piper: %w", err)
		result.Guidance = "Piper binary found but cannot be executed. Check permissions and dependencies."
		return result
	}

	// Check model path configuration
	if config.ModelPath == "" {
		result.Error = fmt.Errorf("Piper model path not configured")
		result.Guidance = buildPiperModelGuidance()
		return result
	}
	result.Details["model_path"] = config.ModelPath

	// Check if model file exists and is readable
	if _, err := os.Stat(config.ModelPath); err != nil {
		result.Error = fmt.Errorf("model file not accessible: %w", err)
		result.Guidance = buildPiperValidationGuidance(err)
		return result
	}

	// Check config file if specified
	if config.ConfigPath != "" {
		if _, err := os.Stat(config.ConfigPath); err != nil {
			// Config file is optional, so just note it
			result.Details["config_note"] = "Config file not found (using model defaults)"
		} else {
			result.Details["config_path"] = config.ConfigPath
		}
	} else {
		// Try to find config file automatically
		configPath := strings.TrimSuffix(config.ModelPath, filepath.Ext(config.ModelPath)) + ".json"
		if _, err := os.Stat(configPath); err == nil {
			result.Details["config_path"] = configPath + " (auto-detected)"
		}
	}

	result.Available = true
	result.Details["status"] = "Ready (full validation requires test synthesis)"
	return result
}

// validateGoogleEngine validates the Google TTS engine configuration and availability
func validateGoogleEngine(config GTTSConfigSection, result *ValidationResult) *ValidationResult {
	result.Details["engine"] = "Google TTS (gTTS - Free)"

	// Check if gtts-cli is available
	gttsPath, err := exec.LookPath("gtts-cli")
	if err != nil {
		result.Error = fmt.Errorf("gTTS not found in PATH: %w", err)
		result.Guidance = buildGTTSInstallGuidance()
		return result
	}
	result.Details["gtts_path"] = gttsPath

	// Check if we can execute gtts-cli
	cmd := exec.Command(gttsPath, "--help")
	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("cannot execute gTTS: %w", err)
		result.Guidance = "gTTS binary found but cannot be executed. Check Python installation and dependencies."
		return result
	}

	// Check if ffmpeg is available
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		result.Error = fmt.Errorf("ffmpeg not found in PATH: %w", err)
		result.Guidance = buildFFmpegInstallGuidance()
		return result
	}
	result.Details["ffmpeg_path"] = ffmpegPath

	// Check if we can execute ffmpeg
	cmd = exec.Command(ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		result.Error = fmt.Errorf("cannot execute ffmpeg: %w", err)
		result.Guidance = "ffmpeg binary found but cannot be executed. Check installation and dependencies."
		return result
	}

	// Validate configuration
	language := config.Language
	if language == "" {
		language = "en" // Default
	}
	result.Details["language"] = language

	if config.Slow {
		result.Details["speed"] = "slow"
	} else {
		result.Details["speed"] = "normal"
	}

	// Check temp directory if specified
	if config.TempDir != "" {
		if _, err := os.Stat(config.TempDir); err != nil {
			result.Error = fmt.Errorf("temp directory not accessible: %w", err)
			result.Guidance = "Check temp directory path and permissions"
			return result
		}
		result.Details["temp_dir"] = config.TempDir
	}

	result.Available = true
	result.Details["status"] = "Ready (full validation requires network test)"
	return result
}

// buildPiperInstallGuidance provides instructions for installing Piper
func buildPiperInstallGuidance() string {
	return `Piper TTS is not installed. To install:

1. Download Piper binary from: https://github.com/rhasspy/piper/releases
2. Extract and add to PATH, or install via package manager:
   
   # Ubuntu/Debian
   sudo apt install piper-tts
   
   # Arch Linux
   yay -S piper-tts
   
   # macOS (Homebrew)
   brew install piper-tts
   
   # Or download release manually:
   wget https://github.com/rhasspy/piper/releases/latest/download/piper_linux_x86_64.tar.gz
   tar -xzf piper_linux_x86_64.tar.gz
   sudo cp piper/piper /usr/local/bin/

3. Download a voice model from: https://github.com/rhasspy/piper/blob/master/VOICES.md
4. Configure the model path in ~/.config/glow/config.yml`
}

// buildPiperModelGuidance provides instructions for configuring Piper models
func buildPiperModelGuidance() string {
	return `Piper model path not configured. To configure:

1. Download a voice model from: https://github.com/rhasspy/piper/blob/master/VOICES.md
   
   Example (English, Amy voice):
   mkdir -p ~/.local/share/piper/models
   cd ~/.local/share/piper/models
   wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx
   wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx.json

2. Configure the model path in ~/.config/glow/config.yml:
   tts:
     engine: piper
   piper:
     model_path: ~/.local/share/piper/models/en_US-amy-medium.onnx`
}

// buildPiperValidationGuidance provides specific guidance based on validation errors
func buildPiperValidationGuidance(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "model file not accessible") {
		return `Model file is not accessible. Please check:

1. File exists and has correct permissions
2. Path is correct in config file
3. Download the model if missing:
   mkdir -p ~/.local/share/piper/models
   cd ~/.local/share/piper/models
   wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx
   wget https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/en/en_US/amy/medium/en_US-amy-medium.onnx.json`
	}

	if strings.Contains(errStr, "test synthesis failed") {
		return `Test synthesis failed. This could indicate:

1. Model file is corrupted - try re-downloading
2. Incompatible Piper version - update to latest
3. Missing dependencies - check piper --help
4. Insufficient permissions - check file access rights

Try running piper manually to diagnose:
  echo "Hello world" | piper --model /path/to/model.onnx --output-raw`
	}

	return "Check Piper installation and model configuration"
}

// buildGTTSInstallGuidance provides instructions for installing gTTS
func buildGTTSInstallGuidance() string {
	return `gTTS (Google Text-to-Speech) is not installed. To install:

1. Install via pip:
   pip install gtts
   
   # Or with pipx (recommended):
   pipx install gtts
   
2. Verify installation:
   gtts-cli --help
   
3. No API key required - gTTS uses Google Translate's free TTS service

Note: gTTS requires an internet connection to function.`
}

// buildFFmpegInstallGuidance provides instructions for installing ffmpeg
func buildFFmpegInstallGuidance() string {
	return `ffmpeg is required for gTTS audio conversion. To install:

# Ubuntu/Debian
sudo apt update && sudo apt install ffmpeg

# CentOS/RHEL/Fedora
sudo dnf install ffmpeg

# macOS (Homebrew)
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg

# Windows (Chocolatey)
choco install ffmpeg

# Or download from: https://ffmpeg.org/download.html`
}

// buildGTTSValidationGuidance provides specific guidance based on gTTS validation errors
func buildGTTSValidationGuidance(err error) string {
	errStr := err.Error()

	if strings.Contains(errStr, "test synthesis failed") {
		return `gTTS test synthesis failed. This could indicate:

1. No internet connection - gTTS requires online access
2. Google's TTS service is unavailable
3. Network firewall blocking requests
4. Rate limiting from Google

Try testing manually:
  gtts-cli "Hello world" -l en -o test.mp3
  
If this works, the issue may be with audio conversion.
If not, check your internet connection and try again later.`
	}

	if strings.Contains(errStr, "cannot execute") {
		return `Cannot execute gTTS or ffmpeg. Please check:

1. Both gtts-cli and ffmpeg are in PATH
2. Correct permissions for execution
3. Dependencies are properly installed

Test manually:
  gtts-cli --help
  ffmpeg -version`
	}

	return "Check gTTS installation and internet connectivity"
}

// QuickValidation performs a fast validation check without test synthesis.
// Useful for UI startup to show immediate feedback.
func QuickValidation(engineType ttypes.EngineType) error {
	switch engineType {
	case ttypes.EnginePiper:
		if _, err := exec.LookPath("piper"); err != nil {
			return fmt.Errorf("Piper not found: %w", err)
		}
		return nil

	case ttypes.EngineGoogle:
		if _, err := exec.LookPath("gtts-cli"); err != nil {
			return fmt.Errorf("gTTS not found: %w", err)
		}
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			return fmt.Errorf("ffmpeg not found: %w", err)
		}
		return nil

	default:
		return ErrInvalidEngine
	}
}
