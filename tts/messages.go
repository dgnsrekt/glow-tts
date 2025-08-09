package tts

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages for Bubble Tea communication between TTS and UI.

// PlayingMsg indicates TTS playback has started.
type PlayingMsg struct {
	Sentence int           // Current sentence index
	Total    int           // Total number of sentences
	Duration time.Duration // Total duration of the current sentence
}

// PausedMsg indicates TTS playback has been paused.
type PausedMsg struct {
	Position time.Duration // Position when paused
	Sentence int           // Current sentence index
}

// ResumedMsg indicates TTS playback has resumed.
type ResumedMsg struct {
	Position time.Duration // Position when resumed
	Sentence int           // Current sentence index
}

// StoppedMsg indicates TTS playback has stopped.
type StoppedMsg struct {
	Reason string // Reason for stopping (user, complete, error)
}

// SentenceChangedMsg indicates the current sentence has changed.
type SentenceChangedMsg struct {
	Index    int           // New sentence index
	Text     string        // Sentence text
	Duration time.Duration // Estimated duration
	Progress float64       // Progress through all sentences (0.0 to 1.0)
}

// TTSStateChangedMsg indicates the TTS state has changed.
type TTSStateChangedMsg struct {
	State     StateType
	PrevState StateType // Previous state for transition context
	Sentence  int
	Total     int
	Timestamp time.Time // When the state change occurred
}

// TTSErrorMsg indicates an error occurred in the TTS system.
type TTSErrorMsg struct {
	Error       error
	Recoverable bool
	Component   string // Which component had the error (engine, player, parser, etc.)
	Action      string // What action was being performed
}

// AudioGeneratedMsg indicates audio has been generated for a sentence.
type AudioGeneratedMsg struct {
	Index    int
	Audio    *Audio
	Sentence string        // The text that was converted
	Duration time.Duration // Actual duration of generated audio
}

// TTSEnabledMsg indicates TTS has been enabled.
type TTSEnabledMsg struct {
	Engine string // Which engine was enabled
}

// TTSDisabledMsg indicates TTS has been disabled.
type TTSDisabledMsg struct {
	Reason string // Why TTS was disabled
}

// TTSInitializingMsg indicates TTS is initializing.
type TTSInitializingMsg struct {
	Engine string // Which engine is being initialized
	Steps  int    // Total initialization steps
	Step   int    // Current step
}

// TTSReadyMsg indicates TTS is ready to use.
type TTSReadyMsg struct {
	Engine       string // Which engine is ready
	VoiceCount   int    // Number of available voices
	SelectedVoice string // Currently selected voice
}

// PositionUpdateMsg provides playback position updates.
type PositionUpdateMsg struct {
	Position         time.Duration // Current position in the sentence
	Duration         time.Duration // Total duration of current sentence
	SentenceIndex    int           // Current sentence index
	SentenceProgress float64       // Progress in current sentence (0.0 to 1.0)
	TotalProgress    float64       // Progress through all sentences (0.0 to 1.0)
}

// BufferStatusMsg indicates audio buffer status.
type BufferStatusMsg struct {
	Buffered  int // Number of sentences buffered
	Capacity  int // Buffer capacity
	IsLoading bool // Whether actively loading more
}

// VoiceChangedMsg indicates the TTS voice has been changed.
type VoiceChangedMsg struct {
	Voice Voice // New voice configuration
}

// SpeedChangedMsg indicates the playback speed has changed.
type SpeedChangedMsg struct {
	Speed float32 // New speed multiplier
}

// VolumeChangedMsg indicates the volume has changed.
type VolumeChangedMsg struct {
	Volume float32 // New volume level (0.0 to 1.0)
}

// NavigationMsg indicates navigation to a different sentence.
type NavigationMsg struct {
	Target    int    // Target sentence index
	Direction string // "next", "previous", "absolute"
}

// Commands for async TTS operations.

// GenerateAudioCmd creates a command to generate audio for text.
func GenerateAudioCmd(engine Engine, text string, index int) tea.Cmd {
	return func() tea.Msg {
		audio, err := engine.GenerateAudio(text)
		if err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "engine",
				Action:      "generate_audio",
			}
		}
		return AudioGeneratedMsg{
			Index:    index,
			Audio:    audio,
			Sentence: text,
			Duration: audio.Duration,
		}
	}
}

// PlayAudioCmd creates a command to play audio.
func PlayAudioCmd(player AudioPlayer, audio *Audio, sentenceIndex int, totalSentences int) tea.Cmd {
	return func() tea.Msg {
		if err := player.Play(audio); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "player",
				Action:      "play_audio",
			}
		}
		return PlayingMsg{
			Sentence: sentenceIndex,
			Total:    totalSentences,
			Duration: audio.Duration,
		}
	}
}

// PauseAudioCmd creates a command to pause audio playback.
func PauseAudioCmd(player AudioPlayer, currentSentence int) tea.Cmd {
	return func() tea.Msg {
		position := player.GetPosition()
		if err := player.Pause(); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "player",
				Action:      "pause_audio",
			}
		}
		return PausedMsg{
			Position: position,
			Sentence: currentSentence,
		}
	}
}

// ResumeAudioCmd creates a command to resume audio playback.
func ResumeAudioCmd(player AudioPlayer, currentSentence int) tea.Cmd {
	return func() tea.Msg {
		position := player.GetPosition()
		if err := player.Resume(); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "player",
				Action:      "resume_audio",
			}
		}
		return ResumedMsg{
			Position: position,
			Sentence: currentSentence,
		}
	}
}

// StopAudioCmd creates a command to stop audio playback.
func StopAudioCmd(player AudioPlayer, reason string) tea.Cmd {
	return func() tea.Msg {
		if err := player.Stop(); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "player",
				Action:      "stop_audio",
			}
		}
		return StoppedMsg{
			Reason: reason,
		}
	}
}

// InitializeTTSCmd creates a command to initialize the TTS system.
func InitializeTTSCmd(engine Engine, config EngineConfig) tea.Cmd {
	return func() tea.Msg {
		// Send initializing message
		if err := engine.Initialize(config); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: false,
				Component:   "engine",
				Action:      "initialize",
			}
		}
		
		// Check if available
		if !engine.IsAvailable() {
			return TTSErrorMsg{
				Error:       ErrEngineNotAvailable,
				Recoverable: false,
				Component:   "engine",
				Action:      "check_availability",
			}
		}
		
		voices := engine.GetVoices()
		selectedVoice := ""
		if len(voices) > 0 {
			selectedVoice = voices[0].Name
		}
		
		return TTSReadyMsg{
			Engine:        "active",
			VoiceCount:    len(voices),
			SelectedVoice: selectedVoice,
		}
	}
}

// MonitorPositionCmd creates a command that periodically updates playback position.
func MonitorPositionCmd(player AudioPlayer, interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		if !player.IsPlaying() {
			return nil
		}
		
		position := player.GetPosition()
		// Note: Additional context would be needed from controller
		// This is a simplified version
		return PositionUpdateMsg{
			Position: position,
		}
	})
}

// NavigateToSentenceCmd creates a command to navigate to a specific sentence.
func NavigateToSentenceCmd(targetIndex int, direction string) tea.Cmd {
	return func() tea.Msg {
		return NavigationMsg{
			Target:    targetIndex,
			Direction: direction,
		}
	}
}

// ChangeVoiceCmd creates a command to change the TTS voice.
func ChangeVoiceCmd(engine Engine, voice Voice) tea.Cmd {
	return func() tea.Msg {
		if err := engine.SetVoice(voice); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
				Component:   "engine",
				Action:      "set_voice",
			}
		}
		return VoiceChangedMsg{
			Voice: voice,
		}
	}
}

// ChangeSpeedCmd creates a command to change playback speed.
func ChangeSpeedCmd(speed float32) tea.Cmd {
	return func() tea.Msg {
		// Note: Actual speed change would need to be implemented
		// in the audio player or engine
		return SpeedChangedMsg{
			Speed: speed,
		}
	}
}

// ChangeVolumeCmd creates a command to change volume.
func ChangeVolumeCmd(volume float32) tea.Cmd {
	return func() tea.Msg {
		// Note: Actual volume change would need to be implemented
		// in the audio player
		return VolumeChangedMsg{
			Volume: volume,
		}
	}
}

// BatchGenerateAudioCmd creates a command to generate audio for multiple sentences.
func BatchGenerateAudioCmd(engine Engine, sentences []Sentence, startIndex int) tea.Cmd {
	return func() tea.Msg {
		for _, sentence := range sentences {
			audio, err := engine.GenerateAudio(sentence.Text)
			if err != nil {
				// Return error but indicate we can continue with other sentences
				return TTSErrorMsg{
					Error:       err,
					Recoverable: true,
					Component:   "engine",
					Action:      "batch_generate",
				}
			}
			
			// In a real implementation, we'd send each audio to a buffer
			// For now, we'll just generate them
			_ = audio
			
			// Could send progress updates here
		}
		
		return BufferStatusMsg{
			Buffered: len(sentences),
			Capacity: len(sentences),
			IsLoading: false,
		}
	}
}