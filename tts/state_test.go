package tts

import (
	"errors"
	"testing"
	"time"
)

// TestStateTypeString tests the String() method for StateType.
func TestStateTypeString(t *testing.T) {
	tests := []struct {
		state    StateType
		expected string
	}{
		{StateIdle, "idle"},
		{StateInitializing, "initializing"},
		{StateReady, "ready"},
		{StatePlaying, "playing"},
		{StatePaused, "paused"},
		{StateStopping, "stopping"},
		{StateError, "error"},
		{StateType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("StateType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStateIsActive tests the IsActive() method.
func TestStateIsActive(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{
			name:     "playing is active",
			state:    State{CurrentState: StatePlaying},
			expected: true,
		},
		{
			name:     "paused is active",
			state:    State{CurrentState: StatePaused},
			expected: true,
		},
		{
			name:     "idle is not active",
			state:    State{CurrentState: StateIdle},
			expected: false,
		},
		{
			name:     "ready is not active",
			state:    State{CurrentState: StateReady},
			expected: false,
		},
		{
			name:     "error is not active",
			state:    State{CurrentState: StateError},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.state.IsActive(); result != tt.expected {
				t.Errorf("State.IsActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStateCanPlay tests the CanPlay() method.
func TestStateCanPlay(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{
			name:     "can play from ready",
			state:    State{CurrentState: StateReady},
			expected: true,
		},
		{
			name:     "can play from paused",
			state:    State{CurrentState: StatePaused},
			expected: true,
		},
		{
			name:     "cannot play from idle",
			state:    State{CurrentState: StateIdle},
			expected: false,
		},
		{
			name:     "cannot play from playing",
			state:    State{CurrentState: StatePlaying},
			expected: false,
		},
		{
			name:     "cannot play from error",
			state:    State{CurrentState: StateError},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.state.CanPlay(); result != tt.expected {
				t.Errorf("State.CanPlay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStateCanPause tests the CanPause() method.
func TestStateCanPause(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{
			name:     "can pause from playing",
			state:    State{CurrentState: StatePlaying},
			expected: true,
		},
		{
			name:     "cannot pause from paused",
			state:    State{CurrentState: StatePaused},
			expected: false,
		},
		{
			name:     "cannot pause from idle",
			state:    State{CurrentState: StateIdle},
			expected: false,
		},
		{
			name:     "cannot pause from ready",
			state:    State{CurrentState: StateReady},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.state.CanPause(); result != tt.expected {
				t.Errorf("State.CanPause() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStateCanStop tests the CanStop() method.
func TestStateCanStop(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected bool
	}{
		{
			name:     "can stop from playing",
			state:    State{CurrentState: StatePlaying},
			expected: true,
		},
		{
			name:     "can stop from paused",
			state:    State{CurrentState: StatePaused},
			expected: true,
		},
		{
			name:     "can stop from ready",
			state:    State{CurrentState: StateReady},
			expected: true,
		},
		{
			name:     "can stop from error",
			state:    State{CurrentState: StateError},
			expected: true,
		},
		{
			name:     "cannot stop from idle",
			state:    State{CurrentState: StateIdle},
			expected: false,
		},
		{
			name:     "cannot stop from stopping",
			state:    State{CurrentState: StateStopping},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.state.CanStop(); result != tt.expected {
				t.Errorf("State.CanStop() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStateWithData tests State struct with actual data.
func TestStateWithData(t *testing.T) {
	state := State{
		CurrentState:   StatePlaying,
		Sentence:       5,
		TotalSentences: 10,
		Position:       3 * time.Second,
		Duration:       10 * time.Second,
		LastError:      errors.New("test error"),
	}

	if !state.IsActive() {
		t.Error("Playing state should be active")
	}

	if state.CanPlay() {
		t.Error("Cannot play when already playing")
	}

	if !state.CanPause() {
		t.Error("Should be able to pause when playing")
	}

	if !state.CanStop() {
		t.Error("Should be able to stop when playing")
	}

	if state.Sentence != 5 {
		t.Errorf("Sentence = %d, want 5", state.Sentence)
	}

	if state.TotalSentences != 10 {
		t.Errorf("TotalSentences = %d, want 10", state.TotalSentences)
	}

	if state.Position != 3*time.Second {
		t.Errorf("Position = %v, want 3s", state.Position)
	}

	if state.Duration != 10*time.Second {
		t.Errorf("Duration = %v, want 10s", state.Duration)
	}

	if state.LastError == nil || state.LastError.Error() != "test error" {
		t.Errorf("LastError = %v, want 'test error'", state.LastError)
	}
}

// TestNewStateMachine tests state machine creation.
func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()
	
	if sm == nil {
		t.Fatal("Expected non-nil state machine")
	}

	if sm.Current() != StateIdle {
		t.Errorf("Initial state = %v, want StateIdle", sm.Current())
	}

	if sm.transitions == nil {
		t.Error("Transitions map should be initialized")
	}

	if sm.onEnter == nil {
		t.Error("OnEnter map should be initialized")
	}

	if sm.onExit == nil {
		t.Error("OnExit map should be initialized")
	}
}

// TestStateMachineTransitions tests valid state transitions.
func TestStateMachineTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from        StateType
		to          StateType
		shouldAllow bool
	}{
		// Valid transitions
		{"idle to initializing", StateIdle, StateInitializing, true},
		{"initializing to ready", StateInitializing, StateReady, true},
		{"initializing to error", StateInitializing, StateError, true},
		{"ready to playing", StateReady, StatePlaying, true},
		{"ready to idle", StateReady, StateIdle, true},
		{"playing to paused", StatePlaying, StatePaused, true},
		{"playing to stopping", StatePlaying, StateStopping, true},
		{"playing to ready", StatePlaying, StateReady, true},
		{"paused to playing", StatePaused, StatePlaying, true},
		{"paused to stopping", StatePaused, StateStopping, true},
		{"stopping to idle", StateStopping, StateIdle, true},
		{"error to idle", StateError, StateIdle, true},
		{"error to initializing", StateError, StateInitializing, true},
		
		// Invalid transitions
		{"idle to playing", StateIdle, StatePlaying, false},
		{"idle to paused", StateIdle, StatePaused, false},
		{"playing to idle", StatePlaying, StateIdle, false},
		{"paused to ready", StatePaused, StateReady, false},
		{"ready to error", StateReady, StateError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()
			
			// Set initial state
			sm.current = tt.from
			
			result := sm.Transition(tt.to)
			if result != tt.shouldAllow {
				t.Errorf("Transition from %v to %v: got %v, want %v",
					tt.from, tt.to, result, tt.shouldAllow)
			}

			// Check state changed only if transition was valid
			if tt.shouldAllow && sm.Current() != tt.to {
				t.Errorf("State not changed: current = %v, expected = %v",
					sm.Current(), tt.to)
			} else if !tt.shouldAllow && sm.Current() != tt.from {
				t.Errorf("State changed on invalid transition: current = %v, expected = %v",
					sm.Current(), tt.from)
			}
		})
	}
}

// TestStateMachineCallbacks tests state enter/exit callbacks.
func TestStateMachineCallbacks(t *testing.T) {
	sm := NewStateMachine()
	
	var enterCalled, exitCalled bool
	var enterState, exitState StateType

	// Register callbacks
	sm.OnEnter(StateInitializing, func() {
		enterCalled = true
		enterState = StateInitializing
	})

	sm.OnExit(StateIdle, func() {
		exitCalled = true
		exitState = StateIdle
	})

	// Perform transition
	result := sm.Transition(StateInitializing)
	if !result {
		t.Fatal("Transition should have succeeded")
	}

	// Check callbacks were called
	if !exitCalled {
		t.Error("Exit callback not called")
	}
	if exitState != StateIdle {
		t.Errorf("Exit callback called for wrong state: %v", exitState)
	}

	if !enterCalled {
		t.Error("Enter callback not called")
	}
	if enterState != StateInitializing {
		t.Errorf("Enter callback called for wrong state: %v", enterState)
	}
}

// TestStateMachineMultipleCallbacks tests multiple callbacks.
func TestStateMachineMultipleCallbacks(t *testing.T) {
	sm := NewStateMachine()
	
	callOrder := []string{}

	// Register multiple callbacks
	sm.OnExit(StateIdle, func() {
		callOrder = append(callOrder, "exit-idle")
	})

	sm.OnEnter(StateInitializing, func() {
		callOrder = append(callOrder, "enter-initializing")
	})

	sm.OnExit(StateInitializing, func() {
		callOrder = append(callOrder, "exit-initializing")
	})

	sm.OnEnter(StateReady, func() {
		callOrder = append(callOrder, "enter-ready")
	})

	// Perform transitions
	sm.Transition(StateInitializing)
	sm.Transition(StateReady)

	// Check call order
	expectedOrder := []string{
		"exit-idle",
		"enter-initializing",
		"exit-initializing",
		"enter-ready",
	}

	if len(callOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d callbacks, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if callOrder[i] != expected {
			t.Errorf("Callback %d: got %s, want %s", i, callOrder[i], expected)
		}
	}
}

// TestStateMachineInvalidTransition tests invalid state transition.
func TestStateMachineInvalidTransition(t *testing.T) {
	sm := NewStateMachine()
	
	// Try invalid transition from idle to playing
	result := sm.Transition(StatePlaying)
	if result {
		t.Error("Should not allow transition from Idle to Playing")
	}

	if sm.Current() != StateIdle {
		t.Errorf("State should remain Idle, got %v", sm.Current())
	}
}

// TestStateMachineSequentialTransitions tests a sequence of transitions.
func TestStateMachineSequentialTransitions(t *testing.T) {
	sm := NewStateMachine()
	
	// Full lifecycle: Idle -> Initializing -> Ready -> Playing -> Paused -> Playing -> Stopping -> Idle
	transitions := []struct {
		to       StateType
		expected bool
	}{
		{StateInitializing, true},
		{StateReady, true},
		{StatePlaying, true},
		{StatePaused, true},
		{StatePlaying, true},
		{StateStopping, true},
		{StateIdle, true},
	}

	for i, trans := range transitions {
		result := sm.Transition(trans.to)
		if result != trans.expected {
			t.Errorf("Transition %d to %v: got %v, want %v",
				i, trans.to, result, trans.expected)
		}
		if trans.expected && sm.Current() != trans.to {
			t.Errorf("After transition %d: state = %v, want %v",
				i, sm.Current(), trans.to)
		}
	}
}

// TestStateMachineErrorRecovery tests error state recovery.
func TestStateMachineErrorRecovery(t *testing.T) {
	sm := NewStateMachine()
	
	// Move to error state
	sm.current = StateError

	// Should be able to recover to idle
	if !sm.Transition(StateIdle) {
		t.Error("Should be able to transition from Error to Idle")
	}

	// Move back to error
	sm.current = StateError

	// Should be able to restart initialization
	if !sm.Transition(StateInitializing) {
		t.Error("Should be able to transition from Error to Initializing")
	}
}

// TestStateMachineNilCallbacks tests that nil callbacks don't crash.
func TestStateMachineNilCallbacks(t *testing.T) {
	sm := NewStateMachine()
	
	// Register nil callbacks (should not panic)
	sm.OnEnter(StateReady, nil)
	sm.OnExit(StateIdle, nil)

	// This should not panic
	result := sm.Transition(StateInitializing)
	if !result {
		t.Error("Transition should succeed even with nil callbacks")
	}
}