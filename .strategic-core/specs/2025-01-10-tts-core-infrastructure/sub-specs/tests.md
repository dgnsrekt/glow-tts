# Test Specification: TTS Core Infrastructure

## Test Strategy

### Testing Pyramid
- **Unit Tests** (70%): Individual component testing
- **Integration Tests** (20%): Component interaction testing  
- **End-to-End Tests** (10%): Full workflow testing

### Coverage Goals
- **Overall**: Minimum 80% code coverage
- **Critical Paths**: 100% coverage (synthesis, playback)
- **Error Paths**: 90% coverage
- **UI Integration**: 60% coverage

## Unit Test Scenarios

### 1. Sentence Parser Tests

```go
// internal/tts/parser_test.go
```

#### Test Cases
- **Basic Parsing**
  - Simple sentences
  - Multiple sentences
  - Single word sentences
  - Empty input

- **Markdown Handling**
  - Bold/italic removal
  - Link text extraction
  - Header processing
  - List item handling

- **Edge Cases**
  - URLs preservation
  - Abbreviations (Dr., Mr., etc.)
  - Numbers and dates
  - Special characters

- **Code Block Handling**
  - Inline code removal
  - Block code skipping
  - Mixed content

### 2. TTS Engine Tests

```go
// internal/tts/engines/engine_test.go
```

#### Test Cases
- **Synthesis**
  - Valid text input
  - Empty text
  - Very long text (>5000 chars)
  - Special characters
  - Multiple languages

- **Configuration**
  - Voice selection
  - Speed adjustment
  - Volume control
  - Sample rate changes

- **Error Handling**
  - Invalid configuration
  - Process crash recovery
  - Network failures (Google TTS)
  - Model not found (Piper)

### 3. Audio Queue Tests

```go
// internal/queue/audio_queue_test.go
```

#### Test Cases
- **Queue Operations**
  - Enqueue/dequeue
  - Priority handling
  - Queue overflow
  - Empty queue

- **Preprocessing**
  - Lookahead management
  - Cache integration
  - Concurrent access

- **Memory Management**
  - Buffer limits
  - Cleanup on clear
  - Leak detection

### 4. Audio Player Tests

```go
// internal/audio/player_test.go
```

#### Test Cases
- **Playback Control**
  - Play/pause/stop
  - Position tracking
  - Speed adjustment
  - Volume control

- **Streaming**
  - Buffer underrun
  - Large audio files
  - Continuous playback

- **Platform Specific**
  - Device initialization
  - Format compatibility
  - Error recovery

### 5. Cache Tests

```go
// internal/cache/audio_cache_test.go
```

#### Test Cases
- **Basic Operations**
  - Get/set/delete
  - Cache hits/misses
  - Key generation

- **Eviction**
  - LRU eviction
  - Size limits
  - Manual clearing

- **Persistence**
  - Save to disk
  - Load from disk
  - Corruption handling

## Integration Test Scenarios

### 1. Engine Integration

```go
// test/integration/engine_integration_test.go
```

#### Test Scenarios
- **Piper Integration**
  - Process lifecycle
  - Model loading
  - Audio generation
  - Error recovery

- **Google TTS Integration**
  - API authentication
  - Request/response
  - Rate limiting
  - Fallback behavior

- **Engine Switching**
  - Fallback on failure
  - Manual switching
  - Configuration changes

### 2. UI Integration

```go
// test/integration/ui_integration_test.go
```

#### Test Scenarios
- **Command Flow**
  - Start TTS command
  - Navigation commands
  - Control commands
  - Stop command

- **Message Updates**
  - Status updates
  - Error messages
  - Progress indicators

- **State Synchronization**
  - UI reflects TTS state
  - Concurrent operations
  - Race conditions

### 3. Queue Processing

```go
// test/integration/queue_integration_test.go
```

#### Test Scenarios
- **Pipeline Flow**
  - Document → Queue → Engine → Player
  - Preprocessing pipeline
  - Cache integration

- **Concurrency**
  - Multiple producers
  - Multiple consumers
  - Synchronization

## End-to-End Test Scenarios

### 1. Basic Workflow

```go
// test/e2e/basic_workflow_test.go
```

#### Test Flow
1. Start Glow with `--tts` flag
2. Open markdown file
3. Verify TTS initialization
4. Play through document
5. Test pause/resume
6. Navigate sentences
7. Stop and exit

### 2. Engine Fallback

```go
// test/e2e/fallback_test.go
```

#### Test Flow
1. Start with Piper configured
2. Simulate Piper failure
3. Verify fallback to Google TTS
4. Continue playback
5. Verify no data loss

### 3. Large Document

```go
// test/e2e/performance_test.go
```

#### Test Flow
1. Load 100+ page document
2. Start TTS playback
3. Monitor memory usage
4. Test navigation performance
5. Verify cache effectiveness

## Performance Test Scenarios

### 1. Synthesis Benchmarks

```go
// test/benchmark/synthesis_bench_test.go
```

#### Benchmarks
- Single sentence synthesis time
- Batch synthesis throughput
- Different engine comparison
- Cache hit performance

### 2. Memory Benchmarks

```go
// test/benchmark/memory_bench_test.go
```

#### Benchmarks
- Memory per sentence
- Cache memory usage
- Queue memory overhead
- Leak detection over time

### 3. Concurrency Benchmarks

```go
// test/benchmark/concurrency_bench_test.go
```

#### Benchmarks
- Parallel synthesis
- Queue throughput
- Lock contention
- Channel performance

## Test Data Requirements

### Text Samples
```
testdata/
├── simple.md          # Basic markdown
├── complex.md         # Full-featured markdown
├── code-heavy.md      # Lots of code blocks
├── special-chars.md   # Unicode, emojis
├── large.md          # 100+ pages
└── multilingual.md   # Multiple languages
```

### Audio Samples
```
testdata/audio/
├── sample-16k.wav    # 16kHz sample
├── sample-22k.wav    # 22kHz sample
├── sample-48k.wav    # 48kHz sample
└── silence.wav       # Silence for testing
```

### Configuration Files
```
testdata/config/
├── default.yaml      # Default config
├── piper-only.yaml   # Piper engine only
├── google-only.yaml  # Google TTS only
└── invalid.yaml      # Invalid config
```

## Mock Implementations

### Mock Engine
```go
type MockEngine struct {
    SynthesizeFunc func(text string) ([]byte, error)
    DelayMs        int
    FailureRate    float64
}
```

### Mock Player
```go
type MockPlayer struct {
    PlayFunc     func(audio []byte) error
    IsPlayingFunc func() bool
    Position     time.Duration
}
```

### Mock Cache
```go
type MockCache struct {
    storage  map[string][]byte
    hits     int
    misses   int
}
```

## Test Execution Plan

### Continuous Integration
```yaml
# .github/workflows/test.yml
test:
  - go test ./...                    # Unit tests
  - go test -race ./...              # Race detection
  - go test -tags=integration ./...  # Integration tests
  - go test -bench=. ./...           # Benchmarks
  - go test -cover ./...             # Coverage
```

### Local Testing
```bash
# Quick tests
task test

# Full test suite
task test:all

# Specific component
go test ./internal/tts/...

# With coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Acceptance Testing

### Manual Test Checklist

#### Basic Functionality
- [ ] TTS starts with --tts flag
- [ ] Audio plays correctly
- [ ] Pause/resume works
- [ ] Stop halts playback
- [ ] Navigation works (next/prev)

#### Edge Cases
- [ ] Empty document handling
- [ ] Very long document (100+ pages)
- [ ] Document with only code
- [ ] Rapid navigation
- [ ] Quick start/stop cycles

#### Performance
- [ ] Startup time <3 seconds
- [ ] First audio <200ms
- [ ] Memory stays under 75MB
- [ ] No UI freezing
- [ ] Smooth playback

#### Error Conditions
- [ ] No audio device
- [ ] Engine not available
- [ ] Network offline (Google TTS)
- [ ] Cache full
- [ ] Invalid configuration