// Package piper provides the Piper TTS engine integration.
package piper

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// Engine implements the TTS Engine interface for Piper.
type Engine struct {
	// Configuration
	config       Config
	engineConfig tts.EngineConfig

	// Process management
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	stderrBuffer  []string
	processLock   sync.Mutex
	
	// State
	running    int32 // atomic
	healthy    int32 // atomic
	generation int32 // atomic, increments on restart

	// Communication
	outputChan chan []byte
	errorChan  chan error
	
	// Health monitoring
	healthTicker   *time.Ticker
	lastHealthTime time.Time
	healthLock     sync.RWMutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	
	// Error tracking
	lastError     error
	errorCount    int
	errorLock     sync.RWMutex
	restartCount  int
	
	// Available voices cache
	voices     []tts.Voice
	voicesLock sync.RWMutex
}

// Config holds Piper engine configuration.
type Config struct {
	// BinaryPath is the path to the Piper executable
	BinaryPath string
	
	// ModelPath is the path to the voice model file (.onnx)
	ModelPath string
	
	// ConfigPath is the path to the model config file (.json)
	ConfigPath string
	
	// WorkDir is the working directory for Piper
	WorkDir string
	
	// OutputRaw outputs raw audio without WAV header
	OutputRaw bool
	
	// SampleRate is the output sample rate (default: 22050)
	SampleRate int
	
	// StartupTimeout is how long to wait for Piper to start
	StartupTimeout time.Duration
	
	// RequestTimeout is how long to wait for audio generation
	RequestTimeout time.Duration
	
	// HealthCheckInterval is how often to check process health
	HealthCheckInterval time.Duration
	
	// MaxRestarts is the maximum number of restart attempts
	MaxRestarts int
	
	// RestartDelay is the delay between restart attempts
	RestartDelay time.Duration
}

// defaultConfigV1 returns the default Piper configuration for V1.
func defaultConfigV1() Config {
	return Config{
		BinaryPath:          "piper",
		ModelPath:           "",
		ConfigPath:          "",
		WorkDir:             "",
		OutputRaw:           true, // Raw PCM for direct playback
		SampleRate:          22050,
		StartupTimeout:      10 * time.Second,
		RequestTimeout:      30 * time.Second,
		HealthCheckInterval: 5 * time.Second,
		MaxRestarts:         3,
		RestartDelay:        2 * time.Second,
	}
}

// newEngineOriginal creates a new Piper TTS engine (original V1 implementation).
func newEngineOriginal(config Config) (*Engine, error) {
	// Validate configuration
	if config.BinaryPath == "" {
		return nil, errors.New("binary path is required")
	}
	
	// Check if Piper binary exists
	if _, err := exec.LookPath(config.BinaryPath); err != nil {
		// Try to find it in common locations
		possiblePaths := []string{
			"/usr/local/bin/piper",
			"/usr/bin/piper",
			"./piper",
			"./bin/piper",
		}
		
		found := false
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				config.BinaryPath = path
				found = true
				break
			}
		}
		
		if !found {
			return nil, fmt.Errorf("piper binary not found in PATH or common locations")
		}
	}
	
	// Validate model path if provided
	if config.ModelPath != "" {
		if _, err := os.Stat(config.ModelPath); err != nil {
			return nil, fmt.Errorf("model file not found: %w", err)
		}
	}
	
	// Set defaults
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
		config.RestartDelay = 2 * time.Second
	}
	if config.SampleRate == 0 {
		config.SampleRate = 22050
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	engine := &Engine{
		config:       config,
		outputChan:   make(chan []byte, 1),
		errorChan:    make(chan error, 10),
		stderrBuffer: make([]string, 0, 100),
		ctx:          ctx,
		cancel:       cancel,
		voices:       make([]tts.Voice, 0),
	}
	
	// Load available voices from model
	engine.loadVoices()
	
	return engine, nil
}

// Initialize prepares the engine for use.
func (e *Engine) Initialize(config tts.EngineConfig) error {
	e.processLock.Lock()
	defer e.processLock.Unlock()
	
	e.engineConfig = config
	
	// Start the Piper process
	if err := e.startProcess(); err != nil {
		return fmt.Errorf("failed to start Piper process: %w", err)
	}
	
	// Start monitoring goroutines
	e.wg.Add(2)
	go e.healthMonitor()
	go e.stderrMonitor()
	
	return nil
}

// startProcess starts the Piper process.
func (e *Engine) startProcess() error {
	// Build command arguments
	args := []string{}
	
	if e.config.ModelPath != "" {
		args = append(args, "--model", e.config.ModelPath)
	}
	
	if e.config.ConfigPath != "" {
		args = append(args, "--config", e.config.ConfigPath)
	}
	
	if e.config.OutputRaw {
		args = append(args, "--output-raw")
		// Note: --output-raw already sends to stdout, no need for --output-file
	}
	
	// Create command
	e.cmd = exec.Command(e.config.BinaryPath, args...)
	
	// Set working directory
	if e.config.WorkDir != "" {
		e.cmd.Dir = e.config.WorkDir
	}
	
	// Set up pipes
	stdin, err := e.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	e.stdin = stdin
	
	stdout, err := e.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	e.stdout = stdout
	
	stderr, err := e.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	e.stderr = stderr
	
	// Start the process
	if err := e.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}
	
	// Mark as running
	atomic.StoreInt32(&e.running, 1)
	atomic.StoreInt32(&e.healthy, 1)
	atomic.AddInt32(&e.generation, 1)
	e.lastHealthTime = time.Now()
	
	// Start output handler
	e.wg.Add(1)
	go e.outputHandler()
	
	return nil
}

// stopProcess stops the Piper process.
func (e *Engine) stopProcess() error {
	if atomic.LoadInt32(&e.running) == 0 {
		return nil
	}
	
	// Mark as not running
	atomic.StoreInt32(&e.running, 0)
	atomic.StoreInt32(&e.healthy, 0)
	
	// Close stdin to signal shutdown
	if e.stdin != nil {
		e.stdin.Close()
	}
	
	// Give process time to exit gracefully
	done := make(chan error, 1)
	go func() {
		if e.cmd != nil && e.cmd.Process != nil {
			done <- e.cmd.Wait()
		} else {
			done <- nil
		}
	}()
	
	// Wait for process to exit or force kill after timeout
	select {
	case <-time.After(5 * time.Second):
		if e.cmd != nil && e.cmd.Process != nil {
			e.cmd.Process.Kill()
		}
	case <-done:
		// Process exited gracefully
	}
	
	// Close pipes
	if e.stdout != nil {
		e.stdout.Close()
	}
	if e.stderr != nil {
		e.stderr.Close()
	}
	
	return nil
}

// restartProcess attempts to restart the Piper process.
func (e *Engine) restartProcess() error {
	e.processLock.Lock()
	defer e.processLock.Unlock()
	
	// Check restart limit
	if e.restartCount >= e.config.MaxRestarts {
		return fmt.Errorf("max restarts (%d) exceeded", e.config.MaxRestarts)
	}
	
	e.restartCount++
	
	// Stop existing process
	e.stopProcess()
	
	// Wait before restarting
	time.Sleep(e.config.RestartDelay)
	
	// Start new process
	if err := e.startProcess(); err != nil {
		return fmt.Errorf("restart failed: %w", err)
	}
	
	// Reset error count on successful restart
	e.errorLock.Lock()
	e.errorCount = 0
	e.errorLock.Unlock()
	
	return nil
}

// outputHandler handles audio output from Piper.
func (e *Engine) outputHandler() {
	defer e.wg.Done()
	
	generation := atomic.LoadInt32(&e.generation)
	reader := bufio.NewReader(e.stdout)
	
	for {
		// Check if this handler is for the current process generation
		if atomic.LoadInt32(&e.generation) != generation {
			return
		}
		
		// Read audio data
		// Piper outputs raw PCM or WAV data to stdout
		buffer := make([]byte, 4096)
		n, err := reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				e.recordError(fmt.Errorf("stdout read error: %w", err))
			}
			return
		}
		
		if n > 0 {
			// Send audio data
			select {
			case e.outputChan <- buffer[:n]:
			case <-e.ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Timeout - output channel might be full
			}
		}
	}
}

// stderrMonitor monitors stderr for errors and warnings.
func (e *Engine) stderrMonitor() {
	defer e.wg.Done()
	
	scanner := bufio.NewScanner(e.stderr)
	
	for scanner.Scan() {
		select {
		case <-e.ctx.Done():
			return
		default:
			line := scanner.Text()
			if line != "" {
				// Store stderr output for debugging
				e.processLock.Lock()
				e.stderrBuffer = append(e.stderrBuffer, line)
				if len(e.stderrBuffer) > 100 {
					e.stderrBuffer = e.stderrBuffer[1:]
				}
				e.processLock.Unlock()
				
				// Check for errors
				if strings.Contains(strings.ToLower(line), "error") {
					e.recordError(fmt.Errorf("piper error: %s", line))
				}
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		e.recordError(fmt.Errorf("stderr scanner error: %w", err))
	}
}

// healthMonitor monitors the health of the Piper process.
func (e *Engine) healthMonitor() {
	defer e.wg.Done()
	
	e.healthTicker = time.NewTicker(e.config.HealthCheckInterval)
	defer e.healthTicker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.healthTicker.C:
			e.checkHealth()
		}
	}
}

// checkHealth checks if the Piper process is healthy.
func (e *Engine) checkHealth() {
	if atomic.LoadInt32(&e.running) == 0 {
		return
	}
	
	// Check if process is still alive
	if e.cmd != nil && e.cmd.Process != nil {
		// Try to send signal 0 (doesn't actually send a signal, just checks)
		if err := e.cmd.Process.Signal(nil); err != nil {
			// Process is dead
			atomic.StoreInt32(&e.healthy, 0)
			e.recordError(errors.New("process died unexpectedly"))
			
			// Attempt restart
			go func() {
				if err := e.restartProcess(); err != nil {
					e.recordError(fmt.Errorf("failed to restart: %w", err))
				}
			}()
			return
		}
	}
	
	// Update last health time
	e.healthLock.Lock()
	e.lastHealthTime = time.Now()
	e.healthLock.Unlock()
	
	atomic.StoreInt32(&e.healthy, 1)
}

// GenerateAudio converts text to audio data synchronously.
func (e *Engine) GenerateAudio(text string) (*tts.Audio, error) {
	if !e.IsAvailable() {
		return nil, errors.New("engine not available")
	}
	
	// Clear output channel
	select {
	case <-e.outputChan:
	default:
	}
	
	// Send text to Piper via stdin
	if _, err := fmt.Fprintln(e.stdin, text); err != nil {
		return nil, fmt.Errorf("failed to send text: %w", err)
	}
	
	// Collect audio output
	var audioData []byte
	timeout := time.After(e.config.RequestTimeout)
	
	for {
		select {
		case data := <-e.outputChan:
			audioData = append(audioData, data...)
			
			// Check if we have enough data (heuristic: wait for a pause in output)
			select {
			case moreData := <-e.outputChan:
				audioData = append(audioData, moreData...)
				continue
			case <-time.After(100 * time.Millisecond):
				// No more data coming, assume generation is complete
				goto done
			}
			
		case err := <-e.errorChan:
			return nil, fmt.Errorf("generation failed: %w", err)
			
		case <-timeout:
			return nil, errors.New("generation timeout")
		}
	}
	
done:
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
	
	return audio, nil
}

// GenerateAudioStream converts text to audio data asynchronously.
func (e *Engine) GenerateAudioStream(text string) (<-chan tts.AudioChunk, error) {
	// Piper doesn't support true streaming, but we can simulate it
	// by chunking the output as it arrives
	chunkChan := make(chan tts.AudioChunk, 10)
	
	go func() {
		defer close(chunkChan)
		
		// Generate audio
		audio, err := e.GenerateAudio(text)
		if err != nil {
			// Send empty chunk to indicate error
			chunkChan <- tts.AudioChunk{
				Data:  nil,
				Final: true,
			}
			return
		}
		
		// Send audio in chunks
		chunkSize := 4096
		position := 0
		
		for position < len(audio.Data) {
			end := position + chunkSize
			if end > len(audio.Data) {
				end = len(audio.Data)
			}
			
			chunk := tts.AudioChunk{
				Data:     audio.Data[position:end],
				Final:    end >= len(audio.Data),
				Position: position,
			}
			
			select {
			case chunkChan <- chunk:
			case <-e.ctx.Done():
				return
			}
			
			position = end
		}
	}()
	
	return chunkChan, nil
}

// IsAvailable checks if the engine is ready for use.
func (e *Engine) IsAvailable() bool {
	return atomic.LoadInt32(&e.running) == 1 && atomic.LoadInt32(&e.healthy) == 1
}

// GetVoices returns the list of available voices.
func (e *Engine) GetVoices() []tts.Voice {
	e.voicesLock.RLock()
	defer e.voicesLock.RUnlock()
	
	voices := make([]tts.Voice, len(e.voices))
	copy(voices, e.voices)
	return voices
}

// SetVoice sets the active voice for audio generation.
func (e *Engine) SetVoice(voice tts.Voice) error {
	// Piper uses models for voices, which are set at startup
	// To change voice, we'd need to restart with a different model
	
	e.voicesLock.RLock()
	defer e.voicesLock.RUnlock()
	
	for _, v := range e.voices {
		if v.ID == voice.ID {
			return nil // Voice already active
		}
	}
	
	return fmt.Errorf("voice %s not available in current model", voice.ID)
}

// GetCapabilities returns the engine's capabilities.
func (e *Engine) GetCapabilities() tts.Capabilities {
	return tts.Capabilities{
		SupportsStreaming: false, // Piper doesn't support true streaming
		SupportedFormats:  []string{"pcm16", "wav"},
		MaxTextLength:     10000, // Reasonable limit
		RequiresNetwork:   false, // Piper runs locally
	}
}

// Shutdown cleanly stops the engine and releases resources.
func (e *Engine) Shutdown() error {
	// Cancel context to stop all goroutines
	e.cancel()
	
	// Stop the process
	e.processLock.Lock()
	err := e.stopProcess()
	e.processLock.Unlock()
	
	// Stop health ticker
	if e.healthTicker != nil {
		e.healthTicker.Stop()
	}
	
	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	
	// Wait with timeout
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		// Timeout waiting for goroutines
	}
	
	// Close channels
	close(e.outputChan)
	close(e.errorChan)
	
	return err
}

// loadVoices loads available voices from the model.
func (e *Engine) loadVoices() {
	// Extract voice info from model path
	if e.config.ModelPath != "" {
		modelName := filepath.Base(e.config.ModelPath)
		modelName = strings.TrimSuffix(modelName, filepath.Ext(modelName))
		
		// Parse model name (e.g., "en_US-lessac-medium")
		parts := strings.Split(modelName, "-")
		
		language := "en-US"
		if len(parts) > 0 && strings.Contains(parts[0], "_") {
			// Convert en_US to en-US
			language = strings.Replace(parts[0], "_", "-", 1)
		}
		
		voice := tts.Voice{
			ID:       modelName,
			Name:     modelName,
			Language: language,
			Gender:   "neutral",
		}
		
		e.voicesLock.Lock()
		e.voices = []tts.Voice{voice}
		e.voicesLock.Unlock()
	}
}

// recordError records an error for tracking.
func (e *Engine) recordError(err error) {
	e.errorLock.Lock()
	defer e.errorLock.Unlock()
	
	e.lastError = err
	e.errorCount++
	
	// Send to error channel if space available
	select {
	case e.errorChan <- err:
	default:
	}
	
	// If too many errors, mark as unhealthy
	if e.errorCount > 10 {
		atomic.StoreInt32(&e.healthy, 0)
	}
}

// GetLastError returns the last recorded error.
func (e *Engine) GetLastError() error {
	e.errorLock.RLock()
	defer e.errorLock.RUnlock()
	return e.lastError
}

// GetStderrOutput returns recent stderr output for debugging.
func (e *Engine) GetStderrOutput() []string {
	e.processLock.Lock()
	defer e.processLock.Unlock()
	
	output := make([]string, len(e.stderrBuffer))
	copy(output, e.stderrBuffer)
	return output
}

// GetStats returns engine statistics.
func (e *Engine) GetStats() Stats {
	e.errorLock.RLock()
	errorCount := e.errorCount
	lastError := e.lastError
	e.errorLock.RUnlock()
	
	e.healthLock.RLock()
	lastHealth := e.lastHealthTime
	e.healthLock.RUnlock()
	
	return Stats{
		Running:      atomic.LoadInt32(&e.running) == 1,
		Healthy:      atomic.LoadInt32(&e.healthy) == 1,
		Generation:   atomic.LoadInt32(&e.generation),
		RestartCount: e.restartCount,
		ErrorCount:   errorCount,
		LastError:    lastError,
		LastHealth:   lastHealth,
	}
}

// Stats holds engine statistics.
type Stats struct {
	Running      bool
	Healthy      bool
	Generation   int32
	RestartCount int
	ErrorCount   int
	LastError    error
	LastHealth   time.Time
}

// EstimateDuration estimates the duration of generated audio for given text.
func (e *Engine) EstimateDuration(text string) time.Duration {
	// Rough estimation: ~150 words per minute
	// Average word length: 5 characters
	words := len(text) / 5
	if words < 1 {
		words = 1
	}
	
	secondsPerWord := 60.0 / 150.0
	return time.Duration(float64(words) * secondsPerWord * float64(time.Second))
}

// decodeBase64Audio decodes base64 encoded audio data.
func decodeBase64Audio(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// encodeBase64Audio encodes audio data to base64.
func encodeBase64Audio(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}