//go:build nocgo
// +build nocgo

package tts

import (
	"errors"
	"sync"
	"time"
)

// Stub implementations for static analysis and builds without CGO

// AudioContext manages the global audio context (stub for nocgo)
type AudioContext struct {
	mu    sync.Mutex
	ready bool
}

var (
	globalAudioContext *AudioContext
	globalAudioPlayer  *TTSAudioPlayer
	audioContextOnce   sync.Once
	playerOnce         sync.Once
)

// GetAudioContext returns a stub audio context
func GetAudioContext() (*AudioContext, error) {
	return nil, errors.New("audio not available in nocgo build")
}

// GetGlobalAudioPlayer returns a stub audio player
func GetGlobalAudioPlayer() *TTSAudioPlayer {
	return &TTSAudioPlayer{}
}

func (ac *AudioContext) Suspend() error {
	return errors.New("audio not available in nocgo build")
}

func (ac *AudioContext) Resume() error {
	return errors.New("audio not available in nocgo build")
}

// AudioStream stub for nocgo builds
type AudioStream struct {
	mu       sync.RWMutex
	state    PlaybackState
	duration time.Duration
}

func NewAudioStream(pcmData []byte) (*AudioStream, error) {
	return nil, errors.New("audio not available in nocgo build")
}

func (as *AudioStream) Play() error {
	return errors.New("audio not available in nocgo build")
}

func (as *AudioStream) Pause() error {
	return errors.New("audio not available in nocgo build")
}

func (as *AudioStream) Stop() error {
	return errors.New("audio not available in nocgo build")
}

func (as *AudioStream) GetState() PlaybackState {
	return PlaybackStopped
}

func (as *AudioStream) GetPosition() time.Duration {
	return 0
}

func (as *AudioStream) GetDuration() time.Duration {
	return 0
}

func (as *AudioStream) IsPlaying() bool {
	return false
}

func (as *AudioStream) Close() error {
	return nil
}

// TTSAudioPlayer stub for nocgo builds
type TTSAudioPlayer struct {
	mu sync.Mutex
}

func NewTTSAudioPlayer() *TTSAudioPlayer {
	return &TTSAudioPlayer{}
}

func (ap *TTSAudioPlayer) PlayPCM(pcmData []byte) error {
	return errors.New("audio not available in nocgo build")
}

func (ap *TTSAudioPlayer) Stop() error {
	return errors.New("audio not available in nocgo build")
}

func (ap *TTSAudioPlayer) Pause() error {
	return errors.New("audio not available in nocgo build")
}

func (ap *TTSAudioPlayer) Resume() error {
	return errors.New("audio not available in nocgo build")
}

func (ap *TTSAudioPlayer) GetState() PlaybackState {
	return PlaybackStopped
}

func (ap *TTSAudioPlayer) Close() error {
	return nil
}