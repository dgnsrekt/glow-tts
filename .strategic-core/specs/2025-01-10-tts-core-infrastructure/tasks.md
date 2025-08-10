# Implementation Tasks: TTS Core Infrastructure

## Phase 1: Core Infrastructure (Setup)

### Task 1: Project Setup and Dependencies

**Type**: setup
**Priority**: high
**Estimated Hours**: 2

#### Pre-Implementation Checklist
- [ ] Review existing codebase structure
- [ ] Identify integration points
- [ ] Plan package organization
- [ ] Check Go version compatibility
- [ ] Review audio library options

#### Description
Set up the project structure for TTS functionality, add required dependencies, and create the basic package layout.

#### Acceptance Criteria
- [ ] Created `internal/tts/` package structure
- [ ] Added oto/v3 audio library dependency
- [ ] Created `internal/audio/` package
- [ ] Created `internal/queue/` package
- [ ] Created `internal/cache/` package
- [ ] Updated go.mod with new dependencies

#### Validation Steps
- [ ] Project compiles without errors
- [ ] Package imports resolve correctly
- [ ] No dependency conflicts
- [ ] Directory structure follows standards

#### Technical Notes
- Use `go get github.com/ebitengine/oto/v3` for audio
- Consider creating interfaces first for testability

---

### Task 2: Define Core Interfaces

**Type**: implementation
**Priority**: high
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Review interface design standards
- [ ] Consider all engine types
- [ ] Plan for extensibility
- [ ] Design for testability
- [ ] Consider error handling patterns

#### Description
Create the core interfaces that define contracts between TTS components.

#### Acceptance Criteria
- [ ] Created `TTSEngine` interface
- [ ] Created `AudioPlayer` interface
- [ ] Created `AudioCache` interface
- [ ] Created `SentenceQueue` interface
- [ ] Added comprehensive godoc comments
- [ ] Defined error types

#### Validation Steps
- [ ] Interfaces compile without errors
- [ ] Godoc comments are complete
- [ ] Error types are well-defined
- [ ] Interfaces follow Go conventions

#### Technical Notes
```go
// internal/tts/interfaces.go
type TTSEngine interface {
    Synthesize(ctx context.Context, text string) ([]byte, error)
    GetInfo() EngineInfo
    Close() error
}
```

---

## Phase 2: Component Implementation

### Task 3: Implement Sentence Parser

**Type**: implementation
**Priority**: high
**Estimated Hours**: 4

#### Pre-Implementation Checklist
- [ ] Review markdown structure in Glamour
- [ ] Understand sentence boundary detection
- [ ] Plan regex patterns
- [ ] Consider internationalization
- [ ] Design for performance

#### Description
Implement the sentence parser that extracts speakable text from markdown content.

#### Acceptance Criteria
- [ ] Strips markdown formatting correctly
- [ ] Preserves sentence boundaries
- [ ] Handles abbreviations properly
- [ ] Skips code blocks
- [ ] Maintains position mapping
- [ ] Handles edge cases (URLs, numbers)

#### Validation Steps
- [ ] Unit tests pass with >90% coverage
- [ ] Handles all markdown elements
- [ ] Performance benchmark acceptable
- [ ] No regex catastrophic backtracking

#### Technical Notes
- Use `github.com/charmbracelet/glamour` for markdown parsing
- Consider using `strings.Builder` for performance
- Handle Unicode properly

---

### Task 4: Implement Audio Queue

**Type**: implementation
**Priority**: high
**Estimated Hours**: 4

#### Pre-Implementation Checklist
- [ ] Review concurrency patterns
- [ ] Design queue data structure
- [ ] Plan priority handling
- [ ] Consider memory constraints
- [ ] Design for thread safety

#### Description
Implement the audio queue that manages sentence processing order and preprocessing.

#### Acceptance Criteria
- [ ] Thread-safe enqueue/dequeue operations
- [ ] Priority queue implementation
- [ ] Lookahead preprocessing (2-3 sentences)
- [ ] Memory-bounded queue
- [ ] Backpressure handling
- [ ] Clear statistics tracking

#### Validation Steps
- [ ] Concurrent access tests pass
- [ ] No race conditions detected
- [ ] Memory stays within bounds
- [ ] Performance benchmarks pass

#### Technical Notes
- Use channels for thread safety
- Implement ring buffer for efficiency
- Consider sync.Pool for buffer reuse

---

### Task 5: Implement Audio Cache

**Type**: implementation
**Priority**: medium
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Design cache key strategy
- [ ] Plan eviction policy
- [ ] Consider persistence options
- [ ] Design for concurrent access
- [ ] Plan memory management

#### Description
Implement LRU cache for synthesized audio with disk persistence.

#### Acceptance Criteria
- [ ] LRU eviction when size limit reached
- [ ] Thread-safe operations
- [ ] Persistent disk cache
- [ ] Configurable size limits
- [ ] Cache hit/miss statistics
- [ ] Compression for stored audio

#### Validation Steps
- [ ] Cache operations are atomic
- [ ] Eviction works correctly
- [ ] Persistence survives restart
- [ ] No memory leaks

#### Technical Notes
- Use SHA256 for cache keys
- Consider zstd for compression
- Implement with sync.Map or custom mutex

---

### Task 6: Implement Mock Audio Player

**Type**: implementation
**Priority**: medium
**Estimated Hours**: 2

#### Pre-Implementation Checklist
- [ ] Design player interface
- [ ] Plan state management
- [ ] Consider testing needs
- [ ] Design position tracking
- [ ] Plan event system

#### Description
Create a mock audio player for testing before implementing real audio.

#### Acceptance Criteria
- [ ] Implements AudioPlayer interface
- [ ] Simulates playback timing
- [ ] Tracks position accurately
- [ ] Handles state transitions
- [ ] Provides test callbacks

#### Validation Steps
- [ ] Interface compliance verified
- [ ] State transitions work correctly
- [ ] Position tracking is accurate
- [ ] Tests can use mock effectively

---

## Phase 3: Engine Integration

### Task 7: Implement Piper TTS Engine

**Type**: implementation
**Priority**: high
**Estimated Hours**: 6

#### Pre-Implementation Checklist
- [ ] Install Piper locally for testing
- [ ] Understand Piper CLI interface
- [ ] Plan process management
- [ ] Design error handling
- [ ] Consider model management

#### Description
Implement the Piper TTS engine wrapper that manages the Piper subprocess.

#### Acceptance Criteria
- [ ] Spawns Piper subprocess correctly
- [ ] Handles stdin/stdout communication
- [ ] Manages process lifecycle
- [ ] Implements error recovery
- [ ] Supports multiple voices
- [ ] Handles process crashes gracefully

#### Validation Steps
- [ ] Synthesis produces valid audio
- [ ] Process management is robust
- [ ] No zombie processes
- [ ] Resource cleanup works
- [ ] Integration tests pass

#### Technical Notes
- Use `exec.CommandContext` for process management
- Implement heartbeat for health checking
- Buffer I/O for efficiency

---

### Task 8: Implement Google TTS Engine

**Type**: implementation  
**Priority**: medium
**Estimated Hours**: 4

#### Pre-Implementation Checklist
- [ ] Set up Google Cloud account
- [ ] Understand API quotas/limits
- [ ] Plan authentication
- [ ] Design rate limiting
- [ ] Consider error handling

#### Description
Implement Google Text-to-Speech API integration with proper error handling.

#### Acceptance Criteria
- [ ] Authenticates with API correctly
- [ ] Synthesizes text successfully
- [ ] Implements rate limiting
- [ ] Handles API errors gracefully
- [ ] Supports voice selection
- [ ] Manages quotas properly

#### Validation Steps
- [ ] API calls succeed
- [ ] Rate limiting works
- [ ] Error handling is robust
- [ ] Voice selection works
- [ ] Integration tests pass

#### Technical Notes
- Use official Google Cloud Go SDK
- Implement exponential backoff
- Cache API responses

---

### Task 9: Implement Fallback Engine

**Type**: implementation
**Priority**: medium
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Design fallback strategy
- [ ] Plan health checking
- [ ] Consider switching logic
- [ ] Design configuration
- [ ] Plan logging strategy

#### Description
Create a fallback engine wrapper that automatically switches between engines on failure.

#### Acceptance Criteria
- [ ] Detects engine failures
- [ ] Switches to fallback seamlessly
- [ ] Maintains playback continuity
- [ ] Logs fallback events
- [ ] Configurable fallback order
- [ ] Health check mechanism

#### Validation Steps
- [ ] Fallback triggers correctly
- [ ] No audio interruption
- [ ] State maintained properly
- [ ] Logs are informative
- [ ] Tests cover failure scenarios

---

## Phase 4: Audio Implementation

### Task 10: Implement Real Audio Player

**Type**: implementation
**Priority**: high
**Estimated Hours**: 5

#### Pre-Implementation Checklist
- [ ] Test oto library setup
- [ ] Understand platform differences
- [ ] Plan buffer management
- [ ] Design streaming approach
- [ ] Consider latency requirements

#### Description
Implement cross-platform audio playback using the oto library.

#### Acceptance Criteria
- [ ] Plays audio on Linux/macOS/Windows
- [ ] Streams audio without gaps
- [ ] Controls volume correctly
- [ ] Tracks position accurately
- [ ] Handles device errors
- [ ] Supports speed adjustment

#### Validation Steps
- [ ] Audio plays without distortion
- [ ] No gaps between chunks
- [ ] Position tracking accurate
- [ ] Platform tests pass
- [ ] Performance acceptable

#### Technical Notes
- Initialize oto context once
- Use ring buffer for streaming
- Handle sample rate conversion

---

### Task 11: Implement TTS Controller

**Type**: implementation
**Priority**: high
**Estimated Hours**: 6

#### Pre-Implementation Checklist
- [ ] Review state machine design
- [ ] Plan component coordination
- [ ] Design command handling
- [ ] Consider error propagation
- [ ] Plan lifecycle management

#### Description
Implement the main TTS controller that orchestrates all components.

#### Acceptance Criteria
- [ ] Manages component lifecycle
- [ ] Coordinates data flow
- [ ] Handles state transitions
- [ ] Processes UI commands
- [ ] Implements error recovery
- [ ] Provides status updates

#### Validation Steps
- [ ] Components initialize correctly
- [ ] State transitions are valid
- [ ] Commands execute properly
- [ ] Errors handled gracefully
- [ ] Integration tests pass

---

## Phase 5: UI Integration

### Task 12: Add CLI Flag Support

**Type**: implementation
**Priority**: high
**Estimated Hours**: 2

#### Pre-Implementation Checklist
- [ ] Review Cobra flag patterns
- [ ] Plan configuration structure
- [ ] Consider validation rules
- [ ] Design help text
- [ ] Plan defaults

#### Description
Add `--tts [engine]` flag to CLI with proper validation and configuration.

#### Acceptance Criteria
- [ ] Flag parses correctly
- [ ] Validates engine choice
- [ ] Forces TUI mode
- [ ] Updates configuration
- [ ] Shows in help text
- [ ] Has sensible defaults

#### Validation Steps
- [ ] CLI accepts flag
- [ ] Validation works correctly
- [ ] TUI mode enforced
- [ ] Help text is clear
- [ ] Integration with config works

---

### Task 13: Integrate TTS with Bubble Tea UI

**Type**: implementation
**Priority**: high
**Estimated Hours**: 5

#### Pre-Implementation Checklist
- [ ] Review Bubble Tea patterns
- [ ] Plan message types
- [ ] Design UI components
- [ ] Consider state synchronization
- [ ] Plan keyboard shortcuts

#### Description
Integrate TTS controls into the Bubble Tea TUI with status display.

#### Acceptance Criteria
- [ ] TTS status bar displays
- [ ] Keyboard shortcuts work
- [ ] Navigation commands function
- [ ] Status updates in real-time
- [ ] Error messages display
- [ ] Progress indicator works

#### Validation Steps
- [ ] UI updates correctly
- [ ] No UI blocking
- [ ] Shortcuts are responsive
- [ ] Status accurate
- [ ] Error handling works

#### Technical Notes
- Create new Bubble Tea messages
- Update pager model
- Add TTS-specific key bindings

---

### Task 14: Add TTS Configuration Support

**Type**: implementation
**Priority**: medium
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Review Viper configuration
- [ ] Design config schema
- [ ] Plan defaults
- [ ] Consider validation
- [ ] Design migration strategy

#### Description
Add TTS configuration to Glow's config file with validation.

#### Acceptance Criteria
- [ ] Config schema defined
- [ ] Viper integration works
- [ ] Environment variables supported
- [ ] Validation implemented
- [ ] Defaults are sensible
- [ ] Migration from old config

#### Validation Steps
- [ ] Config loads correctly
- [ ] Validation catches errors
- [ ] Env vars override file
- [ ] Defaults work properly
- [ ] Old configs migrate

---

## Phase 6: Testing and Polish

### Task 15: Write Comprehensive Unit Tests

**Type**: testing
**Priority**: high
**Estimated Hours**: 6

#### Pre-Implementation Checklist
- [ ] Review test standards
- [ ] Plan test coverage
- [ ] Design test fixtures
- [ ] Create mocks
- [ ] Plan benchmarks

#### Description
Write unit tests for all components achieving >80% coverage.

#### Acceptance Criteria
- [ ] Parser tests complete
- [ ] Engine tests complete
- [ ] Queue tests complete
- [ ] Cache tests complete
- [ ] Player tests complete
- [ ] Coverage >80%

#### Validation Steps
- [ ] All tests pass
- [ ] Coverage goal met
- [ ] No flaky tests
- [ ] Benchmarks run
- [ ] Race detector passes

---

### Task 16: Write Integration Tests

**Type**: testing
**Priority**: high
**Estimated Hours**: 4

#### Pre-Implementation Checklist
- [ ] Plan test scenarios
- [ ] Set up test environment
- [ ] Create test data
- [ ] Design test harness
- [ ] Consider CI integration

#### Description
Write integration tests for component interactions and workflows.

#### Acceptance Criteria
- [ ] Engine integration tests
- [ ] UI integration tests
- [ ] Pipeline tests complete
- [ ] Fallback tests work
- [ ] Performance tests pass

#### Validation Steps
- [ ] Tests are reproducible
- [ ] No external dependencies
- [ ] CI integration works
- [ ] Tests are maintainable

---

### Task 17: Performance Optimization

**Type**: implementation
**Priority**: medium
**Estimated Hours**: 4

#### Pre-Implementation Checklist
- [ ] Profile current performance
- [ ] Identify bottlenecks
- [ ] Plan optimizations
- [ ] Set performance goals
- [ ] Design benchmarks

#### Description
Optimize performance based on profiling results and benchmarks.

#### Acceptance Criteria
- [ ] Startup time <3 seconds
- [ ] First audio <200ms
- [ ] Memory usage <75MB
- [ ] No UI blocking
- [ ] Cache hit rate >80%

#### Validation Steps
- [ ] Benchmarks show improvement
- [ ] Memory profile acceptable
- [ ] CPU usage reasonable
- [ ] No regressions
- [ ] Real-world testing passes

---

### Task 18: Documentation

**Type**: documentation
**Priority**: medium
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Review documentation standards
- [ ] Plan documentation structure
- [ ] Gather examples
- [ ] Create diagrams
- [ ] Write user guide

#### Description
Write comprehensive documentation for TTS functionality.

#### Acceptance Criteria
- [ ] User guide complete
- [ ] API documentation complete
- [ ] Configuration documented
- [ ] Examples provided
- [ ] Troubleshooting guide
- [ ] Architecture documented

#### Validation Steps
- [ ] Documentation builds
- [ ] Examples work
- [ ] No broken links
- [ ] Clear and accurate
- [ ] Reviewed by others

---

### Task 19: Error Handling and Recovery

**Type**: implementation
**Priority**: high
**Estimated Hours**: 3

#### Pre-Implementation Checklist
- [ ] Identify failure points
- [ ] Design recovery strategies
- [ ] Plan error messages
- [ ] Consider logging
- [ ] Design fallback behavior

#### Description
Implement comprehensive error handling and recovery mechanisms.

#### Acceptance Criteria
- [ ] All errors handled gracefully
- [ ] Recovery mechanisms work
- [ ] User-friendly error messages
- [ ] Appropriate logging
- [ ] No crashes or panics
- [ ] Fallback behavior works

#### Validation Steps
- [ ] Error injection tests pass
- [ ] Recovery works correctly
- [ ] Messages are helpful
- [ ] Logs are informative
- [ ] System remains stable

---

### Task 20: Final Integration and Polish

**Type**: implementation
**Priority**: low
**Estimated Hours**: 2

#### Pre-Implementation Checklist
- [ ] Review all components
- [ ] Check integration points
- [ ] Verify configuration
- [ ] Test full workflow
- [ ] Gather feedback

#### Description
Final integration, testing, and polish before release.

#### Acceptance Criteria
- [ ] All components integrated
- [ ] Full workflow tested
- [ ] Performance acceptable
- [ ] Documentation complete
- [ ] No known critical bugs
- [ ] Ready for release

#### Validation Steps
- [ ] Manual testing complete
- [ ] Automated tests pass
- [ ] Performance goals met
- [ ] Documentation reviewed
- [ ] Code review complete

---

## Summary

**Total Tasks**: 20
**Total Estimated Hours**: 75-80 hours
**Estimated Duration**: 7-11 days (with parallel work)

### Critical Path
1. Project Setup → Interfaces → Parser → Queue → Piper Engine → Audio Player → Controller → UI Integration

### Parallel Work Opportunities
- Cache and Mock Player (can be done in parallel)
- Google TTS and Fallback Engine (after Piper)
- Documentation and Testing (ongoing)

### Risk Areas
- Audio device compatibility
- Piper subprocess management
- UI responsiveness
- Memory management