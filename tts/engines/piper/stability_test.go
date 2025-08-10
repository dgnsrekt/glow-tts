package piper

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
	
	"github.com/charmbracelet/glow/v2/tts"
)

// TestV2StabilityUnderLoad tests the V2 engine under concurrent load
func TestV2StabilityUnderLoad(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := Config{
		BinaryPath:          findPiperBinary(),
		SampleRate:          22050,
		MaxRestarts:         5,
		RequestTimeout:      10 * time.Second,
		HealthCheckInterval: 2 * time.Second,
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
	
	// Test concurrent requests
	const numWorkers = 5
	const requestsPerWorker = 10
	
	var wg sync.WaitGroup
	errors := make(chan error, numWorkers*requestsPerWorker)
	
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				text := fmt.Sprintf("Worker %d request %d test", workerID, j)
				
				// Try to generate audio
				_, err := engine.GenerateAudio(text)
				if err != nil {
					errors <- fmt.Errorf("worker %d request %d: %w", workerID, j, err)
				}
				
				// Small delay between requests
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}
	
	// Wait for all workers
	wg.Wait()
	close(errors)
	
	// Check errors
	errorCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}
	
	// Get statistics
	stats := engine.GetStatistics()
	t.Logf("Statistics: %+v", stats)
	
	// Allow some failures but not too many
	maxFailureRate := 0.2 // 20% failure rate acceptable
	totalRequests := numWorkers * requestsPerWorker
	if float64(errorCount)/float64(totalRequests) > maxFailureRate {
		t.Errorf("Too many failures: %d/%d (%.1f%%)", 
			errorCount, totalRequests, 
			float64(errorCount)/float64(totalRequests)*100)
	}
	
	// Check that we had some successful requests
	if stats["success_requests"] == 0 {
		t.Error("No successful requests")
	}
	
	// Check if restarts happened (expected under load)
	if stats["restarts"] > 0 {
		t.Logf("Engine performed %d restarts under load", stats["restarts"])
	}
}

// TestV2ProcessRecovery tests that the V2 engine recovers from process crashes
func TestV2ProcessRecovery(t *testing.T) {
	// Skip if Piper is not available  
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	config := Config{
		BinaryPath:          findPiperBinary(),
		SampleRate:          22050,
		MaxRestarts:         3,
		RequestTimeout:      5 * time.Second,
		HealthCheckInterval: 1 * time.Second,
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
	
	// Simulate process crash by killing a process
	engine.poolLock.RLock()
	if len(engine.processPool) > 0 {
		proc := engine.processPool[0]
		engine.poolLock.RUnlock()
		
		// Kill the process
		if proc.cmd != nil && proc.cmd.Process != nil {
			proc.cmd.Process.Kill()
			log.Printf("Killed process %d for testing", proc.id)
		}
		
		// Wait for health check to detect and recover
		time.Sleep(3 * time.Second)
		
		// Try to use the engine - should work with recovered process
		_, err := engine.GenerateAudio("Test after crash")
		if err != nil {
			t.Errorf("Failed to generate audio after crash recovery: %v", err)
		}
		
		// Check statistics
		stats := engine.GetStatistics()
		if stats["restarts"] == 0 {
			t.Error("Expected at least one restart after crash")
		}
	} else {
		engine.poolLock.RUnlock()
		t.Skip("No processes in pool to test")
	}
}

// TestV2FreshModeStability tests fresh mode stability
func TestV2FreshModeStability(t *testing.T) {
	// Skip if Piper is not available
	if findPiperBinary() == "" {
		t.Skip("Piper binary not found, skipping test")
	}
	
	// Enable fresh mode
	t.Setenv("PIPER_FRESH_MODE", "true")
	
	config := Config{
		BinaryPath:     findPiperBinary(),
		SampleRate:     22050,
		RequestTimeout: 10 * time.Second,
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
	
	// Make several requests - each should use a fresh process
	successCount := 0
	for i := 0; i < 5; i++ {
		text := fmt.Sprintf("Fresh mode test %d", i)
		_, err := engine.GenerateAudio(text)
		if err == nil {
			successCount++
		} else {
			t.Logf("Request %d failed: %v", i, err)
		}
		
		// Delay between requests
		time.Sleep(500 * time.Millisecond)
	}
	
	// In fresh mode, we should have high success rate
	if successCount < 3 {
		t.Errorf("Too few successful requests in fresh mode: %d/5", successCount)
	}
	
	// Check statistics
	stats := engine.GetStatistics()
	t.Logf("Fresh mode statistics: %+v", stats)
	
	// In fresh mode, we should have created many processes
	if stats["processes_created"] < int64(successCount) {
		t.Errorf("Expected at least %d processes created, got %d", 
			successCount, stats["processes_created"])
	}
}