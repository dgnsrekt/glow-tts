package tts

import (
	"fmt"
	"math"
	"sync"
)

// Available speed presets
var (
	DefaultSpeedSteps = []float64{0.5, 0.75, 1.0, 1.25, 1.5, 2.0}
	DefaultSpeed      = 1.0
	MinSpeed          = 0.5
	MaxSpeed          = 2.0
)

// EngineType identifies different TTS engines
type EngineType string

const (
	EngineTypePiper  EngineType = "piper"
	EngineTypeGoogle EngineType = "google"
)

// EngineParameter represents an engine-specific speed parameter
type EngineParameter struct {
	Name  string
	Value interface{}
}

// TTSSpeedController manages TTS playback speed with discrete steps
type TTSSpeedController struct {
	mu              sync.RWMutex
	currentSpeed    float64
	availableSpeeds []float64
	currentIndex    int
}

// NewSpeedController creates a new speed controller with default settings
func NewSpeedController() *TTSSpeedController {
	return NewSpeedControllerWithSteps(DefaultSpeedSteps)
}

// NewSpeedControllerWithSteps creates a speed controller with custom speed steps
func NewSpeedControllerWithSteps(speeds []float64) *TTSSpeedController {
	if len(speeds) == 0 {
		speeds = DefaultSpeedSteps
	}
	
	// Find index of default speed
	defaultIndex := 0
	for i, speed := range speeds {
		if math.Abs(speed-DefaultSpeed) < 0.001 {
			defaultIndex = i
			break
		}
	}
	
	return &TTSSpeedController{
		currentSpeed:    DefaultSpeed,
		availableSpeeds: speeds,
		currentIndex:    defaultIndex,
	}
}

// GetSpeed returns the current speed setting
func (sc *TTSSpeedController) GetSpeed() float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.currentSpeed
}

// SetSpeed sets the speed to the nearest valid discrete value
func (sc *TTSSpeedController) SetSpeed(speed float64) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	// Validate bounds
	if speed < MinSpeed || speed > MaxSpeed {
		return fmt.Errorf("speed %.2f out of range [%.2f, %.2f]", speed, MinSpeed, MaxSpeed)
	}
	
	// Find nearest discrete speed
	nearestIndex := 0
	minDiff := math.MaxFloat64
	for i, availSpeed := range sc.availableSpeeds {
		diff := math.Abs(availSpeed - speed)
		if diff < minDiff {
			minDiff = diff
			nearestIndex = i
		}
	}
	
	sc.currentIndex = nearestIndex
	sc.currentSpeed = sc.availableSpeeds[nearestIndex]
	
	return nil
}

// NextSpeed increases to the next speed step
func (sc *TTSSpeedController) NextSpeed() (float64, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if sc.currentIndex >= len(sc.availableSpeeds)-1 {
		return sc.currentSpeed, fmt.Errorf("already at maximum speed")
	}
	
	sc.currentIndex++
	sc.currentSpeed = sc.availableSpeeds[sc.currentIndex]
	
	return sc.currentSpeed, nil
}

// PreviousSpeed decreases to the previous speed step
func (sc *TTSSpeedController) PreviousSpeed() (float64, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if sc.currentIndex <= 0 {
		return sc.currentSpeed, fmt.Errorf("already at minimum speed")
	}
	
	sc.currentIndex--
	sc.currentSpeed = sc.availableSpeeds[sc.currentIndex]
	
	return sc.currentSpeed, nil
}

// IsValidSpeed checks if a speed value is one of the available discrete steps
func (sc *TTSSpeedController) IsValidSpeed(speed float64) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	for _, availSpeed := range sc.availableSpeeds {
		if math.Abs(availSpeed-speed) < 0.001 {
			return true
		}
	}
	return false
}

// GetAvailableSpeeds returns all available speed steps
func (sc *TTSSpeedController) GetAvailableSpeeds() []float64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	speeds := make([]float64, len(sc.availableSpeeds))
	copy(speeds, sc.availableSpeeds)
	return speeds
}

// GetEngineParameter returns engine-specific parameter for the current speed
func (sc *TTSSpeedController) GetEngineParameter(engineType EngineType, speed float64) EngineParameter {
	switch engineType {
	case EngineTypePiper:
		// Piper uses --length-scale which has inverse relationship with speed
		// speed 2.0 = length-scale 0.5 (faster = shorter)
		// speed 0.5 = length-scale 2.0 (slower = longer)
		lengthScale := 1.0 / speed
		return EngineParameter{
			Name:  "--length-scale",
			Value: lengthScale,
		}
		
	case EngineTypeGoogle:
		// Google TTS will use ffmpeg atempo filter
		// Direct relationship: speed 2.0 = atempo 2.0
		// Note: atempo has range limits (0.5 to 2.0 per filter)
		// For extreme speeds, may need to chain filters
		return EngineParameter{
			Name:  "atempo",
			Value: speed,
		}
		
	default:
		// Default to direct speed value
		return EngineParameter{
			Name:  "speed",
			Value: speed,
		}
	}
}

// GetPiperLengthScale returns the Piper-specific length-scale parameter
func (sc *TTSSpeedController) GetPiperLengthScale() float64 {
	speed := sc.GetSpeed()
	return 1.0 / speed
}

// GetGoogleAtempo returns the ffmpeg atempo parameter for Google TTS
func (sc *TTSSpeedController) GetGoogleAtempo() float64 {
	return sc.GetSpeed()
}

// FormatSpeed returns a formatted string representation of the speed
func (sc *TTSSpeedController) FormatSpeed() string {
	speed := sc.GetSpeed()
	return fmt.Sprintf("%.2fx", speed)
}

// FormatSpeedCompact returns a compact speed representation
func (sc *TTSSpeedController) FormatSpeedCompact() string {
	speed := sc.GetSpeed()
	// Show only significant digits
	if speed == float64(int(speed)) {
		return fmt.Sprintf("%.0fx", speed)
	}
	return fmt.Sprintf("%.2gx", speed)
}

// Reset resets the speed to default
func (sc *TTSSpeedController) Reset() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	// Find default speed index
	for i, speed := range sc.availableSpeeds {
		if math.Abs(speed-DefaultSpeed) < 0.001 {
			sc.currentIndex = i
			sc.currentSpeed = DefaultSpeed
			break
		}
	}
}

// Clone creates a copy of the speed controller
func (sc *TTSSpeedController) Clone() *TTSSpeedController {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	speeds := make([]float64, len(sc.availableSpeeds))
	copy(speeds, sc.availableSpeeds)
	
	clone := &TTSSpeedController{
		currentSpeed:    sc.currentSpeed,
		availableSpeeds: speeds,
		currentIndex:    sc.currentIndex,
	}
	
	return clone
}

// IncreaseSpeed implements SpeedController interface
func (sc *TTSSpeedController) IncreaseSpeed() error {
	_, err := sc.NextSpeed()
	return err
}

// DecreaseSpeed implements SpeedController interface
func (sc *TTSSpeedController) DecreaseSpeed() error {
	_, err := sc.PreviousSpeed()
	return err
}

// GetSpeedSteps implements SpeedController interface
func (sc *TTSSpeedController) GetSpeedSteps() []float64 {
	return sc.GetAvailableSpeeds()
}