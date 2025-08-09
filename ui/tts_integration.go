package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/tts"
)

// TTSController wraps the TTS controller for UI integration.
type TTSController struct {
	controller       *tts.Controller
	enabled          bool
	currentSentence  int
	totalSentences   int
	highlightedRange [2]int // Start and end positions for highlighting
	statusDisplay    *TTSStatusDisplay // Rich status display
}

// NewTTSController creates a new TTS controller wrapper.
func NewTTSController() *TTSController {
	return &TTSController{
		enabled:         false,
		currentSentence: -1,
		totalSentences:  0,
		statusDisplay:   NewTTSStatusDisplay(),
	}
}

// HandleTTSMessage handles TTS-related messages for the pager.
func (tc *TTSController) HandleTTSMessage(msg tea.Msg) (bool, tea.Cmd) {
	if tc == nil {
		return false, nil
	}

	// Update status display for all messages
	if tc.statusDisplay != nil {
		tc.statusDisplay.UpdateFromMessage(msg)
	}

	// Only process if enabled
	if !tc.enabled {
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
		
	case tts.TTSEnabledMsg:
		tc.enabled = true
		return true, nil
		
	case tts.TTSDisabledMsg:
		tc.enabled = false
		tc.statusDisplay.Reset()
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
	if tc == nil {
		return ""
	}

	// Use the new status display for rich status information
	if tc.statusDisplay != nil {
		return tc.statusDisplay.CompactStatus()
	}

	// Fallback to simple status if status display not available
	if !tc.enabled {
		return ""
	}

	if tc.controller == nil {
		return "TTS: initializing..."
	}

	return "TTS: active"
}

// GetDetailedStatus returns detailed TTS status for panels or dialogs.
func (tc *TTSController) GetDetailedStatus(width int) string {
	if tc == nil || tc.statusDisplay == nil {
		return ""
	}
	
	return tc.statusDisplay.DetailedStatus(width)
}

// GetProgressBar returns a visual progress bar for TTS playback.
func (tc *TTSController) GetProgressBar(width int) string {
	if tc == nil || tc.statusDisplay == nil {
		return ""
	}
	
	return tc.statusDisplay.ProgressBar(width)
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