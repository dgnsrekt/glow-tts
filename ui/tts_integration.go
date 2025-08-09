package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/lipgloss"
)

// TTSController wraps the TTS controller for UI integration.
type TTSController struct {
	controller       *tts.Controller
	enabled          bool
	currentSentence  int
	totalSentences   int
	highlightedRange [2]int // Start and end positions for highlighting
}

// NewTTSController creates a new TTS controller wrapper.
func NewTTSController() *TTSController {
	return &TTSController{
		enabled:         false,
		currentSentence: -1,
		totalSentences:  0,
	}
}

// HandleTTSMessage handles TTS-related messages for the pager.
func (tc *TTSController) HandleTTSMessage(msg tea.Msg) (bool, tea.Cmd) {
	if tc == nil || !tc.enabled {
		return false, nil
	}

	switch msg := msg.(type) {
	case tts.SentenceChangedMsg:
		tc.currentSentence = msg.Index
		return true, nil

	case tts.TTSStateChangedMsg:
		tc.totalSentences = msg.Total
		return true, nil

	case tts.PlayingMsg:
		tc.currentSentence = msg.Sentence
		tc.totalSentences = msg.Total
		return true, nil

	case tts.StoppedMsg:
		tc.currentSentence = -1
		return true, nil

	case tts.TTSErrorMsg:
		// Handle error, potentially disable TTS
		if !msg.Recoverable {
			tc.enabled = false
		}
		return true, nil
	}

	return false, nil
}

// HandleTTSKeyPress handles TTS-related keyboard shortcuts.
func (tc *TTSController) HandleTTSKeyPress(key string) tea.Cmd {
	if tc == nil || tc.controller == nil {
		return nil
	}

	switch key {
	case "T", "t":
		// Toggle TTS on/off
		tc.enabled = !tc.enabled
		if tc.enabled && tc.controller != nil {
			// Initialize if needed
			return func() tea.Msg {
				return tts.TTSEnabledMsg{Engine: "active"}
			}
		}
		return func() tea.Msg {
			tc.controller.Stop()
			return tts.TTSDisabledMsg{Reason: "user"}
		}

	case " ":
		// Play/pause
		if tc.enabled && tc.controller != nil {
			state := tc.controller.GetState()
			if state.CurrentState == tts.StatePlaying {
				tc.controller.Pause()
				return func() tea.Msg {
					return tts.PausedMsg{
						Position: state.Position,
						Sentence: tc.currentSentence,
					}
				}
			} else if state.CurrentState == tts.StatePaused || state.CurrentState == tts.StateReady {
				tc.controller.Play()
				return func() tea.Msg {
					return tts.PlayingMsg{
						Sentence: tc.currentSentence,
						Total:    tc.totalSentences,
					}
				}
			}
		}

	case "S", "s":
		// Stop
		if tc.enabled && tc.controller != nil {
			tc.controller.Stop()
			return func() tea.Msg {
				return tts.StoppedMsg{Reason: "user"}
			}
		}

	case "alt+left":
		// Previous sentence
		if tc.enabled && tc.controller != nil {
			tc.controller.PreviousSentence()
			return func() tea.Msg {
				return tts.NavigationMsg{
					Target:    tc.currentSentence - 1,
					Direction: "previous",
				}
			}
		}

	case "alt+right":
		// Next sentence
		if tc.enabled && tc.controller != nil {
			tc.controller.NextSentence()
			return func() tea.Msg {
				return tts.NavigationMsg{
					Target:    tc.currentSentence + 1,
					Direction: "next",
				}
			}
		}
	}

	return nil
}

// GetTTSStatus returns a status string for the TTS system.
func (tc *TTSController) GetTTSStatus() string {
	if tc == nil || !tc.enabled {
		return ""
	}

	if tc.controller == nil {
		return "TTS: initializing..."
	}

	state := tc.controller.GetState()
	
	switch state.CurrentState {
	case tts.StatePlaying:
		if tc.totalSentences > 0 {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Render("▶ TTS: ") + 
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("#888888")).
					Render(fmt.Sprintf("%d/%d", tc.currentSentence+1, tc.totalSentences))
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Render("▶ TTS: playing")

	case tts.StatePaused:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00")).
			Render("⏸ TTS: paused")

	case tts.StateReady:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Render("■ TTS: ready")

	case tts.StateError:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render("✗ TTS: error")

	default:
		return ""
	}
}

// ApplySentenceHighlight applies highlighting to the current sentence in the content.
func (tc *TTSController) ApplySentenceHighlight(content string) string {
	if tc == nil || !tc.enabled || tc.currentSentence < 0 {
		return content
	}

	// This is a simplified version - actual implementation would need to
	// parse sentences and apply highlighting to the correct range
	// For now, we just return the content unchanged
	return content
}

// IsEnabled returns whether TTS is enabled.
func (tc *TTSController) IsEnabled() bool {
	return tc != nil && tc.enabled
}

// GetCurrentSentence returns the current sentence index.
func (tc *TTSController) GetCurrentSentence() int {
	if tc == nil {
		return -1
	}
	return tc.currentSentence
}