# Technical Specification - TTS Core Infrastructure

## Architecture Overview

### System Architecture

```
┌──────────────────────────────────────────┐
│            Glow UI (Bubble Tea)          │
│                                          │
│  ┌────────────┐        ┌──────────────┐ │
│  │   Pager    │←──────→│  TTS Status  │ │
│  │ (modified) │        │   (new)      │ │
│  └──────┬─────┘        └──────────────┘ │
└─────────┼────────────────────────────────┘
          │ Interface Boundary
          ▼
┌──────────────────────────────────────────┐
│           TTS Controller                  │
│         (tts/controller.go)              │
│  ┌─────────────────────────────────────┐ │
│  │  • State Management                 │ │
│  │  • Command Orchestration            │ │
│  │  • Error Handling                   │ │
│  └─────────────────────────────────────┘ │
└────┬────────┬──────────┬─────────┬──────┘
     │        │          │         │
     ▼        ▼          ▼         ▼
┌─────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Engine  │ │ Audio  │ │Sentence│ │  Sync  │
│ (Piper) │ │ Player │ │ Parser │ │Manager │
└─────────┘ └────────┘ └────────┘ └────────┘
```

### Directory Structure

```
tts/
├── controller.go        # Main TTS orchestrator
├── interfaces.go        # Public interfaces
├── messages.go         # Bubble Tea messages
├── state.go           # State management
├── engines/
│   ├── engine.go      # Engine interface
│   └── piper/
│       ├── piper.go   # Piper implementation
│       └── process.go # Process management
├── audio/
│   ├── player.go      # Audio player interface
│   ├── backend.go     # Platform backends
│   └── buffer.go      # Audio buffering
├── sentence/
│   ├── parser.go      # Sentence extraction
│   └── tracker.go     # Position tracking
└── sync/
    ├── manager.go     # Synchronization
    └── drift.go       # Drift correction
```

## Component Design

### TTS Controller

```go
// tts/controller.go
package tts

import (
    "sync"
    "time"
    tea "github.com/charmbracelet/bubbletea"
)

type Controller struct {
    // Core components
    engine   Engine
    player   AudioPlayer
    parser   SentenceParser
    sync     Synchronizer
    
    // State
    state    *State
    mu       sync.RWMutex
    
    // Content
    sentences []Sentence
    current   int
    
    // Control
    stopCh   chan struct{}
    pauseCh  chan struct{}
    
    // Configuration
    config   Config
}

type Config struct {
    Enabled     bool
    Engine      string
    PiperPath   string
    PiperModel  string
    BufferSize  int
    UpdateRate  time.Duration
}

// Public API
func New(config Config) (*Controller, error)
func (c *Controller) Start(content string) error
func (c *Controller) Stop() error
func (c *Controller) Pause() error
func (c *Controller) Resume() error
func (c *Controller) NextSentence() error
func (c *Controller) PrevSentence() error
func (c *Controller) GetState() State
func (c *Controller) Shutdown() error

// Bubble Tea Commands
func (c *Controller) PlayCmd() tea.Cmd
func (c *Controller) GenerateAudioCmd(text string) tea.Cmd
```

### Engine Interface

```go
// tts/engines/engine.go
package engines

type Engine interface {
    // Lifecycle
    Initialize(config EngineConfig) error
    Shutdown() error
    IsAvailable() bool
    
    // Audio generation
    GenerateAudio(text string) (*Audio, error)
    GenerateAudioStream(text string) (<-chan AudioChunk, error)
    
    // Configuration
    GetVoices() []Voice
    SetVoice(voice Voice) error
    GetCapabilities() Capabilities
}

type EngineConfig struct {
    Voice      string
    Rate       float32
    Pitch      float32
    Volume     float32
}

type Audio struct {
    Data       []byte
    Format     AudioFormat
    SampleRate int
    Channels   int
    Duration   time.Duration
}

type AudioFormat int
const (
    FormatPCM16 AudioFormat = iota
    FormatFloat32
    FormatMP3
)
```

### Piper Engine Implementation

```go
// tts/engines/piper/piper.go
package piper

import (
    "bufio"
    "encoding/binary"
    "fmt"
    "io"
    "os/exec"
    "sync"
)

type PiperEngine struct {
    // Process management
    cmd      *exec.Cmd
    stdin    io.WriteCloser
    stdout   io.ReadCloser
    stderr   io.ReadCloser
    
    // Configuration
    binary   string
    model    string
    config   EngineConfig
    
    // State
    mu       sync.Mutex
    running  bool
    
    // Channels
    errorCh  chan error
}

func New(binary, model string) (*PiperEngine, error) {
    engine := &PiperEngine{
        binary: binary,
        model:  model,
    }
    return engine, engine.start()
}

func (e *PiperEngine) start() error {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    if e.running {
        return nil
    }
    
    // Start Piper process
    e.cmd = exec.Command(e.binary,
        "--model", e.model,
        "--output-raw",
        "--quiet",
    )
    
    // Setup pipes
    stdin, err := e.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("stdin pipe: %w", err)
    }
    e.stdin = stdin
    
    stdout, err := e.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("stdout pipe: %w", err)
    }
    e.stdout = stdout
    
    // Start process
    if err := e.cmd.Start(); err != nil {
        return fmt.Errorf("start piper: %w", err)
    }
    
    e.running = true
    
    // Monitor process
    go e.monitor()
    
    return nil
}

func (e *PiperEngine) GenerateAudio(text string) (*Audio, error) {
    e.mu.Lock()
    defer e.mu.Unlock()
    
    if !e.running {
        return nil, ErrEngineNotRunning
    }
    
    // Send text to Piper
    if _, err := fmt.Fprintln(e.stdin, text); err != nil {
        return nil, fmt.Errorf("write to piper: %w", err)
    }
    
    // Read audio data
    audio := &Audio{
        Format:     FormatPCM16,
        SampleRate: 22050,
        Channels:   1,
    }
    
    // Read PCM data from stdout
    var buf []byte
    scanner := bufio.NewScanner(e.stdout)
    scanner.Split(scanAudioFrames)
    
    for scanner.Scan() {
        buf = append(buf, scanner.Bytes()...)
        
        // Check for sentence end marker
        if isSentenceComplete(scanner.Bytes()) {
            break
        }
    }
    
    audio.Data = buf
    audio.Duration = calculateDuration(len(buf), audio.SampleRate)
    
    return audio, nil
}
```

### Audio Player

```go
// tts/audio/player.go
package audio

import (
    "github.com/hajimehoshi/oto/v3"
    "sync"
    "time"
)

type Player struct {
    // Audio context
    context *oto.Context
    player  *oto.Player
    
    // State
    playing  bool
    paused   bool
    position time.Duration
    mu       sync.RWMutex
    
    // Buffer
    buffer   *AudioBuffer
    
    // Control
    stopCh   chan struct{}
    pauseCh  chan struct{}
}

func NewPlayer() (*Player, error) {
    // Initialize audio context
    op := &oto.NewContextOptions{
        SampleRate:   22050,
        ChannelCount: 1,
        Format:       oto.FormatSignedInt16LE,
    }
    
    context, ready, err := oto.NewContext(op)
    if err != nil {
        return nil, fmt.Errorf("audio context: %w", err)
    }
    <-ready
    
    return &Player{
        context: context,
        buffer:  NewAudioBuffer(3), // 3 sentence buffer
        stopCh:  make(chan struct{}),
        pauseCh: make(chan struct{}),
    }, nil
}

func (p *Player) Play(audio *Audio) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.playing && !p.paused {
        return ErrAlreadyPlaying
    }
    
    // Create new player for audio
    p.player = p.context.NewPlayer(audio.Data)
    
    p.playing = true
    p.paused = false
    
    // Start playback goroutine
    go p.playbackLoop(audio)
    
    return nil
}

func (p *Player) playbackLoop(audio *Audio) {
    defer func() {
        p.mu.Lock()
        p.playing = false
        p.position = 0
        p.mu.Unlock()
    }()
    
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    
    startTime := time.Now()
    
    for {
        select {
        case <-p.stopCh:
            p.player.Close()
            return
            
        case <-p.pauseCh:
            p.player.Pause()
            p.mu.Lock()
            p.paused = true
            p.mu.Unlock()
            
            <-p.pauseCh // Wait for resume
            p.player.Resume()
            p.mu.Lock()
            p.paused = false
            p.mu.Unlock()
            
        case <-ticker.C:
            // Update position
            p.mu.Lock()
            p.position = time.Since(startTime)
            p.mu.Unlock()
            
            // Check if finished
            if p.position >= audio.Duration {
                return
            }
        }
    }
}
```

### Sentence Parser

```go
// tts/sentence/parser.go
package sentence

import (
    "regexp"
    "strings"
)

type Parser struct {
    // Sentence detection
    sentenceRegex *regexp.Regexp
    
    // Options
    skipCodeBlocks bool
    skipURLs       bool
    minLength      int
}

type Sentence struct {
    Index    int
    Text     string
    Markdown string
    Start    int    // Character position in original
    End      int
    Duration time.Duration
}

func NewParser() *Parser {
    return &Parser{
        sentenceRegex:  regexp.MustCompile(`[.!?]+[\s\])"']*`),
        skipCodeBlocks: true,
        skipURLs:       true,
        minLength:      3,
    }
}

func (p *Parser) Parse(markdown string) []Sentence {
    // Strip markdown to plain text while tracking positions
    plainText, positionMap := p.stripMarkdown(markdown)
    
    // Find sentence boundaries
    boundaries := p.findBoundaries(plainText)
    
    // Create sentences
    sentences := make([]Sentence, 0, len(boundaries))
    
    for i, boundary := range boundaries {
        start := 0
        if i > 0 {
            start = boundaries[i-1].end
        }
        
        text := strings.TrimSpace(plainText[start:boundary.end])
        if len(text) < p.minLength {
            continue
        }
        
        sentence := Sentence{
            Index:    len(sentences),
            Text:     text,
            Markdown: markdown[positionMap[start]:positionMap[boundary.end]],
            Start:    positionMap[start],
            End:      positionMap[boundary.end],
            Duration: p.estimateDuration(text),
        }
        
        sentences = append(sentences, sentence)
    }
    
    return sentences
}

func (p *Parser) estimateDuration(text string) time.Duration {
    // Estimate ~150 words per minute
    words := len(strings.Fields(text))
    seconds := float64(words) * 60.0 / 150.0
    return time.Duration(seconds * float64(time.Second))
}
```

### Synchronization Manager

```go
// tts/sync/manager.go
package sync

import (
    "sync"
    "time"
)

type Manager struct {
    // Current state
    sentences     []Sentence
    currentIndex  int
    startTime     time.Time
    mu            sync.RWMutex
    
    // Timing
    ticker        *time.Ticker
    updateRate    time.Duration
    
    // Drift correction
    driftThreshold time.Duration
    lastCorrection time.Time
    
    // Callbacks
    onChangeCallbacks []func(int)
    
    // Control
    stopCh chan struct{}
}

func NewManager(updateRate time.Duration) *Manager {
    return &Manager{
        updateRate:     updateRate,
        driftThreshold: 500 * time.Millisecond,
        stopCh:        make(chan struct{}),
    }
}

func (m *Manager) Start(sentences []Sentence, player AudioPlayer) {
    m.mu.Lock()
    m.sentences = sentences
    m.currentIndex = 0
    m.startTime = time.Now()
    m.mu.Unlock()
    
    m.ticker = time.NewTicker(m.updateRate)
    
    go m.syncLoop(player)
}

func (m *Manager) syncLoop(player AudioPlayer) {
    defer m.ticker.Stop()
    
    for {
        select {
        case <-m.stopCh:
            return
            
        case <-m.ticker.C:
            m.update(player)
        }
    }
}

func (m *Manager) update(player AudioPlayer) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Get current audio position
    audioPos := player.GetPosition()
    
    // Calculate expected sentence
    expectedIndex := m.findSentenceAtPosition(audioPos)
    
    // Check if sentence changed
    if expectedIndex != m.currentIndex {
        m.currentIndex = expectedIndex
        m.notifyChange(expectedIndex)
    }
    
    // Check for drift
    if m.needsDriftCorrection(audioPos, expectedIndex) {
        m.correctDrift(player, expectedIndex)
    }
}

func (m *Manager) findSentenceAtPosition(position time.Duration) int {
    elapsed := time.Duration(0)
    
    for i, sentence := range m.sentences {
        elapsed += sentence.Duration
        if elapsed > position {
            return i
        }
    }
    
    return len(m.sentences) - 1
}
```

## Data Flow

### Audio Generation Pipeline

```
Text Input → Sentence Parser → Audio Generator → Buffer → Player
                ↓                    ↓              ↓        ↓
           Sentences[]          Piper Process   Queue    Speakers
                ↓                    ↓              ↓        ↓
           Tracker ←────────── Audio Data ────→ Buffer → Output
```

### Synchronization Flow

```
Audio Player → Position Updates → Sync Manager
                      ↓                ↓
                Current Time    Sentence Index
                      ↓                ↓
                  Compare ──────→ UI Update Message
                      ↓                ↓
                Drift Check       Highlight Change
```

## Integration Points

### Bubble Tea Integration

```go
// tts/messages.go
package tts

// Messages sent to UI
type PlayingMsg struct {
    Sentence int
    Total    int
}

type PausedMsg struct{}
type StoppedMsg struct{}
type ErrorMsg struct {
    Error error
}

type SentenceChangedMsg struct {
    Index    int
    Text     string
}

// Commands for async operations
func GenerateAudioCmd(engine Engine, text string) tea.Cmd {
    return func() tea.Msg {
        audio, err := engine.GenerateAudio(text)
        if err != nil {
            return ErrorMsg{Error: err}
        }
        return AudioGeneratedMsg{Audio: audio}
    }
}
```

### UI Pager Modification

```go
// ui/pager.go modifications (minimal)
type model struct {
    // ... existing fields ...
    
    // TTS addition (only field added)
    tts *tts.Controller
}

// In Update() method, add TTS message handling:
case tts.SentenceChangedMsg:
    m.highlightSentence(msg.Index)
    return m, nil

// Add highlighting method:
func (m *model) highlightSentence(index int) {
    // Apply highlight style to sentence at index
    // Using existing Glamour/Lipgloss styling
}
```

## Performance Considerations

### Concurrency Model

- Audio generation runs in separate goroutine
- Playback has dedicated goroutine
- Synchronization runs on timer-based goroutine
- All communication via channels or mutex-protected state

### Memory Management

```go
// Audio buffer pool for reuse
var audioBufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 32768) // 32KB buffers
    },
}

// Sentence cache to avoid re-parsing
type SentenceCache struct {
    cache map[string][]Sentence
    mu    sync.RWMutex
    maxSize int
}
```

### Resource Limits

- Maximum 3 sentences buffered ahead
- Audio buffers recycled via sync.Pool
- Piper process single instance per session
- Maximum 50MB memory for audio buffers

## Security Considerations

### Process Security

- Piper runs with minimal privileges
- Input sanitization before sending to Piper
- Process resource limits enforced
- No shell execution, direct process spawn

### Input Validation

```go
func sanitizeForTTS(text string) string {
    // Remove control characters
    text = regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllString(text, "")
    
    // Limit length
    if len(text) > 1000 {
        text = text[:1000]
    }
    
    return text
}
```

## Error Handling

### Error Categories

```go
type ErrorCategory int

const (
    ErrorEngine ErrorCategory = iota
    ErrorAudio
    ErrorParsing
    ErrorResource
)

type TTSError struct {
    Category    ErrorCategory
    Err         error
    Recoverable bool
}
```

### Recovery Strategies

1. **Engine Failure**: Restart process, fallback to visual-only
2. **Audio Failure**: Retry with different backend, disable TTS
3. **Parse Failure**: Skip problematic content, continue
4. **Resource Failure**: Clean up, reduce buffer size, retry

## Platform-Specific Considerations

### Linux
- Audio: ALSA/PulseAudio support via oto
- Process: Standard Unix process management
- Paths: Follow XDG specifications

### macOS
- Audio: CoreAudio support
- Process: Standard Unix with macOS specifics
- Paths: Follow macOS conventions

### Windows
- Audio: DirectSound/WASAPI support
- Process: Windows process creation
- Paths: Handle Windows path separators