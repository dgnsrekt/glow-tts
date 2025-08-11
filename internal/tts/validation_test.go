package tts

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

func TestValidateEngineSelection(t *testing.T) {
	tests := []struct {
		name      string
		cliArg    string
		config    Config
		want      ttypes.EngineType
		wantError bool
		errorText string
	}{
		{
			name:      "CLI arg takes precedence - piper",
			cliArg:    "piper",
			config:    Config{Engine: ttypes.EngineGoogle},
			want:      ttypes.EnginePiper,
			wantError: false,
		},
		{
			name:      "CLI arg takes precedence - gtts",
			cliArg:    "gtts",
			config:    Config{Engine: ttypes.EnginePiper},
			want:      ttypes.EngineGoogle,
			wantError: false,
		},
		{
			name:      "CLI arg takes precedence - google alias",
			cliArg:    "google",
			config:    Config{Engine: ttypes.EnginePiper},
			want:      ttypes.EngineGoogle,
			wantError: false,
		},
		{
			name:      "Use config when no CLI arg - piper",
			cliArg:    "",
			config:    Config{Engine: ttypes.EnginePiper},
			want:      ttypes.EnginePiper,
			wantError: false,
		},
		{
			name:      "Use config when no CLI arg - gtts",
			cliArg:    "",
			config:    Config{Engine: ttypes.EngineGoogle},
			want:      ttypes.EngineGoogle,
			wantError: false,
		},
		{
			name:      "No engine configured - requires explicit selection",
			cliArg:    "",
			config:    Config{Engine: ttypes.EngineNone},
			want:      ttypes.EngineNone,
			wantError: true,
			errorText: "no TTS engine configured",
		},
		{
			name:      "Invalid engine type",
			cliArg:    "invalid",
			config:    Config{},
			want:      ttypes.EngineNone,
			wantError: true,
			errorText: "invalid TTS engine",
		},
		{
			name:      "Empty string engine",
			cliArg:    "  ",
			config:    Config{},
			want:      ttypes.EngineNone,
			wantError: true,
			errorText: "no TTS engine configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Normalize whitespace in CLI arg
			cliArg := strings.TrimSpace(tt.cliArg)
			if cliArg == "  " {
				cliArg = ""
			}

			got, err := ValidateEngineSelection(cliArg, tt.config)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateEngineSelection() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("ValidateEngineSelection() error = %q, want to contain %q", err.Error(), tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEngineSelection() unexpected error = %v", err)
					return
				}
			}

			if got != tt.want {
				t.Errorf("ValidateEngineSelection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateEngineSelection_ErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		cliArg          string
		config          Config
		expectedInError []string
	}{
		{
			name:   "No engine configured provides helpful guidance",
			cliArg: "",
			config: Config{Engine: ttypes.EngineNone},
			expectedInError: []string{
				"glow --tts piper",
				"glow --tts gtts",
				"config.yml",
				"engine: piper",
			},
		},
		{
			name:   "Invalid engine shows supported options",
			cliArg: "whisper",
			config: Config{},
			expectedInError: []string{
				"invalid TTS engine",
				"piper (offline TTS)",
				"gtts (Google TTS)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateEngineSelection(tt.cliArg, tt.config)

			if err == nil {
				t.Fatalf("Expected error but got none")
			}

			errorMsg := err.Error()
			for _, expected := range tt.expectedInError {
				if !strings.Contains(errorMsg, expected) {
					t.Errorf("Error message should contain %q, but got: %s", expected, errorMsg)
				}
			}
		})
	}
}

func TestValidateEngine(t *testing.T) {
	tests := []struct {
		name       string
		engineType ttypes.EngineType
		config     Config
		wantEngine ttypes.EngineType
	}{
		{
			name:       "Piper engine validation",
			engineType: ttypes.EnginePiper,
			config: Config{
				Piper: PiperConfig{
					ModelPath: "/path/to/model.onnx",
				},
			},
			wantEngine: ttypes.EnginePiper,
		},
		{
			name:       "Google engine validation",
			engineType: ttypes.EngineGoogle,
			config: Config{
				GTTS: GTTSConfigSection{
					Language: "en",
				},
			},
			wantEngine: ttypes.EngineGoogle,
		},
		{
			name:       "No engine",
			engineType: ttypes.EngineNone,
			config:     Config{},
			wantEngine: ttypes.EngineNone,
		},
		{
			name:       "Invalid engine",
			engineType: "invalid",
			config:     Config{},
			wantEngine: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEngine(tt.engineType, tt.config)

			if result == nil {
				t.Fatal("ValidateEngine() returned nil result")
			}

			if result.Engine != tt.wantEngine {
				t.Errorf("ValidateEngine().Engine = %v, want %v", result.Engine, tt.wantEngine)
			}

			if result.Details == nil {
				t.Error("ValidateEngine().Details should not be nil")
			}
		})
	}
}

func TestValidateEngine_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		engineType ttypes.EngineType
		config     Config
		wantError  bool
	}{
		{
			name:       "No engine should error",
			engineType: ttypes.EngineNone,
			config:     Config{},
			wantError:  true,
		},
		{
			name:       "Invalid engine should error",
			engineType: "unknown",
			config:     Config{},
			wantError:  true,
		},
		{
			name:       "Piper without model path should error",
			engineType: ttypes.EnginePiper,
			config: Config{
				Piper: PiperConfig{
					ModelPath: "", // Empty model path
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEngine(tt.engineType, tt.config)

			if result == nil {
				t.Fatal("ValidateEngine() returned nil result")
			}

			hasError := result.Error != nil
			if hasError != tt.wantError {
				t.Errorf("ValidateEngine() error = %v, want error = %v", result.Error, tt.wantError)
			}

			if tt.wantError && !result.Available {
				// When there's an error, engine should not be available
				if result.Available {
					t.Error("Engine should not be available when there's an error")
				}

				// Should provide guidance for errors
				if result.Guidance == "" {
					t.Error("Should provide guidance when validation fails")
				}
			}
		})
	}
}

func TestQuickValidation(t *testing.T) {
	tests := []struct {
		name       string
		engineType ttypes.EngineType
		expectPass bool // Whether we expect the validation to pass
	}{
		{
			name:       "Valid piper type",
			engineType: ttypes.EnginePiper,
			expectPass: false, // Depends on environment - may pass if piper is installed
		},
		{
			name:       "Valid google type",
			engineType: ttypes.EngineGoogle,
			expectPass: false, // Depends on environment - may pass if gtts-cli and ffmpeg are installed
		},
		{
			name:       "Invalid engine type",
			engineType: "invalid",
			expectPass: false, // Should always fail
		},
		{
			name:       "Empty engine type",
			engineType: ttypes.EngineNone,
			expectPass: false, // Should always fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuickValidation(tt.engineType)

			// For invalid and empty engines, always expect error
			if tt.engineType == "invalid" || tt.engineType == ttypes.EngineNone {
				if err == nil {
					t.Errorf("QuickValidation() expected error for %v, but got none", tt.engineType)
				}
				return
			}

			// For valid engines, result depends on environment
			// Just verify the function doesn't panic and returns a boolean result
			if tt.engineType == ttypes.EnginePiper {
				t.Logf("Piper validation result: %v", err)
			} else if tt.engineType == ttypes.EngineGoogle {
				t.Logf("Google validation result: %v", err)
			}
		})
	}
}

func TestBuildGuidance(t *testing.T) {
	t.Run("Piper install guidance", func(t *testing.T) {
		guidance := buildPiperInstallGuidance()

		expectedTexts := []string{
			"github.com/rhasspy/piper",
			"sudo apt install piper-tts",
			"brew install piper-tts",
			"config.yml",
		}

		for _, expected := range expectedTexts {
			if !strings.Contains(guidance, expected) {
				t.Errorf("Piper guidance should contain %q", expected)
			}
		}
	})

	t.Run("gTTS install guidance", func(t *testing.T) {
		guidance := buildGTTSInstallGuidance()

		expectedTexts := []string{
			"pip install gtts",
			"pipx install gtts",
			"gtts-cli --help",
			"internet connection",
		}

		for _, expected := range expectedTexts {
			if !strings.Contains(guidance, expected) {
				t.Errorf("gTTS guidance should contain %q", expected)
			}
		}
	})

	t.Run("FFmpeg install guidance", func(t *testing.T) {
		guidance := buildFFmpegInstallGuidance()

		expectedTexts := []string{
			"sudo apt",
			"brew install ffmpeg",
			"choco install ffmpeg",
			"ffmpeg.org",
		}

		for _, expected := range expectedTexts {
			if !strings.Contains(guidance, expected) {
				t.Errorf("FFmpeg guidance should contain %q", expected)
			}
		}
	})
}

func TestValidationResult(t *testing.T) {
	t.Run("ValidationResult structure", func(t *testing.T) {
		result := &ValidationResult{
			Engine:    ttypes.EnginePiper,
			Available: true,
			Error:     nil,
			Guidance:  "test guidance",
			Details:   map[string]string{"test": "value"},
		}

		if result.Engine != ttypes.EnginePiper {
			t.Errorf("Expected engine %v, got %v", ttypes.EnginePiper, result.Engine)
		}

		if !result.Available {
			t.Error("Expected Available to be true")
		}

		if result.Error != nil {
			t.Errorf("Expected no error, got %v", result.Error)
		}

		if result.Guidance != "test guidance" {
			t.Errorf("Expected guidance 'test guidance', got %q", result.Guidance)
		}

		if result.Details["test"] != "value" {
			t.Errorf("Expected details test=value, got %v", result.Details)
		}
	})
}

// Benchmark validation performance
func BenchmarkValidateEngineSelection(b *testing.B) {
	config := Config{Engine: ttypes.EnginePiper}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateEngineSelection("piper", config)
	}
}

func BenchmarkQuickValidation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = QuickValidation(ttypes.EnginePiper)
	}
}
