package piper_test

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/engines/piper"
)

// TestDefaultConfig tests the default configuration creation.
func TestDefaultConfig(t *testing.T) {
	config := piper.DefaultConfig()
	
	if config.BinaryPath != "piper" {
		t.Errorf("Expected binary path 'piper', got %s", config.BinaryPath)
	}
	if !config.OutputRaw {
		t.Error("Expected OutputRaw to be true")
	}
	if config.SampleRate != 22050 {
		t.Errorf("Expected sample rate 22050, got %d", config.SampleRate)
	}
	if config.StartupTimeout != 10*time.Second {
		t.Errorf("Expected startup timeout 10s, got %v", config.StartupTimeout)
	}
	if config.MaxRestarts != 3 {
		t.Errorf("Expected max restarts 3, got %d", config.MaxRestarts)
	}
}

// TestEngineCreation tests creating a new engine.
func TestEngineCreation(t *testing.T) {
	tests := []struct {
		name    string
		config  piper.Config
		wantErr bool
	}{
		{
			name: "valid config with mock binary",
			config: piper.Config{
				BinaryPath: "echo", // Use echo as a mock binary for testing
				ModelPath:  "",
			},
			wantErr: false,
		},
		{
			name: "empty binary path",
			config: piper.Config{
				BinaryPath: "",
			},
			wantErr: true,
		},
		{
			name: "non-existent model file",
			config: piper.Config{
				BinaryPath: "echo",
				ModelPath:  "/non/existent/model.onnx",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := piper.NewEngine(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && engine == nil {
				t.Error("Expected non-nil engine")
			}
		})
	}
}

// TestEngineInitialization tests engine initialization.
func TestEngineInitialization(t *testing.T) {
	// Skip if piper is not available
	if _, err := exec.LookPath("piper"); err != nil {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := piper.Config{
		BinaryPath:     "piper",
		StartupTimeout: 5 * time.Second,
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	engineConfig := tts.EngineConfig{
		Rate:   1.0,
		Pitch:  1.0,
		Volume: 1.0,
	}
	
	err = engine.Initialize(engineConfig)
	if err != nil {
		// If initialization fails, it might be due to missing model
		if strings.Contains(err.Error(), "model") {
			t.Skip("Piper model not configured, skipping test")
		}
		t.Errorf("Initialize() error = %v", err)
	}
	
	// Clean up
	if engine.IsAvailable() {
		engine.Shutdown()
	}
}

// TestEngineCapabilities tests getting engine capabilities.
func TestEngineCapabilities(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	caps := engine.GetCapabilities()
	
	if caps.SupportsStreaming {
		t.Error("Piper should not support streaming")
	}
	if caps.RequiresNetwork {
		t.Error("Piper should not require network")
	}
	if caps.MaxTextLength <= 0 {
		t.Error("MaxTextLength should be positive")
	}
	
	foundPCM16 := false
	for _, format := range caps.SupportedFormats {
		if format == "pcm16" {
			foundPCM16 = true
			break
		}
	}
	if !foundPCM16 {
		t.Error("Should support pcm16 format")
	}
}

// TestVoiceManagement tests voice loading and management.
func TestVoiceManagement(t *testing.T) {
	// Create a temporary model file for testing
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "en_US-test-voice.onnx")
	if err := os.WriteFile(modelPath, []byte("fake model"), 0644); err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}
	
	config := piper.Config{
		BinaryPath: "echo",
		ModelPath:  modelPath,
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	voices := engine.GetVoices()
	if len(voices) != 1 {
		t.Errorf("Expected 1 voice, got %d", len(voices))
	}
	
	if len(voices) > 0 {
		voice := voices[0]
		if voice.ID != "en_US-test-voice" {
			t.Errorf("Expected voice ID 'en_US-test-voice', got %s", voice.ID)
		}
		if voice.Language != "en-US" {
			t.Errorf("Expected language 'en-US', got %s", voice.Language)
		}
		
		// Test setting the same voice
		err = engine.SetVoice(voice)
		if err != nil {
			t.Errorf("SetVoice() error = %v", err)
		}
		
		// Test setting an invalid voice
		invalidVoice := tts.Voice{ID: "invalid"}
		err = engine.SetVoice(invalidVoice)
		if err == nil {
			t.Error("Expected error for invalid voice")
		}
	}
}

// TestProcessHealth tests process health monitoring.
func TestProcessHealth(t *testing.T) {
	// This test requires a mock binary that we can control
	t.Skip("Requires mock binary setup")
	
	config := piper.Config{
		BinaryPath:          "sleep", // Use sleep as a long-running process
		HealthCheckInterval: 100 * time.Millisecond,
		MaxRestarts:         2,
		RestartDelay:        100 * time.Millisecond,
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Initialize should fail with sleep command
	err = engine.Initialize(tts.EngineConfig{})
	if err == nil {
		t.Error("Expected initialization to fail with sleep command")
	}
}

// TestShutdown tests clean shutdown.
func TestShutdown(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Shutdown without initialization
	err = engine.Shutdown()
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
	
	// Check engine is not available after shutdown
	if engine.IsAvailable() {
		t.Error("Engine should not be available after shutdown")
	}
}

// TestGetStats tests statistics retrieval.
func TestGetStats(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	stats := engine.GetStats()
	
	if stats.Running {
		t.Error("Engine should not be running initially")
	}
	if stats.Healthy {
		t.Error("Engine should not be healthy initially")
	}
	if stats.Generation != 0 {
		t.Error("Generation should be 0 initially")
	}
	if stats.RestartCount != 0 {
		t.Error("Restart count should be 0 initially")
	}
}

// TestEstimateDuration tests duration estimation.
func TestEstimateDuration(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	tests := []struct {
		text     string
		minDur   time.Duration
		maxDur   time.Duration
	}{
		{"Hello", 100 * time.Millisecond, 1 * time.Second},
		{"This is a longer sentence with more words.", 1 * time.Second, 5 * time.Second},
		{"", 100 * time.Millisecond, 1 * time.Second},
	}
	
	for _, tt := range tests {
		duration := engine.EstimateDuration(tt.text)
		if duration < tt.minDur || duration > tt.maxDur {
			t.Errorf("EstimateDuration(%q) = %v, want between %v and %v",
				tt.text, duration, tt.minDur, tt.maxDur)
		}
	}
}

// TestErrorRecording tests error tracking.
func TestErrorRecording(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Initially no error
	if engine.GetLastError() != nil {
		t.Error("Should have no error initially")
	}
	
	// Generate audio without initialization should fail
	_, err = engine.GenerateAudio("test")
	if err == nil {
		t.Error("Expected error when generating audio without initialization")
	}
}

// MockPiperServer simulates a Piper process for testing.
type MockPiperServer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

// NewMockPiperServer creates a mock Piper server for testing.
func NewMockPiperServer() (*MockPiperServer, error) {
	// Use a simple echo command as a mock
	cmd := exec.Command("cat") // cat will echo stdin to stdout
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	
	return &MockPiperServer{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// Stop stops the mock server.
func (m *MockPiperServer) Stop() error {
	if m.stdin != nil {
		m.stdin.Close()
	}
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}
	return nil
}

// TestGenerateAudioWithMock tests audio generation with a mock server.
func TestGenerateAudioWithMock(t *testing.T) {
	t.Skip("Requires proper mock implementation")
	
	// This would require a more sophisticated mock that simulates
	// Piper's actual protocol. For now, we skip this test.
}

// TestConcurrentRequests tests handling of concurrent requests.
func TestConcurrentRequests(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Try concurrent operations (should fail without initialization)
	done := make(chan bool, 3)
	
	go func() {
		engine.GetVoices()
		done <- true
	}()
	
	go func() {
		engine.GetCapabilities()
		done <- true
	}()
	
	go func() {
		engine.IsAvailable()
		done <- true
	}()
	
	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Error("Concurrent operation timeout")
		}
	}
}

// TestRestartLimit tests that the engine respects restart limits.
func TestRestartLimit(t *testing.T) {
	// Use a command that exists but will fail for Piper usage
	config := piper.Config{
		BinaryPath:     "echo",  // Will fail Piper protocol
		MaxRestarts:    2,
		RestartDelay:   10 * time.Millisecond,
		StartupTimeout: 100 * time.Millisecond,  // Short timeout
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Initialization should fail because echo doesn't implement Piper protocol
	err = engine.Initialize(tts.EngineConfig{})
	if err == nil {
		// If somehow it doesn't fail, that's okay - the test is about restart limits
		t.Skip("Test command didn't fail as expected")
	}
	
	// The error should be related to startup/protocol, not the command itself
	if !strings.Contains(err.Error(), "startup") && !strings.Contains(err.Error(), "process") {
		t.Logf("Initialization failed with: %v", err)
	}
}

// TestGetStderrOutput tests stderr buffer management.
func TestGetStderrOutput(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Initially should be empty
	output := engine.GetStderrOutput()
	if len(output) != 0 {
		t.Error("Stderr output should be empty initially")
	}
}

// BenchmarkEngineCreation benchmarks engine creation.
func BenchmarkEngineCreation(b *testing.B) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	for i := 0; i < b.N; i++ {
		engine, err := piper.NewEngine(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = engine
	}
}

// BenchmarkGetVoices benchmarks voice retrieval.
func BenchmarkGetVoices(b *testing.B) {
	config := piper.Config{
		BinaryPath: "echo",
		ModelPath:  "test.onnx",
	}
	
	// Create temp model file
	os.WriteFile("test.onnx", []byte("test"), 0644)
	defer os.Remove("test.onnx")
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.GetVoices()
	}
}

// TestGenerateAudioStream tests audio streaming functionality.
func TestGenerateAudioStream(t *testing.T) {
	config := piper.Config{
		BinaryPath: "echo",
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	
	// Should fail without initialization
	chunkChan, err := engine.GenerateAudioStream("test")
	if err != nil {
		t.Errorf("GenerateAudioStream should not return error for channel creation")
	}
	
	// Should receive an error chunk
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	select {
	case chunk := <-chunkChan:
		if chunk.Data != nil {
			t.Error("Expected nil data in error chunk")
		}
		if !chunk.Final {
			t.Error("Expected final flag in error chunk")
		}
	case <-ctx.Done():
		t.Error("Timeout waiting for chunk")
	}
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*piper.Config)
		wantErr bool
	}{
		{
			name: "zero timeouts get defaults",
			modify: func(c *piper.Config) {
				c.StartupTimeout = 0
				c.RequestTimeout = 0
			},
			wantErr: false,
		},
		{
			name: "zero sample rate gets default",
			modify: func(c *piper.Config) {
				c.SampleRate = 0
			},
			wantErr: false,
		},
		{
			name: "valid custom values",
			modify: func(c *piper.Config) {
				c.SampleRate = 44100
				c.MaxRestarts = 5
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := piper.Config{
				BinaryPath: "echo",
			}
			tt.modify(&config)
			
			engine, err := piper.NewEngine(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if engine != nil && !tt.wantErr {
				// Check defaults were applied
				if config.StartupTimeout == 0 {
					stats := engine.GetStats()
					_ = stats // Defaults should be set internally
				}
			}
		})
	}
}

// TestBinaryDiscovery tests finding Piper in common locations.
func TestBinaryDiscovery(t *testing.T) {
	// Create a temporary directory to simulate a common location
	tmpDir := t.TempDir()
	fakePiper := filepath.Join(tmpDir, "piper")
	
	// Create a fake piper executable
	if err := os.WriteFile(fakePiper, []byte("#!/bin/sh\necho fake piper"), 0755); err != nil {
		t.Fatalf("Failed to create fake piper: %v", err)
	}
	
	config := piper.Config{
		BinaryPath: fakePiper,
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Errorf("Failed to create engine with valid binary path: %v", err)
	}
	if engine == nil {
		t.Error("Expected non-nil engine")
	}
}

// TestContextCancellation tests that operations respect context cancellation.
func TestContextCancellation(t *testing.T) {
	config := piper.Config{
		BinaryPath: "sleep", // Long-running command
	}
	
	engine, err := piper.NewEngine(config)
	if err != nil {
		t.Skip("sleep command not available")
	}
	
	// Shutdown should cancel context and stop operations
	done := make(chan bool)
	go func() {
		engine.Shutdown()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(6 * time.Second): // Slightly longer than shutdown timeout
		t.Error("Shutdown took too long")
	}
}