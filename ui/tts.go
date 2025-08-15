package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/pkg/tts"
	"github.com/charmbracelet/glow/v2/pkg/tts/engines"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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
	
	// Loading states
	isSynthesizing bool
	isBuffering    bool
	loadingSpinner spinner.Model
	loadingMessage string
	
	// Playback timer
	playbackTimer  timer.Model
	playbackStart  time.Time

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
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	
	// Create a timer that counts up (using a large timeout)
	t := timer.NewWithInterval(24 * time.Hour, time.Second)
	
	return &TTSState{
		engine:          engine,
		isInitializing:  true,  // Start as initializing since we'll init immediately
		isInitialized:   false,
		isStopped:       true,
		speedController: tts.NewSpeedController(),
		loadingSpinner:  s,
		loadingMessage:  "Initializing TTS engine",
		playbackTimer:   t,
	}
}

// IsEnabled returns true if TTS is enabled
func (t *TTSState) IsEnabled() bool {
	return t != nil && t.engine != ""
}

// Update handles spinner and timer updates and returns any commands
func (t *TTSState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Update spinner if we're in a loading state
	if t.isInitializing || t.isSynthesizing || t.isBuffering {
		var cmd tea.Cmd
		t.loadingSpinner, cmd = t.loadingSpinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	// Update timer if playing
	if t.isPlaying && !t.isPaused {
		var cmd tea.Cmd
		t.playbackTimer, cmd = t.playbackTimer.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return nil, tea.Batch(cmds...)
}

// SetLoadingState updates the loading state and message
func (t *TTSState) SetLoadingState(synthesizing, buffering bool, message string) {
	t.isSynthesizing = synthesizing
	t.isBuffering = buffering
	if message != "" {
		t.loadingMessage = message
	}
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

// ttsPlaybackFinishedMsg is sent when playback completes
type ttsPlaybackFinishedMsg struct{}

// ttsClearErrorMsg is sent to clear TTS error messages
type ttsClearErrorMsg struct{}

// TTS Commands - These are the async commands that perform TTS operations

// ttsTick sends periodic tick messages to refresh UI
func ttsTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return ttsTickMsg{}
	})
}

// clearTTSErrorCmd clears the TTS error after a delay
func clearTTSErrorCmd(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return ttsClearErrorMsg{}
	})
}

// initTTSCmd initializes the TTS controller with timeout
func initTTSCmd(engine string, ttsState *TTSState) tea.Cmd {
	return func() tea.Msg {
		log.Debug("initTTSCmd executing", "engine", engine)
		
		// Create a channel to receive the result
		resultChan := make(chan tea.Msg, 1)
		
		// Run initialization in a goroutine
		go func() {
			resultChan <- initTTSWithTimeout(engine, ttsState)
		}()
		
		// Wait with timeout
		select {
		case result := <-resultChan:
			log.Debug("initTTSCmd completed")
			return result
		case <-time.After(10 * time.Second):
			log.Error("TTS initialization timeout", "timeout", "10 seconds")
			return ttsInitMsg{err: fmt.Errorf("TTS initialization timed out after 10 seconds")}
		}
	}
}

// initTTSWithTimeout performs the actual initialization
func initTTSWithTimeout(engine string, ttsState *TTSState) tea.Msg {
		// Don't modify state here - it should be done in the Update function
		
		log.Debug("starting TTS initialization", "engine", engine)
		
		// Validate engine availability first
		if err := tts.ValidateEngineAvailability(engine); err != nil {
			log.Error("TTS engine validation failed", "engine", engine, "error", err)
			return ttsInitMsg{err: err}
		}
		
		// Initialize TTS controller
		cfg := tts.ControllerConfig{
			Engine:             engine,
			EnableCache:        false,  // Disable cache for now
			LookaheadSentences: 3,
			DefaultSpeed:       1.0,
		}

		log.Debug("creating TTS controller")
		controller, err := tts.NewController(cfg)
		if err != nil {
			log.Error("TTS controller creation failed", "error", err)
			return ttsInitMsg{err: fmt.Errorf("controller creation failed: %w", err)}
		}
		log.Debug("TTS controller created successfully")

		// Set up the engine based on the engine string
		var ttsEngine tts.TTSEngine
		switch engine {
		case "piper":
			log.Debug("creating Piper engine")
			piperEngine, err := engines.NewPiperEngine()
			if err != nil {
				log.Error("Piper engine creation failed", "error", err)
				return ttsInitMsg{err: fmt.Errorf("failed to create Piper engine: %w", err)}
			}
			log.Debug("Piper engine created")
			ttsEngine = piperEngine
		case "gtts":
			log.Debug("creating Google TTS engine")
			gttsEngine, err := engines.NewGTTSEngine()
			if err != nil {
				log.Error("Google TTS engine creation failed", "error", err)
				return ttsInitMsg{err: fmt.Errorf("failed to create Google TTS engine: %w", err)}
			}
			log.Debug("Google TTS engine created")
			ttsEngine = gttsEngine
		default:
			return ttsInitMsg{err: fmt.Errorf("unsupported engine: %s", engine)}
		}

		// Set the engine
		log.Debug("setting engine on controller")
		if err := controller.SetEngine(ttsEngine); err != nil {
			log.Error("failed to set engine", "error", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set engine: %w", err)}
		}
		log.Debug("engine set successfully")

		// Set the parser
		log.Debug("creating parser")
		parserConfig := &tts.ParserConfig{
			MinSentenceLength: 3,
			MaxSentenceLength: 500,
		}
		parser, err := tts.NewSentenceParser(parserConfig)
		if err != nil {
			log.Error("parser creation failed", "error", err)
			return ttsInitMsg{err: fmt.Errorf("failed to create parser: %w", err)}
		}
		log.Debug("setting parser on controller")
		if err := controller.SetParser(parser); err != nil {
			log.Error("failed to set parser", "error", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set parser: %w", err)}
		}
		log.Debug("parser set successfully")

		// Set the speed controller
		log.Debug("setting speed controller")
		if err := controller.SetSpeedController(ttsState.speedController); err != nil {
			log.Error("failed to set speed controller", "error", err)
			return ttsInitMsg{err: fmt.Errorf("failed to set speed controller: %w", err)}
		}
		log.Debug("speed controller set successfully")

		// Initialize the controller
		log.Debug("initializing controller")
		if err := controller.Initialize(); err != nil {
			log.Error("controller initialization failed", "error", err)
			return ttsInitMsg{err: fmt.Errorf("failed to initialize controller: %w", err)}
		}
		log.Debug("controller initialized successfully")

		// Store the controller in the TTS state
		ttsState.controller = controller
		
		// Register components for lifecycle management
		lifecycle := tts.GetLifecycleManager()
		
		// Register engine for cleanup
		lifecycle.Register(tts.NewEngineLifecycle(selectedEngine, engine))
		
		// Register queue if it exists
		if queue := controller.GetQueue(); queue != nil {
			lifecycle.Register(tts.NewQueueLifecycle(queue))
		}
		
		// Register audio player
		if player := tts.GetGlobalAudioPlayer(); player != nil {
			lifecycle.Register(tts.NewPlayerLifecycle(player))
		}
		
		// Register cache if enabled
		if cache := controller.GetCache(); cache != nil {
			lifecycle.Register(tts.NewCacheLifecycle(cache, true))
		}
		
		// Don't modify state flags here - let the Update function handle it
		
		log.Info("TTS initialization complete", "engine", engine)
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

// ttsMonitorMsg is sent periodically during playback monitoring
type ttsMonitorMsg struct {
	continueMonitoring bool
}

// monitorPlaybackCmd monitors playback and sends updates when it finishes
func monitorPlaybackCmd(controller *tts.Controller) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return nil
		}
		
		// Check the audio player state once
		player := tts.GetGlobalAudioPlayer()
		if player != nil {
			state := player.GetState()
			if state == tts.PlaybackStopped {
				// Try to play the next segment
				log.Debug("TTS: Current segment finished, attempting to play next")
				err := controller.Next()
				if err != nil {
					// No more segments or error, stop playback
					log.Debug("TTS: No more segments to play", "error", err)
					return ttsPlaybackFinishedMsg{}
				}
				// Continue monitoring for the next segment
				log.Debug("TTS: Playing next segment, continuing monitor")
				return ttsMonitorMsg{continueMonitoring: true}
			}
			// Still playing, continue monitoring
			return ttsMonitorMsg{continueMonitoring: true}
		}
		
		return nil
	}
}

// monitorDelayCmd waits before checking playback status again
func monitorDelayCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
		return ttsMonitorMsg{continueMonitoring: true}
	})
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
		if controller == nil {
			return ttsNextMsg{
				sentenceIndex: currentIndex,
				err:           fmt.Errorf("TTS controller not initialized"),
			}
		}
		
		// Use the controller's Next method to navigate
		err := controller.Next()
		if err != nil {
			// Check if we're at the end - this is informational, not an error
			if strings.Contains(err.Error(), "end of queue") {
				// Just return current index, no error - user is at the end
				return ttsNextMsg{
					sentenceIndex: currentIndex,
					err:           nil, // Don't show error for boundary
				}
			}
			return ttsNextMsg{
				sentenceIndex: currentIndex,
				err:           err,
			}
		}
		
		// Update the index
		newIndex := currentIndex + 1
		if newIndex >= totalSentences {
			newIndex = totalSentences - 1
		}
		
		return ttsNextMsg{
			sentenceIndex: newIndex,
			err:           nil,
		}
	}
}

// prevSentenceCmd moves to the previous sentence
func prevSentenceCmd(controller *tts.Controller, currentIndex int) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return ttsPrevMsg{
				sentenceIndex: currentIndex,
				err:           fmt.Errorf("TTS controller not initialized"),
			}
		}
		
		// Use the controller's Previous method to navigate
		err := controller.Previous()
		if err != nil {
			// Check if we're at the beginning - this is informational, not an error
			if strings.Contains(err.Error(), "beginning of queue") {
				// Just return current index, no error - user is at the beginning
				return ttsPrevMsg{
					sentenceIndex: 0,
					err:           nil, // Don't show error for boundary
				}
			}
			return ttsPrevMsg{
				sentenceIndex: currentIndex,
				err:           err,
			}
		}
		
		// Update the index
		newIndex := currentIndex - 1
		if newIndex < 0 {
			newIndex = 0
		}
		
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

	// Debug log the render call (verbose - normally disabled)
	// log.Debug("RenderStatus called", 
	// 	"isInitializing", t.isInitializing, 
	// 	"isInitialized", t.isInitialized)

	var parts []string

	// Engine indicator (styled like Glow logo)
	engineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("39")).  // Blue background
		Foreground(lipgloss.Color("15")).  // White text
		Padding(0, 1).                     // Add padding like Glow logo
		Bold(true)
	engineText := fmt.Sprintf("TTS: %s", strings.ToUpper(t.engine))
	
	// Add (Not Ready) only if not initialized and not initializing
	if !t.isInitialized && !t.isInitializing {
		engineText += " (Not Ready)"
	}
	
	parts = append(parts, engineStyle.Render(engineText))

	// Playback state or spinner (only show if initialized or loading)
	if t.isInitialized || t.isInitializing || t.isSynthesizing || t.isBuffering {
		stateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Width(3).  // Fixed width of 3 chars
			Align(lipgloss.Center)  // Center align within the width
		
		var stateIcon string
		// Show spinner during any loading state
		if t.isInitializing || t.isSynthesizing || t.isBuffering {
			stateIcon = t.loadingSpinner.View()
		} else if t.isPlaying {
			stateIcon = "▶"
		} else if t.isPaused {
			stateIcon = "⏸"
		} else {
			stateIcon = "■"
		}
		parts = append(parts, stateStyle.Render(stateIcon))
	}

	// Speed indicator (show during init and after)
	if t.isInitialized || t.isInitializing || t.isSynthesizing || t.isBuffering {
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

	// Loading status text (show after speed when loading)
	if t.isInitializing || t.isSynthesizing || t.isBuffering {
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("247")).
			Italic(true)
		
		var statusText string
		if t.isInitializing {
			statusText = "Initializing..."
		} else if t.isSynthesizing {
			statusText = "Synthesizing..."
		} else if t.isBuffering {
			statusText = "Buffering..."
		}
		
		if t.loadingMessage != "" && t.loadingMessage != "Initializing TTS engine" && t.loadingMessage != "Synthesizing audio..." {
			// Use custom message if it's different from defaults
			statusText = t.loadingMessage
		}
		
		parts = append(parts, statusStyle.Render(statusText))
	} else if t.isPlaying && !t.isPaused {
		// Show timer when playing
		timerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("247"))
		
		elapsed := time.Since(t.playbackStart)
		timerText := fmt.Sprintf("%02d:%02d", int(elapsed.Minutes()), int(elapsed.Seconds())%60)
		parts = append(parts, timerStyle.Render(timerText))
	}

	// Sentence position (only show when not loading and not playing)
	if t.totalSentences > 0 && !t.isInitializing && !t.isSynthesizing && !t.isBuffering && (!t.isPlaying || t.isPaused) {
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