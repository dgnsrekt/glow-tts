package tts

import (
	"math"
	"testing"
)

func TestNewSpeedController(t *testing.T) {
	sc := NewSpeedController()
	
	if sc == nil {
		t.Fatal("Expected SpeedController to be created")
	}
	
	if sc.GetSpeed() != DefaultSpeed {
		t.Errorf("Expected default speed %.2f, got %.2f", DefaultSpeed, sc.GetSpeed())
	}
	
	speeds := sc.GetAvailableSpeeds()
	if len(speeds) != len(DefaultSpeedSteps) {
		t.Errorf("Expected %d speed steps, got %d", len(DefaultSpeedSteps), len(speeds))
	}
}

func TestSpeedControllerSetSpeed(t *testing.T) {
	sc := NewSpeedController()
	
	tests := []struct {
		name      string
		speed     float64
		expected  float64
		wantError bool
	}{
		{"exact match 0.5", 0.5, 0.5, false},
		{"exact match 1.0", 1.0, 1.0, false},
		{"exact match 2.0", 2.0, 2.0, false},
		{"nearest to 0.6", 0.6, 0.5, false},
		{"nearest to 0.8", 0.8, 0.75, false},
		{"nearest to 1.3", 1.3, 1.25, false},
		{"out of range low", 0.3, 1.0, true},
		{"out of range high", 3.0, 1.0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.SetSpeed(tt.speed)
			if (err != nil) != tt.wantError {
				t.Errorf("SetSpeed() error = %v, wantError %v", err, tt.wantError)
			}
			
			if !tt.wantError && math.Abs(sc.GetSpeed()-tt.expected) > 0.001 {
				t.Errorf("Expected speed %.2f, got %.2f", tt.expected, sc.GetSpeed())
			}
		})
	}
}

func TestSpeedControllerNavigation(t *testing.T) {
	sc := NewSpeedController()
	
	// Start at default (1.0)
	if sc.GetSpeed() != 1.0 {
		t.Errorf("Expected starting speed 1.0, got %.2f", sc.GetSpeed())
	}
	
	// Increase speed
	newSpeed, err := sc.NextSpeed()
	if err != nil {
		t.Fatalf("NextSpeed failed: %v", err)
	}
	if newSpeed != 1.25 {
		t.Errorf("Expected speed 1.25 after increase, got %.2f", newSpeed)
	}
	
	// Increase again
	newSpeed, err = sc.NextSpeed()
	if err != nil {
		t.Fatalf("NextSpeed failed: %v", err)
	}
	if newSpeed != 1.5 {
		t.Errorf("Expected speed 1.5 after second increase, got %.2f", newSpeed)
	}
	
	// Decrease speed
	newSpeed, err = sc.PreviousSpeed()
	if err != nil {
		t.Fatalf("PreviousSpeed failed: %v", err)
	}
	if newSpeed != 1.25 {
		t.Errorf("Expected speed 1.25 after decrease, got %.2f", newSpeed)
	}
	
	// Test bounds
	sc.SetSpeed(2.0)
	_, err = sc.NextSpeed()
	if err == nil {
		t.Error("Expected error at maximum speed")
	}
	
	sc.SetSpeed(0.5)
	_, err = sc.PreviousSpeed()
	if err == nil {
		t.Error("Expected error at minimum speed")
	}
}

func TestSpeedControllerValidation(t *testing.T) {
	sc := NewSpeedController()
	
	tests := []struct {
		speed float64
		valid bool
	}{
		{0.5, true},
		{0.75, true},
		{1.0, true},
		{1.25, true},
		{1.5, true},
		{2.0, true},
		{0.6, false},
		{1.1, false},
		{1.75, false},
	}
	
	for _, tt := range tests {
		if sc.IsValidSpeed(tt.speed) != tt.valid {
			t.Errorf("IsValidSpeed(%.2f) = %v, expected %v", tt.speed, !tt.valid, tt.valid)
		}
	}
}

func TestEngineParameterMapping(t *testing.T) {
	sc := NewSpeedController()
	
	tests := []struct {
		engineType     EngineType
		speed          float64
		expectedName   string
		expectedValue  float64
	}{
		// Piper has inverse relationship
		{EngineTypePiper, 2.0, "--length-scale", 0.5},
		{EngineTypePiper, 1.0, "--length-scale", 1.0},
		{EngineTypePiper, 0.5, "--length-scale", 2.0},
		
		// Google has direct relationship
		{EngineTypeGoogle, 2.0, "atempo", 2.0},
		{EngineTypeGoogle, 1.0, "atempo", 1.0},
		{EngineTypeGoogle, 0.5, "atempo", 0.5},
	}
	
	for _, tt := range tests {
		param := sc.GetEngineParameter(tt.engineType, tt.speed)
		if param.Name != tt.expectedName {
			t.Errorf("Engine %s: expected parameter name %s, got %s", 
				tt.engineType, tt.expectedName, param.Name)
		}
		
		value, ok := param.Value.(float64)
		if !ok {
			t.Errorf("Expected float64 value, got %T", param.Value)
			continue
		}
		
		if math.Abs(value-tt.expectedValue) > 0.001 {
			t.Errorf("Engine %s at speed %.2f: expected value %.2f, got %.2f",
				tt.engineType, tt.speed, tt.expectedValue, value)
		}
	}
}

func TestSpeedControllerHelpers(t *testing.T) {
	sc := NewSpeedController()
	
	t.Run("GetPiperLengthScale", func(t *testing.T) {
		sc.SetSpeed(2.0)
		if ls := sc.GetPiperLengthScale(); math.Abs(ls-0.5) > 0.001 {
			t.Errorf("Expected length scale 0.5 for speed 2.0, got %.2f", ls)
		}
		
		sc.SetSpeed(0.5)
		if ls := sc.GetPiperLengthScale(); math.Abs(ls-2.0) > 0.001 {
			t.Errorf("Expected length scale 2.0 for speed 0.5, got %.2f", ls)
		}
	})
	
	t.Run("GetGoogleAtempo", func(t *testing.T) {
		sc.SetSpeed(1.5)
		if at := sc.GetGoogleAtempo(); math.Abs(at-1.5) > 0.001 {
			t.Errorf("Expected atempo 1.5, got %.2f", at)
		}
	})
}

func TestSpeedControllerFormatting(t *testing.T) {
	sc := NewSpeedController()
	
	tests := []struct {
		speed          float64
		expectedFormat string
		expectedCompact string
	}{
		{1.0, "1.00x", "1x"},
		{1.25, "1.25x", "1.2x"},
		{1.5, "1.50x", "1.5x"},
		{2.0, "2.00x", "2x"},
		{0.75, "0.75x", "0.75x"},
	}
	
	for _, tt := range tests {
		sc.SetSpeed(tt.speed)
		
		formatted := sc.FormatSpeed()
		if formatted != tt.expectedFormat {
			t.Errorf("FormatSpeed() for %.2f = %s, expected %s", 
				tt.speed, formatted, tt.expectedFormat)
		}
		
		compact := sc.FormatSpeedCompact()
		if compact != tt.expectedCompact {
			t.Errorf("FormatSpeedCompact() for %.2f = %s, expected %s",
				tt.speed, compact, tt.expectedCompact)
		}
	}
}

func TestSpeedControllerReset(t *testing.T) {
	sc := NewSpeedController()
	
	// Change speed
	sc.SetSpeed(1.5)
	if sc.GetSpeed() != 1.5 {
		t.Errorf("Failed to set speed to 1.5")
	}
	
	// Reset
	sc.Reset()
	if sc.GetSpeed() != DefaultSpeed {
		t.Errorf("Expected speed to reset to %.2f, got %.2f", DefaultSpeed, sc.GetSpeed())
	}
}

func TestSpeedControllerClone(t *testing.T) {
	sc := NewSpeedController()
	sc.SetSpeed(1.5)
	
	clone := sc.Clone()
	
	// Check clone has same state
	if clone.GetSpeed() != sc.GetSpeed() {
		t.Errorf("Clone has different speed: %.2f vs %.2f", clone.GetSpeed(), sc.GetSpeed())
	}
	
	// Verify independence
	clone.SetSpeed(2.0)
	if sc.GetSpeed() == 2.0 {
		t.Error("Original controller was affected by clone modification")
	}
}

func TestCustomSpeedSteps(t *testing.T) {
	customSteps := []float64{0.25, 0.5, 1.0, 1.5, 3.0}
	sc := NewSpeedControllerWithSteps(customSteps)
	
	speeds := sc.GetAvailableSpeeds()
	if len(speeds) != len(customSteps) {
		t.Errorf("Expected %d custom steps, got %d", len(customSteps), len(speeds))
	}
	
	// Verify custom steps are used
	for i, expected := range customSteps {
		if math.Abs(speeds[i]-expected) > 0.001 {
			t.Errorf("Custom step %d: expected %.2f, got %.2f", i, expected, speeds[i])
		}
	}
	
	// Test navigation with custom steps
	sc.SetSpeed(1.0)
	newSpeed, _ := sc.NextSpeed()
	if newSpeed != 1.5 {
		t.Errorf("Expected next speed 1.5 with custom steps, got %.2f", newSpeed)
	}
}

func BenchmarkSpeedControllerGetSpeed(b *testing.B) {
	sc := NewSpeedController()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sc.GetSpeed()
	}
}

func BenchmarkSpeedControllerSetSpeed(b *testing.B) {
	sc := NewSpeedController()
	speeds := []float64{0.5, 1.0, 1.5, 2.0}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sc.SetSpeed(speeds[i%len(speeds)])
	}
}

func BenchmarkEngineParameterMapping(b *testing.B) {
	sc := NewSpeedController()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sc.GetEngineParameter(EngineTypePiper, 1.5)
	}
}