// Package sync provides audio-visual synchronization for TTS.
package sync

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// Config holds configuration for the synchronization manager.
type Config struct {
	// UpdateRate is how often to check synchronization
	UpdateRate time.Duration
	
	// DriftThreshold is the maximum allowed drift before correction
	DriftThreshold time.Duration
	
	// CorrectionBackoff is the minimum time between corrections
	CorrectionBackoff time.Duration
	
	// HistorySize is the number of drift samples to keep
	HistorySize int
	
	// SmoothingFactor for position updates (0.0 to 1.0)
	SmoothingFactor float64
}

// DefaultConfig returns the default synchronization configuration.
func DefaultConfig() Config {
	return Config{
		UpdateRate:        50 * time.Millisecond, // 20 Hz update rate
		DriftThreshold:    200 * time.Millisecond, // Tighter threshold for better sync
		CorrectionBackoff: 500 * time.Millisecond,
		HistorySize:       20,
		SmoothingFactor:   0.3,
	}
}

// DriftSample represents a single drift measurement.
type DriftSample struct {
	Timestamp time.Time
	AudioPos  time.Duration
	Expected  time.Duration
	Drift     time.Duration
	Corrected bool
}

// Manager coordinates audio playback with visual highlighting.
type Manager struct {
	// Configuration
	config Config
	
	// Current state
	sentences      []tts.Sentence
	currentIndex   int32 // atomic
	lastIndex      int
	startTime      time.Time
	totalDuration  time.Duration
	mu             sync.RWMutex
	
	// Audio player reference
	player tts.AudioPlayer
	
	// Position tracking
	lastAudioPos   time.Duration
	smoothedPos    time.Duration
	positionOffset time.Duration // For drift correction
	
	// Timing
	ticker  *time.Ticker
	running int32 // atomic
	
	// Drift correction
	driftHistory   []DriftSample
	lastCorrection time.Time
	correctionCount int
	backoffMultiplier float64
	
	// Callbacks
	onChangeCallbacks []func(int)
	callbackMu        sync.RWMutex
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
	wg     sync.WaitGroup
	
	// Statistics
	stats SyncStats
}

// SyncStats holds synchronization statistics.
type SyncStats struct {
	TotalUpdates      uint64
	SentenceChanges   uint64
	DriftCorrections  uint64
	AverageDrift      time.Duration
	MaxDrift          time.Duration
	LastUpdate        time.Time
}

// NewManager creates a new synchronization manager.
func NewManager(config Config) *Manager {
	// Validate config
	if config.UpdateRate <= 0 {
		config.UpdateRate = 50 * time.Millisecond
	}
	if config.DriftThreshold <= 0 {
		config.DriftThreshold = 200 * time.Millisecond
	}
	if config.CorrectionBackoff <= 0 {
		config.CorrectionBackoff = 500 * time.Millisecond
	}
	if config.HistorySize <= 0 {
		config.HistorySize = 20
	}
	if config.SmoothingFactor <= 0 || config.SmoothingFactor > 1 {
		config.SmoothingFactor = 0.3
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Manager{
		config:            config,
		stopCh:            make(chan struct{}),
		driftHistory:      make([]DriftSample, 0, config.HistorySize),
		backoffMultiplier: 1.0,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start begins synchronization tracking.
func (m *Manager) Start(sentences []tts.Sentence, player tts.AudioPlayer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if already running
	if atomic.LoadInt32(&m.running) == 1 {
		m.stopInternal()
	}
	
	// Set up state
	m.sentences = sentences
	m.player = player
	m.lastIndex = -1
	m.startTime = time.Now()
	m.positionOffset = 0
	m.smoothedPos = 0
	m.lastAudioPos = 0
	
	// Calculate total duration
	m.totalDuration = 0
	for _, s := range sentences {
		m.totalDuration += s.Duration
	}
	
	// Reset statistics
	m.stats = SyncStats{}
	m.correctionCount = 0
	m.backoffMultiplier = 1.0
	
	// Clear drift history
	m.driftHistory = m.driftHistory[:0]
	
	// Set initial index
	atomic.StoreInt32(&m.currentIndex, 0)
	
	// Mark as running
	atomic.StoreInt32(&m.running, 1)
	
	// Start update ticker
	m.ticker = time.NewTicker(m.config.UpdateRate)
	
	// Start sync loop
	m.wg.Add(1)
	go m.syncLoop()
}

// Stop halts synchronization tracking.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.stopInternal()
}

// stopInternal stops synchronization without locking (must be called with lock held).
func (m *Manager) stopInternal() {
	// Check if running
	if atomic.LoadInt32(&m.running) == 0 {
		return
	}
	
	// Mark as stopped
	atomic.StoreInt32(&m.running, 0)
	
	// Stop ticker
	if m.ticker != nil {
		m.ticker.Stop()
		m.ticker = nil
	}
	
	// Signal stop (non-blocking)
	select {
	case m.stopCh <- struct{}{}:
	default:
	}
	
	// We need to wait for the goroutine but can't hold the lock
	// Store what we need to clean up
	needsWait := true
	
	// Unlock, wait, then re-lock
	if needsWait {
		m.mu.Unlock()
		m.wg.Wait()
		m.mu.Lock()
	}
	
	// Reset state
	atomic.StoreInt32(&m.currentIndex, 0)
	m.sentences = nil
	m.player = nil
	
	// Recreate stop channel
	m.stopCh = make(chan struct{})
}

// GetCurrentSentence returns the index of the current sentence.
func (m *Manager) GetCurrentSentence() int {
	return int(atomic.LoadInt32(&m.currentIndex))
}

// OnSentenceChange registers a callback for sentence changes.
func (m *Manager) OnSentenceChange(callback func(int)) {
	m.callbackMu.Lock()
	defer m.callbackMu.Unlock()
	m.onChangeCallbacks = append(m.onChangeCallbacks, callback)
}

// GetStats returns synchronization statistics.
func (m *Manager) GetStats() SyncStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Calculate average drift
	if len(m.driftHistory) > 0 {
		total := time.Duration(0)
		for _, sample := range m.driftHistory {
			drift := sample.Drift
			if drift < 0 {
				drift = -drift
			}
			total += drift
		}
		m.stats.AverageDrift = total / time.Duration(len(m.driftHistory))
	}
	
	return m.stats
}

// syncLoop continuously updates synchronization.
func (m *Manager) syncLoop() {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		if m.ticker != nil {
			m.ticker.Stop()
			m.ticker = nil
		}
		m.mu.Unlock()
	}()
	
	// Get local reference to ticker
	m.mu.RLock()
	ticker := m.ticker
	m.mu.RUnlock()
	
	if ticker == nil {
		return
	}
	
	for {
		select {
		case <-m.ctx.Done():
			return
			
		case <-m.stopCh:
			return
			
		case <-ticker.C:
			m.update()
		}
	}
}

// update checks and updates the current sentence based on audio position.
func (m *Manager) update() {
	// Check if still running
	if atomic.LoadInt32(&m.running) == 0 {
		return
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.sentences) == 0 || m.player == nil {
		return
	}
	
	// Get current audio position
	audioPos := m.player.GetPosition()
	
	// Apply position offset for drift correction
	adjustedPos := audioPos + m.positionOffset
	
	// Apply smoothing to reduce jitter
	m.smoothedPos = m.smoothPosition(adjustedPos)
	
	// Calculate expected sentence
	expectedIndex, expectedPos := m.findSentenceAtPosition(m.smoothedPos)
	
	// Record drift sample (use raw audio position for drift calculation)
	drift := audioPos - expectedPos
	m.recordDrift(audioPos, expectedPos, drift)
	
	// Update statistics
	atomic.AddUint64(&m.stats.TotalUpdates, 1)
	m.stats.LastUpdate = time.Now()
	if absDrift := drift; absDrift < 0 {
		absDrift = -absDrift
		if absDrift > m.stats.MaxDrift {
			m.stats.MaxDrift = absDrift
		}
	} else if drift > m.stats.MaxDrift {
		m.stats.MaxDrift = drift
	}
	
	// Check if sentence changed
	currentIdx := int(atomic.LoadInt32(&m.currentIndex))
	if expectedIndex != currentIdx {
		atomic.StoreInt32(&m.currentIndex, int32(expectedIndex))
		atomic.AddUint64(&m.stats.SentenceChanges, 1)
		m.notifyChange(expectedIndex)
		m.lastIndex = expectedIndex
	}
	
	// Check for drift and correct if needed
	if m.needsDriftCorrection(drift) {
		m.correctDrift(drift)
	}
	
	// Update last position
	m.lastAudioPos = audioPos
}

// smoothPosition applies exponential smoothing to reduce position jitter.
func (m *Manager) smoothPosition(currentPos time.Duration) time.Duration {
	if m.smoothedPos == 0 {
		return currentPos
	}
	
	// Exponential moving average
	alpha := m.config.SmoothingFactor
	smoothed := time.Duration(float64(m.smoothedPos)*(1-alpha) + float64(currentPos)*alpha)
	
	return smoothed
}

// findSentenceAtPosition determines which sentence should be playing.
func (m *Manager) findSentenceAtPosition(position time.Duration) (int, time.Duration) {
	if position < 0 {
		return 0, 0
	}
	
	elapsed := time.Duration(0)
	
	for i, sentence := range m.sentences {
		nextElapsed := elapsed + sentence.Duration
		if position < nextElapsed {
			return i, elapsed
		}
		elapsed = nextElapsed
	}
	
	// Past the end, return last sentence
	if len(m.sentences) > 0 {
		return len(m.sentences) - 1, elapsed - m.sentences[len(m.sentences)-1].Duration
	}
	return 0, 0
}

// recordDrift adds a drift sample to history.
func (m *Manager) recordDrift(audioPos, expectedPos, drift time.Duration) {
	sample := DriftSample{
		Timestamp: time.Now(),
		AudioPos:  audioPos,
		Expected:  expectedPos,
		Drift:     drift,
		Corrected: false,
	}
	
	// Add to history
	m.driftHistory = append(m.driftHistory, sample)
	
	// Maintain history size
	if len(m.driftHistory) > m.config.HistorySize {
		m.driftHistory = m.driftHistory[1:]
	}
}

// needsDriftCorrection checks if synchronization has drifted too much.
func (m *Manager) needsDriftCorrection(drift time.Duration) bool {
	// Get absolute drift
	absDrift := drift
	if absDrift < 0 {
		absDrift = -absDrift
	}
	
	// Check if drift exceeds threshold
	if absDrift < m.config.DriftThreshold {
		return false
	}
	
	// Check backoff period with exponential backoff
	backoffDuration := time.Duration(float64(m.config.CorrectionBackoff) * m.backoffMultiplier)
	if time.Since(m.lastCorrection) < backoffDuration {
		return false
	}
	
	// Check for consistent drift pattern
	if !m.hasConsistentDrift() {
		return false
	}
	
	return true
}

// hasConsistentDrift checks if drift is consistent across recent samples.
func (m *Manager) hasConsistentDrift() bool {
	if len(m.driftHistory) < 3 {
		return true // Not enough history, allow correction
	}
	
	// Check last 3 samples for consistent drift direction
	recentSamples := m.driftHistory[len(m.driftHistory)-3:]
	positive := 0
	negative := 0
	
	for _, sample := range recentSamples {
		if sample.Drift > 0 {
			positive++
		} else if sample.Drift < 0 {
			negative++
		}
	}
	
	// Consistent if mostly in same direction
	return positive >= 2 || negative >= 2
}

// correctDrift attempts to correct synchronization drift.
func (m *Manager) correctDrift(drift time.Duration) {
	// Calculate correction amount (gradual correction)
	correction := time.Duration(float64(drift) * 0.5)
	
	// Apply correction to position offset
	m.positionOffset -= correction
	
	// Update correction tracking
	m.lastCorrection = time.Now()
	m.correctionCount++
	atomic.AddUint64(&m.stats.DriftCorrections, 1)
	
	// Update backoff multiplier (exponential backoff)
	if m.correctionCount > 3 {
		m.backoffMultiplier = min(m.backoffMultiplier*1.5, 10.0)
	}
	
	// Mark recent samples as corrected
	if len(m.driftHistory) > 0 {
		m.driftHistory[len(m.driftHistory)-1].Corrected = true
	}
	
	// Reset backoff if correction is successful (drift reduces)
	if len(m.driftHistory) >= 2 {
		prevDrift := m.driftHistory[len(m.driftHistory)-2].Drift
		if absDrift(drift) < absDrift(prevDrift) {
			m.backoffMultiplier = max(m.backoffMultiplier*0.8, 1.0)
		}
	}
}

// notifyChange calls all registered callbacks for sentence changes.
func (m *Manager) notifyChange(index int) {
	m.callbackMu.RLock()
	callbacks := make([]func(int), len(m.onChangeCallbacks))
	copy(callbacks, m.onChangeCallbacks)
	m.callbackMu.RUnlock()
	
	for _, callback := range callbacks {
		if callback != nil {
			// Call in goroutine to avoid blocking
			go func(cb func(int), idx int) {
				defer func() {
					// Recover from panic in callback
					if r := recover(); r != nil {
						// Callback panicked, ignore
					}
				}()
				cb(idx)
			}(callback, index)
		}
	}
}

// GetDriftHistory returns recent drift samples for analysis.
func (m *Manager) GetDriftHistory() []DriftSample {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	history := make([]DriftSample, len(m.driftHistory))
	copy(history, m.driftHistory)
	return history
}

// Reset clears the manager state without stopping.
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	atomic.StoreInt32(&m.currentIndex, 0)
	m.lastIndex = -1
	m.positionOffset = 0
	m.smoothedPos = 0
	m.lastAudioPos = 0
	m.driftHistory = m.driftHistory[:0]
	m.correctionCount = 0
	m.backoffMultiplier = 1.0
}

// IsRunning returns whether synchronization is active.
func (m *Manager) IsRunning() bool {
	return atomic.LoadInt32(&m.running) == 1
}

// Helper functions

func absDrift(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}