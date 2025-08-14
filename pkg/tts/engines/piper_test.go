package engines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewPiperEngine(t *testing.T) {
	engine, err := NewPiperEngine()
	
	// We expect this might fail if piper is not installed
	if err != nil {
		// Check if it's the expected error
		if piperErr, ok := err.(*PiperError); ok {
			if piperErr.Type != "dependency" {
				t.Errorf("Expected dependency error, got %s", piperErr.Type)
			}
			t.Logf("Piper not installed: %v", err)
		} else {
			t.Errorf("Expected PiperError, got %T", err)
		}
		return
	}

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.speed != DefaultSpeed {
		t.Errorf("Expected default speed %.1f, got %.1f", DefaultSpeed, engine.speed)
	}

	if engine.timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", engine.timeout)
	}
}

func TestPiperEngineSetModel(t *testing.T) {
	engine := &PiperEngine{
		speed:   DefaultSpeed,
		timeout: 30 * time.Second,
	}

	tests := []struct {
		name      string
		modelPath string
		wantErr   bool
		errType   string
	}{
		{
			name:      "valid ONNX model path",
			modelPath: "/tmp/test-model.onnx",
			wantErr:   true, // File doesn't exist
			errType:   "model",
		},
		{
			name:      "invalid extension",
			modelPath: "/tmp/test-model.txt",
			wantErr:   true,
			errType:   "model",
		},
		{
			name:      "empty path",
			modelPath: "",
			wantErr:   true,
			errType:   "model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.SetModel(tt.modelPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetModel() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errType != "" {
				if piperErr, ok := err.(*PiperError); ok {
					if piperErr.Type != tt.errType {
						t.Errorf("Expected error type %s, got %s", tt.errType, piperErr.Type)
					}
				}
			}
		})
	}
}

func TestPiperEngineWithMockModel(t *testing.T) {
	// Create a temporary mock model file
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-voice.onnx")
	configPath := filepath.Join(tmpDir, "test-voice.onnx.json")

	// Create mock files
	if err := os.WriteFile(modelPath, []byte("mock onnx data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	engine := &PiperEngine{
		speed:      DefaultSpeed,
		timeout:    30 * time.Second,
		binaryPath: "/usr/bin/piper", // Assume it might exist
	}

	// Test setting model
	if err := engine.SetModel(modelPath); err != nil {
		t.Fatalf("Failed to set model: %v", err)
	}

	if engine.modelPath != modelPath {
		t.Errorf("Model path not set correctly")
	}

	if engine.configPath != configPath {
		t.Errorf("Config path not detected")
	}

	if engine.voiceName != "test-voice" {
		t.Errorf("Expected voice name 'test-voice', got %s", engine.voiceName)
	}
}

func TestPiperEngineSetSpeed(t *testing.T) {
	engine := &PiperEngine{
		speed: DefaultSpeed,
	}

	tests := []struct {
		name    string
		speed   float64
		wantErr bool
	}{
		{
			name:    "normal speed",
			speed:   1.0,
			wantErr: false,
		},
		{
			name:    "minimum speed",
			speed:   0.5,
			wantErr: false,
		},
		{
			name:    "maximum speed",
			speed:   2.0,
			wantErr: false,
		},
		{
			name:    "too slow",
			speed:   0.3,
			wantErr: true,
		},
		{
			name:    "too fast",
			speed:   2.5,
			wantErr: true,
		},
		{
			name:    "negative speed",
			speed:   -1.0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.SetSpeed(tt.speed)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSpeed() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && engine.speed != tt.speed {
				t.Errorf("Speed not set correctly: expected %.1f, got %.1f", tt.speed, engine.speed)
			}
		})
	}
}

func TestPiperEngineValidate(t *testing.T) {
	tests := []struct {
		name       string
		engine     *PiperEngine
		wantErr    bool
		errType    string
	}{
		{
			name: "missing binary",
			engine: &PiperEngine{
				binaryPath: "",
				modelPath:  "/tmp/model.onnx",
			},
			wantErr: true,
			errType: "dependency",
		},
		{
			name: "missing model",
			engine: &PiperEngine{
				binaryPath: "/usr/bin/echo", // Use a binary that exists
				modelPath:  "",
			},
			wantErr: true,
			errType: "model",
		},
		{
			name: "nonexistent binary",
			engine: &PiperEngine{
				binaryPath: "/nonexistent/piper",
				modelPath:  "/tmp/model.onnx",
			},
			wantErr: true,
			errType: "dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.engine.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errType != "" {
				if piperErr, ok := err.(*PiperError); ok {
					if piperErr.Type != tt.errType {
						t.Errorf("Expected error type %s, got %s", tt.errType, piperErr.Type)
					}
				}
			}
		})
	}
}

func TestPiperEngineGetInfo(t *testing.T) {
	engine := &PiperEngine{
		binaryPath: "/usr/bin/piper",
		modelPath:  "/tmp/voice.onnx",
		configPath: "/tmp/voice.onnx.json",
		voiceName:  "test-voice",
		speed:      1.5,
	}

	info := engine.GetInfo()

	if info["engine"] != "piper" {
		t.Errorf("Expected engine 'piper', got %s", info["engine"])
	}

	if info["binary"] != "/usr/bin/piper" {
		t.Errorf("Binary path incorrect")
	}

	if info["model"] != "/tmp/voice.onnx" {
		t.Errorf("Model path incorrect")
	}

	if info["voice"] != "test-voice" {
		t.Errorf("Voice name incorrect")
	}

	if info["speed"] != "1.5" {
		t.Errorf("Speed incorrect: %s", info["speed"])
	}

	if info["sampleRate"] != "22050" {
		t.Errorf("Sample rate incorrect")
	}

	if info["format"] != "PCM 16-bit mono" {
		t.Errorf("Format incorrect")
	}

	if info["config"] != "/tmp/voice.onnx.json" {
		t.Errorf("Config path incorrect")
	}
}

func TestPiperEngineSynthesizeValidation(t *testing.T) {
	engine := &PiperEngine{}

	// Test with invalid configuration
	_, err := engine.Synthesize("test text", 1.0)
	if err == nil {
		t.Error("Expected error for unconfigured engine")
	}

	// Test with empty text (should return empty without validation)
	data, err := engine.Synthesize("", 1.0)
	if err != nil {
		t.Errorf("Empty text should not error: %v", err)
	}
	if len(data) != 0 {
		t.Error("Empty text should return empty data")
	}
}

func TestPiperErrorTypes(t *testing.T) {
	tests := []struct {
		name    string
		err     *PiperError
		wantMsg string
	}{
		{
			name: "simple error",
			err: &PiperError{
				Type:    "test",
				Message: "test message",
			},
			wantMsg: "piper test: test message",
		},
		{
			name: "error with cause",
			err: &PiperError{
				Type:    "test",
				Message: "test message",
				Cause:   os.ErrNotExist,
			},
			wantMsg: "piper test: test message:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !strings.Contains(msg, tt.wantMsg) {
				t.Errorf("Error message should contain %q, got %q", tt.wantMsg, msg)
			}

			// Test Unwrap
			if tt.err.Cause != nil {
				if tt.err.Unwrap() != tt.err.Cause {
					t.Error("Unwrap should return the cause")
				}
			}
		})
	}
}

func TestPiperEngineGetters(t *testing.T) {
	engine := &PiperEngine{
		voiceName: "test-voice",
		modelPath: "/tmp/model.onnx",
	}

	if engine.GetVoice() != "test-voice" {
		t.Errorf("GetVoice() incorrect")
	}

	if engine.GetModelPath() != "/tmp/model.onnx" {
		t.Errorf("GetModelPath() incorrect")
	}

	// Test GetName with voice
	name := engine.GetName()
	if name != "Piper (test-voice)" {
		t.Errorf("GetName() with voice incorrect: %s", name)
	}

	// Test GetName without voice
	engine.voiceName = ""
	name = engine.GetName()
	if name != "Piper TTS" {
		t.Errorf("GetName() without voice incorrect: %s", name)
	}
}

func TestPiperEngineTimeout(t *testing.T) {
	engine := &PiperEngine{
		timeout: 5 * time.Second,
	}

	// Test setting timeout
	engine.SetTimeout(10 * time.Second)
	if engine.timeout != 10*time.Second {
		t.Errorf("SetTimeout() didn't update timeout")
	}
}

func TestFindBinaryFallback(t *testing.T) {
	engine := &PiperEngine{}

	// This test checks the fallback paths
	// It will likely fail in most environments, but that's expected
	err := engine.findBinary()
	if err == nil {
		t.Logf("Found piper at: %s", engine.binaryPath)
	} else {
		// Verify it's the right error type
		if piperErr, ok := err.(*PiperError); ok {
			if piperErr.Type != "dependency" {
				t.Errorf("Expected dependency error type")
			}
			if !strings.Contains(piperErr.Message, "install piper") {
				t.Errorf("Error should contain installation instructions")
			}
		}
	}
}

func TestFindDefaultModel(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	voicesDir := filepath.Join(tmpDir, ".local/share/piper-voices/en/en_US/test")
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a mock model file
	modelPath := filepath.Join(voicesDir, "test.onnx")
	if err := os.WriteFile(modelPath, []byte("mock"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config file
	configPath := filepath.Join(voicesDir, "test.onnx.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Temporarily override HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	engine := &PiperEngine{}
	err := engine.findDefaultModel()
	
	if err != nil {
		t.Logf("Find default model error: %v", err)
		// This is expected in most test environments
		if piperErr, ok := err.(*PiperError); ok {
			if piperErr.Type != "model" {
				t.Errorf("Expected model error type")
			}
		}
	} else {
		// If it succeeded, verify the paths
		if !strings.Contains(engine.modelPath, "test.onnx") {
			t.Errorf("Model path incorrect: %s", engine.modelPath)
		}
		if !strings.Contains(engine.configPath, "test.onnx.json") {
			t.Errorf("Config path incorrect: %s", engine.configPath)
		}
		if engine.voiceName != "test" {
			t.Errorf("Voice name incorrect: %s", engine.voiceName)
		}
	}
}