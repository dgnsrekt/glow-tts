# TTS Core Infrastructure Specification

## Feature Overview

The TTS Core Infrastructure provides the foundational text-to-speech system for Glow-TTS, enabling sentence-level synchronized audio playback of markdown documents. This feature implements the essential components needed for all TTS functionality while maintaining complete isolation from the existing Glow codebase.

## User Stories

### As a Developer
- I want to consume technical documentation audibly while coding
- I want synchronized highlighting so I can follow along visually
- I want to navigate between sentences easily
- I want TTS to work offline without internet dependency

### As a User with Visual Impairment
- I want to access terminal-based markdown content through audio
- I want clear audio feedback for my navigation actions
- I want the system to gracefully handle TTS failures
- I want consistent playback controls similar to media players

### As a Productivity User
- I want to listen to documentation while performing other tasks
- I want to pause and resume playback easily
- I want to jump to specific sentences quickly
- I want clear visual indication of playback progress

## Acceptance Criteria

### Core Functionality
- [ ] TTS system initializes without errors when Piper is available
- [ ] System gracefully degrades when Piper is unavailable
- [ ] Markdown content is correctly parsed into sentences
- [ ] Audio generation completes within 2 seconds for typical sentences
- [ ] Playback controls (play, pause, stop) function correctly
- [ ] Sentence navigation (next, previous) works accurately
- [ ] Current sentence is visually highlighted during playback
- [ ] Audio-visual synchronization maintains <500ms accuracy

### Integration Requirements
- [ ] All TTS code is contained within `tts/` directory
- [ ] Changes to existing Glow files are minimal (<100 lines total)
- [ ] TTS can be disabled via configuration
- [ ] System maintains Glow's existing performance characteristics
- [ ] Keyboard shortcuts follow established patterns

### Error Handling
- [ ] System handles Piper process failures gracefully
- [ ] Audio device errors don't crash the application
- [ ] Memory usage remains stable during long sessions
- [ ] Resource cleanup occurs properly on exit

## Success Metrics

### Performance Metrics
- Audio generation latency: <2 seconds for sentences up to 100 words
- Synchronization accuracy: <500ms drift between audio and highlighting
- Memory overhead: <50MB for TTS components
- CPU usage: <10% increase during playback

### Quality Metrics
- Zero crashes due to TTS failures
- 100% of sentences correctly parsed
- Playback controls respond within 200ms
- Clean shutdown in all scenarios

### User Experience Metrics
- Learning curve: <5 minutes for existing Glow users
- All TTS functions accessible via keyboard
- Clear visual feedback for all state changes
- Intuitive navigation between sentences

## Dependencies

### External Dependencies
- **Piper TTS**: External binary for text-to-speech generation
  - Version: Latest stable release
  - Installation: User-provided or bundled
  - Fallback: Graceful degradation if unavailable

### Go Dependencies
- **Audio Library**: Cross-platform audio playback
  - Options: `github.com/hajimehoshi/oto/v3` or `github.com/faiface/beep`
  - Requirement: PCM audio support
  - Platform: Linux, macOS, Windows

### Internal Dependencies
- **Bubble Tea**: Event system integration
- **Glamour**: Markdown parsing for sentence extraction
- **Lipgloss**: Styling for highlighted sentences

## Constraints

### Technical Constraints
- Must work within terminal environment limitations
- Cannot modify core Glow rendering pipeline
- Must maintain compatibility with all Glow-supported terminals
- Audio playback depends on system audio availability

### Design Constraints
- All TTS code must be in `tts/` directory
- Interfaces must be clearly defined and minimal
- UI changes limited to highlighting and status display
- Configuration extends existing Viper schema

### Performance Constraints
- Audio generation must not block UI updates
- Memory usage must scale linearly with document size
- Sentence parsing must complete in <1 second for 10MB files
- Synchronization updates must occur at least every 100ms

## Risks and Mitigations

### Risk: Piper Binary Availability
- **Impact**: Core functionality unavailable
- **Mitigation**: Clear installation instructions, binary detection, graceful fallback

### Risk: Audio Device Access
- **Impact**: No audio playback possible
- **Mitigation**: Multiple audio backend options, clear error messages

### Risk: Synchronization Drift
- **Impact**: Poor user experience
- **Mitigation**: Regular drift correction, timing calibration

### Risk: Memory Leaks
- **Impact**: Application instability
- **Mitigation**: Proper resource management, regular cleanup, testing

## Feature Scope

### In Scope
- Piper TTS integration
- Sentence-level parsing and tracking
- Audio playback with controls
- Visual highlighting synchronization
- Keyboard navigation
- Basic configuration options
- Error handling and recovery

### Out of Scope (Future Phases)
- Google TTS integration
- Word-level synchronization
- Speech rate adjustment
- Voice selection UI
- Audio export functionality
- Bookmark management
- Advanced configuration UI

## Technical Approach

### Architecture Pattern
- Clean Architecture with dependency injection
- Interface-driven design for testability
- Event-driven communication via Bubble Tea
- Concurrent audio pipeline for performance

### Implementation Strategy
1. Build isolated TTS components in `tts/` directory
2. Define minimal interfaces for Glow integration
3. Implement Piper engine with process management
4. Create audio playback system with platform abstraction
5. Develop sentence parser using existing Glamour
6. Build synchronization manager with drift correction
7. Add minimal UI integration for highlighting
8. Implement comprehensive error handling

## Configuration Schema

```yaml
tts:
  enabled: true
  engine: "piper"
  piper:
    binary: "piper"  # Path to Piper executable
    model: "en_US-lessac-medium"  # Default voice model
    modelPath: "~/.local/share/piper/models"
  playback:
    bufferSize: 3  # Sentences to buffer ahead
  sync:
    updateInterval: 100ms  # Highlight update frequency
    driftThreshold: 500ms  # Maximum allowed drift
```

## Testing Requirements

### Unit Tests
- Sentence parser with various markdown inputs
- Audio buffer management
- Synchronization timing calculations
- Error handling paths

### Integration Tests
- Piper process lifecycle management
- Audio playback on different platforms
- Bubble Tea event integration
- Configuration loading and validation

### End-to-End Tests
- Complete TTS flow from markdown to audio
- Navigation during playback
- Error recovery scenarios
- Resource cleanup verification

## Documentation Requirements

### User Documentation
- Installation guide for Piper TTS
- Configuration options explanation
- Keyboard shortcuts reference
- Troubleshooting guide

### Developer Documentation
- Architecture overview
- Interface definitions
- Testing procedures
- Contribution guidelines

## Timeline Estimate

### Phase 1: Core Components (Week 1-2)
- TTS controller and interfaces
- Piper engine integration
- Basic audio playback

### Phase 2: Parsing & Sync (Week 2-3)
- Sentence parser implementation
- Synchronization manager
- Drift correction

### Phase 3: UI Integration (Week 3-4)
- Minimal pager modifications
- Status display component
- Keyboard handlers

### Phase 4: Testing & Polish (Week 4-5)
- Comprehensive testing
- Performance optimization
- Documentation completion