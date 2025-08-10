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
}
```

**Responsibilities:**
- Cross-platform audio playback
- Non-blocking audio streaming
- Position tracking
- Volume control

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
- **LRU Cache** with 100MB limit
- **Key Format**: `hash(text + voice + speed)`
- **Compression**: zstd for cached audio
- **Preemptive Loading**: Next 3 sentences

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

// Commands
func StartTTSCmd(content string) tea.Cmd
func StopTTSCmd() tea.Cmd
func NavigateTTSCmd(direction Direction) tea.Cmd
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
// ✅ CORRECT - No race possible
func (e *PiperEngine) Synthesize(ctx context.Context, text string) ([]byte, error) {
    // Check cache first
    if audio := e.getFromCache(text); audio != nil {
        return audio, nil
    }
    
    cmd := exec.CommandContext(ctx, "piper",
        "--model", e.modelPath,
        "--config", e.configPath,
        "--output-raw")
    
    // Critical: Pre-set stdin before starting
    cmd.Stdin = strings.NewReader(text)
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Run synchronously
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("piper failed: %w, stderr: %s", err, stderr.String())
    }
    
    audio := stdout.Bytes()
    e.saveToCache(text, audio)
    return audio, nil
}
```

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