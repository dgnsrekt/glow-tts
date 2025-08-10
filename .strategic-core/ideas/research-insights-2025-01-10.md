# Research Insights and Improvements for TTS Implementation

> Compiled from web search of references in idea.md
> Date: 2025-01-10

## Executive Summary

Based on research of the latest best practices for Piper TTS, Google Cloud TTS, subprocess management, audio playback, and Bubble Tea integration, we've identified several key improvements that can enhance our TTS implementation beyond what's currently documented.

## Key Findings and Suggested Improvements

### 1. Piper TTS Optimizations

#### Current Approach
We're using fresh process per synthesis with pre-configured stdin.

#### Research Finding
Piper can be loaded once and kept waiting for input on stdin, with output streamed to audio players.

#### **Suggested Improvement**
Consider a hybrid approach for frequently-used documents:
- Keep a Piper process alive for the session duration for the current document
- Use fresh processes for one-off syntheses
- This could reduce latency from ~100ms to ~20ms for active reading

#### Implementation Pattern
```go
type PiperSession struct {
    cmd     *exec.Cmd
    stdin   io.WriteCloser
    stdout  io.ReadCloser
    active  bool
    timeout time.Duration
}

// Keep alive for active document, timeout after 5 minutes idle
```

### 2. Google Cloud TTS Streaming

#### Current Approach
Standard synthesis requests with caching.

#### Research Finding
Google now supports bidirectional streaming with Chirp 3: HD voices (2024 feature).

#### **Suggested Improvement**
Implement streaming synthesis for Google TTS:
- Use StreamingSynthesizeRequest for real-time synthesis
- Reduces first-audio latency significantly
- Only available in specific regions (us, eu, asia-southeast1)

#### Configuration Update
```yaml
google:
  api_key: ${GOOGLE_TTS_API_KEY}
  voice: en-US-Journey-F  # New Journey voices with streaming
  region: us  # Required for streaming
  streaming: true
```

### 3. OTO Audio Library Enhancements

#### Current Approach
Basic audio playback with oto/v3.

#### Research Finding
Critical requirement: Keep file references alive during streaming playback.

#### **Suggested Improvement**
Implement proper lifecycle management for audio streams:
```go
type AudioStream struct {
    data     []byte      // Keep reference alive
    reader   io.Reader   // For streaming
    player   oto.Player
    position int64
}

// Never let data go out of scope while playing
```

#### Buffer Size Optimization
- Use 100ms frames as optimal balance between latency and efficiency
- Implement BufferSizeSetter interface for dynamic tuning
- Sample rate: Strictly 44100 or 48000 Hz (other values cause distortion)

### 4. Bubble Tea Integration Pattern

#### Current Approach
Basic message passing for TTS status.

#### Research Finding
**Never use goroutines directly** - always use Commands.

#### **Suggested Improvement**
Refactor all async operations to use Bubble Tea commands:
```go
// ❌ WRONG - Never do this
go func() {
    audio := engine.Synthesize(text)
    program.Send(AudioReadyMsg{audio})
}()

// ✅ CORRECT - Use commands
func synthesizeCmd(text string) tea.Cmd {
    return func() tea.Msg {
        audio := engine.Synthesize(text)
        return AudioReadyMsg{audio}
    }
}
```

### 5. Race Condition Prevention Enhancement

#### Current Approach
Using pre-configured stdin with cmd.Run().

#### Research Finding
Additional race conditions exist with Wait() and file operations.

#### **Suggested Improvement**
Implement comprehensive synchronization:
```go
func (e *PiperEngine) Synthesize(ctx context.Context, text string) ([]byte, error) {
    cmd := exec.CommandContext(ctx, "piper", args...)
    cmd.Stdin = strings.NewReader(text)
    
    // Add timeout protection
    done := make(chan error, 1)
    go func() {
        done <- cmd.Run()
    }()
    
    select {
    case err := <-done:
        return handleResult(err)
    case <-time.After(5 * time.Second):
        cmd.Process.Kill()
        return nil, ErrTimeout
    }
}
```

### 6. Advanced Caching Strategy

#### Current Approach
Two-level cache with TTL and size limits.

#### Research Finding
Audio applications commonly use session-based caching with predictive preloading.

#### **Suggested Improvement**
Add predictive caching based on reading patterns:
```go
type PredictiveCache struct {
    history     []string  // Recent access pattern
    predictions []string  // Predicted next accesses
    confidence  float64   // Prediction confidence
}

// Preload next 5 sentences with 90% confidence
// Preload next 10 sentences with 70% confidence
```

### 7. Process Management Pattern

#### Current Approach
Fresh process per synthesis.

#### Research Finding
Long-running processes can be reliable with proper health checks.

#### **Suggested Improvement**
Implement health-checked process pool:
```go
type ProcessPool struct {
    processes []*PiperProcess
    health    map[*PiperProcess]HealthStatus
    maxAge    time.Duration  // Recycle after 1 hour
    maxUses   int            // Recycle after 1000 uses
}

// Health check every 30 seconds
// Automatic restart on failure
// Graceful degradation to fresh processes
```

## Implementation Priority

Based on impact and complexity:

### High Priority (Immediate)
1. **Bubble Tea Command Refactoring** - Critical for stability
2. **OTO Lifecycle Management** - Prevents audio glitches
3. **Timeout Protection** - Prevents hanging processes

### Medium Priority (Next Sprint)
4. **Google TTS Streaming** - Significant latency improvement
5. **Predictive Caching** - Better user experience
6. **Process Pool** - Performance optimization

### Low Priority (Future)
7. **Piper Session Mode** - Complex but beneficial for power users

## Risk Mitigation

### For Each Improvement
- Implement behind feature flags
- Extensive testing with race detector
- Gradual rollout with metrics
- Fallback to current implementation

## Testing Requirements

### New Test Scenarios
```go
// Test streaming Google TTS fallback
func TestGoogleStreamingFallback(t *testing.T)

// Test OTO lifecycle management
func TestAudioStreamLifecycle(t *testing.T)

// Test Bubble Tea command patterns
func TestCommandBasedSynthesis(t *testing.T)

// Test predictive cache accuracy
func TestPredictiveCacheHitRate(t *testing.T)
```

## Configuration Schema Updates

```yaml
tts:
  # New options
  mode: fresh|session|pool  # Process management mode
  
  piper:
    session_timeout: 5m     # Keep alive duration
    pool_size: 3           # Process pool size
    
  google:
    streaming: true        # Enable streaming synthesis
    region: us            # Regional endpoint
    
  cache:
    predictive: true      # Enable predictive caching
    prediction_depth: 10  # Sentences to predict
    
  audio:
    buffer_ms: 100       # Frame size in milliseconds
    keep_alive: true     # Keep references during playback
```

## Conclusion

These improvements, derived from the latest 2024 best practices and real-world implementations, can significantly enhance our TTS system's performance, reliability, and user experience. The key themes are:

1. **Proper async patterns** - Use framework-appropriate concurrency
2. **Lifecycle management** - Keep resources alive when needed
3. **Predictive optimization** - Anticipate user behavior
4. **Graceful degradation** - Always have fallbacks
5. **Regional optimization** - Use nearest endpoints

Implementing these improvements incrementally will create a more robust and performant TTS system while maintaining our current stability.

## References

- Piper TTS GitHub Discussions and rhasspy3 implementation
- Google Cloud TTS Streaming Documentation (2024)
- OTO v3 Audio Library Best Practices
- Bubble Tea Command Patterns
- Go Race Detector Documentation