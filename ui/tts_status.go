package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
)

// TTSStatusDisplay provides rich TTS status information for the UI.
type TTSStatusDisplay struct {
	state           tts.StateType
	currentSentence int
	totalSentences  int
	position        time.Duration
	duration        time.Duration
	progress        float64
	isBuffering     bool
	bufferCount     int
	errorMessage    string
}

// NewTTSStatusDisplay creates a new TTS status display.
func NewTTSStatusDisplay() *TTSStatusDisplay {
	return &TTSStatusDisplay{
		state:           tts.StateIdle,
		currentSentence: -1,
		totalSentences:  0,
	}
}

// Update updates the status display with current TTS state.
func (s *TTSStatusDisplay) Update(state tts.State) {
	s.state = state.CurrentState
	s.currentSentence = state.Sentence
	s.totalSentences = state.TotalSentences
	s.position = state.Position
	s.duration = state.Duration
	
	// Calculate progress
	if s.totalSentences > 0 {
		s.progress = float64(s.currentSentence) / float64(s.totalSentences)
	} else {
		s.progress = 0
	}
	
	// Clear error if state is not error
	if state.CurrentState != tts.StateError {
		s.errorMessage = ""
	} else if state.LastError != nil {
		s.errorMessage = state.LastError.Error()
	}
}

// UpdateFromMessage updates the display from a TTS message.
func (s *TTSStatusDisplay) UpdateFromMessage(msg interface{}) {
	switch m := msg.(type) {
	case tts.PlayingMsg:
		s.state = tts.StatePlaying
		s.currentSentence = m.Sentence
		s.totalSentences = m.Total
		s.duration = m.Duration
		
	case tts.PausedMsg:
		s.state = tts.StatePaused
		s.position = m.Position
		s.currentSentence = m.Sentence
		
	case tts.ResumedMsg:
		s.state = tts.StatePlaying
		s.position = m.Position
		s.currentSentence = m.Sentence
		
	case tts.StoppedMsg:
		s.state = tts.StateIdle
		s.currentSentence = -1
		s.position = 0
		
	case tts.SentenceChangedMsg:
		s.currentSentence = m.Index
		s.duration = m.Duration
		s.progress = m.Progress
		
	case tts.TTSStateChangedMsg:
		s.state = m.State
		s.currentSentence = m.Sentence
		s.totalSentences = m.Total
		
	case tts.PositionUpdateMsg:
		s.position = m.Position
		s.duration = m.Duration
		s.currentSentence = m.SentenceIndex
		s.progress = m.TotalProgress
		
	case tts.BufferStatusMsg:
		s.isBuffering = m.IsLoading
		s.bufferCount = m.Buffered
		
	case tts.TTSErrorMsg:
		s.state = tts.StateError
		s.errorMessage = m.Error.Error()
	}
}

// CompactStatus returns a compact status string for the status bar.
func (s *TTSStatusDisplay) CompactStatus() string {
	if s.state == tts.StateIdle {
		return ""
	}
	
	var icon string
	var stateText string
	var color lipgloss.Color
	
	switch s.state {
	case tts.StatePlaying:
		icon = "▶"
		stateText = "TTS"
		color = lipgloss.Color("#00FF00") // Green
		
	case tts.StatePaused:
		icon = "⏸"
		stateText = "TTS"
		color = lipgloss.Color("#FFFF00") // Yellow
		
	case tts.StateReady:
		icon = "■"
		stateText = "TTS"
		color = lipgloss.Color("#888888") // Gray
		
	case tts.StateInitializing:
		icon = "⟳"
		stateText = "TTS"
		color = lipgloss.Color("#00AAFF") // Blue
		
	case tts.StateError:
		icon = "✗"
		stateText = "TTS"
		color = lipgloss.Color("#FF0000") // Red
		
	case tts.StateStopping:
		icon = "◼"
		stateText = "TTS"
		color = lipgloss.Color("#FF8800") // Orange
		
	default:
		return ""
	}
	
	// Build status string
	statusStyle := lipgloss.NewStyle().Foreground(color)
	status := statusStyle.Render(fmt.Sprintf("%s %s", icon, stateText))
	
	// Add sentence counter if playing/paused
	if (s.state == tts.StatePlaying || s.state == tts.StatePaused) && s.totalSentences > 0 {
		counterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		counter := counterStyle.Render(fmt.Sprintf(" %d/%d", s.currentSentence+1, s.totalSentences))
		status += counter
	}
	
	// Add buffer indicator if buffering
	if s.isBuffering {
		bufferStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
		status += bufferStyle.Render(" ⟳")
	}
	
	return status
}

// DetailedStatus returns a detailed multi-line status for display panels.
func (s *TTSStatusDisplay) DetailedStatus(width int) string {
	if s.state == tts.StateIdle {
		return ""
	}
	
	var lines []string
	
	// Header line with state
	headerStyle := lipgloss.NewStyle().Bold(true)
	lines = append(lines, headerStyle.Render("TTS Status"))
	
	// State line
	stateStyle := lipgloss.NewStyle().Foreground(s.getStateColor())
	stateLine := fmt.Sprintf("State: %s %s", s.getStateIcon(), s.state.String())
	lines = append(lines, stateStyle.Render(stateLine))
	
	// Progress line
	if s.totalSentences > 0 {
		progressLine := fmt.Sprintf("Sentence: %d of %d", s.currentSentence+1, s.totalSentences)
		lines = append(lines, progressLine)
		
		// Progress bar
		if width > 20 {
			bar := s.renderProgressBar(width - 4)
			lines = append(lines, bar)
		}
	}
	
	// Position line
	if s.duration > 0 {
		posLine := fmt.Sprintf("Position: %s / %s", formatDuration(s.position), formatDuration(s.duration))
		lines = append(lines, posLine)
	}
	
	// Buffer status
	if s.isBuffering {
		bufferStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00AAFF"))
		bufferLine := fmt.Sprintf("Buffering: %d sentences ready", s.bufferCount)
		lines = append(lines, bufferStyle.Render(bufferLine))
	}
	
	// Error message
	if s.errorMessage != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		errorLine := truncate.StringWithTail(s.errorMessage, uint(width-2), "...")
		lines = append(lines, errorStyle.Render("Error: "+errorLine))
	}
	
	return strings.Join(lines, "\n")
}

// ProgressBar returns a visual progress bar.
func (s *TTSStatusDisplay) ProgressBar(width int) string {
	if s.totalSentences <= 0 || width < 10 {
		return ""
	}
	
	return s.renderProgressBar(width)
}

// renderProgressBar creates a visual progress bar.
func (s *TTSStatusDisplay) renderProgressBar(width int) string {
	if width < 10 {
		return ""
	}
	
	// Calculate filled width
	filledWidth := int(s.progress * float64(width))
	if filledWidth > width {
		filledWidth = width
	}
	
	// Create bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", width-filledWidth)
	
	// Style the bar
	filledStyle := lipgloss.NewStyle().Foreground(s.getStateColor())
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
	
	return filledStyle.Render(filled) + emptyStyle.Render(empty)
}

// getStateColor returns the appropriate color for the current state.
func (s *TTSStatusDisplay) getStateColor() lipgloss.Color {
	switch s.state {
	case tts.StatePlaying:
		return lipgloss.Color("#00FF00") // Green
	case tts.StatePaused:
		return lipgloss.Color("#FFFF00") // Yellow
	case tts.StateReady:
		return lipgloss.Color("#888888") // Gray
	case tts.StateInitializing:
		return lipgloss.Color("#00AAFF") // Blue
	case tts.StateError:
		return lipgloss.Color("#FF0000") // Red
	case tts.StateStopping:
		return lipgloss.Color("#FF8800") // Orange
	default:
		return lipgloss.Color("#666666") // Dark gray
	}
}

// getStateIcon returns an icon for the current state.
func (s *TTSStatusDisplay) getStateIcon() string {
	switch s.state {
	case tts.StatePlaying:
		return "▶"
	case tts.StatePaused:
		return "⏸"
	case tts.StateReady:
		return "■"
	case tts.StateInitializing:
		return "⟳"
	case tts.StateError:
		return "✗"
	case tts.StateStopping:
		return "◼"
	default:
		return "○"
	}
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0:00"
	}
	
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// IsActive returns true if TTS is in an active state.
func (s *TTSStatusDisplay) IsActive() bool {
	return s.state != tts.StateIdle && s.state != tts.StateError
}

// NeedsUpdate returns true if the display should be updated.
func (s *TTSStatusDisplay) NeedsUpdate() bool {
	// Update when playing, buffering, or initializing
	return s.state == tts.StatePlaying || 
		s.state == tts.StateInitializing || 
		s.isBuffering
}

// Reset resets the status display to initial state.
func (s *TTSStatusDisplay) Reset() {
	s.state = tts.StateIdle
	s.currentSentence = -1
	s.totalSentences = 0
	s.position = 0
	s.duration = 0
	s.progress = 0
	s.isBuffering = false
	s.bufferCount = 0
	s.errorMessage = ""
}

// Clone creates a copy of the status display.
func (s *TTSStatusDisplay) Clone() *TTSStatusDisplay {
	return &TTSStatusDisplay{
		state:           s.state,
		currentSentence: s.currentSentence,
		totalSentences:  s.totalSentences,
		position:        s.position,
		duration:        s.duration,
		progress:        s.progress,
		isBuffering:     s.isBuffering,
		bufferCount:     s.bufferCount,
		errorMessage:    s.errorMessage,
	}
}