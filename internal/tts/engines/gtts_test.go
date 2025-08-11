package engines

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/internal/cache"
)

// TestGTTSEngine_NewGTTSEngine tests engine creation with various configurations
func TestGTTSEngine_NewGTTSEngine(t *testing.T) {
	tests := []struct {
		name        string
		config      GTTSConfig
		expectError bool
	}{
		{
			name:        "default configuration",
			config:      GTTSConfig{},
			expectError: false,
		},
		{
			name: "custom language",
			config: GTTSConfig{
				Language: "es",
			},
			expectError: false,
		},
		{
			name: "slow speech enabled",
			config: GTTSConfig{
				Language: "en",
				Slow:     true,
			},
			expectError: false,
		},
		{
			name: "custom temp directory",
			config: GTTSConfig{
				Language: "fr",
				TempDir:  "/tmp/test-gtts",
			},
			expectError: false,
		},
		{
			name: "with cache configuration",
			config: GTTSConfig{
				Language: "en",
				CacheConfig: &cache.CacheConfig{
					MemoryCapacity: 10 * 1024 * 1024, // 10MB
					DiskCapacity:   100 * 1024 * 1024, // 100MB
					DiskPath:       "/tmp/test-cache",
				},
			},
			expectError: false,
		},
		{
			name: "custom rate limiting",
			config: GTTSConfig{
				Language:          "en",
				RequestsPerMinute: 30,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewGTTSEngine(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if engine == nil {
				t.Fatal("Engine should not be nil")
			}
			
			// Check defaults
			if engine.language == "" {
				t.Error("Language should have default value")
			}
			
			if engine.sampleRate == 0 {
				t.Error("Sample rate should have default value")
			}
			
			if engine.rateLimiter == nil {
				t.Error("Rate limiter should not be nil")
			}
			
			// Clean up
			engine.Close()
		})
	}
}

// TestGTTSEngine_GetInfo tests engine info
func TestGTTSEngine_GetInfo(t *testing.T) {
	engine, err := NewGTTSEngine(GTTSConfig{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	info := engine.GetInfo()

	if info.Name != "gtts" {
		t.Errorf("Expected name 'gtts', got '%s'", info.Name)
	}

	if info.SampleRate != 22050 {
		t.Errorf("Expected sample rate 22050, got %d", info.SampleRate)
	}

	if info.Channels != 1 {
		t.Errorf("Expected 1 channel, got %d", info.Channels)
	}

	if info.BitDepth != 16 {
		t.Errorf("Expected 16-bit depth, got %d", info.BitDepth)
	}

	if info.MaxTextSize != 5000 {
		t.Errorf("Expected max text size 5000, got %d", info.MaxTextSize)
	}

	if !info.IsOnline {
		t.Error("gTTS should be marked as online")
	}
}

// TestGTTSEngine_Validate tests engine validation
func TestGTTSEngine_Validate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation test in short mode (requires gtts-cli and ffmpeg)")
	}

	engine, err := NewGTTSEngine(GTTSConfig{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	err = engine.Validate()
	
	// If gtts-cli or ffmpeg is not installed, the test should fail with a helpful message
	if err != nil {
		if strings.Contains(err.Error(), "not found in PATH") {
			t.Skipf("Skipping validation test: %v", err)
		}
		if strings.Contains(err.Error(), "test synthesis failed") && strings.Contains(err.Error(), "Check internet connection") {
			t.Skipf("Skipping validation test: no internet connection or gTTS blocked: %v", err)
		}
		t.Errorf("Validation failed: %v", err)
	}
}

// TestGTTSEngine_SynthesizeWithMockCommand tests synthesis with basic input validation
func TestGTTSEngine_SynthesizeWithMockCommand(t *testing.T) {
	engine, err := NewGTTSEngine(GTTSConfig{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test empty text
	_, err = engine.Synthesize(context.Background(), "", 1.0)
	if err == nil || !strings.Contains(err.Error(), "text cannot be empty") {
		t.Error("Expected error for empty text")
	}

	// Test text too long
	longText := strings.Repeat("a", 5001)
	_, err = engine.Synthesize(context.Background(), longText, 1.0)
	if err == nil || !strings.Contains(err.Error(), "text too long") {
		t.Error("Expected error for text too long")
	}
}

// TestGTTSEngine_SynthesizeIntegration tests actual synthesis (requires network and tools)
func TestGTTSEngine_SynthesizeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if gtts-cli and ffmpeg are available
	if _, err := exec.LookPath("gtts-cli"); err != nil {
		t.Skip("Skipping integration test: gtts-cli not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("Skipping integration test: ffmpeg not available")
	}

	engine, err := NewGTTSEngine(GTTSConfig{
		Language:          "en",
		RequestsPerMinute: 10, // Conservative for testing
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test synthesis
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	audio, err := engine.Synthesize(ctx, "Hello, this is a test.", 1.0)
	if err != nil {
		// If this fails due to network issues, skip rather than fail
		if strings.Contains(err.Error(), "timeout") || 
		   strings.Contains(err.Error(), "network") ||
		   strings.Contains(err.Error(), "connection") {
			t.Skipf("Skipping synthesis test due to network issues: %v", err)
		}
		t.Fatalf("Synthesis failed: %v", err)
	}

	if len(audio) == 0 {
		t.Error("Synthesis should produce audio data")
	}

	// Verify it's reasonable size (PCM audio should be substantial)
	if len(audio) < 1000 {
		t.Errorf("Audio data seems too small: %d bytes", len(audio))
	}
}

// TestGTTSEngine_SynthesizeWithSpeed tests speed adjustment
func TestGTTSEngine_SynthesizeWithSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping speed test in short mode")
	}

	// Check if tools are available
	if _, err := exec.LookPath("gtts-cli"); err != nil {
		t.Skip("Skipping speed test: gtts-cli not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("Skipping speed test: ffmpeg not available")
	}

	engine, err := NewGTTSEngine(GTTSConfig{
		Language:          "en",
		RequestsPerMinute: 5, // Very conservative for testing multiple requests
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	testText := "Speed test."
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	speeds := []float64{0.5, 1.0, 1.5, 2.0}
	
	for _, speed := range speeds {
		t.Run(fmt.Sprintf("speed_%.1f", speed), func(t *testing.T) {
			audio, err := engine.Synthesize(ctx, testText, speed)
			if err != nil {
				if strings.Contains(err.Error(), "timeout") || 
				   strings.Contains(err.Error(), "network") {
					t.Skipf("Skipping speed test due to network issues: %v", err)
				}
				t.Errorf("Synthesis at speed %.1f failed: %v", speed, err)
				return
			}

			if len(audio) == 0 {
				t.Errorf("Synthesis at speed %.1f produced no audio", speed)
			}
		})
		
		// Add delay between requests to respect rate limiting
		if speed != speeds[len(speeds)-1] {
			time.Sleep(15 * time.Second)
		}
	}
}

// TestGTTSEngine_SynthesizeWithCaching tests caching functionality
func TestGTTSEngine_SynthesizeWithCaching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping caching test in short mode")
	}

	// Check if tools are available
	if _, err := exec.LookPath("gtts-cli"); err != nil {
		t.Skip("Skipping caching test: gtts-cli not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("Skipping caching test: ffmpeg not available")
	}

	// Create temp cache directory
	cacheDir, err := os.MkdirTemp("", "gtts-test-cache-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache dir: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	engine, err := NewGTTSEngine(GTTSConfig{
		Language:          "en",
		RequestsPerMinute: 10,
		CacheConfig: &cache.CacheConfig{
			MemoryCapacity: 10 * 1024 * 1024, // 10MB
			DiskCapacity:   50 * 1024 * 1024,  // 50MB
			DiskPath:       cacheDir,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create engine with cache: %v", err)
	}
	defer engine.Close()

	testText := "Cache test."
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First synthesis (should hit network)
	start := time.Now()
	audio1, err := engine.Synthesize(ctx, testText, 1.0)
	firstDuration := time.Since(start)
	
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || 
		   strings.Contains(err.Error(), "network") {
			t.Skip("Skipping caching test due to network issues")
		}
		t.Fatalf("First synthesis failed: %v", err)
	}

	// Second synthesis (should hit cache)
	start = time.Now()
	audio2, err := engine.Synthesize(ctx, testText, 1.0)
	secondDuration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Second synthesis failed: %v", err)
	}

	// Verify results match
	if len(audio1) != len(audio2) {
		t.Errorf("Cached audio length differs: %d vs %d", len(audio1), len(audio2))
	}

	// Second should be much faster (cache hit)
	if secondDuration >= firstDuration {
		t.Logf("First synthesis: %v, Second: %v", firstDuration, secondDuration)
		// Don't fail the test as network timing can vary
		// t.Errorf("Second synthesis should be faster (cached): %v >= %v", secondDuration, firstDuration)
	}

	// Check cache stats
	stats := engine.GetCacheStats()
	if stats == nil {
		t.Error("Cache stats should not be nil")
	}
}

// TestGTTSEngine_LanguageAndSlow tests language and slow speech settings
func TestGTTSEngine_LanguageAndSlow(t *testing.T) {
	engine, err := NewGTTSEngine(GTTSConfig{
		Language: "es",
		Slow:     true,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test getters
	if engine.GetLanguage() != "es" {
		t.Errorf("Expected language 'es', got '%s'", engine.GetLanguage())
	}

	if !engine.GetSlow() {
		t.Error("Expected slow speech to be enabled")
	}

	// Test setters
	engine.SetLanguage("fr")
	if engine.GetLanguage() != "fr" {
		t.Errorf("Expected language 'fr' after set, got '%s'", engine.GetLanguage())
	}

	engine.SetSlow(false)
	if engine.GetSlow() {
		t.Error("Expected slow speech to be disabled after set")
	}
}

// TestGTTSEngine_RaceConditions tests concurrent synthesis requests
func TestGTTSEngine_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	engine, err := NewGTTSEngine(GTTSConfig{
		Language:          "en",
		RequestsPerMinute: 1, // Very low to test rate limiting
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// This test is mainly to ensure no race conditions in the engine itself
	// Multiple rapid calls should be rate-limited properly
	results := make([]error, 3)
	
	for i := 0; i < 3; i++ {
		go func(idx int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			_, err := engine.Synthesize(ctx, "test", 1.0)
			results[idx] = err
		}(i)
	}

	// Wait for all goroutines to complete
	time.Sleep(15 * time.Second)
	
	// At least one should succeed, others may fail due to rate limiting or timeout
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
		} else {
			t.Logf("Request %d failed (expected due to rate limiting): %v", i, err)
		}
	}

	// We expect some failures due to aggressive rate limiting
	if successCount == 0 {
		t.Error("At least one request should succeed")
	}
}

// TestGTTSEngine_ContextCancellation tests context cancellation
func TestGTTSEngine_ContextCancellation(t *testing.T) {
	engine, err := NewGTTSEngine(GTTSConfig{})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = engine.Synthesize(ctx, "test", 1.0)
	if err == nil {
		t.Error("Expected error due to cancelled context")
	}

	if !strings.Contains(err.Error(), "rate limit wait cancelled") &&
	   !strings.Contains(err.Error(), "timeout") &&
	   !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected cancellation error, got: %v", err)
	}
}

// TestGTTSEngine_CacheOperations tests cache-specific operations
func TestGTTSEngine_CacheOperations(t *testing.T) {
	// Create temp cache directory
	cacheDir, err := os.MkdirTemp("", "gtts-cache-ops-*")
	if err != nil {
		t.Fatalf("Failed to create temp cache dir: %v", err)
	}
	defer os.RemoveAll(cacheDir)

	engine, err := NewGTTSEngine(GTTSConfig{
		CacheConfig: &cache.CacheConfig{
			MemoryCapacity: 1024 * 1024, // 1MB
			DiskCapacity:   10 * 1024 * 1024, // 10MB
			DiskPath:       cacheDir,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Test cache stats (should work even without synthesis)
	stats := engine.GetCacheStats()
	if stats == nil {
		t.Error("Cache stats should not be nil when cache is enabled")
	}

	// Test cache clearing
	err = engine.ClearCache()
	if err != nil {
		t.Errorf("Cache clear failed: %v", err)
	}
}

// BenchmarkGTTSEngine_Synthesize benchmarks synthesis performance
func BenchmarkGTTSEngine_Synthesize(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	// Check if tools are available
	if _, err := exec.LookPath("gtts-cli"); err != nil {
		b.Skip("Skipping benchmark: gtts-cli not available")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		b.Skip("Skipping benchmark: ffmpeg not available")
	}

	engine, err := NewGTTSEngine(GTTSConfig{
		Language:          "en",
		RequestsPerMinute: 10,
	})
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	b.ResetTimer()
	
	// Note: This benchmark may be rate-limited, so results may vary
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, err := engine.Synthesize(ctx, "Benchmark test.", 1.0)
		cancel()
		
		if err != nil {
			b.Logf("Synthesis %d failed (may be rate limited): %v", i, err)
		}
		
		// Add delay to respect rate limits
		time.Sleep(time.Minute / 10) // 6 seconds between requests
	}
}