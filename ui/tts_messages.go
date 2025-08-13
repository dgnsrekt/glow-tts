package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glow/v2/internal/audio"
	"github.com/charmbracelet/glow/v2/internal/cache"
	"github.com/charmbracelet/glow/v2/internal/queue"
	"github.com/charmbracelet/glow/v2/internal/tts"
	"github.com/charmbracelet/glow/v2/internal/ttypes"
)

// audioCacheWrapper adapts cache.MemoryCache to ttypes.AudioCache interface
type audioCacheWrapper struct {
	cache *cache.MemoryCache
}

func (w *audioCacheWrapper) Get(key string) ([]byte, bool) {
	return w.cache.Get(key)
}

func (w *audioCacheWrapper) Put(key string, audio []byte) error {
	return w.cache.Put(key, audio)
}

func (w *audioCacheWrapper) Delete(key string) error {
	return w.cache.Delete(key)
}

func (w *audioCacheWrapper) Clear() error {
	return w.cache.Clear()
}

func (w *audioCacheWrapper) Size() int64 {
	return w.cache.Size()
}

func (w *audioCacheWrapper) Stats() ttypes.CacheStats {
	stats := w.cache.Stats()
	return ttypes.CacheStats{
		Hits:      stats.Hits,
		Misses:    stats.Misses,
		Evictions: stats.Evictions,
		Size:      stats.Size,
	}
}

// TTS Message Types for Bubble Tea Command pattern

// TTSInitMsg is sent when TTS controller initialization starts
type TTSInitMsg struct{}

// TTSInitDoneMsg is sent when TTS controller initialization completes
type TTSInitDoneMsg struct {
	Controller *tts.TTSController
	Err        error
}

// TTSPlayMsg is sent to start TTS playback
type TTSPlayMsg struct {
	Content string
}

// TTSPlayDoneMsg is sent when TTS playback command completes
type TTSPlayDoneMsg struct {
	Err error
}

// TTSPauseMsg is sent to pause TTS playback
type TTSPauseMsg struct{}

// TTSPauseDoneMsg is sent when TTS pause command completes
type TTSPauseDoneMsg struct {
	Err error
}

// TTSStopMsg is sent to stop TTS playback
type TTSStopMsg struct{}

// TTSStopDoneMsg is sent when TTS stop command completes
type TTSStopDoneMsg struct {
	Err error
}

// TTSNextMsg is sent to move to next sentence
type TTSNextMsg struct{}

// TTSNextDoneMsg is sent when next sentence command completes
type TTSNextDoneMsg struct {
	Err error
}

// TTSPrevMsg is sent to move to previous sentence
type TTSPrevMsg struct{}

// TTSPrevDoneMsg is sent when previous sentence command completes
type TTSPrevDoneMsg struct {
	Err error
}

// TTSSpeedMsg is sent to change TTS speed
type TTSSpeedMsg struct {
	Speed float64
}

// TTSSpeedDoneMsg is sent when speed change command completes
type TTSSpeedDoneMsg struct {
	Err error
}

// TTSStatusMsg is sent for TTS status updates
type TTSStatusMsg struct {
	State    string
	Current  int
	Total    int
	Progress float64
	Error    string
}

// TTSErrorMsg is sent when TTS encounters an error
type TTSErrorMsg struct {
	Err error
}

// TTS Commands (these return tea.Cmd functions)

// initTTSCmd initializes the TTS controller
func initTTSCmd(engine string, cfg Config) tea.Cmd {
	return func() tea.Msg {
		// Convert string to EngineType
		var engineType ttypes.EngineType
		switch engine {
		case "piper":
			engineType = ttypes.EnginePiper
		case "gtts", "google":
			engineType = ttypes.EngineGoogle
		default:
			return TTSInitDoneMsg{Controller: nil, Err: errors.New("invalid engine: " + engine)}
		}

		// Determine cache directory
		cacheDir := cfg.TTSCacheDir
		if cacheDir == "" {
			// Use default if not configured
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return TTSInitDoneMsg{Controller: nil, Err: err}
			}
			cacheDir = filepath.Join(homeDir, ".cache", "glow-tts")
		}

		// Create TTS configuration
		config := &tts.Config{
			Engine:       engineType,
			CacheDir:     cacheDir,
			MaxCacheSize: int64(cfg.TTSMaxCacheSize) * 1024 * 1024, // Convert MB to bytes
		}

		// Create components - use configured cache size
		memoryCacheSize := int64(cfg.TTSMaxCacheSize) * 1024 * 1024 // Convert MB to bytes
		memoryCache := cache.NewMemoryCache(memoryCacheSize)
		audioCache := &audioCacheWrapper{cache: memoryCache}

		player, err := audio.NewPlayer(audio.PlayerConfig{
			SampleRate: 44100, // OTO requires 44100 or 48000 Hz
			Channels:   1,
			BitDepth:   16,
			BufferSize: 4096,
		})
		if err != nil {
			return TTSInitDoneMsg{Controller: nil, Err: err}
		}

		audioQueue := queue.NewAudioQueue(50, 3, 10*1024*1024) // 10MB limit

		// Create controller
		controller, err := tts.NewController(config, player, audioQueue, audioCache)
		if err != nil {
			return TTSInitDoneMsg{Controller: nil, Err: err}
		}

		// Start controller
		ctx := context.Background()
		err = controller.Start(ctx, engineType)
		if err != nil {
			return TTSInitDoneMsg{Controller: nil, Err: err}
		}

		return TTSInitDoneMsg{Controller: controller, Err: nil}
	}
}

// ttsPlayCmd starts TTS playback using Commands
func ttsPlayCmd(controller *tts.TTSController, content string) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSPlayDoneMsg{Err: nil} // No-op if no controller
		}

		// First process the document, then play
		err := controller.ProcessDocument(content)
		if err != nil {
			return TTSPlayDoneMsg{Err: err}
		}

		err = controller.Play()
		return TTSPlayDoneMsg{Err: err}
	}
}

// ttsPauseCmd pauses TTS playback
func ttsPauseCmd(controller *tts.TTSController) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSPauseDoneMsg{Err: nil} // No-op if no controller
		}

		err := controller.Pause()
		return TTSPauseDoneMsg{Err: err}
	}
}

// ttsStopCmd stops TTS playback
func ttsStopCmd(controller *tts.TTSController) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSStopDoneMsg{Err: nil} // No-op if no controller
		}

		err := controller.Stop()
		return TTSStopDoneMsg{Err: err}
	}
}

// ttsNextCmd moves to next sentence
func ttsNextCmd(controller *tts.TTSController) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSNextDoneMsg{Err: nil} // No-op if no controller
		}

		err := controller.NextSentence()
		return TTSNextDoneMsg{Err: err}
	}
}

// ttsPrevCmd moves to previous sentence
func ttsPrevCmd(controller *tts.TTSController) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSPrevDoneMsg{Err: nil} // No-op if no controller
		}

		err := controller.PreviousSentence()
		return TTSPrevDoneMsg{Err: err}
	}
}

// ttsSetSpeedCmd changes TTS speed
func ttsSetSpeedCmd(controller *tts.TTSController, speed float64) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSSpeedDoneMsg{Err: nil} // No-op if no controller
		}

		err := controller.SetSpeed(speed)
		return TTSSpeedDoneMsg{Err: err}
	}
}

// ttsStatusCmd polls TTS status (for periodic updates)
func ttsStatusCmd(controller *tts.TTSController) tea.Cmd {
	return func() tea.Msg {
		if controller == nil {
			return TTSStatusMsg{State: "inactive"}
		}

		state := controller.GetState()
		progress := controller.GetProgress()
		controllerErr := controller.GetError()

		errorMsg := ""
		if controllerErr != nil {
			errorMsg = controllerErr.Error()
		}

		// Calculate progress percentage
		var progressPercent float64
		if progress.TotalSentences > 0 {
			progressPercent = float64(progress.CurrentSentence) / float64(progress.TotalSentences)
		}

		return TTSStatusMsg{
			State:    state.String(),
			Current:  progress.CurrentSentence,
			Total:    progress.TotalSentences,
			Progress: progressPercent,
			Error:    errorMsg,
		}
	}
}
