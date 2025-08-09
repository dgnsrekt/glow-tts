# TTS Architecture Standards - Glow-TTS

> Architectural patterns and design principles for TTS feature implementation
> Ensures clean integration while maintaining feature isolation

## Core Architecture Principles

### 1. Separation of Concerns

The TTS system is divided into distinct layers:

```
┌─────────────────────────────────────┐
│         UI Layer (Bubble Tea)       │  ← Minimal changes
├─────────────────────────────────────┤
│      TTS Controller (Interface)     │  ← Clean boundary
├─────────────────────────────────────┤
│         TTS Core Components         │  ← Isolated in tts/
│  ┌──────────┬──────────┬─────────┐ │
│  │ Sentence │  Audio   │ Sync    │ │
│  │ Parser   │ Playback │ Manager │ │
│  └──────────┴──────────┴─────────┘ │
├─────────────────────────────────────┤
│         TTS Engine Layer            │
│  ┌──────────────┬────────────────┐ │
│  │  Piper TTS   │  Google TTS    │ │
│  │   (Local)    │   (Cloud)       │ │
│  └──────────────┴────────────────┘ │
└─────────────────────────────────────┘
```

### 2. Interface-Driven Design

All major components communicate through interfaces:

```go
// Core interfaces in tts/interfaces.go
type Engine interface {
    GenerateAudio(text string) (Audio, error)
    Configure(options EngineOptions) error
    IsAvailable() bool
    Shutdown() error
}

type AudioPlayer interface {
    Play(audio Audio) error
    Pause() error
    Resume() error
    Stop() error
    GetPosition() time.Duration
}

type SentenceTracker interface {
    Parse(markdown string) []Sentence
    GetCurrent() *Sentence
    Next() bool
    Previous() bool
    JumpTo(index int) error
}

type Synchronizer interface {
    Start(sentences []Sentence, player AudioPlayer)
    GetCurrentSentence() int
    OnSentenceChange(callback func(int))
}
```

### 3. Event-Driven Communication

Use Bubble Tea's message system for UI updates:

```go
// TTS messages in tts/messages.go
type TTSMessages struct{}

// Sent when TTS state changes
type TTSStateChangedMsg struct {
    Playing  bool
    Sentence int
    Total    int
}

// Sent when sentence changes
type SentenceChangedMsg struct {
    Index    int
    Text     string
    Duration time.Duration
}

// Sent on TTS errors (non-fatal)
type TTSErrorMsg struct {
    Error   error
    Recoverable bool
}

// Commands that return messages
func generateAudioCmd(text string) tea.Cmd {
    return func() tea.Msg {
        audio, err := generateAudio(text)
        if err != nil {
            return TTSErrorMsg{Error: err, Recoverable: true}
        }
        return AudioGeneratedMsg{Audio: audio}
    }
}
```

## Component Architecture

### TTS Controller

The main orchestrator that coordinates all TTS operations:

```go
// tts/controller.go
type Controller struct {
    // Core components
    engine     Engine
    player     AudioPlayer
    parser     SentenceTracker
    sync       Synchronizer
    
    // State management
    state      State
    mu         sync.RWMutex
    
    // Configuration
    config     Config
    
    // Channels for async operations
    audioQueue chan Audio
    stopCh     chan struct{}
}

// Public API - used by UI layer
func (c *Controller) Start(content string) error
func (c *Controller) Stop() error
func (c *Controller) Pause() error
func (c *Controller) Resume() error
func (c *Controller) NextSentence() error
func (c *Controller) PreviousSentence() error
func (c *Controller) GetState() State
func (c *Controller) SetEngine(engine string) error
```

### Sentence Parser

Handles markdown parsing and sentence extraction:

```go
// tts/sentence/parser.go
type Parser struct {
    // Sentence detection
    boundaries []string // [. ! ? ...]
    
    // Markdown awareness
    skipCodeBlocks bool
    skipLinks      bool
    
    // Caching
    cache map[string][]Sentence
}

type Sentence struct {
    Index     int
    Text      string
    Start     int    // Character position in original
    End       int
    Markdown  string // Original markdown
    Duration  time.Duration // Estimated
}

// Parsing strategy
func (p *Parser) Parse(markdown string) []Sentence {
    // 1. Strip markdown to plain text
    // 2. Detect sentence boundaries
    // 3. Map back to original positions
    // 4. Estimate durations
    return sentences
}
```

### Audio System

Manages audio playback across platforms:

```go
// tts/audio/player.go
type Player struct {
    // Platform-specific implementation
    backend AudioBackend
    
    // Playback state
    current   *Audio
    position  time.Duration
    playing   bool
    
    // Buffering
    buffer    *ring.Ring
    bufferSize int
}

// Audio representation
type Audio struct {
    Data       []byte
    Format     Format
    SampleRate int
    Duration   time.Duration
    Metadata   map[string]interface{}
}

// Platform backends
type AudioBackend interface {
    Init() error
    Play(audio []byte) error
    Pause() error
    Resume() error
    Stop() error
    Close() error
}
```

### Synchronization Manager

Coordinates audio playback with visual highlighting:

```go
// tts/sync/manager.go
type Manager struct {
    // Timing
    ticker    *time.Ticker
    startTime time.Time
    
    // Tracking
    sentences []Sentence
    current   int
    
    // Callbacks
    onchange  []func(int)
    
    // Drift correction
    driftThreshold time.Duration
    lastCorrection time.Time
}

// Synchronization algorithm
func (m *Manager) sync() {
    // 1. Check audio position
    // 2. Calculate expected sentence
    // 3. Update if changed
    // 4. Correct drift if needed
}
```

## Engine Integration Patterns

### Piper TTS Integration

```go
// tts/engines/piper/piper.go
type PiperEngine struct {
    // Process management
    cmd        *exec.Cmd
    stdin      io.WriteCloser
    stdout     io.ReadCloser
    
    // Configuration
    binary     string
    model      string
    voice      string
    
    // Resource management
    mu         sync.Mutex
    running    bool
}

// Lifecycle management
func (e *PiperEngine) Start() error {
    e.cmd = exec.Command(e.binary, 
        "--model", e.model,
        "--output_raw",
    )
    // Setup pipes
    // Start process
    // Verify ready
}

func (e *PiperEngine) GenerateAudio(text string) (Audio, error) {
    // Send text to stdin
    // Read audio from stdout
    // Convert to Audio struct
}
```

### Google TTS Integration

```go
// tts/engines/google/google.go
type GoogleEngine struct {
    // API client
    client     *http.Client
    apiKey     string
    endpoint   string
    
    // Configuration
    voice      Voice
    audioConfig AudioConfig
    
    // Rate limiting
    limiter    *rate.Limiter
    
    // Caching
    cache      Cache
}

// API interaction
func (e *GoogleEngine) GenerateAudio(text string) (Audio, error) {
    // Check cache first
    if audio, ok := e.cache.Get(text); ok {
        return audio, nil
    }
    
    // Rate limit
    e.limiter.Wait(context.Background())
    
    // Build request
    req := TextToSpeechRequest{
        Input: SynthesisInput{Text: text},
        Voice: e.voice,
        AudioConfig: e.audioConfig,
    }
    
    // Send request
    // Parse response
    // Cache result
    // Return audio
}
```

## State Management

### TTS State Machine

```go
// tts/state.go
type State int

const (
    StateIdle State = iota
    StateInitializing
    StateReady
    StatePlaying
    StatePaused
    StateStopping
    StateError
)

type StateMachine struct {
    current State
    mu      sync.RWMutex
    
    // State transitions
    transitions map[State][]State
    
    // Callbacks
    onEnter map[State]func()
    onExit  map[State]func()
}

// Valid transitions
func init() {
    transitions = map[State][]State{
        StateIdle:         {StateInitializing},
        StateInitializing: {StateReady, StateError},
        StateReady:        {StatePlaying, StateIdle},
        StatePlaying:      {StatePaused, StateStopping, StateReady},
        StatePaused:       {StatePlaying, StateStopping},
        StateStopping:     {StateIdle},
        StateError:        {StateIdle, StateInitializing},
    }
}
```

## Error Handling Strategy

### Graceful Degradation

```go
// Error categories
type ErrorCategory int

const (
    ErrorEngine ErrorCategory = iota  // Engine not available
    ErrorAudio                        // Audio device issues
    ErrorNetwork                      // Network issues (cloud TTS)
    ErrorResource                     // Resource exhaustion
)

// Error handling
func (c *Controller) handleError(err error, category ErrorCategory) {
    switch category {
    case ErrorEngine:
        // Try fallback engine
        if c.tryFallbackEngine() {
            log.Warn("Switched to fallback engine")
            return
        }
        // Disable TTS
        c.disable("TTS engines unavailable")
        
    case ErrorAudio:
        // Retry with different backend
        if c.retryAudioBackend() {
            return
        }
        // Disable with message
        c.disable("Audio playback unavailable")
        
    case ErrorNetwork:
        // Switch to offline engine if available
        if c.switchToOffline() {
            return
        }
        // Continue with visual only
        
    case ErrorResource:
        // Clean up and retry
        c.cleanup()
        if c.retry() {
            return
        }
        // Disable feature
    }
}
```

## Performance Optimization

### Audio Pipeline

```go
// Buffered audio generation
type AudioPipeline struct {
    // Pipeline stages
    generator  chan string      // Text to generate
    processor  chan RawAudio    // Raw audio to process
    player     chan Audio       // Processed audio to play
    
    // Workers
    genWorkers  int
    procWorkers int
    
    // Buffering
    maxBuffer   int
    currentBuffer int
}

// Efficient pipeline
func (p *AudioPipeline) Start() {
    // Start generator workers
    for i := 0; i < p.genWorkers; i++ {
        go p.generateWorker()
    }
    
    // Start processor workers
    for i := 0; i < p.procWorkers; i++ {
        go p.processWorker()
    }
    
    // Start player
    go p.playWorker()
}
```

### Memory Management

```go
// Resource pooling
var audioBufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 16384) // 16KB buffers
    },
}

// Reuse buffers
func getAudioBuffer() []byte {
    return audioBufferPool.Get().([]byte)
}

func putAudioBuffer(buf []byte) {
    // Clear sensitive data
    for i := range buf {
        buf[i] = 0
    }
    audioBufferPool.Put(buf)
}
```

## Testing Architecture

### Mock Implementations

```go
// tts/testing/mocks.go
type MockEngine struct {
    GenerateFunc func(string) (Audio, error)
    Available    bool
}

func (m *MockEngine) GenerateAudio(text string) (Audio, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(text)
    }
    // Return test audio
    return Audio{
        Data: []byte("mock audio"),
        Duration: time.Second * time.Duration(len(text)/10),
    }, nil
}

// Test helpers
func NewTestController() *Controller {
    return &Controller{
        engine: &MockEngine{Available: true},
        player: &MockPlayer{},
        parser: &MockParser{},
    }
}
```

### Integration Testing

```go
// tts/integration_test.go
func TestTTSIntegration(t *testing.T) {
    // Skip if no TTS engine available
    if !isPiperAvailable() {
        t.Skip("Piper TTS not available")
    }
    
    // Full integration test
    controller := New(Config{
        Engine: "piper",
        Voice:  "test-voice",
    })
    
    // Test full flow
    err := controller.Start("Hello. World. Test.")
    require.NoError(t, err)
    
    // Verify state changes
    assert.Eventually(t, func() bool {
        return controller.GetState().Playing
    }, 5*time.Second, 100*time.Millisecond)
    
    // Test navigation
    err = controller.NextSentence()
    require.NoError(t, err)
    
    // Cleanup
    controller.Stop()
}
```

## Configuration Schema

### TTS Configuration Structure

```yaml
# Complete TTS configuration
tts:
  # Core settings
  enabled: true
  engine: "piper"  # piper|google|auto
  
  # Playback settings
  playback:
    rate: 1.0           # Speech rate multiplier
    volume: 0.8         # 0.0 to 1.0
    bufferSize: 3       # Sentences to buffer ahead
    
  # Synchronization
  sync:
    updateInterval: 100ms   # Highlight update frequency
    driftThreshold: 500ms   # Max allowed drift
    
  # Engine configurations
  engines:
    piper:
      binary: "piper"
      model: "en_US-lessac-medium"
      modelPath: "~/.local/share/piper/models"
      speaker: 0
      
    google:
      apiKey: "${GOOGLE_TTS_API_KEY}"  # From environment
      voice:
        languageCode: "en-US"
        name: "en-US-Neural2-F"
      audioConfig:
        audioEncoding: "MP3"
        speakingRate: 1.0
        pitch: 0.0
        
  # Feature flags
  features:
    sentenceHighlight: true
    autoPlay: false
    keyboardShortcuts: true
    statusDisplay: true
    
  # Error handling
  errors:
    fallbackToVisual: true
    retryAttempts: 3
    retryDelay: 1s
```

## Deployment Considerations

### Binary Distribution

```makefile
# Build with TTS support
build-with-tts:
	go build -tags tts -o glow-tts

# Build without TTS (lighter binary)
build-no-tts:
	go build -o glow

# Platform-specific builds
build-all:
	GOOS=linux GOARCH=amd64 go build -tags tts -o dist/glow-tts-linux-amd64
	GOOS=darwin GOARCH=arm64 go build -tags tts -o dist/glow-tts-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -tags tts -o dist/glow-tts-windows-amd64.exe
```

### Dependency Management

```bash
# Runtime dependencies check
check-deps:
	@echo "Checking TTS dependencies..."
	@command -v piper >/dev/null 2>&1 || echo "Warning: Piper not found"
	@echo "Audio system check..."
	@go run tools/check_audio.go
```

## Migration Path from Glow

### Phase 1: Non-Breaking Addition
- All TTS code in isolated `tts/` directory
- No modifications to core Glow behavior
- TTS disabled by default

### Phase 2: UI Integration Points
- Add hooks in pager for highlighting
- Extend keyboard handler for TTS keys
- Add status line for TTS state

### Phase 3: Configuration Extension
- Extend Viper config schema
- Add CLI flags for TTS
- Update help documentation

### Phase 4: Full Integration
- Enable TTS by default (if available)
- Add to standard builds
- Update all documentation