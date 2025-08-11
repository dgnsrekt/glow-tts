package tts

import (
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrSpeedOutOfRange is returned when speed is outside valid range
	ErrSpeedOutOfRange = errors.New("speed must be between 0.5 and 2.0")
)

// speedController manages playback speed adjustments with predefined steps.
type speedController struct {
	currentSpeed float64
	steps        []float64
	mu           sync.RWMutex
}

// NewSpeedController creates a new speed controller with default speed steps.
func NewSpeedController() SpeedController {
	return &speedController{
		currentSpeed: 1.0, // Normal speed
		steps: []float64{
			0.5,  // Half speed
			0.75, // Three-quarter speed
			1.0,  // Normal speed
			1.25, // Quarter faster
			1.5,  // Half faster
			1.75, // Three-quarter faster
			2.0,  // Double speed
		},
	}
}

// GetSpeed returns the current speed multiplier.
func (s *speedController) GetSpeed() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSpeed
}

// SetSpeed sets the speed multiplier (0.5 to 2.0).
func (s *speedController) SetSpeed(speed float64) error {
	if speed < 0.5 || speed > 2.0 {
		return ErrSpeedOutOfRange
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentSpeed = speed
	return nil
}

// Increase increments to the next speed step.
// Returns the new speed value.
func (s *speedController) Increase() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the next higher speed step
	for _, speed := range s.steps {
		if speed > s.currentSpeed {
			s.currentSpeed = speed
			return s.currentSpeed
		}
	}

	// Already at maximum speed
	return s.currentSpeed
}

// Decrease decrements to the previous speed step.
// Returns the new speed value.
func (s *speedController) Decrease() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the next lower speed step
	for i := len(s.steps) - 1; i >= 0; i-- {
		if s.steps[i] < s.currentSpeed {
			s.currentSpeed = s.steps[i]
			return s.currentSpeed
		}
	}

	// Already at minimum speed
	return s.currentSpeed
}

// ToPiperScale converts speed to Piper's length-scale parameter.
// Piper uses inverse scaling: faster speed = smaller length-scale.
func (s *speedController) ToPiperScale() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Piper's length-scale is inversely proportional to speed
	// Normal speed (1.0) -> length-scale 1.0
	// Faster speed (1.5) -> length-scale 0.67
	// Slower speed (0.75) -> length-scale 1.33
	scale := 1.0 / s.currentSpeed
	return fmt.Sprintf("%.2f", scale)
}

// ToGoogleRate converts speed to Google's speaking_rate parameter.
// Google TTS uses direct scaling: faster speed = higher speaking_rate.
func (s *speedController) ToGoogleRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Google's speaking_rate is directly proportional to speed
	// Normal speed (1.0) -> speaking_rate 1.0
	// Faster speed (1.5) -> speaking_rate 1.5
	// Slower speed (0.75) -> speaking_rate 0.75
	return s.currentSpeed
}

// ToGTTSSlow returns whether gTTS should use slow mode.
// gTTS only supports normal and slow speeds, so we consider
// anything below 0.8x as "slow".
func (s *speedController) ToGTTSSlow() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentSpeed < 0.8
}

// GetSpeedDisplay returns a human-readable speed description.
func (s *speedController) GetSpeedDisplay() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.currentSpeed {
	case 0.5:
		return "0.5x (Half Speed)"
	case 0.75:
		return "0.75x (Slow)"
	case 1.0:
		return "1.0x (Normal)"
	case 1.25:
		return "1.25x (Fast)"
	case 1.5:
		return "1.5x (Faster)"
	case 1.75:
		return "1.75x (Very Fast)"
	case 2.0:
		return "2.0x (Double Speed)"
	default:
		return fmt.Sprintf("%.2fx", s.currentSpeed)
	}
}

// IsAtMinimum returns true if speed is at the minimum value.
func (s *speedController) IsAtMinimum() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSpeed <= s.steps[0]
}

// IsAtMaximum returns true if speed is at the maximum value.
func (s *speedController) IsAtMaximum() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentSpeed >= s.steps[len(s.steps)-1]
}
