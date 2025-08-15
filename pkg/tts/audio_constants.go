package tts

// Audio format constants for TTS
// These constants are used by both CGO and non-CGO builds
const (
	// SampleRate is the audio sample rate in Hz (22050Hz for TTS)
	SampleRate = 22050
	// Channels is the number of audio channels (1 = mono)
	Channels = 1
	// BitDepth is the bit depth per sample (16-bit)
	BitDepth = 16
	// BytesPerSample is the number of bytes per sample
	BytesPerSample = BitDepth / 8
)

// PlaybackState represents the current state of audio playback
type PlaybackState int

const (
	// PlaybackStopped indicates no audio is playing
	PlaybackStopped PlaybackState = iota
	// PlaybackPlaying indicates audio is currently playing
	PlaybackPlaying
	// PlaybackPaused indicates audio is paused
	PlaybackPaused
)