package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/tts"
	"github.com/charmbracelet/glow/v2/tts/audio"
	"github.com/charmbracelet/glow/v2/tts/engines"
	"github.com/charmbracelet/glow/v2/tts/engines/mock"
	"github.com/charmbracelet/glow/v2/tts/engines/piper"
	"github.com/charmbracelet/glow/v2/tts/sentence"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

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
	// Setup debug logging to file
	debugFile, err := os.OpenFile("/tmp/glow_tts_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(debugFile)
		log.Println("=== TTS Debug Session Started ===")
		debugFile.Sync()
	}
	
	return &TTSController{
		controller:      nil,
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
		log.Printf("[DEBUG TTS] TTS enabled via message")
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
	if tc == nil {
		return nil
	}

	switch key {
	case "T", "t":
		// Toggle TTS on/off
		tc.enabled = !tc.enabled
		// Debug: log the state change
		if tc.enabled {
			// Enable TTS - create controller if needed
			var engine tts.Engine
			var engineName string
			
			if tc.controller == nil {
				// Initialize with real Piper TTS
				
				// Try to create Piper engine first
				homeDir, _ := os.UserHomeDir()
				
				// Try to find Piper in multiple locations
				piperBinary := ""
				possiblePaths := []string{
					"piper", // In PATH
					filepath.Join(homeDir, ".local", "bin", "piper"),
					filepath.Join(homeDir, "bin", "piper"),
					"/usr/local/bin/piper",
					"/usr/bin/piper",
				}
				
				for _, path := range possiblePaths {
					if _, err := os.Stat(path); err == nil {
						piperBinary = path
						break
					}
				}
				
				if piperBinary == "" {
					// Try with exec.LookPath
					if path, err := exec.LookPath("piper"); err == nil {
						piperBinary = path
					}
				}
				
				log.Printf("[DEBUG TTS] Piper binary path: %s", piperBinary)
				
				piperModel := filepath.Join(homeDir, "piper-voices", "en_US-amy-medium.onnx")
				piperConfigPath := filepath.Join(homeDir, "piper-voices", "en_US-amy-medium.onnx.json")
				
				log.Printf("[DEBUG TTS] Piper model path: %s (exists: %v)", piperModel, fileExists(piperModel))
				
				piperConfig := piper.Config{
					BinaryPath:          piperBinary,
					ModelPath:           piperModel,
					ConfigPath:          piperConfigPath,
					OutputRaw:           true, // Critical for SimpleEngine
					SampleRate:          22050,
					MaxRestarts:         5, // Triggers V2 with better stability
					HealthCheckInterval: 5 * time.Second,
					RequestTimeout:      30 * time.Second,
				}
				
				piperEngine, err := piper.NewEngine(piperConfig)
				if err != nil {
					// Fallback to mock if Piper setup fails completely
					log.Printf("[WARNING TTS] Piper setup failed: %v, using mock engine only", err)
					engine = mock.New()
					engineName = "mock (Piper unavailable)"
				} else {
					// Wrap Piper with automatic fallback to mock
					log.Printf("[INFO TTS] Piper engine created, wrapping with fallback capability")
					mockEngine := mock.New()
					engine = engines.NewFallbackEngine(piperEngine, mockEngine, 3)
					engineName = "piper (with mock fallback)"
					log.Printf("[INFO TTS] FallbackEngine created successfully")
				}
				
				// Initialize audio player
				var player tts.AudioPlayer
				realPlayer, err := audio.NewPlayer()
				if err != nil {
					// Fallback to mock player if real one fails
					log.Printf("[DEBUG TTS] Real audio player failed: %v, using mock\n", err)
					player = audio.NewMockPlayer()
				} else {
					log.Printf("[DEBUG TTS] Real audio player initialized successfully\n")
					player = realPlayer
				}
				
				parser := sentence.NewParser()
				tc.controller = tts.NewController(engine, player, parser)
				
				// Initialize the engine with appropriate config
				var config tts.EngineConfig
				// Check for FallbackEngine
				if _, isFallback := engine.(*engines.FallbackEngine); isFallback {
					log.Printf("[DEBUG TTS] Initializing FallbackEngine with config")
					config = tts.EngineConfig{
						Voice:  "amy",  // Piper voice (fallback will handle mock internally)
						Rate:   1.0,
						Pitch:  1.0,
						Volume: 1.0,
					}
				} else if _, isPiper := engine.(*piper.Engine); isPiper {
					config = tts.EngineConfig{
						Voice:  "amy",  // Piper voice
						Rate:   1.0,
						Pitch:  1.0,
						Volume: 1.0,
					}
				} else {
					config = tts.EngineConfig{
						Voice:  "mock-voice-1",
						Rate:   1.0,
						Pitch:  1.0,
						Volume: 1.0,
					}
				}
				tc.controller.Initialize(config)
			}
			// Update status display immediately
			if tc.statusDisplay != nil {
				tc.statusDisplay.UpdateFromMessage(tts.TTSEnabledMsg{Engine: engineName})
			}
			
			// Log engine status if it's a fallback engine
			if fe, ok := engine.(*engines.FallbackEngine); ok {
				log.Printf("[INFO TTS] Engine status: %s", fe.GetStatus())
			}
			
			// Return a command that will trigger content loading
			return func() tea.Msg {
				log.Printf("[INFO TTS] TTS enabled with engine: %s", engineName)
				return tts.TTSEnabledMsg{Engine: engineName}
			}
		} else {
			// Disable TTS
			if tc.controller != nil {
				tc.controller.Stop()
				tc.controller.Shutdown()
				tc.controller = nil
			}
			// Update status display immediately
			if tc.statusDisplay != nil {
				tc.statusDisplay.UpdateFromMessage(tts.TTSDisabledMsg{Reason: "user"})
			}
			return func() tea.Msg {
				return tts.TTSDisabledMsg{Reason: "user"}
			}
		}

	case " ":
		// Play/pause
		if tc.enabled && tc.controller != nil {
			state := tc.controller.GetState()
			log.Printf("[DEBUG TTS] Space pressed, current state: %v\n", state.CurrentState)
			
			if state.CurrentState == tts.StatePlaying {
				log.Printf("[DEBUG TTS] Pausing TTS\n")
				tc.controller.Pause()
				return func() tea.Msg {
					return tts.PausedMsg{
						Position: state.Position,
						Sentence: tc.currentSentence,
					}
				}
			} else if state.CurrentState == tts.StatePaused || state.CurrentState == tts.StateReady {
				log.Printf("[DEBUG TTS] Starting/Resuming TTS\n")
				tc.controller.Play()
				return func() tea.Msg {
					return tts.PlayingMsg{
						Sentence: tc.currentSentence,
						Total:    tc.totalSentences,
					}
				}
			} else {
				log.Printf("[DEBUG TTS] State %v - no action taken\n", state.CurrentState)
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

// LoadContent loads markdown content into the TTS system.
func (tc *TTSController) LoadContent(content string) error {
	if tc == nil || tc.controller == nil || !tc.enabled {
		log.Printf("[DEBUG TTS] LoadContent skipped - enabled=%v, controller=%v", tc.enabled, tc.controller != nil)
		return nil // No error, just not enabled
	}
	
	log.Printf("[DEBUG TTS] Loading content (%d chars)", len(content))
	err := tc.controller.SetContent(content)
	if err != nil {
		log.Printf("[DEBUG TTS] SetContent failed: %v", err)
	} else {
		state := tc.controller.GetState()
		log.Printf("[DEBUG TTS] Content loaded, total sentences: %d", state.TotalSentences)
	}
	return err
}

// LoadContentIfEnabled loads content only if TTS is enabled and controller exists.
func (tc *TTSController) LoadContentIfEnabled(content string) {
	if tc != nil && tc.enabled && tc.controller != nil {
		log.Printf("[DEBUG TTS] LoadContentIfEnabled called")
		tc.LoadContent(content)
	} else {
		log.Printf("[DEBUG TTS] LoadContentIfEnabled skipped - enabled=%v, controller=%v", tc != nil && tc.enabled, tc != nil && tc.controller != nil)
	}
}

// GetContentForTTS gets current content for TTS from the controller.
func (tc *TTSController) GetContentForTTS() string {
	if tc == nil || tc.controller == nil {
		return ""
	}
	
	// For now, we'll return empty - content is stored internally in controller
	return ""
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

// GetTotalSentences returns the total number of sentences.
func (tc *TTSController) GetTotalSentences() int {
	if tc == nil || tc.controller == nil {
		return 0
	}
	return tc.controller.GetTotalSentences()
}