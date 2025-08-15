package tts

import (
	"errors"
	"testing"
	"time"
)

// Mock implementations for testing

type mockEngine struct {
	name        string
	available   bool
	validateErr error
	synthErr    error
	synthData   []byte
}

func (m *mockEngine) Synthesize(text string, speed float64) ([]byte, error) {
	if m.synthErr != nil {
		return nil, m.synthErr
	}
	return m.synthData, nil
}

func (m *mockEngine) SetSpeed(speed float64) error {
	if speed < 0.5 || speed > 2.0 {
		return errors.New("speed out of range")
	}
	return nil
}

func (m *mockEngine) Validate() error {
	return m.validateErr
}

func (m *mockEngine) GetName() string {
	return m.name
}

func (m *mockEngine) IsAvailable() bool {
	return m.available
}

type mockParser struct {
	sentences []Sentence
	parseErr  error
}

func (m *mockParser) ParseSentences(text string) ([]Sentence, error) {
	if m.parseErr != nil {
		return nil, m.parseErr
	}
	return m.sentences, nil
}

type mockCache struct {
	data     map[string][]byte
	getErr   error
	setErr   error
	clearErr error
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string][]byte),
	}
}

func (m *mockCache) Get(key string) ([]byte, bool) {
	if m.getErr != nil {
		return nil, false
	}
	data, ok := m.data[key]
	return data, ok
}

func (m *mockCache) Set(key string, data []byte) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.data[key] = data
	return nil
}

func (m *mockCache) Clear() error {
	if m.clearErr != nil {
		return m.clearErr
	}
	m.data = make(map[string][]byte)
	return nil
}

func (m *mockCache) GenerateKey(text string, voice string, speed float64) string {
	return text + voice + string(rune(speed))
}

type mockSpeedController struct {
	speed    float64
	setErr   error
	steps    []float64
}

func newMockSpeedController() *mockSpeedController {
	return &mockSpeedController{
		speed: 1.0,
		steps: []float64{0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0},
	}
}

func (m *mockSpeedController) GetSpeed() float64 {
	return m.speed
}

func (m *mockSpeedController) SetSpeed(speed float64) error {
	if m.setErr != nil {
		return m.setErr
	}
	if speed < 0.5 || speed > 2.0 {
		return errors.New("speed out of range")
	}
	m.speed = speed
	return nil
}

func (m *mockSpeedController) IncreaseSpeed() error {
	for i, s := range m.steps {
		if s == m.speed && i < len(m.steps)-1 {
			m.speed = m.steps[i+1]
			return nil
		}
	}
	return errors.New("already at maximum speed")
}

func (m *mockSpeedController) DecreaseSpeed() error {
	for i, s := range m.steps {
		if s == m.speed && i > 0 {
			m.speed = m.steps[i-1]
			return nil
		}
	}
	return errors.New("already at minimum speed")
}

func (m *mockSpeedController) GetSpeedSteps() []float64 {
	return m.steps
}

// Controller tests

func TestNewController(t *testing.T) {
	cfg := ControllerConfig{
		Engine:             "piper",
		EnableCache:        true,
		LookaheadSentences: 5,
		DefaultSpeed:       1.5,
	}

	controller, err := NewController(cfg)
	if err != nil {
		t.Fatalf("NewController failed: %v", err)
	}

	if controller == nil {
		t.Fatal("NewController returned nil")
	}

	if controller.config.LookaheadSentences != 5 {
		t.Errorf("Expected lookahead sentences 5, got %d", controller.config.LookaheadSentences)
	}

	if controller.config.DefaultSpeed != 1.5 {
		t.Errorf("Expected default speed 1.5, got %f", controller.config.DefaultSpeed)
	}

	if controller.GetState() != StateUninitialized {
		t.Errorf("Expected state Uninitialized, got %s", controller.GetState())
	}
}

func TestControllerStateTransitions(t *testing.T) {
	// Test state string representations
	states := []ControllerState{
		StateUninitialized,
		StateReady,
		StateRunning,
		StateStopping,
		StateStopped,
	}

	expectedStrings := []string{
		"uninitialized",
		"ready",
		"running",
		"stopping",
		"stopped",
	}

	for i, state := range states {
		if state.String() != expectedStrings[i] {
			t.Errorf("State %d: expected '%s', got '%s'", state, expectedStrings[i], state.String())
		}
	}
}

func TestControllerInitialize(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*Controller)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful initialization",
			setupFunc: func(c *Controller) {
				c.SetEngine(&mockEngine{name: "test", available: true})
				c.SetParser(&mockParser{})
				c.SetCache(newMockCache())
				c.SetSpeedController(newMockSpeedController())
			},
			expectError: false,
		},
		{
			name: "missing engine",
			setupFunc: func(c *Controller) {
				c.SetParser(&mockParser{})
				c.SetCache(newMockCache())
				c.SetSpeedController(newMockSpeedController())
			},
			expectError: true,
			errorMsg:    "TTS engine not set",
		},
		{
			name: "engine validation failure",
			setupFunc: func(c *Controller) {
				c.SetEngine(&mockEngine{
					name:        "test",
					validateErr: errors.New("validation failed"),
				})
				c.SetParser(&mockParser{})
				c.SetCache(newMockCache())
				c.SetSpeedController(newMockSpeedController())
			},
			expectError: true,
			errorMsg:    "engine validation failed",
		},
		{
			name: "missing parser",
			setupFunc: func(c *Controller) {
				c.SetEngine(&mockEngine{name: "test", available: true})
				c.SetCache(newMockCache())
				c.SetSpeedController(newMockSpeedController())
			},
			expectError: true,
			errorMsg:    "text parser not set",
		},
		{
			name: "missing cache when enabled",
			setupFunc: func(c *Controller) {
				c.config.EnableCache = true
				c.SetEngine(&mockEngine{name: "test", available: true})
				c.SetParser(&mockParser{})
				c.SetSpeedController(newMockSpeedController())
			},
			expectError: true,
			errorMsg:    "cache enabled but cache manager not set",
		},
		{
			name: "missing speed controller",
			setupFunc: func(c *Controller) {
				c.SetEngine(&mockEngine{name: "test", available: true})
				c.SetParser(&mockParser{})
			},
			expectError: true,
			errorMsg:    "speed controller not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ControllerConfig{}
			controller, _ := NewController(cfg)
			tt.setupFunc(controller)

			err := controller.Initialize()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if controller.GetState() != StateReady {
					t.Errorf("Expected state Ready, got %s", controller.GetState())
				}
			}
		})
	}
}

func TestControllerStartStop(t *testing.T) {
	cfg := ControllerConfig{}
	controller, _ := NewController(cfg)

	// Setup mock components
	controller.SetEngine(&mockEngine{name: "test", available: true})
	controller.SetParser(&mockParser{})
	controller.SetCache(newMockCache())
	controller.SetSpeedController(newMockSpeedController())

	// Try to start without initialization
	err := controller.Start(context.TODO())
	if err == nil {
		t.Error("Expected error starting uninitialized controller")
	}

	// Initialize
	if err := controller.Initialize(); err != nil {
		t.Fatalf("Initialization failed: %v", err)
	}

	// Start controller
	if err := controller.Start(context.TODO()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !controller.IsRunning() {
		t.Error("Controller should be running")
	}

	// Try to start again
	err = controller.Start(context.TODO())
	if err == nil {
		t.Error("Expected error starting already running controller")
	}

	// Stop controller
	if err := controller.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if controller.GetState() != StateStopped {
		t.Errorf("Expected state Stopped, got %s", controller.GetState())
	}

	// Stop again should be idempotent
	if err := controller.Stop(); err != nil {
		t.Error("Second stop should not error")
	}
}

func TestControllerWaitForReady(t *testing.T) {
	cfg := ControllerConfig{}
	controller, _ := NewController(cfg)

	// Setup mock components
	controller.SetEngine(&mockEngine{name: "test", available: true})
	controller.SetParser(&mockParser{})
	controller.SetCache(newMockCache())
	controller.SetSpeedController(newMockSpeedController())

	// Test timeout when not ready
	err := controller.WaitForReady(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Initialize in background
	go func() {
		time.Sleep(50 * time.Millisecond)
		controller.Initialize()
	}()

	// Wait for ready
	err = controller.WaitForReady(200 * time.Millisecond)
	if err != nil {
		t.Errorf("WaitForReady failed: %v", err)
	}

	if controller.GetState() != StateReady {
		t.Errorf("Expected state Ready, got %s", controller.GetState())
	}
}

func TestControllerSetters(t *testing.T) {
	cfg := ControllerConfig{}
	controller, _ := NewController(cfg)

	// Test setting engine
	engine := &mockEngine{name: "test"}
	if err := controller.SetEngine(engine); err != nil {
		t.Errorf("SetEngine failed: %v", err)
	}

	// Test setting parser
	parser := &mockParser{}
	if err := controller.SetParser(parser); err != nil {
		t.Errorf("SetParser failed: %v", err)
	}

	// Test setting cache
	cache := newMockCache()
	if err := controller.SetCache(cache); err != nil {
		t.Errorf("SetCache failed: %v", err)
	}

	// Test setting speed controller
	speedCtrl := newMockSpeedController()
	if err := controller.SetSpeedController(speedCtrl); err != nil {
		t.Errorf("SetSpeedController failed: %v", err)
	}

	// Initialize and start
	controller.Initialize()
	controller.Start(context.TODO())

	// Test that setters fail when running
	if err := controller.SetEngine(engine); err == nil {
		t.Error("Expected error setting engine while running")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && s[:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || (len(substr) > 0 && len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}