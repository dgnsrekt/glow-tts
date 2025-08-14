package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/pkg/tts"
	"github.com/charmbracelet/glow/v2/pkg/tts/engines"
	"github.com/charmbracelet/lipgloss"
)

// TTSState represents the TTS subsystem state in the UI
type TTSState struct {
	// Controller manages the TTS pipeline
	controller *tts.Controller

	// Engine being used (piper or gtts)
	engine string

	// Initialization state
	isInitializing bool
	isInitialized  bool

	// Playback state
	isPlaying  bool
	isPaused   bool
	isStopped  bool

	// Navigation state
	sentences            []tts.Sentence
	currentSentenceIndex int
	totalSentences       int

	// Speed control
	speedController *tts.TTSSpeedController

	// Error state
	lastError error
}

// NewTTSState creates a new TTS state instance
func NewTTSState(engine string) *TTSState {
	return &TTSState{
		engine:          engine,
		isInitializing:  true,  // Start as initializing since we'll init immediately
		isInitialized:   false,
		isStopped:       true,
		speedController: tts.NewSpeedController(),
	}
}

// IsEnabled returns true if TTS is enabled
func (t *TTSState) IsEnabled() bool {
	return t != nil && t.engine != ""
}

// TTS Commands - These follow the Bubble Tea command pattern

// ttsInitMsg is sent when TTS initialization completes
type ttsInitMsg struct {
	err error
}

// ttsTickMsg is sent periodically to refresh UI during initialization
type ttsTickMsg struct{}

// ttsPlayMsg is sent when play operation completes
type ttsPlayMsg struct {
	err error
}

// ttsPauseMsg is sent when pause operation completes
type ttsPauseMsg struct {
	err error
}

// ttsStopMsg is sent when stop operation completes
type ttsStopMsg struct {
	err error
}

// ttsNextMsg is sent when next sentence operation completes
type ttsNextMsg struct {
	sentenceIndex int
	err           error
}

// ttsPrevMsg is sent when previous sentence operation completes
type ttsPrevMsg struct {
	sentenceIndex int
	err           error
}

// ttsSpeedChangeMsg is sent when speed change completes
type ttsSpeedChangeMsg struct {
	newSpeed float64
	err      error
}

// ttsSentencesParsedMsg is sent when sentences are parsed from document
type ttsSentencesParsedMsg struct {
	sentences []tts.Sentence
	err       error
}

// ttsStatusUpdateMsg is sent periodically to update playback status
type ttsStatusUpdateMsg struct {
	sentenceIndex int
	isPlaying     bool
}

// TTS Commands - These are the async commands that perform TTS operations

// ttsTick sends periodic tick messages to refresh UI
func ttsTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return ttsTickMsg{}
	})
}

// initTTSCmd initializes the TTS controller with timeout
func initTTSCmd(engine string, ttsState *TTSState) tea.Cmd {
	return func() tea.Msg {
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] initTTSCmd executing for engine: %s\n", engine)
		
		// Create a channel to receive the result
		resultChan := make(chan tea.Msg, 1)
		
		// Run initialization in a goroutine
		go func() {
			resultChan <- initTTSWithTimeout(engine, ttsState)
		}()
		
		// Wait with timeout
		select {
		case result := <-resultChan:
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] initTTSCmd completed, returning result\n")
			return result
		case <-time.After(10 * time.Second):
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Initialization timeout after 10 seconds\n")
			return ttsInitMsg{err: fmt.Errorf("TTS initialization timed out after 10 seconds")}
		}
	}
}

// initTTSWithTimeout performs the actual initialization
func initTTSWithTimeout(engine string, ttsState *TTSState) tea.Msg {
		// Don't modify state here - it should be done in the Update function
		
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Starting initialization for engine: %s\n", engine)
		
		// Initialize TTS controller
		cfg := tts.ControllerConfig{
			Engine:             engine,
			EnableCache:        false,  // Disable cache for now
			LookaheadSentences: 3,
			DefaultSpeed:       1.0,
		}

		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Creating controller...\n")
		controller, err := tts.NewController(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Controller creation failed: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("controller creation failed: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Controller created successfully\n")

		// Set up the engine based on the engine string
		var ttsEngine tts.TTSEngine
		switch engine {
		case "piper":
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Creating Piper engine...\n")
			piperEngine, err := engines.NewPiperEngine()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[TTS DEBUG] Piper engine creation failed: %v\n", err)
				return ttsInitMsg{err: fmt.Errorf("failed to create Piper engine: %w", err)}
			}
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Piper engine created\n")
			ttsEngine = piperEngine
		default:
			return ttsInitMsg{err: fmt.Errorf("unsupported engine: %s", engine)}
		}

		// Set the engine
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Setting engine on controller...\n")
		if err := controller.SetEngine(ttsEngine); err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Failed to set engine: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set engine: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Engine set successfully\n")

		// Set the parser
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Creating parser...\n")
		parserConfig := &tts.ParserConfig{
			MinSentenceLength: 3,
			MaxSentenceLength: 500,
		}
		parser, err := tts.NewSentenceParser(parserConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Parser creation failed: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("failed to create parser: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Setting parser on controller...\n")
		if err := controller.SetParser(parser); err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Failed to set parser: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set parser: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Parser set successfully\n")

		// Set the speed controller
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Setting speed controller...\n")
		if err := controller.SetSpeedController(ttsState.speedController); err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Failed to set speed controller: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set speed controller: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Speed controller set successfully\n")

		// Initialize the controller
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Initializing controller...\n")
		if err := controller.Initialize(); err != nil {
			fmt.Fprintf(os.Stderr, "[TTS DEBUG] Controller initialization failed: %v\n", err)
			return ttsInitMsg{err: fmt.Errorf("failed to initialize controller: %w", err)}
		}
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] Controller initialized successfully\n")

		// Store the controller in the TTS state
		ttsState.controller = controller
		
		// Don't modify state flags here - let the Update function handle it
		
		fmt.Fprintf(os.Stderr, "[TTS DEBUG] TTS initialization complete!\n")
		return ttsInitMsg{err: nil}
}

// parseSentencesCmd parses sentences from markdown content
func parseSentencesCmd(content string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual sentence parsing
		// For now, return empty sentences
		return ttsSentencesParsedMsg{
			sentences: []tts.Sentence{},
			err:       nil,
		}
	}
}

// playTTSCmd starts or resumes TTS playback
func playTTSCmd(controller *tts.Controller, text string) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return ttsPlayMsg{err: fmt.Errorf("TTS controller not initialized")}
		}
		
		// Debug: log text length (commented out for production)
		// fmt.Fprintf(os.Stderr, "[DEBUG] Playing text of length: %d\n", len(text))
		
		// Check if text is empty
		if len(text) == 0 {
			return ttsPlayMsg{err: fmt.Errorf("no text to play")}
		}
		
		// Play the text using the controller
		err := controller.Play(text)
		if err != nil {
			return ttsPlayMsg{err: fmt.Errorf("failed to play: %w", err)}
		}
		
		return ttsPlayMsg{err: nil}
	}
}

// pauseTTSCmd pauses TTS playback
func pauseTTSCmd(controller *tts.Controller) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return ttsPauseMsg{err: fmt.Errorf("TTS controller not initialized")}
		}
		
		err := controller.Pause()
		if err != nil {
			return ttsPauseMsg{err: fmt.Errorf("failed to pause: %w", err)}
		}
		
		return ttsPauseMsg{err: nil}
	}
}

// stopTTSCmd stops TTS playback
func stopTTSCmd(controller *tts.Controller) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return ttsStopMsg{err: fmt.Errorf("TTS controller not initialized")}
		}
		
		err := controller.Stop()
		if err != nil {
			return ttsStopMsg{err: fmt.Errorf("failed to stop: %w", err)}
		}
		
		return ttsStopMsg{err: nil}
	}
}

// nextSentenceCmd moves to the next sentence
func nextSentenceCmd(controller *tts.Controller, currentIndex int, totalSentences int) tea.Cmd {
	return func() tea.Msg {
		if currentIndex >= totalSentences-1 {
			return ttsNextMsg{
				sentenceIndex: currentIndex,
				err:           fmt.Errorf("already at last sentence"),
			}
		}
		newIndex := currentIndex + 1
		// TODO: Implement actual navigation logic
		return ttsNextMsg{
			sentenceIndex: newIndex,
			err:           nil,
		}
	}
}

// prevSentenceCmd moves to the previous sentence
func prevSentenceCmd(controller *tts.Controller, currentIndex int) tea.Cmd {
	return func() tea.Msg {
		if currentIndex <= 0 {
			return ttsPrevMsg{
				sentenceIndex: 0,
				err:           fmt.Errorf("already at first sentence"),
			}
		}
		newIndex := currentIndex - 1
		// TODO: Implement actual navigation logic
		return ttsPrevMsg{
			sentenceIndex: newIndex,
			err:           nil,
		}
	}
}

// changeSpeedCmd changes the playback speed
func changeSpeedCmd(controller *tts.Controller, speed float64) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return ttsSpeedChangeMsg{
				newSpeed: speed,
				err:      fmt.Errorf("TTS controller not initialized"),
			}
		}
		// TODO: Implement actual speed change logic
		return ttsSpeedChangeMsg{
			newSpeed: speed,
			err:      nil,
		}
	}
}

// increaseSpeedCmd increases playback speed by one step
func (t *TTSState) increaseSpeedCmd() tea.Cmd {
	if t.speedController == nil {
		return nil
	}
	newSpeed, err := t.speedController.NextSpeed()
	if err != nil {
		return nil // Already at max speed
	}
	return changeSpeedCmd(t.controller, newSpeed)
}

// decreaseSpeedCmd decreases playback speed by one step
func (t *TTSState) decreaseSpeedCmd() tea.Cmd {
	if t.speedController == nil {
		return nil
	}
	newSpeed, err := t.speedController.PreviousSpeed()
	if err != nil {
		return nil // Already at min speed
	}
	return changeSpeedCmd(t.controller, newSpeed)
}

// RenderStatus renders the TTS status bar
func (t *TTSState) RenderStatus() string {
	if !t.IsEnabled() {
		return ""
	}

	// Debug log the render call
	fmt.Fprintf(os.Stderr, "[TTS DEBUG] RenderStatus called - isInitializing: %v, isInitialized: %v\n", 
		t.isInitializing, t.isInitialized)

	var parts []string

	// Engine indicator with initialization state
	engineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
	engineText := fmt.Sprintf("TTS: %s", strings.ToUpper(t.engine))
	
	// Add initialization indicator
	if t.isInitializing {
		engineText += " (Initializing...)"
	} else if !t.isInitialized {
		engineText += " (Not Ready)"
	}
	
	parts = append(parts, engineStyle.Render(engineText))

	// Playback state (only show if initialized)
	if t.isInitialized {
		stateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
		
		var stateIcon string
		if t.isPlaying {
			stateIcon = "▶"
		} else if t.isPaused {
			stateIcon = "⏸"
		} else {
			stateIcon = "■"
		}
		parts = append(parts, stateStyle.Render(stateIcon))
	}

	// Speed indicator (only show if initialized)
	if t.isInitialized {
		speedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("141"))
		var speedStr string
		if t.speedController != nil {
			speedStr = t.speedController.FormatSpeedCompact()
		} else {
			speedStr = "1.0x"
		}
		parts = append(parts, speedStyle.Render(speedStr))
	}

	// Sentence position
	if t.totalSentences > 0 {
		posStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("247"))
		parts = append(parts, posStyle.Render(
			fmt.Sprintf("%d/%d", t.currentSentenceIndex+1, t.totalSentences),
		))
	}

	// Error indicator
	if t.lastError != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
		// Include error message for debugging
		parts = append(parts, errorStyle.Render(fmt.Sprintf("⚠ %v", t.lastError)))
	}

	// Join with separators
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(" │ ")
	
	return strings.Join(parts, separator)
}

// GetKeyboardHelp returns keyboard shortcuts help text
func (t *TTSState) GetKeyboardHelp() string {
	if !t.IsEnabled() {
		return ""
	}

	help := []string{
		"Space: Play/Pause",
		"←/→: Prev/Next sentence",
		"+/-: Speed up/down",
		"S: Stop",
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	return helpStyle.Render(strings.Join(help, " • "))
}