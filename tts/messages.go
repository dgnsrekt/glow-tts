package tts

import tea "github.com/charmbracelet/bubbletea"

// Messages for Bubble Tea communication between TTS and UI.

// PlayingMsg indicates TTS playback has started.
type PlayingMsg struct {
	Sentence int // Current sentence index
	Total    int // Total number of sentences
}

// PausedMsg indicates TTS playback has been paused.
type PausedMsg struct{}

// StoppedMsg indicates TTS playback has stopped.
type StoppedMsg struct{}

// SentenceChangedMsg indicates the current sentence has changed.
type SentenceChangedMsg struct {
	Index int    // New sentence index
	Text  string // Sentence text
}

// TTSStateChangedMsg indicates the TTS state has changed.
type TTSStateChangedMsg struct {
	State    StateType
	Sentence int
	Total    int
}

// TTSErrorMsg indicates an error occurred in the TTS system.
type TTSErrorMsg struct {
	Error       error
	Recoverable bool
}

// AudioGeneratedMsg indicates audio has been generated for a sentence.
type AudioGeneratedMsg struct {
	Index int
	Audio *Audio
}

// TTSEnabledMsg indicates TTS has been enabled.
type TTSEnabledMsg struct{}

// TTSDisabledMsg indicates TTS has been disabled.
type TTSDisabledMsg struct{}

// Commands for async TTS operations.

// GenerateAudioCmd creates a command to generate audio for text.
func GenerateAudioCmd(engine Engine, text string, index int) tea.Cmd {
	return func() tea.Msg {
		audio, err := engine.GenerateAudio(text)
		if err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
			}
		}
		return AudioGeneratedMsg{
			Index: index,
			Audio: audio,
		}
	}
}

// PlayAudioCmd creates a command to play audio.
func PlayAudioCmd(player AudioPlayer, audio *Audio) tea.Cmd {
	return func() tea.Msg {
		if err := player.Play(audio); err != nil {
			return TTSErrorMsg{
				Error:       err,
				Recoverable: true,
			}
		}
		return PlayingMsg{}
	}
}