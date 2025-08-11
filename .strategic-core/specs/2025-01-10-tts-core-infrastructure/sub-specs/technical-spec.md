# Technical Specification: TTS Core Infrastructure

## Architecture Design

### Component Overview

The TTS system consists of five main components working in concert:

1. **TTS Controller** - Orchestrates the entire pipeline
2. **Sentence Parser** - Extracts speakable text from markdown
3. **Audio Queue** - Manages preprocessing and playback order
4. **TTS Engines** - Performs actual synthesis (Piper/Google)
5. **Audio Player** - Handles cross-platform playback

### Component Details

#### 1. TTS Controller (`internal/tts/controller.go`)

```go
type Controller struct {
    engine      TTSEngine
    queue       *AudioQueue
    player      AudioPlayer
    parser      *SentenceParser
    state       State
    config      Config
    mu          sync.RWMutex
    done        chan struct{}
}
```

**Responsibilities:**
- Manages lifecycle of all components
- Coordinates between UI and TTS subsystem
- Handles state transitions
- Processes commands from UI

**Key Methods:**
- `Start(ctx context.Context) error`
- `Stop() error`
- `ProcessDocument(content string) error`
- `SetPlaybackSpeed(speed float64) error`
- `NavigateSentence(direction Direction) error`

#### 2. Sentence Parser (`internal/tts/parser.go`)

```go
type SentenceParser struct {
    stripMarkdown bool
    skipCodeBlocks bool
    minLength     int
}
```

**Responsibilities:**
- Strip markdown formatting while preserving structure
- Split text into speakable sentences
- Handle edge cases (URLs, abbreviations, code blocks)
- Maintain sentence position mapping

**Key Methods:**
- `Parse(markdown string) []Sentence`
- `StripMarkdown(text string) string`
- `SplitSentences(text string) []string`

#### 3. Audio Queue (`internal/queue/audio_queue.go`)

```go
type AudioQueue struct {
    sentences   chan Sentence
    audio      chan AudioData
    processing map[string]*AudioData
    cache      AudioCache
    lookahead  int
}
```

**Responsibilities:**
- Maintain 2-3 sentence lookahead
- Prioritize user navigation
- Manage memory efficiently
- Coordinate with cache

**Key Methods:**
- `Enqueue(sentence Sentence) error`
- `Dequeue() (AudioData, error)`
- `Preprocess(count int) error`
- `Clear() error`

#### 4. TTS Engines

##### Piper Engine (`internal/tts/engines/piper.go`)

```go
type PiperEngine struct {
    modelPath  string
    configPath string
    voice      string
    cache      map[string][]byte
    cacheMu    sync.RWMutex
    config     PiperConfig
}
```

**Implementation:**
- Fresh process per synthesis request
- Pre-configured stdin to avoid race condition
- Synchronous execution with cmd.Run()
- Aggressive caching for performance
- **Critical**: Never uses StdinPipe()

##### Google TTS Engine (`internal/tts/engines/google.go`)

```go
type GoogleTTSEngine struct {
    client     *texttospeech.Client
    voice      *texttospeechpb.VoiceSelectionParams
    audioConfig *texttospeechpb.AudioConfig
    rateLimit  *rate.Limiter
}
```

**Implementation:**
- Uses Google Cloud TTS API
- Implements rate limiting
- Handles authentication
- Supports SSML markup

#### 5. Audio Player (`internal/audio/player.go`)

```go
type AudioPlayer struct {
    context    *oto.Context
    player     oto.Player
    buffer     *RingBuffer
    playing    atomic.Bool
    position   atomic.Int64
    // CRITICAL: Keep audio data alive during playback
    activeStream *AudioStream
}

// CRITICAL: OTO Memory Management Pattern
type AudioStream struct {
    data      []byte         // Must stay alive during playback!
    reader    *bytes.Reader  // Recreated from data
    player    oto.Player     // Active player
    position  int64
    closeOnce sync.Once
    mu        sync.RWMutex
}
```

**Responsibilities:**
- Cross-platform audio playback
- Non-blocking audio streaming
- Position tracking
- Volume control
- **CRITICAL: Maintain audio data references during playback**

**⚠️ OTO Memory Management Requirements:**

OTO streams audio data - it does NOT load everything into memory first. If the underlying data source is garbage collected or closed while playing, you'll get:
- Static noise instead of audio
- Complete silence
- Segmentation faults/crashes

```go
// ❌ BROKEN - Data will be GC'd, causing static/crashes
func (p *AudioPlayer) Play(audio []byte) error {
    reader := bytes.NewReader(audio)
    p.player = p.context.NewPlayer(reader)
    p.player.Play()
    return nil
    // audio goes out of scope, gets GC'd
    // reader now points to freed memory!
}

// ✅ CORRECT - Keep data alive during playback
func (p *AudioPlayer) Play(audio []byte) error {
    // Store reference to prevent GC
    p.activeStream = &AudioStream{
        data:   audio,  // Keep alive!
        reader: bytes.NewReader(audio),
    }
    
    p.player = p.context.NewPlayer(p.activeStream.reader)
    p.player.Play()
    return nil
}

func (p *AudioPlayer) Stop() {
    if p.activeStream != nil {
        p.player.Close()
        // NOW safe to release memory
        p.activeStream = nil
    }
}
```

## Data Flow

### 1. Document Processing Flow

```
Document → Parser → Sentences → Queue → Engine → Cache → Player
```

1. User opens markdown document
2. Parser extracts sentences
3. Sentences queued for processing
4. Engine synthesizes audio
5. Audio cached for reuse
6. Player outputs audio

### 2. Navigation Flow

```
User Input → UI → Controller → Queue (Priority) → Player
```

1. User presses navigation key
2. UI sends command to controller
3. Controller updates queue priority
4. Target sentence processed immediately
5. Player jumps to new position

### 3. Background Processing Flow

```
Queue → Lookahead → Engine → Cache (Async)
```

1. Queue monitors upcoming sentences
2. Initiates preprocessing for next 2-3
3. Engine synthesizes in background
4. Results stored in cache
5. Ready for instant playback

## State Management

### Controller States

```go
type State int

const (
    StateIdle State = iota
    StateInitializing
    StateReady
    StateProcessing
    StatePlaying
    StatePaused
    StateStopping
)
```

### State Transitions

```
Idle → Initializing → Ready → Processing → Playing ⟷ Paused
                                    ↓           ↓        ↓
                                    └─────→ Stopping → Idle
```

## Concurrency Model

### Goroutine Architecture

1. **Main Controller** - Coordinates all operations
2. **Queue Processor** - Manages sentence queue
3. **Engine Worker Pool** - Parallel synthesis (2-4 workers)
4. **Audio Streamer** - Feeds audio to player
5. **Cache Manager** - Background cache maintenance

### Synchronization

- **Channels** for inter-component communication
- **Mutexes** for shared state protection
- **Context** for cancellation propagation
- **WaitGroups** for lifecycle management

## Error Handling

### Error Categories

1. **Recoverable Errors**
   - Engine timeout → Retry with fallback
   - Cache miss → Synthesize on demand
   - Network failure → Use offline engine

2. **Fatal Errors**
   - Audio device unavailable
   - All engines failed
   - Out of memory

### Error Recovery Strategy

```go
type ErrorHandler struct {
    retryCount   int
    retryDelay   time.Duration
    fallbackEngine TTSEngine
}

func (h *ErrorHandler) Handle(err error) error {
    switch e := err.(type) {
    case *EngineError:
        return h.handleEngineError(e)
    case *AudioError:
        return h.handleAudioError(e)
    default:
        return err
    }
}
```

## Performance Optimizations

### Caching Strategy

#### Two-Level Cache Architecture

1. **Memory Cache (L1)**
   - **Size Limit**: 100MB
   - **Eviction**: LRU (Least Recently Used)
   - **Purpose**: Ultra-fast access for active session
   - **Hit Latency**: <1ms
   - **Typical Hit Rate**: 40-50%

2. **Disk Cache (L2)**
   - **Size Limit**: 1GB
   - **Location**: `~/.cache/glow-tts/audio/`
   - **TTL**: 7 days per entry
   - **Purpose**: Persistent storage across sessions
   - **Hit Latency**: ~10-20ms
   - **Typical Hit Rate**: 30-40%
   - **Compression**: zstd level 3

#### Cache Key Generation
```go
func generateCacheKey(text, voice string, speed float64) string {
    data := fmt.Sprintf("%s|%s|%.2f", text, voice, speed)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

#### Cache Cleanup Strategy

##### Cleanup Triggers
1. **Size-Based Eviction**
   - Memory: When approaching 100MB limit
   - Disk: When approaching 1GB limit
   - Uses smart scoring: `score = age × size / frequency`

2. **Time-Based Cleanup**
   - **Periodic**: Every hour, remove entries older than 7 days
   - **On Startup**: Clean expired entries
   - **On Shutdown**: Optional aggressive cleanup

3. **Session-Based Cleanup**
   - **Session Cache**: Separate 50MB allocation
   - **On Exit**: Clear session cache completely
   - **Crash Recovery**: Stale session detection and cleanup

##### Cleanup Implementation
```go
type CacheManager struct {
    memoryCache *LRUCache      // 100MB limit
    diskCache   *DiskCache     // 1GB limit
    sessionCache *SessionCache // 50MB limit
    
    cleanupTicker *time.Ticker // Hourly cleanup
    metrics      *CacheMetrics
}

func (c *CacheManager) startCleanupRoutine() {
    c.cleanupTicker = time.NewTicker(1 * time.Hour)
    go func() {
        for range c.cleanupTicker.C {
            c.cleanupExpired()
            c.enforeceSizeLimits()
        }
    }()
}

func (c *CacheManager) cleanupExpired() {
    cutoff := time.Now().Add(-7 * 24 * time.Hour)
    c.diskCache.RemoveOlderThan(cutoff)
}
```

#### Cache Performance Targets
- **Combined Hit Rate**: >80%
- **Memory Usage**: <100MB for cache
- **Disk Usage**: <1GB maximum
- **Cleanup Time**: <100ms
- **Zero Memory Leaks**: Validated with pprof

### Memory Management
- **Ring Buffers** for audio streaming
- **Pool Allocation** for temporary buffers
- **Lazy Loading** of voice models
- **Garbage Collection** tuning

### CPU Optimization
- **Parallel Processing** for batch synthesis
- **SIMD** for audio processing (where available)
- **Worker Pool** to limit concurrent operations

## Integration Points

### UI Integration

#### CRITICAL: Bubble Tea Command Pattern

**⚠️ NEVER use goroutines directly in Bubble Tea programs!**

Bubble Tea has its own scheduler for managing concurrency. Using goroutines directly bypasses this scheduler and causes:
- Race conditions with internal state
- UI freezes and unresponsive behavior
- Missed re-renders after state changes
- Unpredictable crashes

```go
// ❌ NEVER DO THIS - Breaks Bubble Tea completely
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case StartTTSMsg:
        go func() {  // CATASTROPHIC ERROR!
            audio := m.engine.Synthesize(text)
            m.audioReady <- audio
        }()
        return m, nil
}

// ✅ ALWAYS DO THIS - Use Commands for ALL async operations
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case StartTTSMsg:
        return m, synthesizeCmd(msg.Text)  // Return a command
    case AudioReadyMsg:
        m.audio = msg.Audio
        return m, playAudioCmd(m.audio)
}

// All async operations must be commands
func synthesizeCmd(text string) tea.Cmd {
    return func() tea.Msg {
        audio := engine.Synthesize(text)
        return AudioReadyMsg{Audio: audio}
    }
}

func playAudioCmd(audio []byte) tea.Cmd {
    return func() tea.Msg {
        player.Play(audio)
        return AudioPlayingMsg{}
    }
}
```

#### Messages and Commands

```go
// Bubble Tea messages
type TTSStatusMsg struct {
    State      State
    Sentence   string
    Position   int
    TotalCount int
}

type TTSErrorMsg struct {
    Error error
    Fatal bool
}

type AudioReadyMsg struct {
    Audio    []byte
    Sentence string
}

// Commands - ALL I/O operations must be commands
func StartTTSCmd(content string) tea.Cmd
func StopTTSCmd() tea.Cmd
func NavigateTTSCmd(direction Direction) tea.Cmd
func SynthesizeCmd(text string) tea.Cmd
func PlayAudioCmd(audio []byte) tea.Cmd
func CacheLoadCmd(key string) tea.Cmd
func CacheSaveCmd(key string, data []byte) tea.Cmd
```

### Configuration

```yaml
tts:
  engine: piper  # or "google"
  cache_dir: ~/.cache/glow-tts
  max_cache_size: 100MB
  lookahead: 3
  
piper:
  model_path: ~/.local/share/piper/models
  voice: en_US-amy-medium
  
google:
  api_key: ${GOOGLE_TTS_API_KEY}
  voice: en-US-Wavenet-F
  language: en-US
```

## Critical Implementation Patterns

### Avoiding the Stdin Race Condition

**NEVER DO THIS:**
```go
// ❌ BROKEN - Race condition
cmd := exec.Command("piper", args...)
cmd.Start()
stdin, _ := cmd.StdinPipe()
stdin.Write(text)  // Too late!
```

**ALWAYS DO THIS:**
```go
// ✅ COMPLETE - Handles ALL race conditions with timeout protection
func (e *PiperEngine) Synthesize(ctx context.Context, text string) ([]byte, error) {
    // Check cache first
    if audio := e.getFromCache(text); audio != nil {
        return audio, nil
    }
    
    // CRITICAL: Add timeout protection
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, "piper",
        "--model", e.modelPath,
        "--config", e.configPath,
        "--output-raw")
    
    // Critical: Pre-set stdin before starting
    cmd.Stdin = strings.NewReader(text)
    
    // Capture outputs before starting
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Use channel for completion notification
    done := make(chan error, 1)
    
    // Start process with proper synchronization
    go func() {
        done <- cmd.Run()
        close(done)
    }()
    
    // Wait with timeout protection
    select {
    case err := <-done:
        // Process completed normally
        if err != nil {
            return nil, fmt.Errorf("piper failed: %w, stderr: %s", 
                err, stderr.String())
        }
        
        audio := stdout.Bytes()
        
        // Validate output
        if len(audio) == 0 {
            return nil, fmt.Errorf("piper produced no audio")
        }
        if len(audio) > 10*1024*1024 { // 10MB sanity check
            return nil, fmt.Errorf("piper output too large: %d bytes", len(audio))
        }
        
        e.saveToCache(text, audio)
        return audio, nil
        
    case <-ctx.Done():
        // Timeout or cancellation
        // Try graceful shutdown first
        if cmd.Process != nil {
            cmd.Process.Signal(os.Interrupt)
            
            // Give it a moment to clean up
            select {
            case <-done:
                // Exited gracefully
            case <-time.After(100 * time.Millisecond):
                // Force kill
                cmd.Process.Kill()
                <-done // Wait for goroutine to finish
            }
        }
        
        return nil, fmt.Errorf("synthesis timeout after 5s: %w", ctx.Err())
    }
}
```

### Additional Race Condition Protections

Beyond the stdin race, we must protect against:

1. **Process Hang Protection**: Always use timeouts
2. **Wait/Write Race**: Complete all I/O before Wait()
3. **File Descriptor Reuse**: Never access cmd after Wait()
4. **Graceful Shutdown**: Try SIGINT before SIGKILL
5. **Output Validation**: Check for sane output sizes

## Security Considerations

### API Key Management
- Environment variable storage
- Never logged or displayed
- Encrypted in memory

### Cache Security
- Files created with 0600 permissions
- User-specific cache directory
- No PII in cache keys

### Process Isolation
- Fresh process per request (more secure)
- No long-running processes to exploit
- Limited lifetime per synthesis