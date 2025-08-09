package tts

import "time"

// StateType represents the current state of the TTS system.
type StateType int

const (
	// StateIdle indicates TTS is not active.
	StateIdle StateType = iota
	// StateInitializing indicates TTS is starting up.
	StateInitializing
	// StateReady indicates TTS is ready to play.
	StateReady
	// StatePlaying indicates TTS is actively playing audio.
	StatePlaying
	// StatePaused indicates TTS playback is paused.
	StatePaused
	// StateStopping indicates TTS is shutting down.
	StateStopping
	// StateError indicates TTS encountered an error.
	StateError
)

// String returns the string representation of the state.
func (s StateType) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateStopping:
		return "stopping"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// State holds the complete TTS system state.
type State struct {
	CurrentState   StateType     // Current state of the system
	Sentence       int           // Current sentence index (0-based)
	TotalSentences int           // Total number of sentences
	Position       time.Duration // Position within current sentence
	Duration       time.Duration // Duration of current sentence
	LastError      error         // Last error encountered
}

// IsActive returns true if TTS is in an active state.
func (s *State) IsActive() bool {
	return s.CurrentState == StatePlaying || s.CurrentState == StatePaused
}

// CanPlay returns true if TTS can start or resume playback.
func (s *State) CanPlay() bool {
	return s.CurrentState == StateReady || s.CurrentState == StatePaused
}

// CanPause returns true if TTS can be paused.
func (s *State) CanPause() bool {
	return s.CurrentState == StatePlaying
}

// CanStop returns true if TTS can be stopped.
func (s *State) CanStop() bool {
	return s.CurrentState != StateIdle && s.CurrentState != StateStopping
}

// StateMachine manages state transitions for the TTS system.
type StateMachine struct {
	current     StateType
	transitions map[StateType][]StateType
	onEnter     map[StateType]func()
	onExit      map[StateType]func()
}

// NewStateMachine creates a new state machine with valid transitions.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: StateIdle,
		transitions: map[StateType][]StateType{
			StateIdle:         {StateInitializing},
			StateInitializing: {StateReady, StateError},
			StateReady:        {StatePlaying, StateIdle},
			StatePlaying:      {StatePaused, StateStopping, StateReady},
			StatePaused:       {StatePlaying, StateStopping},
			StateStopping:     {StateIdle},
			StateError:        {StateIdle, StateInitializing},
		},
		onEnter: make(map[StateType]func()),
		onExit:  make(map[StateType]func()),
	}
}

// Transition attempts to transition to the specified state.
func (sm *StateMachine) Transition(to StateType) bool {
	// Check if transition is valid
	validTransitions, ok := sm.transitions[sm.current]
	if !ok {
		return false
	}

	valid := false
	for _, state := range validTransitions {
		if state == to {
			valid = true
			break
		}
	}

	if !valid {
		return false
	}

	// Execute exit callback for current state
	if exitFn, ok := sm.onExit[sm.current]; ok && exitFn != nil {
		exitFn()
	}

	// Transition to new state
	sm.current = to

	// Execute enter callback for new state
	if enterFn, ok := sm.onEnter[to]; ok && enterFn != nil {
		enterFn()
	}

	return true
}

// Current returns the current state.
func (sm *StateMachine) Current() StateType {
	return sm.current
}

// OnEnter registers a callback for entering a state.
func (sm *StateMachine) OnEnter(state StateType, fn func()) {
	sm.onEnter[state] = fn
}

// OnExit registers a callback for exiting a state.
func (sm *StateMachine) OnExit(state StateType, fn func()) {
	sm.onExit[state] = fn
}