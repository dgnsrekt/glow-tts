package piper

import (
	"os"
	"testing"
	"time"
	
	"github.com/charmbracelet/glow/v2/tts"
)

func TestEngineV2ProcessPool(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	// Create engine with small pool
	config := Config{
		BinaryPath:     findPiperBinary(),
		SampleRate:     22050,
		MaxRestarts:    3,
		RequestTimeout: 5 * time.Second,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Initialize
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Check pool size
	engine.poolLock.RLock()
	poolSize := len(engine.processPool)
	engine.poolLock.RUnlock()
	
	if poolSize != 2 {
		t.Errorf("Expected pool size 2, got %d", poolSize)
	}
	
	// Check all processes are healthy
	healthyCount := 0
	engine.poolLock.RLock()
	for _, proc := range engine.processPool {
		if engine.isProcessHealthy(proc) {
			healthyCount++
		}
	}
	engine.poolLock.RUnlock()
	
	if healthyCount != poolSize {
		t.Errorf("Expected %d healthy processes, got %d", poolSize, healthyCount)
	}
}

func TestEngineV2FreshMode(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	// Enable fresh mode
	os.Setenv("PIPER_FRESH_MODE", "true")
	defer os.Unsetenv("PIPER_FRESH_MODE")
	
	config := Config{
		BinaryPath:     findPiperBinary(),
		SampleRate:     22050,
		RequestTimeout: 5 * time.Second,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Check fresh mode is enabled
	if !engine.freshMode {
		t.Error("Fresh mode should be enabled")
	}
	
	// Initialize
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// In fresh mode, pool should be empty
	engine.poolLock.RLock()
	poolSize := len(engine.processPool)
	engine.poolLock.RUnlock()
	
	if poolSize != 0 {
		t.Errorf("Expected empty pool in fresh mode, got %d", poolSize)
	}
	
	// Should still be available
	if !engine.IsAvailable() {
		t.Error("Engine should be available in fresh mode")
	}
}

func TestEngineV2ProcessRecycling(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := Config{
		BinaryPath:     findPiperBinary(),
		SampleRate:     22050,
		MaxRestarts:    3,
		RequestTimeout: 5 * time.Second,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Initialize
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Get a process
	proc, err := engine.getAvailableProcess()
	if err != nil {
		t.Fatalf("Failed to get process: %v", err)
	}
	
	// Simulate many requests to trigger recycling
	for i := 0; i < 101; i++ {
		proc.requestCount = int64(i)
	}
	
	initialGeneration := proc.generation
	
	// Release process (should trigger recycling)
	engine.releaseProcess(proc)
	
	// Wait for recycling
	time.Sleep(2 * time.Second)
	
	// Check if process was recycled (generation increased)
	if proc.generation <= initialGeneration {
		t.Error("Process should have been recycled")
	}
}

func TestEngineV2HealthMonitoring(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := Config{
		BinaryPath:          findPiperBinary(),
		SampleRate:          22050,
		HealthCheckInterval: 500 * time.Millisecond,
		MaxRestarts:         3,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Initialize
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Wait for health check to run
	time.Sleep(1 * time.Second)
	
	// Check that health monitoring is working
	engine.poolLock.RLock()
	for _, proc := range engine.processPool {
		if !engine.isProcessHealthy(proc) {
			t.Errorf("Process %d is not healthy", proc.id)
		}
	}
	engine.poolLock.RUnlock()
}

func TestEngineV2Statistics(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := Config{
		BinaryPath: findPiperBinary(),
		SampleRate: 22050,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	// Initialize
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	
	// Get initial statistics
	stats := engine.GetStatistics()
	
	if stats["total_requests"] != 0 {
		t.Errorf("Expected 0 total requests, got %d", stats["total_requests"])
	}
	
	if stats["processes_created"] < 2 {
		t.Errorf("Expected at least 2 processes created, got %d", stats["processes_created"])
	}
}

// BenchmarkEngineV2ProcessPool benchmarks the process pool performance
func BenchmarkEngineV2ProcessPool(b *testing.B) {
	if findPiperBinary() == "" {
		b.Skip("Piper binary not found, skipping benchmark")
	}
	
	config := Config{
		BinaryPath:     findPiperBinary(),
		SampleRate:     22050,
		RequestTimeout: 5 * time.Second,
	}
	
	engine, err := NewEngineV2(config)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Shutdown()
	
	engineConfig := tts.EngineConfig{
		Voice:  "test",
		Rate:   1.0,
		Volume: 1.0,
	}
	
	if err := engine.Initialize(engineConfig); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			proc, err := engine.getAvailableProcess()
			if err != nil {
				b.Errorf("Failed to get process: %v", err)
				continue
			}
			engine.releaseProcess(proc)
		}
	})
}