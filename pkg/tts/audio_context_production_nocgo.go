//go:build nocgo
// +build nocgo

package tts

import (
	"errors"
	"io"
	"time"
)

// Stub implementations for static analysis and builds without CGO

// ProductionAudioContext stub for nocgo builds
type ProductionAudioContext struct {
	ready bool
}

// NewProductionAudioContext creates a stub production audio context
func NewProductionAudioContext() (*ProductionAudioContext, error) {
	return nil, errors.New("audio not available in nocgo build")
}

// NewProductionAudioContextWithRetry creates a stub production audio context with retry logic
func NewProductionAudioContextWithRetry(platform *PlatformInfo) (*ProductionAudioContext, error) {
	return nil, errors.New("audio not available in nocgo build")
}

func (pac *ProductionAudioContext) NewPlayer(r io.Reader) (AudioPlayerInterface, error) {
	return nil, errors.New("audio not available in nocgo build")
}

func (pac *ProductionAudioContext) Close() error {
	return nil
}

func (pac *ProductionAudioContext) IsReady() bool {
	return false
}

func (pac *ProductionAudioContext) SampleRate() int {
	return SampleRate
}

func (pac *ProductionAudioContext) ChannelCount() int {
	return Channels
}

// ProductionAudioPlayer stub for nocgo builds
type ProductionAudioPlayer struct{}

func (pap *ProductionAudioPlayer) Play() {}

func (pap *ProductionAudioPlayer) Pause() {}

func (pap *ProductionAudioPlayer) IsPlaying() bool {
	return false
}

func (pap *ProductionAudioPlayer) Reset() error {
	return nil
}

func (pap *ProductionAudioPlayer) Close() error {
	return nil
}

func (pap *ProductionAudioPlayer) SetVolume(volume float64) {}

func (pap *ProductionAudioPlayer) Volume() float64 {
	return 1.0
}

func (pap *ProductionAudioPlayer) Seek(offset int64, whence int) (int64, error) {
	return 0, errors.New("audio not available in nocgo build")
}

func (pap *ProductionAudioPlayer) BufferedDuration() time.Duration {
	return 0
}