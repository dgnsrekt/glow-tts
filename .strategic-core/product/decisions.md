# Glow-TTS Architectural Decisions

## Observed Patterns (Current Implementation)

### 1. Terminal-First Design
**Decision**: Build exclusively for terminal environments
**Rationale**: 
- Glow's core user base consists of terminal power users
- Maintains consistency with the original Glow philosophy
- Reduces complexity by avoiding GUI considerations
**Trade-offs**: 
- Limited to terminal-capable environments
- No web or mobile support

### 2. Bubble Tea Framework
**Decision**: Use Bubble Tea for all UI interactions
**Rationale**:
- Provides a robust, event-driven architecture
- Excellent terminal rendering capabilities
- Strong ecosystem with Bubbles components
- Maintained by the same team (Charm)
**Trade-offs**:
- Learning curve for contributors unfamiliar with Elm architecture
- Tied to a specific UI paradigm

### 3. Message-Based Architecture
**Decision**: Use message passing for all state updates
**Rationale**:
- Clean separation of concerns
- Predictable state management
- Easy to test and debug
- Natural fit for async operations
**Pattern**: All UI updates flow through Update() method with typed messages

### 4. Glamour for Markdown Rendering
**Decision**: Use Glamour instead of direct terminal rendering
**Rationale**:
- Purpose-built for terminal markdown
- Handles styling and ANSI codes
- Supports themes and customization
- Same maintainer ensures compatibility

### 5. Configuration via Viper
**Decision**: Use Viper for configuration management
**Rationale**:
- Industry standard in Go ecosystem
- Supports multiple config formats
- Environment variable integration
- Hierarchical configuration

## TTS Architectural Decisions (Updated with Lessons Learned)

### 6. Subprocess Execution Model for TTS
**Decision**: Use synchronous subprocess execution with pre-configured stdin
**Rationale**:
- Avoids stdin race condition discovered in experimental branch
- Simpler than long-running processes or process pools
- More reliable and predictable
- Easier to debug and maintain
**Implementation**: 
```go
cmd.Stdin = strings.NewReader(text)  // Pre-set stdin
cmd.Run()  // Synchronous execution
```
**Trade-offs**:
- Higher process spawn overhead (mitigated by caching)
- No process reuse (but more stable)

### 7. Queue-Based Audio Processing
**Decision**: Use a queue system for sentence processing
**Rationale**:
- Enables look-ahead preprocessing
- Smooth playback without gaps
- Better control over playback state
- Efficient memory usage
**Pattern**: Ring buffer with 2-3 sentence look-ahead

### 8. Multi-Engine Support
**Decision**: Support both offline (Piper) and cloud (Google TTS) engines
**Rationale**:
- Flexibility for different use cases
- Offline capability for privacy/security
- Cloud option for better quality
- Graceful fallback mechanisms
**Trade-offs**: Increased complexity in engine abstraction

### 9. CLI Flag Integration
**Decision**: Use `--tts [engine]` flag pattern
**Rationale**:
- Consistent with existing Glow CLI patterns
- Clear and explicit activation
- Engine selection at startup
- Forces TUI mode automatically

### 10. Caching Strategy for TTS
**Decision**: Implement aggressive in-memory and disk caching
**Rationale**:
- Mitigates process spawn overhead
- Provides instant response for repeated content
- Reduces CPU usage significantly
- Improves user experience
**Implementation**:
- LRU memory cache with 100MB limit
- Persistent disk cache in temp directory
- SHA256-based cache keys
- 80% expected cache hit rate

### 11. Stdin Race Prevention
**Decision**: Never use StdinPipe() for subprocess communication
**Rationale**:
- StdinPipe() creates race conditions with programs that read immediately
- Piper reads stdin on startup before pipe is ready
- Race is non-deterministic and platform-dependent
**Lesson Learned**: Experimental branch discovered this critical issue
**Correct Pattern**:
```go
// Always use this pattern for stdin
cmd.Stdin = strings.NewReader(text)
cmd.Run()  // Not Start()
```

## Code Organization Patterns

### Package Structure
**Pattern**: Functional grouping with clear boundaries
```
main.go           - Entry point and CLI setup
ui/              - All UI-related code
  ├── pager.go   - Document viewing
  ├── stash.go   - File browsing
  └── ...
utils/           - Shared utilities
```

### Error Handling
**Pattern**: Explicit error returns, wrapped with context
- No panic in library code
- Errors bubble up to UI layer
- User-friendly error messages

### State Management
**Pattern**: Centralized state in model struct
- Immutable updates via messages
- No direct state mutation
- Clear state transitions

### Async Operations
**Pattern**: Goroutines with channels for communication
- File discovery runs async
- Network operations are non-blocking
- Results delivered via messages

## Testing Strategy

### Current Approach
- Unit tests for pure functions
- Integration tests for CLI commands
- No UI testing (manual testing only)

### Planned TTS Testing
- Mock audio players for unit tests
- Interface-based engine testing
- Queue operation verification
- Process management testing

## Performance Considerations

### Startup Performance
**Decision**: Lazy loading where possible
**Implementation**:
- Don't scan all files on startup
- Progressive file discovery
- On-demand markdown rendering

### Memory Management
**Decision**: Stream large files rather than loading entirely
**Implementation**:
- Chunked reading for large documents
- Bounded buffers for audio (planned)
- Cache eviction for old audio files (planned)

## Security Considerations

### File Access
**Decision**: Respect OS file permissions
**Implementation**: 
- No privilege escalation
- User-space operation only
- Standard file system APIs

### Network Security (Planned)
**Decision**: HTTPS only for cloud TTS
**Implementation**:
- Certificate validation
- API key encryption
- No credential logging

## Future Considerations

### Extensibility Points
1. **Engine Interface**: Clean abstraction for adding new TTS engines
2. **Message Protocol**: Extensible message types for new features
3. **Plugin System**: Potential for user-defined processors
4. **Theme System**: Already supports custom themes via JSON

### Technical Debt to Address
1. **Test Coverage**: Increase automated testing
2. **Documentation**: Add inline API documentation
3. **Performance Profiling**: Identify bottlenecks
4. **Accessibility**: Beyond TTS, consider screen reader support

### Migration Path
1. **Gradual Feature Addition**: TTS as optional feature
2. **Backward Compatibility**: Maintain all existing Glow functionality
3. **Configuration Migration**: Support old config formats
4. **Experimental Flags**: Test new features before stable release