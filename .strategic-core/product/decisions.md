# Architectural Decisions - Glow-TTS

## Existing Architecture (Inherited from Glow)

### Decision: Bubble Tea Framework for TUI
**Status**: Implemented (Base Glow)
**Context**: Need for a robust, event-driven terminal UI framework
**Decision**: Use Bubble Tea for all interactive UI components
**Rationale**: 
- Excellent terminal compatibility
- Composable component architecture
- Active development and community
- Same organization (Charm.sh) for consistency
**Consequences**: 
- All UI components follow Model-Update-View pattern
- Event-driven architecture throughout
- Need to integrate TTS into event loop

### Decision: Glamour for Markdown Rendering
**Status**: Implemented (Base Glow)
**Context**: Need to render markdown beautifully in terminals
**Decision**: Use Glamour as the markdown rendering engine
**Rationale**:
- Purpose-built for terminal rendering
- Supports syntax highlighting via Chroma
- Customizable styling system
- ANSI color support
**Consequences**:
- Consistent markdown rendering
- Need to extend for sentence highlighting
- Style system can be leveraged for TTS highlighting

### Decision: Cobra for CLI Structure
**Status**: Implemented (Base Glow)
**Context**: Need robust command-line argument parsing
**Decision**: Use Cobra for CLI commands and flags
**Rationale**:
- Industry standard for Go CLIs
- Auto-generated help and completion
- Subcommand support
- Well-tested and mature
**Consequences**:
- Standardized CLI interface
- Easy to add TTS-related flags
- Consistent with Go ecosystem

## Planned TTS Architecture Decisions

### Decision: Sentence-Level Synchronization
**Status**: Planned
**Context**: Need to synchronize audio with visual content
**Decision**: Implement sentence-level (not word-level) synchronization
**Rationale**:
- Simpler implementation for v1.0
- Sufficient granularity for most use cases
- Easier timing estimation
- Less computational overhead
**Trade-offs**:
- Less precise than word-level
- May feel less smooth for very long sentences
- Acceptable for initial release
**Future**: Consider word-level in v2.0

### Decision: Dual TTS Engine Support
**Status**: Planned
**Context**: Need both offline and online TTS capabilities
**Decision**: Support both Piper (local) and Google TTS (cloud)
**Rationale**:
- Piper for privacy and offline use
- Google TTS for quality and language variety
- User choice based on needs
- Fallback options available
**Trade-offs**:
- More complex integration layer
- Need to handle different APIs
- Worth it for flexibility

### Decision: External Process for TTS
**Status**: Planned
**Context**: TTS engines run as separate binaries/services
**Decision**: Use os/exec for Piper, HTTP client for Google TTS
**Rationale**:
- Piper is a standalone binary
- Clean separation of concerns
- Easier to swap engines
- Process isolation for stability
**Trade-offs**:
- Process management complexity
- IPC overhead
- Acceptable for audio generation

### Decision: TTS Code Isolation in Dedicated Directory
**Status**: Planned
**Context**: Need to add TTS without disrupting existing Glow codebase
**Decision**: All TTS code lives in `tts/` directory with clean interfaces
**Rationale**:
- Maintains clear separation of concerns
- Makes feature optional/removable
- Simplifies testing and maintenance
- Potential for upstream contribution
- Minimal changes to original Glow files
**Trade-offs**:
- Slightly more complex import paths
- Need to define clear interfaces
- Worth it for maintainability
**Implementation**:
- Create `tts/` directory for all TTS logic
- Define interfaces in `tts/interfaces.go`
- Only modify ui/pager.go for highlighting
- Add single ui/tts_status.go for status display

### Decision: Integration Pattern with Bubble Tea
**Status**: Planned
**Context**: Need to integrate audio playback with existing TUI
**Decision**: Extend pager model with TTS state, use tea.Cmd for async audio
**Rationale**:
- Maintains Bubble Tea patterns
- Non-blocking audio operations
- Clean state management
- Preserves existing functionality
**Implementation**:
- Add minimal TTSState to pager model
- Use tea.Cmd for audio operations
- Update view for highlighting (minimal changes)
- New keyboard handlers for TTS (isolated)

### Decision: Highlighting Implementation
**Status**: Planned  
**Context**: Need to highlight current sentence being spoken
**Decision**: Extend Glamour renderer with sentence tracking
**Rationale**:
- Leverage existing styling system
- ANSI escape codes for highlighting
- Maintain theme compatibility
- Minimal performance impact
**Implementation**:
- Track sentence boundaries during parsing
- Apply highlight style to current sentence
- Update on audio position changes

## Code Organization Patterns

### Pattern: Feature Isolation with Directory Structure
**Observed**: Features organized in separate files (github.go, gitlab.go)
**Application**: TTS features should be fully isolated in dedicated directory
**Recommendation**: Create `tts/` directory with all TTS code:
- `tts/controller.go` - Main TTS orchestrator
- `tts/engines/` - Engine implementations (piper/, google/)
- `tts/audio/` - Audio playback system
- `tts/sentence/` - Sentence parsing
- `tts/sync/` - Synchronization management

### Pattern: Minimal UI Integration
**Observed**: Each UI component in separate file under ui/
**Application**: TTS UI changes should be minimal
**Recommendation**: 
- Add only `ui/tts_status.go` for status display
- Extend existing `ui/pager.go` minimally for highlighting
- Avoid creating multiple new UI files

### Pattern: Configuration Integration
**Observed**: Viper used for all configuration
**Application**: TTS settings should use same system
**Recommendation**: Extend existing config with TTS section

### Pattern: Error Handling
**Observed**: Explicit error returns, graceful degradation
**Application**: TTS failures shouldn't crash application
**Recommendation**: Fallback to visual-only mode on TTS failure

## Performance Considerations

### Decision: Lazy Audio Generation
**Status**: Planned
**Context**: Don't want to pre-generate all audio
**Decision**: Generate audio on-demand per sentence
**Rationale**:
- Reduces memory usage
- Faster initial load
- Better for large documents
- Can buffer ahead intelligently

### Decision: Concurrent Audio Pipeline
**Status**: Planned
**Context**: Audio generation can be slow
**Decision**: Use goroutines for audio generation pipeline
**Rationale**:
- Leverage Go's concurrency
- Generate next sentence while playing current
- Smooth playback experience
- Efficient resource usage

## Testing Strategy

### Decision: Integration Testing Focus
**Status**: Planned
**Context**: TTS involves external processes and timing
**Decision**: Emphasize integration tests over unit tests for TTS
**Rationale**:
- Audio synchronization is timing-sensitive
- External process interaction critical
- End-to-end validation important
- Unit tests for pure functions only

## Future Considerations

### Potential: Plugin Architecture
**Context**: May want to support more TTS engines
**Consideration**: Design TTS interface for pluggability
**Benefit**: Easy to add new engines
**Cost**: Additional abstraction complexity

### Potential: Caching Layer
**Context**: Re-generating audio is wasteful
**Consideration**: Cache generated audio
**Benefit**: Faster repeated playback
**Cost**: Memory/disk usage

### Potential: Streaming TTS
**Context**: Some TTS engines support streaming
**Consideration**: Support streaming audio generation
**Benefit**: Lower latency for long text
**Cost**: More complex audio pipeline

## Implementation Phases

### Phase 1: Core TTS Infrastructure (Isolated)
**Goal**: Build TTS functionality without touching Glow
**Approach**:
- Create entire `tts/` directory structure
- Implement all TTS components with interfaces
- Test independently of Glow UI
- No changes to existing Glow files

### Phase 2: Minimal UI Integration
**Goal**: Connect TTS to Glow with minimal changes
**Approach**:
- Add `ui/tts_status.go` for status display
- Add ~50 lines to `ui/pager.go` for highlighting
- Add ~20 lines to `ui/keys.go` for keyboard handlers
- Total changes to existing files: <100 lines

### Phase 3: Configuration & Polish
**Goal**: Full integration with Glow ecosystem
**Approach**:
- Extend Viper configuration
- Add CLI flags to main.go
- Update help and documentation
- Performance optimization

### Phase 4: Testing & Release
**Goal**: Production-ready TTS feature
**Approach**:
- Comprehensive test coverage
- Cross-platform validation
- Documentation updates
- Binary distribution setup

## Technical Debt Acknowledgment

### Debt: No Current Audio Support
**Impact**: Starting from scratch for audio
**Mitigation**: Use established Go audio libraries
**Priority**: Core requirement, must address

### Debt: No Accessibility Features
**Impact**: Limited usability for vision-impaired users
**Mitigation**: TTS is first step, consider screen reader support
**Priority**: High, aligns with mission

### Debt: Synchronization Complexity
**Impact**: Hardest part of implementation
**Mitigation**: Start simple, iterate
**Priority**: Critical for user experience