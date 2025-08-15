package tts

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// Mock components for testing
type mockQueueEngine struct {
	synthesizeFunc func(string, float64) ([]byte, error)
	name           string
	available      bool
}

func (m *mockQueueEngine) Synthesize(text string, speed float64) ([]byte, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(text, speed)
	}
	// Return mock audio data
	audio := make([]byte, 100)
	for i := range audio {
		audio[i] = byte(i % 256)
	}
	return audio, nil
}

func (m *mockQueueEngine) SetSpeed(speed float64) error {
	return nil
}

func (m *mockQueueEngine) Validate() error {
	if !m.available {
		return fmt.Errorf("engine not available")
	}
	return nil
}

func (m *mockQueueEngine) GetName() string {
	if m.name != "" {
		return m.name
	}
	return "mock-engine"
}

func (m *mockQueueEngine) IsAvailable() bool {
	return m.available
}

type mockQueueParser struct {
	parseFunc func(string) ([]Sentence, error)
}

func (m *mockQueueParser) ParseSentences(text string) ([]Sentence, error) {
	if m.parseFunc != nil {
		return m.parseFunc(text)
	}
	// Simple sentence splitting
	sentences := []Sentence{
		{Text: text, Position: 0},
	}
	return sentences, nil
}

func TestNewAudioQueue(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
		LookaheadSize: 3,
		WorkerCount: 2,
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create audio queue: %v", err)
	}
	defer queue.Stop()

	if queue == nil {
		t.Fatal("Expected queue to be created")
	}

	if queue.GetState() != QueueStateIdle {
		t.Errorf("Expected initial state to be Idle, got %v", queue.GetState())
	}

	if queue.GetQueueDepth() != 0 {
		t.Errorf("Expected empty queue, got depth %d", queue.GetQueueDepth())
	}
}

func TestQueueValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *QueueConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing engine",
			config: &QueueConfig{
				Parser: &mockQueueParser{},
			},
			wantErr: true,
		},
		{
			name: "missing parser",
			config: &QueueConfig{
				Engine: &mockQueueEngine{},
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &QueueConfig{
				Engine: &mockQueueEngine{available: true},
				Parser: &mockQueueParser{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queue, err := NewAudioQueue(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAudioQueue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if queue != nil {
				queue.Stop()
			}
		})
	}
}

func TestAddText(t *testing.T) {
	parser := &mockQueueParser{
		parseFunc: func(text string) ([]Sentence, error) {
			return []Sentence{
				{Text: "First sentence.", Position: 0},
				{Text: "Second sentence.", Position: 1},
				{Text: "Third sentence.", Position: 2},
			}, nil
		},
	}

	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: parser,
		LookaheadSize: 2,
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add text
	err = queue.AddText("First sentence. Second sentence. Third sentence.")
	if err != nil {
		t.Fatalf("Failed to add text: %v", err)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	if queue.GetQueueDepth() != 3 {
		t.Errorf("Expected 3 segments, got %d", queue.GetQueueDepth())
	}
}

func TestQueueNavigation(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add multiple segments
	for i := 0; i < 5; i++ {
		err = queue.AddText(fmt.Sprintf("Sentence %d", i))
		if err != nil {
			t.Fatalf("Failed to add text: %v", err)
		}
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	t.Run("Next", func(t *testing.T) {
		segment, err := queue.Next()
		if err != nil {
			t.Fatalf("Next() failed: %v", err)
		}
		if segment == nil {
			t.Error("Expected segment, got nil")
		}
	})

	t.Run("Previous", func(t *testing.T) {
		// Move forward first
		_, _ = queue.Next()
		
		segment, err := queue.Previous()
		if err != nil {
			t.Fatalf("Previous() failed: %v", err)
		}
		if segment == nil {
			t.Error("Expected segment, got nil")
		}
	})

	t.Run("Skip forward", func(t *testing.T) {
		segment, err := queue.Skip(2)
		if err != nil {
			t.Fatalf("Skip(2) failed: %v", err)
		}
		if segment == nil {
			t.Error("Expected segment, got nil")
		}
	})

	t.Run("Skip backward", func(t *testing.T) {
		segment, err := queue.Skip(-1)
		if err != nil {
			t.Fatalf("Skip(-1) failed: %v", err)
		}
		if segment == nil {
			t.Error("Expected segment, got nil")
		}
	})
}

func TestLookaheadBuffer(t *testing.T) {
	synthesisCount := 0
	var mu sync.Mutex

	engine := &mockQueueEngine{
		available: true,
		synthesizeFunc: func(text string, speed float64) ([]byte, error) {
			mu.Lock()
			synthesisCount++
			mu.Unlock()
			
			// Simulate synthesis time
			time.Sleep(10 * time.Millisecond)
			return make([]byte, 100), nil
		},
	}

	config := &QueueConfig{
		Engine:        engine,
		Parser:        &mockQueueParser{},
		LookaheadSize: 2,
		WorkerCount:   2,
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add 5 segments
	for i := 0; i < 5; i++ {
		err = queue.AddText(fmt.Sprintf("Sentence %d", i))
		if err != nil {
			t.Fatalf("Failed to add text: %v", err)
		}
	}

	// Wait for lookahead synthesis
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	count := synthesisCount
	mu.Unlock()

	// Should synthesize at least lookahead size initially
	if count < 2 {
		t.Errorf("Expected at least 2 segments synthesized, got %d", count)
	}

	// Move forward and check if more synthesis happens
	_, _ = queue.Next()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	newCount := synthesisCount
	mu.Unlock()

	if newCount <= count {
		t.Error("Expected additional synthesis after navigation")
	}
}

func TestAudioPreprocessing(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	t.Run("Trim silence", func(t *testing.T) {
		// Create audio with silence at beginning and end
		audio := make([]byte, 200)
		// Add silence at beginning (0-50)
		// Add signal in middle (50-150)
		for i := 50; i < 150; i += 2 {
			binary.LittleEndian.PutUint16(audio[i:i+2], uint16(1000))
		}
		// Silence at end (150-200)

		trimmed := queue.trimSilence(audio)
		
		if len(trimmed) >= len(audio) {
			t.Error("Expected trimmed audio to be shorter")
		}
	})

	t.Run("Normalize audio", func(t *testing.T) {
		// Create quiet audio
		audio := make([]byte, 100)
		for i := 0; i < len(audio); i += 2 {
			binary.LittleEndian.PutUint16(audio[i:i+2], uint16(100)) // Very quiet
		}

		normalized := queue.normalizeAudio(audio)
		
		// Check if amplitude increased
		originalSample := int16(binary.LittleEndian.Uint16(audio[0:2]))
		normalizedSample := int16(binary.LittleEndian.Uint16(normalized[0:2]))
		
		if normalizedSample <= originalSample {
			t.Error("Expected normalization to increase amplitude")
		}
	})
}

func TestQueueMemoryManagement(t *testing.T) {
	config := &QueueConfig{
		Engine:          &mockQueueEngine{available: true},
		Parser:          &mockQueueParser{},
		MaxMemoryMB:     1, // Very small limit
		RetentionPeriod: 1,
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add many segments to trigger memory cleanup
	for i := 0; i < 20; i++ {
		err = queue.AddText(fmt.Sprintf("Sentence %d", i))
		if err != nil {
			t.Fatalf("Failed to add text: %v", err)
		}
	}

	// Wait for synthesis
	time.Sleep(500 * time.Millisecond)

	// Navigate through queue to mark segments as played
	for i := 0; i < 10; i++ {
		_, _ = queue.Next()
	}

	// Trigger cleanup
	queue.cleanupMemory()

	// Check memory usage
	memUsage := queue.GetMemoryUsage()
	maxMemory := int64(config.MaxMemoryMB * 1024 * 1024)
	
	if memUsage > maxMemory {
		t.Errorf("Memory usage %d exceeds limit %d", memUsage, maxMemory)
	}
}

func TestQueueState(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Test state transitions
	stateChanges := make([]QueueState, 0)
	var stateChangesMu sync.Mutex
	queue.SetCallbacks(
		func(state QueueState) {
			stateChangesMu.Lock()
			stateChanges = append(stateChanges, state)
			stateChangesMu.Unlock()
		},
		nil,
		nil,
	)

	// Add text should trigger processing
	_ = queue.AddText("Test")
	time.Sleep(100 * time.Millisecond)

	// Navigate should trigger playing
	_, _ = queue.Next()
	
	// Pause
	queue.Pause()
	if queue.GetState() != QueueStatePaused {
		t.Error("Expected paused state")
	}

	// Resume
	queue.Resume()
	if queue.GetState() != QueueStatePlaying {
		t.Error("Expected playing state")
	}

	// Stop
	queue.Stop()
	
	// Wait for stopped state with retry logic
	if !waitForQueueState(t, queue, QueueStateStopped, 500*time.Millisecond) {
		t.Errorf("Queue did not reach stopped state")
	}

	stateChangesMu.Lock()
	numChanges := len(stateChanges)
	stateChangesMu.Unlock()
	
	if numChanges == 0 {
		t.Error("Expected state change callbacks")
	}
}

func TestCrossfade(t *testing.T) {
	// Create two audio segments
	audio1 := make([]byte, 100)
	audio2 := make([]byte, 100)
	
	for i := 0; i < len(audio1); i += 2 {
		binary.LittleEndian.PutUint16(audio1[i:i+2], uint16(1000))
		binary.LittleEndian.PutUint16(audio2[i:i+2], uint16(2000))
	}

	// Apply crossfade
	result := CrossfadeAudio(audio1, audio2, 10, 22050)
	
	// Result should be shorter than sum due to overlap
	if len(result) > len(audio1)+len(audio2) {
		t.Error("Crossfaded audio should not be longer than sum")
	}

	// Check edge cases
	t.Run("Empty first audio", func(t *testing.T) {
		result := CrossfadeAudio([]byte{}, audio2, 10, 22050)
		if len(result) != len(audio2) {
			t.Error("Should return second audio when first is empty")
		}
	})

	t.Run("Empty second audio", func(t *testing.T) {
		result := CrossfadeAudio(audio1, []byte{}, 10, 22050)
		if len(result) != len(audio1) {
			t.Error("Should return first audio when second is empty")
		}
	})
}

func TestWaitForReady(t *testing.T) {
	// Adjust delay for CI environments
	synthesizeDelay := 200 * time.Millisecond
	if os.Getenv("CI") == "true" {
		synthesizeDelay = 300 * time.Millisecond
	}
	
	slowEngine := &mockQueueEngine{
		available: true,
		synthesizeFunc: func(text string, speed float64) ([]byte, error) {
			time.Sleep(synthesizeDelay)
			return make([]byte, 100), nil
		},
	}

	config := &QueueConfig{
		Engine: slowEngine,
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add text
	_ = queue.AddText("Test")

	// Wait for ready with adequate timeout
	// Need to account for: worker startup + synthesis time + polling interval (100ms) + buffer
	timeout := 1*time.Second
	if os.Getenv("CI") == "true" {
		// CI needs more time for worker startup and processing
		timeout = 2*time.Second
	}
	err = queue.WaitForReady(timeout)
	if err != nil {
		t.Errorf("WaitForReady failed after %v: %v", timeout, err)
	}

	// Test timeout - use a timeout shorter than polling interval
	queue2, _ := NewAudioQueue(config)
	defer queue2.Stop()
	
	// This should timeout since no text is added and polling is 100ms
	err = queue2.WaitForReady(50 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestQueueMetrics(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add and process some segments
	for i := 0; i < 3; i++ {
		_ = queue.AddText(fmt.Sprintf("Sentence %d", i))
	}

	time.Sleep(200 * time.Millisecond)

	metrics := queue.GetMetrics()
	
	if metrics["total_processed"].(int64) == 0 {
		t.Error("Expected some segments to be processed")
	}

	if metrics["queue_depth"].(int) != 3 {
		t.Errorf("Expected queue depth 3, got %d", metrics["queue_depth"])
	}

	if metrics["worker_count"].(int) == 0 {
		t.Error("Expected workers to be running")
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := &QueueConfig{
		Engine:      &mockQueueEngine{available: true},
		Parser:      &mockQueueParser{},
		WorkerCount: 4,
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	var wg sync.WaitGroup
	
	// Concurrent adds
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = queue.AddText(fmt.Sprintf("Text %d", idx))
		}(i)
	}

	// Concurrent navigation
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			_, _ = queue.Next()
		}()
	}

	// Concurrent metrics access
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = queue.GetMetrics()
			_ = queue.GetMemoryUsage()
		}()
	}

	wg.Wait()

	// If we get here without deadlock/panic, concurrent access works
	t.Log("Concurrent access successful")
}

func TestQueueClear(t *testing.T) {
	config := &QueueConfig{
		Engine: &mockQueueEngine{available: true},
		Parser: &mockQueueParser{},
	}

	queue, err := NewAudioQueue(config)
	if err != nil {
		t.Fatalf("Failed to create queue: %v", err)
	}
	defer queue.Stop()

	// Add segments
	for i := 0; i < 5; i++ {
		_ = queue.AddText(fmt.Sprintf("Text %d", i))
	}

	time.Sleep(100 * time.Millisecond)

	// Clear queue
	queue.Clear()

	if queue.GetQueueDepth() != 0 {
		t.Error("Expected empty queue after clear")
	}

	if queue.GetMemoryUsage() != 0 {
		t.Error("Expected zero memory usage after clear")
	}

	if queue.GetState() != QueueStateIdle {
		t.Error("Expected idle state after clear")
	}
}

func BenchmarkQueueThroughput(b *testing.B) {
	config := &QueueConfig{
		Engine:      &mockQueueEngine{available: true},
		Parser:      &mockQueueParser{},
		WorkerCount: 4,
	}

	queue, _ := NewAudioQueue(config)
	defer queue.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.AddText(fmt.Sprintf("Sentence %d", i))
	}
}

func BenchmarkAudioPreprocessing(b *testing.B) {
	config := DefaultQueueConfig()
	config.Engine = &mockQueueEngine{available: true}
	config.Parser = &mockQueueParser{}
	
	queue, _ := NewAudioQueue(config)
	defer queue.Stop()

	audio := make([]byte, 44100) // 1 second at 22050Hz, 16-bit
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = queue.preprocessAudio(audio)
	}
}