//go:build !nocgo
// +build !nocgo

package tts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Format specifies the OTO format for our audio
const Format = oto.FormatSignedInt16LE

// positionTrackingReader wraps a reader and tracks position atomically
type positionTrackingReader struct {
	reader   *bytes.Reader
	position int64 // atomic
	mu       sync.Mutex // protects reader operations
}

func newPositionTrackingReader(data []byte) *positionTrackingReader {
	return &positionTrackingReader{
		reader: bytes.NewReader(data),
	}
}

func (ptr *positionTrackingReader) Read(p []byte) (n int, err error) {
	ptr.mu.Lock()
	defer ptr.mu.Unlock()
	
	n, err = ptr.reader.Read(p)
	if n > 0 {
		atomic.AddInt64(&ptr.position, int64(n))
	}
	return n, err
}

func (ptr *positionTrackingReader) Seek(offset int64, whence int) (int64, error) {
	ptr.mu.Lock()
	defer ptr.mu.Unlock()
	
	newPos, err := ptr.reader.Seek(offset, whence)
	if err == nil {
		atomic.StoreInt64(&ptr.position, newPos)
	}
	return newPos, err
}

func (ptr *positionTrackingReader) GetPosition() int64 {
	return atomic.LoadInt64(&ptr.position)
}

// AudioContext manages the global OTO audio context
type AudioContext struct {
	context *oto.Context
	mu      sync.Mutex
	ready   bool
}

// globalAudioContext is a singleton audio context
var (
	globalAudioContext *AudioContext
	globalAudioPlayer  *TTSAudioPlayer
	audioContextOnce   sync.Once
	playerOnce         sync.Once
)

// GetAudioContext returns the global audio context, initializing it if needed
func GetAudioContext() (*AudioContext, error) {
	var initErr error
	audioContextOnce.Do(func() {
		globalAudioContext = &AudioContext{}
		initErr = globalAudioContext.initialize()
	})
	
	if initErr != nil {
		return nil, initErr
	}
	
	return globalAudioContext, nil
}

// GetGlobalAudioPlayer returns the global audio player, initializing it if needed
func GetGlobalAudioPlayer() *TTSAudioPlayer {
	playerOnce.Do(func() {
		globalAudioPlayer = NewTTSAudioPlayer()
	})
	return globalAudioPlayer
}

// initialize creates the OTO audio context
func (ac *AudioContext) initialize() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.ready {
		return nil
	}

	// Create OTO context with our TTS audio format
	options := &oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: Channels,
		Format:       Format,
	}

	// Platform-specific buffer size adjustments
	switch runtime.GOOS {
	case "darwin":
		// macOS benefits from larger buffers
		options.BufferSize = time.Millisecond * 100
	case "windows":
		// Windows WASAPI works well with moderate buffers
		options.BufferSize = time.Millisecond * 50
	default:
		// Linux/ALSA default
		options.BufferSize = time.Millisecond * 50
	}

	context, readyChan, err := oto.NewContext(options)
	if err != nil {
		return fmt.Errorf("failed to create audio context: %w", err)
	}

	// Wait for context to be ready
	<-readyChan

	ac.context = context
	ac.ready = true
	return nil
}

// Suspend suspends the audio context (mobile platforms)
func (ac *AudioContext) Suspend() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.ready || ac.context == nil {
		return errors.New("audio context not initialized")
	}

	return ac.context.Suspend()
}

// Resume resumes the audio context (mobile platforms)
func (ac *AudioContext) Resume() error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if !ac.ready || ac.context == nil {
		return errors.New("audio context not initialized")
	}

	return ac.context.Resume()
}

// AudioStream manages audio playback with proper memory lifecycle
type AudioStream struct {
	// data holds the PCM audio data in memory
	data []byte
	
	// reader provides streaming access to the audio data with position tracking
	reader *positionTrackingReader
	
	// player is the audio player instance
	player AudioPlayerInterface
	
	// State management
	mu       sync.RWMutex
	state    PlaybackState
	position int64 // Current playback position in bytes
	duration time.Duration
	
	// Memory management
	pinned    bool // Indicates if data is pinned in memory
	refCount  int  // Reference counter for cleanup
	closeOnce sync.Once
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAudioStream creates a new audio stream from PCM data
func NewAudioStream(pcmData []byte) (*AudioStream, error) {
	if len(pcmData) == 0 {
		return nil, errors.New("empty audio data")
	}

	// Validate PCM format
	if len(pcmData)%BytesPerSample != 0 {
		return nil, fmt.Errorf("invalid PCM data length: %d bytes (not aligned to %d-byte samples)", 
			len(pcmData), BytesPerSample)
	}

	// Get or create audio context using new interface
	audioCtx, err := GetGlobalAudioContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get audio context: %w", err)
	}

	if !audioCtx.IsReady() {
		return nil, errors.New("audio context not ready")
	}

	// Calculate duration
	numSamples := len(pcmData) / BytesPerSample
	duration := time.Duration(numSamples) * time.Second / SampleRate

	ctx, cancel := context.WithCancel(context.Background())

	stream := &AudioStream{
		data:     pcmData,
		reader:   newPositionTrackingReader(pcmData),
		state:    PlaybackStopped,
		duration: duration,
		refCount: 1,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Pin data in memory to prevent GC during playback
	stream.pinMemory()

	// Set finalizer for cleanup if Close() is not called
	runtime.SetFinalizer(stream, (*AudioStream).finalize)

	return stream, nil
}

// pinMemory prevents the audio data from being garbage collected
func (as *AudioStream) pinMemory() {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	as.pinned = true
	// Keep a reference to ensure data stays in memory
	// The data slice is already referenced by the struct
}

// unpinMemory allows the audio data to be garbage collected
func (as *AudioStream) unpinMemory() {
	as.mu.Lock()
	defer as.mu.Unlock()
	
	as.pinned = false
}

// Play starts or resumes audio playback
func (as *AudioStream) Play() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	switch as.state {
	case PlaybackPlaying:
		return nil // Already playing
	case PlaybackPaused:
		return as.resume()
	case PlaybackStopped:
		return as.start()
	}
	
	return nil
}

// start begins playback from the beginning or current position
func (as *AudioStream) start() error {
	// Get audio context using new interface
	audioCtx, err := GetGlobalAudioContext()
	if err != nil {
		return err
	}

	// Create new player using interface
	player, err := audioCtx.NewPlayer(as.reader)
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}
	as.player = player
	
	// Start playback
	as.player.Play()
	as.state = PlaybackPlaying
	
	// Start monitoring goroutine
	go as.monitorPlayback()
	
	return nil
}

// resume continues playback from paused position
func (as *AudioStream) resume() error {
	if as.player == nil {
		// Need to recreate player at current position
		as.reader.Seek(as.position, io.SeekStart)
		return as.start()
	}
	
	as.player.Play()
	as.state = PlaybackPlaying
	return nil
}

// Pause pauses audio playback
func (as *AudioStream) Pause() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.state != PlaybackPlaying {
		return errors.New("not playing")
	}

	if as.player != nil {
		as.player.Pause()
		// Store current position
		as.position, _ = as.reader.Seek(0, io.SeekCurrent)
	}
	
	as.state = PlaybackPaused
	return nil
}

// Stop stops audio playback and resets position
func (as *AudioStream) Stop() error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.state == PlaybackStopped {
		return nil
	}

	if as.player != nil {
		as.player.Pause()
		as.player.Close()
		as.player = nil
	}

	// Reset position
	as.position = 0
	as.reader.Seek(0, io.SeekStart)
	as.state = PlaybackStopped
	
	return nil
}

// GetState returns the current playback state
func (as *AudioStream) GetState() PlaybackState {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.state
}

// GetPosition returns the current playback position as a duration
func (as *AudioStream) GetPosition() time.Duration {
	as.mu.RLock()
	defer as.mu.RUnlock()
	
	// Use the thread-safe position from our tracking reader
	if as.reader != nil {
		pos := as.reader.GetPosition()
		samples := pos / BytesPerSample
		return time.Duration(samples) * time.Second / SampleRate
	}
	
	return 0
}

// GetDuration returns the total duration of the audio
func (as *AudioStream) GetDuration() time.Duration {
	return as.duration
}

// IsPlaying returns true if audio is currently playing
func (as *AudioStream) IsPlaying() bool {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.state == PlaybackPlaying
}

// monitorPlayback monitors the playback and updates state when finished
func (as *AudioStream) monitorPlayback() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.mu.RLock()
			playing := as.state == PlaybackPlaying
			player := as.player
			as.mu.RUnlock()
			
			if !playing {
				return
			}
			
			// Check if the player is still playing
			if player != nil && !player.IsPlaying() {
				// Check if we've actually reached the end of the data
				as.mu.Lock()
				// Use thread-safe position tracking
				pos := as.reader.GetPosition()
				if pos >= int64(len(as.data)) {
					// Playback truly finished
					as.state = PlaybackStopped
					if as.player != nil {
						as.player.Close()
						as.player = nil
					}
					as.position = 0
					as.reader.Seek(0, io.SeekStart)
					as.mu.Unlock()
					return
				}
				as.mu.Unlock()
			}
		}
	}
}

// Close releases all resources associated with the audio stream
func (as *AudioStream) Close() error {
	var err error
	
	as.closeOnce.Do(func() {
		// Cancel monitoring goroutine
		if as.cancel != nil {
			as.cancel()
		}
		
		// Stop playback
		if stopErr := as.Stop(); stopErr != nil {
			err = stopErr
		}
		
		// Clear finalizer
		runtime.SetFinalizer(as, nil)
		
		// Unpin memory
		as.unpinMemory()
		
		// Decrement reference count
		as.mu.Lock()
		as.refCount--
		if as.refCount <= 0 {
			// Clear data reference to allow GC
			as.data = nil
			as.reader = nil
		}
		as.mu.Unlock()
	})
	
	return err
}

// finalize is called by the garbage collector if Close() wasn't called
func (as *AudioStream) finalize() {
	// Best effort cleanup
	_ = as.Close()
}

// TTSAudioPlayer is the high-level audio player used by the TTS controller
type TTSAudioPlayer struct {
	currentStream *AudioStream
	mu            sync.Mutex
}

// NewTTSAudioPlayer creates a new audio player
func NewTTSAudioPlayer() *TTSAudioPlayer {
	return &TTSAudioPlayer{}
}

// PlayPCM plays PCM audio data
func (ap *TTSAudioPlayer) PlayPCM(pcmData []byte) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Stop current stream if playing
	if ap.currentStream != nil {
		ap.currentStream.Stop()
		ap.currentStream.Close()
	}

	// Create new stream
	stream, err := NewAudioStream(pcmData)
	if err != nil {
		return err
	}

	ap.currentStream = stream
	return stream.Play()
}

// Stop stops the current audio playback
func (ap *TTSAudioPlayer) Stop() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentStream != nil {
		return ap.currentStream.Stop()
	}
	return nil
}

// Pause pauses the current audio playback
func (ap *TTSAudioPlayer) Pause() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentStream != nil {
		return ap.currentStream.Pause()
	}
	return nil
}

// Resume resumes the current audio playback
func (ap *TTSAudioPlayer) Resume() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentStream != nil {
		return ap.currentStream.Play()
	}
	return nil
}

// GetState returns the current playback state
func (ap *TTSAudioPlayer) GetState() PlaybackState {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentStream != nil {
		return ap.currentStream.GetState()
	}
	return PlaybackStopped
}

// Close releases all resources
func (ap *TTSAudioPlayer) Close() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentStream != nil {
		return ap.currentStream.Close()
	}
	return nil
}