// Package piper provides the Piper TTS engine integration.
// This is version 2 with improved process management and stability fixes.
package piper

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// EngineV2 implements the TTS Engine interface for Piper with improved stability.
type EngineV2 struct {
	// Configuration
	config       Config
	engineConfig tts.EngineConfig

	// Process management
	processPool   []*processInstance
	poolSize      int
	currentIndex  int32
	poolLock      sync.RWMutex
	
	// Fresh process per request mode
	freshMode     bool
	
	// State
	initialized   int32 // atomic
	shutdownFlag  int32 // atomic
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	
	// Statistics
	stats struct {
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		restarts        int64
		processCreated  int64
	}
	
	// Available voices cache
	voices     []tts.Voice
	voicesLock sync.RWMutex
}

// processInstance represents a single Piper process
type processInstance struct {
	id            int
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	
	running       int32 // atomic
	busy          int32 // atomic
	healthy       int32 // atomic
	lastUsed      time.Time
	generation    int32 // atomic, increments on restart
	requestCount  int64 // atomic
	
	outputChan    chan []byte
	errorChan     chan error
	
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	
	stderrBuffer  []string
	bufferLock    sync.Mutex
}

// NewEngineV2 creates a new Piper engine with improved stability.
func NewEngineV2(config Config) (*EngineV2, error) {
	// Validate configuration
	if config.BinaryPath == "" {
		// Try to find Piper in common locations
		config.BinaryPath = findPiperBinary()
		if config.BinaryPath == "" {
			return nil, errors.New("piper binary not found")
		}
	}
	
	// Check if binary exists
	if _, err := os.Stat(config.BinaryPath); err != nil {
		return nil, fmt.Errorf("piper binary not accessible: %w", err)
	}
	
	// Set defaults
	if config.SampleRate == 0 {
		config.SampleRate = 22050
	}
	if config.StartupTimeout == 0 {
		config.StartupTimeout = 10 * time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 5 * time.Second
	}
	if config.MaxRestarts == 0 {
		config.MaxRestarts = 3
	}
	if config.RestartDelay == 0 {
		config.RestartDelay = time.Second
	}
	
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Determine pool size (default to 2 for redundancy)
	poolSize := 2
	if config.MaxRestarts > 2 {
		poolSize = 3 // More processes for higher availability
	}
	
	// Check if we should use fresh process mode
	freshMode := false
	if envMode := os.Getenv("PIPER_FRESH_MODE"); envMode == "true" || envMode == "1" {
		freshMode = true
		poolSize = 1 // Only need 1 in fresh mode
		log.Printf("[INFO Piper] Fresh process mode enabled")
	}
	
	engine := &EngineV2{
		config:      config,
		poolSize:    poolSize,
		freshMode:   freshMode,
		ctx:         ctx,
		cancel:      cancel,
		processPool: make([]*processInstance, 0, poolSize),
	}
	
	return engine, nil
}

// Initialize prepares the engine for use with the given configuration.
func (e *EngineV2) Initialize(config tts.EngineConfig) error {
	if atomic.LoadInt32(&e.initialized) == 1 {
		return nil // Already initialized
	}
	
	e.engineConfig = config
	
	// In fresh mode, we don't pre-create processes
	if e.freshMode {
		atomic.StoreInt32(&e.initialized, 1)
		log.Printf("[INFO Piper] Initialized in fresh process mode")
		return nil
	}
	
	// Create initial process pool
	for i := 0; i < e.poolSize; i++ {
		proc, err := e.createProcess(i)
		if err != nil {
			// Clean up any created processes
			e.Shutdown()
			return fmt.Errorf("failed to create process %d: %w", i, err)
		}
		
		e.poolLock.Lock()
		e.processPool = append(e.processPool, proc)
		e.poolLock.Unlock()
		
		// Start the process
		if err := e.startProcess(proc); err != nil {
			e.Shutdown()
			return fmt.Errorf("failed to start process %d: %w", i, err)
		}
	}
	
	atomic.StoreInt32(&e.initialized, 1)
	log.Printf("[INFO Piper] Initialized with %d processes", e.poolSize)
	
	// Start health monitor for the pool
	go e.poolHealthMonitor()
	
	return nil
}

// createProcess creates a new process instance.
func (e *EngineV2) createProcess(id int) (*processInstance, error) {
	ctx, cancel := context.WithCancel(e.ctx)
	
	proc := &processInstance{
		id:         id,
		ctx:        ctx,
		cancel:     cancel,
		outputChan: make(chan []byte, 100),
		errorChan:  make(chan error, 10),
		lastUsed:   time.Now(),
	}
	
	atomic.AddInt64(&e.stats.processCreated, 1)
	return proc, nil
}

// startProcess starts a Piper process instance.
func (e *EngineV2) startProcess(proc *processInstance) error {
	// Build command arguments
	args := []string{}
	
	if e.config.ModelPath != "" {
		args = append(args, "--model", e.config.ModelPath)
	}
	
	// Only add config if it exists (it's optional for Piper)
	if e.config.ConfigPath != "" {
		// Check if config file actually exists
		if _, err := os.Stat(e.config.ConfigPath); err == nil {
			args = append(args, "--config", e.config.ConfigPath)
		} else {
			log.Printf("[DEBUG Piper] Config file not found, skipping: %s", e.config.ConfigPath)
		}
	}
	
	if e.config.OutputRaw {
		args = append(args, "--output-raw")
		// Note: --output-raw already sends to stdout, no need for --output-file
	}
	
	// Create command with context for better cleanup
	proc.cmd = exec.CommandContext(proc.ctx, e.config.BinaryPath, args...)
	
	// Set working directory
	if e.config.WorkDir != "" {
		proc.cmd.Dir = e.config.WorkDir
	}
	
	// Set up pipes
	stdin, err := proc.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	proc.stdin = stdin
	
	stdout, err := proc.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	proc.stdout = stdout
	
	stderr, err := proc.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	proc.stderr = stderr
	
	// Start the process
	if err := proc.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}
	
	log.Printf("[DEBUG Piper] Process %d started (PID: %d)", proc.id, proc.cmd.Process.Pid)
	
	// Mark as running and healthy
	atomic.StoreInt32(&proc.running, 1)
	atomic.StoreInt32(&proc.healthy, 1)
	atomic.AddInt32(&proc.generation, 1)
	proc.lastUsed = time.Now()
	
	// Start output handlers
	proc.wg.Add(2)
	go e.processOutputHandler(proc)
	go e.processStderrHandler(proc)
	
	// In fresh mode, don't wait or health check - we'll know if it fails when we use it
	if !e.freshMode {
		// Wait a moment to ensure process is stable
		time.Sleep(500 * time.Millisecond)
		
		// Check if still running
		if !e.isProcessHealthy(proc) {
			return errors.New("process died immediately after starting")
		}
	} else {
		// For fresh mode, just mark as healthy - we'll find out on first use
		atomic.StoreInt32(&proc.healthy, 1)
		log.Printf("[DEBUG Piper] Fresh mode: skipping health check for process %d", proc.id)
	}
	
	return nil
}

// processOutputHandler handles stdout from the process.
func (e *EngineV2) processOutputHandler(proc *processInstance) {
	defer proc.wg.Done()
	defer close(proc.outputChan)
	
	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := proc.stdout.Read(buffer)
		if n > 0 {
			// Send a copy of the data
			data := make([]byte, n)
			copy(data, buffer[:n])
			
			select {
			case proc.outputChan <- data:
			case <-proc.ctx.Done():
				return
			default:
				// Channel full, drop oldest data
				select {
				case <-proc.outputChan:
					proc.outputChan <- data
				default:
				}
			}
		}
		
		if err != nil {
			if err != io.EOF {
				select {
				case proc.errorChan <- fmt.Errorf("stdout read error: %w", err):
				default:
				}
			}
			return
		}
	}
}

// processStderrHandler handles stderr from the process.
func (e *EngineV2) processStderrHandler(proc *processInstance) {
	defer proc.wg.Done()
	
	scanner := bufio.NewScanner(proc.stderr)
	for scanner.Scan() {
		select {
		case <-proc.ctx.Done():
			return
		default:
			line := scanner.Text()
			if line != "" {
				// Store stderr output for debugging
				proc.bufferLock.Lock()
				proc.stderrBuffer = append(proc.stderrBuffer, line)
				if len(proc.stderrBuffer) > 50 {
					proc.stderrBuffer = proc.stderrBuffer[1:]
				}
				proc.bufferLock.Unlock()
				
				// Check for errors
				if strings.Contains(strings.ToLower(line), "error") || 
				   strings.Contains(strings.ToLower(line), "failed") {
					log.Printf("[WARNING Piper] Process %d stderr: %s", proc.id, line)
					select {
					case proc.errorChan <- fmt.Errorf("piper error: %s", line):
					default:
					}
				}
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("[ERROR Piper] Process %d stderr scanner: %v", proc.id, err)
	}
}

// stopProcess stops a process instance.
func (e *EngineV2) stopProcess(proc *processInstance) error {
	if atomic.LoadInt32(&proc.running) == 0 {
		return nil
	}
	
	log.Printf("[DEBUG Piper] Stopping process %d", proc.id)
	
	// Mark as not running
	atomic.StoreInt32(&proc.running, 0)
	atomic.StoreInt32(&proc.healthy, 0)
	
	// Cancel context to stop goroutines
	proc.cancel()
	
	// Close stdin to signal shutdown
	if proc.stdin != nil {
		proc.stdin.Close()
	}
	
	// Give process time to exit gracefully
	done := make(chan error, 1)
	go func() {
		if proc.cmd != nil && proc.cmd.Process != nil {
			done <- proc.cmd.Wait()
		} else {
			done <- nil
		}
	}()
	
	// Wait for process to exit or force kill after timeout
	select {
	case <-time.After(2 * time.Second):
		if proc.cmd != nil && proc.cmd.Process != nil {
			log.Printf("[WARNING Piper] Force killing process %d", proc.id)
			proc.cmd.Process.Kill()
		}
	case <-done:
		// Process exited gracefully
	}
	
	// Wait for goroutines
	proc.wg.Wait()
	
	// Close pipes
	if proc.stdout != nil {
		proc.stdout.Close()
	}
	if proc.stderr != nil {
		proc.stderr.Close()
	}
	
	return nil
}

// isProcessHealthy checks if a process is healthy.
func (e *EngineV2) isProcessHealthy(proc *processInstance) bool {
	if atomic.LoadInt32(&proc.running) == 0 {
		return false
	}
	
	// Check if process is still alive
	if proc.cmd != nil && proc.cmd.Process != nil {
		// Try to send signal 0 (doesn't actually send a signal, just checks)
		if err := proc.cmd.Process.Signal(nil); err != nil {
			// Process is dead
			atomic.StoreInt32(&proc.healthy, 0)
			atomic.StoreInt32(&proc.running, 0)
			return false
		}
		return true
	}
	
	return false
}

// getAvailableProcess returns an available process from the pool.
func (e *EngineV2) getAvailableProcess() (*processInstance, error) {
	// In fresh mode, always create a new process
	if e.freshMode {
		proc, err := e.createProcess(0)
		if err != nil {
			return nil, err
		}
		if err := e.startProcess(proc); err != nil {
			return nil, err
		}
		return proc, nil
	}
	
	// Try to find a healthy, non-busy process
	e.poolLock.RLock()
	defer e.poolLock.RUnlock()
	
	// First pass: find healthy, non-busy process
	for _, proc := range e.processPool {
		if atomic.LoadInt32(&proc.healthy) == 1 && 
		   atomic.CompareAndSwapInt32(&proc.busy, 0, 1) {
			proc.lastUsed = time.Now()
			return proc, nil
		}
	}
	
	// Second pass: restart unhealthy processes
	for _, proc := range e.processPool {
		if atomic.LoadInt32(&proc.healthy) == 0 {
			log.Printf("[INFO Piper] Restarting unhealthy process %d", proc.id)
			e.stopProcess(proc)
			if err := e.startProcess(proc); err != nil {
				log.Printf("[ERROR Piper] Failed to restart process %d: %v", proc.id, err)
				continue
			}
			if atomic.CompareAndSwapInt32(&proc.busy, 0, 1) {
				proc.lastUsed = time.Now()
				atomic.AddInt64(&e.stats.restarts, 1)
				return proc, nil
			}
		}
	}
	
	return nil, errors.New("no available process in pool")
}

// releaseProcess releases a process back to the pool.
func (e *EngineV2) releaseProcess(proc *processInstance) {
	// In fresh mode, stop the process immediately
	if e.freshMode {
		e.stopProcess(proc)
		return
	}
	
	// Mark as not busy
	atomic.StoreInt32(&proc.busy, 0)
	
	// Check if process should be recycled (after N requests)
	if atomic.LoadInt64(&proc.requestCount) > 100 {
		log.Printf("[INFO Piper] Recycling process %d after %d requests", 
			proc.id, proc.requestCount)
		e.stopProcess(proc)
		e.startProcess(proc) // Restart it for next use
	}
}

// poolHealthMonitor monitors the health of the process pool.
func (e *EngineV2) poolHealthMonitor() {
	ticker := time.NewTicker(e.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkPoolHealth()
		}
	}
}

// checkPoolHealth checks and maintains pool health.
func (e *EngineV2) checkPoolHealth() {
	if e.freshMode {
		return // No pool in fresh mode
	}
	
	e.poolLock.RLock()
	defer e.poolLock.RUnlock()
	
	healthyCount := 0
	for _, proc := range e.processPool {
		if e.isProcessHealthy(proc) {
			healthyCount++
		} else if atomic.LoadInt32(&proc.busy) == 0 {
			// Restart unhealthy, non-busy processes
			go func(p *processInstance) {
				log.Printf("[INFO Piper] Health check restarting process %d", p.id)
				e.stopProcess(p)
				if err := e.startProcess(p); err != nil {
					log.Printf("[ERROR Piper] Failed to restart process %d: %v", p.id, err)
				}
			}(proc)
		}
	}
	
	if healthyCount == 0 {
		log.Printf("[WARNING Piper] No healthy processes in pool!")
	}
}

// GenerateAudio converts text to audio data synchronously.
func (e *EngineV2) GenerateAudio(text string) (*tts.Audio, error) {
	if atomic.LoadInt32(&e.shutdownFlag) == 1 {
		return nil, errors.New("engine is shut down")
	}
	
	if atomic.LoadInt32(&e.initialized) == 0 {
		return nil, errors.New("engine not initialized")
	}
	
	atomic.AddInt64(&e.stats.totalRequests, 1)
	
	// Get an available process
	proc, err := e.getAvailableProcess()
	if err != nil {
		atomic.AddInt64(&e.stats.failedRequests, 1)
		return nil, fmt.Errorf("no available process: %w", err)
	}
	defer e.releaseProcess(proc)
	
	// Increment request count
	atomic.AddInt64(&proc.requestCount, 1)
	
	// Clear output channel
	for {
		select {
		case <-proc.outputChan:
		default:
			goto send
		}
	}
	
send:
	// In fresh mode, give Piper a moment to initialize
	if e.freshMode {
		time.Sleep(200 * time.Millisecond) // Increased delay for stability
	}
	
	// Send text to Piper via stdin
	if proc.stdin != nil {
		_, err := io.WriteString(proc.stdin, text + "\n")
		if err != nil {
			atomic.AddInt64(&e.stats.failedRequests, 1)
			atomic.StoreInt32(&proc.healthy, 0)
			return nil, fmt.Errorf("failed to send text: %w", err)
		}
		
		// Close stdin to signal EOF - critical for Piper to start processing
		if err := proc.stdin.Close(); err != nil && !strings.Contains(err.Error(), "file already closed") {
			log.Printf("[WARNING Piper] Failed to close stdin: %v", err)
		}
		proc.stdin = nil // Prevent double close
	} else {
		return nil, errors.New("stdin is nil")
	}
	
	// Collect audio output with timeout
	var audioData []byte
	timeout := time.After(e.config.RequestTimeout)
	outputTimeout := time.NewTimer(500 * time.Millisecond)
	
	for {
		select {
		case data := <-proc.outputChan:
			audioData = append(audioData, data...)
			// Reset output timeout
			outputTimeout.Reset(500 * time.Millisecond)
			
		case <-outputTimeout.C:
			// No more data coming, assume generation is complete
			if len(audioData) > 0 {
				goto done
			}
			// Continue waiting if no data yet
			outputTimeout.Reset(500 * time.Millisecond)
			
		case err := <-proc.errorChan:
			atomic.AddInt64(&e.stats.failedRequests, 1)
			// Mark process as unhealthy
			atomic.StoreInt32(&proc.healthy, 0)
			return nil, fmt.Errorf("generation failed: %w", err)
			
		case <-timeout:
			atomic.AddInt64(&e.stats.failedRequests, 1)
			// Mark process as unhealthy
			atomic.StoreInt32(&proc.healthy, 0)
			return nil, errors.New("generation timeout")
		}
	}
	
done:
	outputTimeout.Stop()
	
	// Validate audio data
	if len(audioData) == 0 {
		atomic.AddInt64(&e.stats.failedRequests, 1)
		return nil, errors.New("no audio data generated")
	}
	
	// Estimate duration based on sample rate and data size
	// PCM16: 2 bytes per sample
	samples := len(audioData) / 2
	duration := time.Duration(float64(samples) / float64(e.config.SampleRate) * float64(time.Second))
	
	// Create audio object
	audio := &tts.Audio{
		Data:       audioData,
		Format:     tts.FormatPCM16,
		SampleRate: e.config.SampleRate,
		Channels:   1, // Piper outputs mono
		Duration:   duration,
	}
	
	atomic.AddInt64(&e.stats.successRequests, 1)
	return audio, nil
}

// GenerateAudioStream converts text to audio data asynchronously.
func (e *EngineV2) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	if !e.IsAvailable() {
		return nil, errors.New("engine not available")
	}
	
	// Piper doesn't support true streaming, but we can simulate it
	// by chunking the output as it arrives
	chunkChan := make(chan tts.AudioChunk, 10)
	
	go func() {
		defer close(chunkChan)
		
		// Generate audio
		audio, err := e.GenerateAudio(text)
		if err != nil {
			// Send error as final chunk
			chunkChan <- tts.AudioChunk{
				Data:  nil,
				Final: true,
			}
			return
		}
		
		// Split into chunks
		chunkSize := 8192 // 8KB chunks
		for i := 0; i < len(audio.Data); i += chunkSize {
			end := i + chunkSize
			if end > len(audio.Data) {
				end = len(audio.Data)
			}
			
			chunk := tts.AudioChunk{
				Data:     audio.Data[i:end],
				Position: i,
				Final:    end >= len(audio.Data),
			}
			
			select {
			case chunkChan <- chunk:
			case <-e.ctx.Done():
				return
			}
		}
	}()
	
	return chunkChan, nil
}

// IsAvailable checks if the engine is ready for use.
func (e *EngineV2) IsAvailable() bool {
	if atomic.LoadInt32(&e.shutdownFlag) == 1 {
		return false
	}
	
	if atomic.LoadInt32(&e.initialized) == 0 {
		return false
	}
	
	// In fresh mode, we're always available (will create process on demand)
	if e.freshMode {
		return true
	}
	
	// Check if at least one process is healthy
	e.poolLock.RLock()
	defer e.poolLock.RUnlock()
	
	for _, proc := range e.processPool {
		if atomic.LoadInt32(&proc.healthy) == 1 {
			return true
		}
	}
	
	return false
}

// GetVoices returns the list of available voices.
func (e *EngineV2) GetVoices() []tts.Voice {
	e.voicesLock.RLock()
	defer e.voicesLock.RUnlock()
	
	if len(e.voices) == 0 {
		// Return default voice based on model
		modelName := "default"
		if e.config.ModelPath != "" {
			modelName = filepath.Base(e.config.ModelPath)
			modelName = strings.TrimSuffix(modelName, filepath.Ext(modelName))
		}
		
		return []tts.Voice{
			{
				ID:       modelName,
				Name:     modelName,
				Language: "en-US",
				Gender:   "neutral",
			},
		}
	}
	
	return e.voices
}

// SetVoice sets the active voice for audio generation.
func (e *EngineV2) SetVoice(voice tts.Voice) error {
	// Piper uses models for voices, so this would require loading a different model
	// For now, we just validate that the voice ID matches our model
	voices := e.GetVoices()
	for _, v := range voices {
		if v.ID == voice.ID {
			return nil
		}
	}
	return fmt.Errorf("voice %s not available", voice.ID)
}

// GetCapabilities returns the engine's capabilities.
func (e *EngineV2) GetCapabilities() tts.Capabilities {
	return tts.Capabilities{
		SupportsStreaming: false, // Piper doesn't support true streaming
		SupportedFormats:  []string{"pcm16", "wav"},
		MaxTextLength:     10000, // Reasonable limit
		RequiresNetwork:   false,
	}
}

// Shutdown cleanly stops the engine and releases resources.
func (e *EngineV2) Shutdown() error {
	if atomic.CompareAndSwapInt32(&e.shutdownFlag, 0, 1) {
		log.Printf("[INFO Piper] Shutting down engine")
		
		// Cancel context to stop all goroutines
		e.cancel()
		
		// Stop all processes
		e.poolLock.Lock()
		var wg sync.WaitGroup
		for _, proc := range e.processPool {
			wg.Add(1)
			go func(p *processInstance) {
				defer wg.Done()
				e.stopProcess(p)
			}(proc)
		}
		e.poolLock.Unlock()
		
		// Wait for all processes to stop
		wg.Wait()
		
		// Log statistics
		log.Printf("[INFO Piper] Engine statistics: requests=%d, success=%d, failed=%d, restarts=%d, processes=%d",
			e.stats.totalRequests,
			e.stats.successRequests,
			e.stats.failedRequests,
			e.stats.restarts,
			e.stats.processCreated)
	}
	
	return nil
}

// GetStatistics returns engine statistics for monitoring.
func (e *EngineV2) GetStatistics() map[string]int64 {
	return map[string]int64{
		"total_requests":   atomic.LoadInt64(&e.stats.totalRequests),
		"success_requests": atomic.LoadInt64(&e.stats.successRequests),
		"failed_requests":  atomic.LoadInt64(&e.stats.failedRequests),
		"restarts":         atomic.LoadInt64(&e.stats.restarts),
		"processes_created": atomic.LoadInt64(&e.stats.processCreated),
	}
}

// findPiperBinary tries to find the Piper binary in common locations.
func findPiperBinary() string {
	// Check common locations
	locations := []string{
		"./piper",
		"piper",
		"/usr/local/bin/piper",
		"/usr/bin/piper",
	}
	
	// Add user home directory locations
	if home, err := os.UserHomeDir(); err == nil {
		locations = append(locations,
			filepath.Join(home, ".local", "bin", "piper"),
			filepath.Join(home, "bin", "piper"),
		)
	}
	
	// Check each location
	for _, loc := range locations {
		if path, err := exec.LookPath(loc); err == nil {
			return path
		}
	}
	
	return ""
}