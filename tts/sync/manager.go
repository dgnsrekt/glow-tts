// Package sync provides audio-visual synchronization for TTS.
package sync

import (
	"sync"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
)

// Manager coordinates audio playback with visual highlighting.
type Manager struct {
	// Current state
	sentences    []tts.Sentence
	currentIndex int
	startTime    time.Time
	mu           sync.RWMutex

	// Timing
	ticker     *time.Ticker
	updateRate time.Duration

	// Drift correction
	driftThreshold time.Duration
	lastCorrection time.Time

	// Callbacks
	onChangeCallbacks []func(int)

	// Control
	stopCh chan struct{}
}

// NewManager creates a new synchronization manager.
func NewManager(updateRate time.Duration) *Manager {
	return &Manager{
		updateRate:     updateRate,
		driftThreshold: 500 * time.Millisecond,
		stopCh:         make(chan struct{}),
	}
}

// Start begins synchronization tracking.
func (m *Manager) Start(sentences []tts.Sentence, player tts.AudioPlayer) {
	m.mu.Lock()
	m.sentences = sentences
	m.currentIndex = 0
	m.startTime = time.Now()
	m.mu.Unlock()

	m.ticker = time.NewTicker(m.updateRate)
	go m.syncLoop(player)
}

// Stop halts synchronization tracking.
func (m *Manager) Stop() {
	close(m.stopCh)
	if m.ticker != nil {
		m.ticker.Stop()
	}

	// Reset state
	m.mu.Lock()
	m.currentIndex = 0
	m.sentences = nil
	m.mu.Unlock()

	// Recreate stop channel for next use
	m.stopCh = make(chan struct{})
}

// GetCurrentSentence returns the index of the current sentence.
func (m *Manager) GetCurrentSentence() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentIndex
}

// OnSentenceChange registers a callback for sentence changes.
func (m *Manager) OnSentenceChange(callback func(int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChangeCallbacks = append(m.onChangeCallbacks, callback)
}

// syncLoop continuously updates synchronization.
func (m *Manager) syncLoop(player tts.AudioPlayer) {
	defer func() {
		if m.ticker != nil {
			m.ticker.Stop()
		}
	}()

	for {
		select {
		case <-m.stopCh:
			return

		case <-m.ticker.C:
			m.update(player)
		}
	}
}

// update checks and updates the current sentence based on audio position.
func (m *Manager) update(player tts.AudioPlayer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sentences) == 0 {
		return
	}

	// Get current audio position
	audioPos := player.GetPosition()

	// Calculate expected sentence
	expectedIndex := m.findSentenceAtPosition(audioPos)

	// Check if sentence changed
	if expectedIndex != m.currentIndex {
		m.currentIndex = expectedIndex
		m.notifyChange(expectedIndex)
	}

	// Check for drift and correct if needed
	if m.needsDriftCorrection(audioPos, expectedIndex) {
		m.correctDrift(player, expectedIndex)
	}
}

// findSentenceAtPosition determines which sentence should be playing.
func (m *Manager) findSentenceAtPosition(position time.Duration) int {
	elapsed := time.Duration(0)

	for i, sentence := range m.sentences {
		elapsed += sentence.Duration
		if elapsed > position {
			return i
		}
	}

	return len(m.sentences) - 1
}

// notifyChange calls all registered callbacks for sentence changes.
func (m *Manager) notifyChange(index int) {
	for _, callback := range m.onChangeCallbacks {
		if callback != nil {
			// Call in goroutine to avoid blocking
			go callback(index)
		}
	}
}

// needsDriftCorrection checks if synchronization has drifted too much.
func (m *Manager) needsDriftCorrection(audioPos time.Duration, sentenceIndex int) bool {
	if sentenceIndex >= len(m.sentences) {
		return false
	}

	// Calculate expected position for current sentence
	expectedPos := time.Duration(0)
	for i := 0; i < sentenceIndex; i++ {
		expectedPos += m.sentences[i].Duration
	}

	// Check drift
	drift := audioPos - expectedPos
	if drift < 0 {
		drift = -drift
	}

	return drift > m.driftThreshold
}

// correctDrift attempts to correct synchronization drift.
func (m *Manager) correctDrift(player tts.AudioPlayer, expectedIndex int) {
	// TODO: Implement drift correction
	// This might involve adjusting playback position or speed
	m.lastCorrection = time.Now()
}