# TTS Implementation Guide

> Quick reference for implementing TTS with the optimal approach

## 🚀 Quick Start

### The Golden Rule

**NEVER use `StdinPipe()` - ALWAYS use `strings.NewReader()`**

### Correct Implementation Pattern

```go
package tts

import (
    "bytes"
    "context"
    "fmt"
    "os/exec"
    "strings"
)

type PiperEngine struct {
    modelPath string
    cache     map[string][]byte
}

func (e *PiperEngine) Synthesize(ctx context.Context, text string) ([]byte, error) {
    // 1. Check cache first
    if audio, ok := e.cache[text]; ok {
        return audio, nil
    }
    
    // 2. Prepare command
    cmd := exec.CommandContext(ctx, "piper",
        "--model", e.modelPath,
        "--output-raw")  // Raw PCM output
    
    // 3. CRITICAL: Pre-set stdin
    cmd.Stdin = strings.NewReader(text)
    
    // 4. Capture outputs
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // 5. Run synchronously
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("piper failed: %w, stderr: %s", 
            err, stderr.String())
    }
    
    // 6. Cache and return
    audio := stdout.Bytes()
    e.cache[text] = audio
    return audio, nil
}
```

## 📁 Project Structure

```
internal/
├── tts/
│   ├── controller.go      # Main orchestrator
│   ├── interfaces.go      # Core interfaces
│   ├── parser.go         # Sentence extraction
│   └── engines/
│       ├── piper.go      # Piper implementation
│       ├── google.go     # Google TTS
│       └── fallback.go  # Fallback logic
├── audio/
│   ├── player.go         # Audio playback (oto/v3)
│   └── buffer.go         # Ring buffer
├── queue/
│   └── queue.go          # Sentence queue
└── cache/
    ├── manager.go        # Cache orchestration
    ├── memory.go         # L1 memory cache (100MB)
    ├── disk.go           # L2 disk cache (1GB)
    └── session.go        # Session cache (50MB)
```

## ⚠️ Critical Lessons Learned

### 1. The Stdin Race Condition

**Problem**: Piper reads stdin immediately on startup
```go
// ❌ BROKEN
cmd.Start()              // Piper starts reading NOW
stdin := cmd.StdinPipe() // Too late!
stdin.Write(text)        // Writes to nothing
```

**Solution**: Pre-configure stdin with timeout protection
```go
// ✅ CORRECT with full protection
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "piper", args...)
cmd.Stdin = strings.NewReader(text)  // Ready before start

done := make(chan error, 1)
go func() {
    done <- cmd.Run()
}()

select {
case err := <-done:
    return handleResult(err)
case <-ctx.Done():
    cmd.Process.Kill()
    return ErrTimeout
}
```

### 2. Bubble Tea Command Pattern (CRITICAL!)

**Problem**: Using goroutines directly breaks Bubble Tea
```go
// ❌ CATASTROPHIC - Breaks everything
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    go func() {  // NEVER DO THIS!
        result := doWork()
        m.channel <- result
    }()
    return m, nil
}
```

**Solution**: ALWAYS use Commands for async operations
```go
// ✅ MANDATORY PATTERN
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return m, doWorkCmd()  // Return a command
}

func doWorkCmd() tea.Cmd {
    return func() tea.Msg {
        result := doWork()
        return WorkDoneMsg{result}
    }
}
```

### 3. OTO Audio Memory Management

**Problem**: Audio data gets GC'd during playback
```go
// ❌ BROKEN - Causes static/crashes
func Play(audio []byte) {
    reader := bytes.NewReader(audio)
    player.Play(reader)
    // audio gets GC'd, reader points to freed memory!
}
```

**Solution**: Keep references alive during playback
```go
// ✅ CORRECT - Prevents GC
type AudioStream struct {
    data []byte  // Keep alive!
    reader *bytes.Reader
}

func (s *AudioStream) Play() {
    s.reader = bytes.NewReader(s.data)
    player.Play(s.reader)
    // data stays alive as long as AudioStream exists
}
```

## 🎯 Implementation Checklist

### Phase 1: Core Setup
- [ ] Create package structure
- [ ] Define interfaces (TTSEngine, AudioPlayer, etc.)
- [ ] Add dependencies (`go get github.com/ebitengine/oto/v3`)
- [ ] Set up error types

### Phase 2: Piper Engine
- [ ] Implement using optimal pattern (no StdinPipe!)
- [ ] Add caching (memory + disk)
- [ ] Handle errors properly (capture stderr)
- [ ] Validate output size

### Phase 3: Audio Playback
- [ ] Implement oto player
- [ ] Add ring buffer for streaming
- [ ] Handle platform differences
- [ ] Test audio format compatibility

### Phase 4: Controller
- [ ] Coordinate components
- [ ] Manage state transitions
- [ ] Handle UI commands
- [ ] Implement error recovery

### Phase 5: UI Integration
- [ ] Add `--tts` flag
- [ ] Create Bubble Tea messages
- [ ] Add status display
- [ ] Implement keyboard controls

### Phase 6: Testing
- [ ] Unit tests for each component
- [ ] Race condition tests (100+ iterations)
- [ ] Integration tests
- [ ] Performance benchmarks

## 🔧 Configuration

```yaml
# ~/.config/glow/config.yml
tts:
  engine: piper
  cache_dir: ~/.cache/glow-tts
  
  # Cache configuration
  cache:
    memory_limit: 100MB     # L1 memory cache
    disk_limit: 1GB        # L2 disk cache
    session_limit: 50MB    # Session-specific cache
    ttl_days: 7           # Keep cached audio for 7 days
    cleanup_interval: 1h   # Run cleanup every hour
  
piper:
  model_path: ~/.local/share/piper/models/en_US-amy-medium.onnx
  sample_rate: 22050
  
google:
  api_key: ${GOOGLE_TTS_API_KEY}
  voice: en-US-Wavenet-F
```

## 📊 Performance Targets

- **First audio**: <500ms (new), <5ms (cached)
- **Cache hit rate**: >80% combined (L1+L2)
- **Memory usage**: <75MB total (excluding cache)
- **Cache memory**: <100MB (L1) + session
- **Disk usage**: <1GB for cache
- **Process spawn**: ~100ms per synthesis
- **No UI blocking**: All synthesis async
- **Cache cleanup**: <100ms per run

## 🧪 Testing Strategy

### Critical Test: Race Condition

```go
func TestNoRaceCondition(t *testing.T) {
    engine := NewPiperEngine("model.onnx")
    
    // Must pass 100+ times
    for i := 0; i < 100; i++ {
        audio, err := engine.Synthesize(ctx, "test")
        require.NoError(t, err)
        require.NotEmpty(t, audio)
    }
}
```

### Benchmark

```go
func BenchmarkSynthesisWithCache(b *testing.B) {
    engine := NewPiperEngine("model.onnx")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Should hit cache after first iteration
        engine.Synthesize(ctx, "benchmark text")
    }
}
```

## 🚫 Common Pitfalls to Avoid

1. **Using StdinPipe()** - Causes race condition
2. **Using goroutines in Bubble Tea** - Breaks UI completely
3. **Not keeping audio data alive** - Static noise/crashes
4. **No timeout protection** - Hanging processes forever
5. **Long-running processes** - Unstable, memory leaks
6. **Process pools** - Over-complicated, still unreliable
7. **No caching** - Terrible performance
8. **Forgetting stderr** - Lost error messages
9. **Ignoring output validation** - Silent failures
10. **Not handling graceful shutdown** - Orphaned processes

## 💾 Cache Implementation Details

### Two-Level Cache Design

```go
// internal/cache/manager.go
type CacheManager struct {
    l1Memory    *MemoryCache  // Fast, limited size
    l2Disk      *DiskCache    // Slower, persistent
    session     *SessionCache // Current session only
    cleanupStop chan struct{}
}

func (cm *CacheManager) Get(key string) ([]byte, bool) {
    // Check L1 memory first
    if data, ok := cm.l1Memory.Get(key); ok {
        return data, true
    }
    
    // Check L2 disk
    if data, ok := cm.l2Disk.Get(key); ok {
        // Promote to L1
        cm.l1Memory.Put(key, data)
        return data, true
    }
    
    // Check session cache
    if data, ok := cm.session.Get(key); ok {
        return data, true
    }
    
    return nil, false
}
```

### Cleanup Strategy

```go
func (cm *CacheManager) StartCleanup() {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for {
            select {
            case <-ticker.C:
                cm.performCleanup()
            case <-cm.cleanupStop:
                ticker.Stop()
                return
            }
        }
    }()
}

func (cm *CacheManager) performCleanup() {
    // Remove expired entries (>7 days)
    cm.l2Disk.RemoveExpired(7 * 24 * time.Hour)
    
    // Enforce size limits with smart eviction
    if cm.l2Disk.Size() > 1*GB {
        cm.l2Disk.EvictLRU()
    }
    
    // Clear old session data
    cm.session.ClearIfStale(24 * time.Hour)
}
```

### Key Generation

```go
func GenerateCacheKey(text, voice string, speed float64) string {
    // Normalize text (trim, lowercase for consistency)
    normalized := strings.ToLower(strings.TrimSpace(text))
    
    // Create unique key
    data := fmt.Sprintf("%s|%s|%.2f", normalized, voice, speed)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:16]) // Use first 16 bytes
}
```

## ✅ Best Practices

1. **Always cache** - 80% of content is repeated
2. **Implement two-level caching** - Memory for speed, disk for persistence
3. **Clean up periodically** - Don't let cache grow unbounded
4. **Use TTL policies** - 7 days is reasonable for most use cases
5. **Validate everything** - Check output size, format
6. **Handle errors gracefully** - Log, fallback, recover
7. **Test concurrency** - Run tests in parallel
8. **Profile performance** - Use pprof, benchmarks
9. **Document race condition** - Future developers need to know
10. **Keep it simple** - Optimal solution is simplest

## 🔗 Key Files to Reference

- **Subprocess standard**: `.strategic-core/standards/active/2025-01-10-0010-subprocess-handling.md`
- **Lessons learned**: `.strategic-core/ideas/tts-lessons-learned.md`
- **Technical spec**: `.strategic-core/specs/2025-01-10-tts-core-infrastructure/sub-specs/technical-spec.md`
- **Implementation tasks**: `.strategic-core/specs/2025-01-10-tts-core-infrastructure/tasks.md`

## 💡 Remember

The experimental branch spent weeks debugging the stdin race condition. The solution is simple:

```go
cmd.Stdin = strings.NewReader(text)  // This one line saves weeks
cmd.Run()                            // Synchronous = reliable
```

Cache aggressively, keep it simple, and never use `StdinPipe()` with programs that read stdin immediately.

---

*When in doubt, refer to this guide. The optimal approach has been proven through painful experience.*