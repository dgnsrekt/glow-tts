package tts

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
)

// MockAudioContext implements AudioContextInterface for testing without real audio
type MockAudioContext struct {
	mu         sync.Mutex
	ready      bool
	players    []*MockAudioPlayer
	sampleRate int
	channels   int
	
	// Test helpers
	PlayersCreated int
	PlayersClosed  int
}

// NewMockAudioContext creates a new mock audio context
func NewMockAudioContext() (*MockAudioContext, error) {
	log.Debug("Creating mock audio context for testing")
	return &MockAudioContext{
		ready:      true,
		sampleRate: SampleRate,
		channels:   Channels,
		players:    make([]*MockAudioPlayer, 0),
	}, nil
}

// NewPlayer creates a new mock audio player
func (mac *MockAudioContext) NewPlayer(r io.Reader) (AudioPlayerInterface, error) {
	mac.mu.Lock()
	defer mac.mu.Unlock()

	if !mac.ready {
		return nil, fmt.Errorf("mock audio context not ready")
	}

	// Read all data from reader to simulate consumption
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}
	
	// Check if the reader is seekable (like bytes.Reader)
	var originalReader io.ReadSeeker
	if seeker, ok := r.(io.ReadSeeker); ok {
		originalReader = seeker
		// Reset position after reading all data
		_, _ = originalReader.Seek(0, io.SeekStart)
	}

	player := &MockAudioPlayer{
		context:        mac,
		reader:         bytes.NewReader(data),
		originalReader: originalReader,
		data:           data,
		volume:         1.0,
		position:       0,
	}

	mac.players = append(mac.players, player)
	mac.PlayersCreated++
	
	log.Debug("Created mock audio player", 
		"data_size", len(data),
		"players_created", mac.PlayersCreated)
	
	return player, nil
}

// Close closes the mock audio context
func (mac *MockAudioContext) Close() error {
	mac.mu.Lock()
	defer mac.mu.Unlock()

	// Close all players
	for _, player := range mac.players {
		_ = player.Close()
	}

	mac.ready = false
	mac.players = nil
	log.Debug("Mock audio context closed")
	return nil
}

// IsReady returns whether the context is ready
func (mac *MockAudioContext) IsReady() bool {
	mac.mu.Lock()
	defer mac.mu.Unlock()
	return mac.ready
}

// SampleRate returns the sample rate
func (mac *MockAudioContext) SampleRate() int {
	return mac.sampleRate
}

// ChannelCount returns the number of channels
func (mac *MockAudioContext) ChannelCount() int {
	return mac.channels
}

// GetPlayersCreated returns the number of players created (for testing)
func (mac *MockAudioContext) GetPlayersCreated() int {
	mac.mu.Lock()
	defer mac.mu.Unlock()
	return mac.PlayersCreated
}

// MockAudioPlayer implements AudioPlayerInterface for testing
type MockAudioPlayer struct {
	context        *MockAudioContext
	reader         *bytes.Reader
	originalReader io.ReadSeeker  // The original reader from AudioStream
	data           []byte
	mu             sync.Mutex
	
	// State
	playing  atomic.Bool
	paused   atomic.Bool
	closed   atomic.Bool
	volume   float64
	position int64
	
	// Playback tracking
	startTime    time.Time
	pausedTime   time.Duration
	lastPauseTime time.Time
	
	// Test helpers
	PlayCount  int
	PauseCount int
	ResetCount int
	SeekCount  int
}

// Play starts or resumes playback
func (m *MockAudioPlayer) Play() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.playing.Load() {
		m.playing.Store(true)
		m.paused.Store(false)
		m.PlayCount++
		
		// Track timing for position calculation
		if m.startTime.IsZero() {
			m.startTime = time.Now()
		} else if m.paused.Load() {
			// Resuming from pause
			m.pausedTime += time.Since(m.lastPauseTime)
		}
		
		// Simulate audio playback in background
		go m.simulatePlayback()
		
		log.Debug("Mock player started", "play_count", m.PlayCount)
	} else if m.paused.Load() {
		// Resume from pause
		m.paused.Store(false)
		m.pausedTime += time.Since(m.lastPauseTime)
		log.Debug("Mock player resumed")
	}
}

// simulatePlayback simulates audio playback
func (m *MockAudioPlayer) simulatePlayback() {
	// Calculate duration based on data size
	// 16-bit mono at 22050Hz = 44100 bytes per second
	bytesPerSecond := float64(SampleRate * BytesPerSample * Channels)
	duration := time.Duration(float64(len(m.data)) / bytesPerSecond * float64(time.Second))
	
	// Simulate playback time
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if !m.playing.Load() || m.closed.Load() {
				return
			}
			
			if m.paused.Load() {
				// Don't advance position while paused
				continue
			}
			
			// Calculate current playback position
			m.mu.Lock()
			if !m.startTime.IsZero() {
				elapsed := time.Since(m.startTime) - m.pausedTime
				bytesPlayed := int64(elapsed.Seconds() * bytesPerSecond)
				
				// Update reader position to match playback
				if bytesPlayed < int64(len(m.data)) {
					_, _ = m.reader.Seek(bytesPlayed, io.SeekStart)
					// Also update the original reader if it exists
					if m.originalReader != nil {
						_, _ = m.originalReader.Seek(bytesPlayed, io.SeekStart)
					}
					m.position = bytesPlayed
				} else {
					// Playback completed
					m.playing.Store(false)
					_, _ = m.reader.Seek(0, io.SeekStart)
					if m.originalReader != nil {
						_, _ = m.originalReader.Seek(0, io.SeekStart)
					}
					m.position = 0
					m.mu.Unlock()
					log.Debug("Mock playback completed", "duration", duration)
					return
				}
			}
			m.mu.Unlock()
		}
	}
}

// Pause pauses playback
func (m *MockAudioPlayer) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.playing.Load() && !m.paused.Load() {
		m.paused.Store(true)
		m.lastPauseTime = time.Now()
		m.PauseCount++
		log.Debug("Mock player paused", "pause_count", m.PauseCount)
	}
}

// IsPlaying returns whether audio is currently playing
func (m *MockAudioPlayer) IsPlaying() bool {
	return m.playing.Load() && !m.paused.Load()
}

// Reset resets the player to the beginning
func (m *MockAudioPlayer) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	_, _ = m.reader.Seek(0, io.SeekStart)
	if m.originalReader != nil {
		_, _ = m.originalReader.Seek(0, io.SeekStart)
	}
	m.position = 0
	m.startTime = time.Time{}
	m.pausedTime = 0
	m.lastPauseTime = time.Time{}
	m.ResetCount++
	log.Debug("Mock player reset", "reset_count", m.ResetCount)
	return nil
}

// Close closes the player
func (m *MockAudioPlayer) Close() error {
	if !m.closed.Load() {
		m.closed.Store(true)
		m.playing.Store(false)
		m.context.mu.Lock()
		m.context.PlayersClosed++
		m.context.mu.Unlock()
		log.Debug("Mock player closed")
	}
	return nil
}

// SetVolume sets the playback volume (0.0 to 1.0)
func (m *MockAudioPlayer) SetVolume(volume float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.volume = volume
	log.Debug("Mock player volume set", "volume", volume)
}

// Volume returns the current volume
func (m *MockAudioPlayer) Volume() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.volume
}

// Seek seeks to a specific position
func (m *MockAudioPlayer) Seek(offset int64, whence int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	pos, err := m.reader.Seek(offset, whence)
	if err == nil {
		m.position = pos
		// Also update the original reader if it exists
		if m.originalReader != nil {
			m.originalReader.Seek(offset, whence)
		}
		m.SeekCount++
		log.Debug("Mock player seeked", "position", pos, "seek_count", m.SeekCount)
	}
	return pos, err
}

// BufferedDuration returns the duration of buffered audio
func (m *MockAudioPlayer) BufferedDuration() time.Duration {
	// Mock implementation: return a simulated buffer duration
	return 100 * time.Millisecond
}

// GetPlayCount returns the number of times Play was called (for testing)
func (m *MockAudioPlayer) GetPlayCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.PlayCount
}

// GetDataSize returns the size of audio data (for testing)
func (m *MockAudioPlayer) GetDataSize() int {
	return len(m.data)
}