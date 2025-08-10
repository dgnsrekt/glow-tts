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
    └── lru.go            # LRU cache
```

## ⚠️ Critical Lessons Learned

### The Stdin Race Condition

**Problem**: Piper reads stdin immediately on startup
```go
// ❌ BROKEN
cmd.Start()              // Piper starts reading NOW
stdin := cmd.StdinPipe() // Too late!
stdin.Write(text)        // Writes to nothing
```

**Solution**: Pre-configure stdin
```go
// ✅ CORRECT
cmd.Stdin = strings.NewReader(text)  // Ready before start
cmd.Run()                            // Synchronous, no race
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
  max_cache_size: 100MB
  
piper:
  model_path: ~/.local/share/piper/models/en_US-amy-medium.onnx
  sample_rate: 22050
  
google:
  api_key: ${GOOGLE_TTS_API_KEY}
  voice: en-US-Wavenet-F
```

## 📊 Performance Targets

- **First audio**: <500ms (new), <5ms (cached)
- **Cache hit rate**: >80%
- **Memory usage**: <75MB total
- **Process spawn**: ~100ms per synthesis
- **No UI blocking**: All synthesis async

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
2. **Long-running processes** - Unstable, memory leaks
3. **Process pools** - Over-complicated, still unreliable
4. **No caching** - Terrible performance
5. **Forgetting stderr** - Lost error messages
6. **No timeout** - Hanging processes
7. **Ignoring output validation** - Silent failures

## ✅ Best Practices

1. **Always cache** - 80% of content is repeated
2. **Validate everything** - Check output size, format
3. **Handle errors gracefully** - Log, fallback, recover
4. **Test concurrency** - Run tests in parallel
5. **Profile performance** - Use pprof, benchmarks
6. **Document race condition** - Future developers need to know
7. **Keep it simple** - Optimal solution is simplest

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