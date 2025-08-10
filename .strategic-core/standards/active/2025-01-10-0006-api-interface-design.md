# API & Interface Design Standards

> Clean interface definitions for TTS engines, IPC protocols, and queue systems
> Focus on extensibility, testability, and clear contracts

## Interface Design Principles

### Core Principles
1. **Interface Segregation** - Many specific interfaces over few general ones
2. **Dependency Inversion** - Depend on abstractions, not concretions
3. **Single Responsibility** - Each interface has one reason to change
4. **Explicit Contracts** - Clear inputs, outputs, and error conditions

## TTS Engine Interfaces

### Core Engine Interface
```go
// TTSEngine defines the contract for text-to-speech engines
type TTSEngine interface {
    // Synthesize converts text to audio data
    // Returns audio in PCM format (16-bit, mono, sample rate per config)
    Synthesize(ctx context.Context, text string) ([]byte, error)
    
    // GetInfo returns engine capabilities and configuration
    GetInfo() EngineInfo
    
    // Close releases any resources held by the engine
    Close() error
}

// EngineInfo describes engine capabilities
type EngineInfo struct {
    Name        string
    Version     string
    SampleRate  int
    Channels    int
    BitDepth    int
    MaxTextSize int
    IsOnline    bool
}
```

### Advanced Engine Features
```go
// ConfigurableEngine supports runtime configuration
type ConfigurableEngine interface {
    TTSEngine
    
    // Configure updates engine settings
    Configure(config EngineConfig) error
    
    // GetConfig returns current configuration
    GetConfig() EngineConfig
}

// EngineConfig holds engine-specific settings
type EngineConfig struct {
    Voice      string
    Speed      float64  // 0.5 to 2.0
    Pitch      float64  // 0.5 to 2.0
    Volume     float64  // 0.0 to 1.0
    Language   string
    SampleRate int
}

// StreamingEngine supports streaming synthesis
type StreamingEngine interface {
    TTSEngine
    
    // SynthesizeStream returns audio chunks as they're generated
    SynthesizeStream(ctx context.Context, text string) (<-chan AudioChunk, error)
}

// AudioChunk represents a piece of audio data
type AudioChunk struct {
    Data     []byte
    Duration time.Duration
    Final    bool  // Last chunk in stream
}
```

### Engine Factory Pattern
```go
// EngineFactory creates TTS engines
type EngineFactory interface {
    // CreateEngine creates a new engine instance
    CreateEngine(engineType string, config map[string]interface{}) (TTSEngine, error)
    
    // SupportedEngines returns list of available engine types
    SupportedEngines() []string
}

// EngineRegistry manages available engines
type EngineRegistry struct {
    factories map[string]EngineFactory
    mu        sync.RWMutex
}

func (r *EngineRegistry) Register(name string, factory EngineFactory) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.factories[name] = factory
}

func (r *EngineRegistry) Create(name string, config map[string]interface{}) (TTSEngine, error) {
    r.mu.RLock()
    factory, exists := r.factories[name]
    r.mu.RUnlock()
    
    if !exists {
        return nil, fmt.Errorf("unknown engine: %s", name)
    }
    
    return factory.CreateEngine(name, config)
}
```

## Audio Player Interfaces

### Basic Player
```go
// AudioPlayer defines audio playback interface
type AudioPlayer interface {
    // Play starts playing audio data
    Play(data []byte) error
    
    // Stop stops current playback
    Stop() error
    
    // Pause pauses playback
    Pause() error
    
    // Resume resumes paused playback
    Resume() error
    
    // IsPlaying returns true if audio is currently playing
    IsPlaying() bool
    
    // Close releases audio resources
    Close() error
}
```

### Advanced Player Features
```go
// StreamingPlayer supports streaming audio playback
type StreamingPlayer interface {
    AudioPlayer
    
    // PlayStream plays audio from a channel
    PlayStream(stream <-chan []byte) error
    
    // GetPosition returns current playback position
    GetPosition() time.Duration
    
    // Seek jumps to specific position
    Seek(position time.Duration) error
}

// BufferedPlayer provides buffering capabilities
type BufferedPlayer interface {
    AudioPlayer
    
    // QueueAudio adds audio to playback queue
    QueueAudio(data []byte) error
    
    // ClearQueue removes all queued audio
    ClearQueue() error
    
    // QueueSize returns number of queued items
    QueueSize() int
}
```

## Queue System Interfaces

### Generic Queue
```go
// Queue defines a generic queue interface
type Queue[T any] interface {
    // Enqueue adds an item to the queue
    Enqueue(item T) error
    
    // Dequeue removes and returns the next item
    Dequeue() (T, error)
    
    // Peek returns the next item without removing it
    Peek() (T, error)
    
    // Size returns the number of items in queue
    Size() int
    
    // IsEmpty returns true if queue is empty
    IsEmpty() bool
    
    // IsFull returns true if queue is at capacity
    IsFull() bool
    
    // Clear removes all items from queue
    Clear()
}
```

### Sentence Queue
```go
// SentenceQueue manages sentence processing
type SentenceQueue interface {
    // EnqueueSentence adds a sentence for processing
    EnqueueSentence(sentence Sentence) error
    
    // DequeueSentence gets next sentence to process
    DequeueSentence() (Sentence, error)
    
    // GetProcessing returns currently processing sentences
    GetProcessing() []Sentence
    
    // MarkComplete marks a sentence as completed
    MarkComplete(id string) error
    
    // GetStats returns queue statistics
    GetStats() QueueStats
}

// Sentence represents a text sentence for TTS
type Sentence struct {
    ID       string
    Text     string
    Position int      // Position in document
    Priority Priority // Processing priority
}

// Priority defines processing priority
type Priority int

const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityImmediate
)

// QueueStats provides queue metrics
type QueueStats struct {
    TotalEnqueued   int
    TotalProcessed  int
    CurrentSize     int
    AverageWaitTime time.Duration
}
```

## IPC Protocol Interfaces

### Message Protocol
```go
// IPCMessage defines inter-process communication message
type IPCMessage interface {
    // GetType returns the message type
    GetType() MessageType
    
    // GetPayload returns the message data
    GetPayload() interface{}
    
    // GetTimestamp returns when message was created
    GetTimestamp() time.Time
    
    // Serialize converts message to bytes
    Serialize() ([]byte, error)
}

// MessageType identifies the type of IPC message
type MessageType string

const (
    MessageTypeCommand  MessageType = "command"
    MessageTypeResponse MessageType = "response"
    MessageTypeEvent    MessageType = "event"
    MessageTypeError    MessageType = "error"
)

// IPCChannel defines bidirectional communication
type IPCChannel interface {
    // Send sends a message through the channel
    Send(msg IPCMessage) error
    
    // Receive receives a message from the channel
    Receive() (IPCMessage, error)
    
    // ReceiveWithTimeout receives with timeout
    ReceiveWithTimeout(timeout time.Duration) (IPCMessage, error)
    
    // Close closes the channel
    Close() error
}
```

### Command Messages
```go
// Command represents an IPC command
type Command struct {
    ID      string
    Action  string
    Params  map[string]interface{}
    Context context.Context
}

// CommandHandler processes commands
type CommandHandler interface {
    // HandleCommand processes a command and returns response
    HandleCommand(cmd Command) (Response, error)
    
    // SupportedCommands returns list of supported commands
    SupportedCommands() []string
}

// Response represents command response
type Response struct {
    ID      string
    Success bool
    Data    interface{}
    Error   string
}
```

## Cache Interfaces

### Basic Cache
```go
// Cache defines caching interface
type Cache[K comparable, V any] interface {
    // Get retrieves a value from cache
    Get(key K) (V, bool)
    
    // Set stores a value in cache
    Set(key K, value V) error
    
    // Delete removes a value from cache
    Delete(key K) error
    
    // Clear removes all values
    Clear() error
    
    // Size returns number of cached items
    Size() int
}
```

### Audio Cache
```go
// AudioCache specializes in caching audio data
type AudioCache interface {
    // GetAudio retrieves cached audio
    GetAudio(text string, config EngineConfig) ([]byte, bool)
    
    // SetAudio stores audio in cache
    SetAudio(text string, config EngineConfig, audio []byte) error
    
    // GetSize returns total cache size in bytes
    GetSize() int64
    
    // Evict removes least recently used items to free space
    Evict(bytes int64) error
    
    // SetMaxSize sets maximum cache size
    SetMaxSize(bytes int64)
}

// CacheKey generates cache keys for audio
type CacheKey struct {
    Text     string
    Voice    string
    Speed    float64
    Language string
}

func (k CacheKey) String() string {
    h := sha256.New()
    h.Write([]byte(fmt.Sprintf("%s:%s:%.2f:%s", 
        k.Text, k.Voice, k.Speed, k.Language)))
    return hex.EncodeToString(h.Sum(nil))
}
```

## Process Management Interfaces

### Process Controller
```go
// ProcessController manages background processes
type ProcessController interface {
    // Start starts the background process
    Start(ctx context.Context) error
    
    // Stop gracefully stops the process
    Stop() error
    
    // Restart restarts the process
    Restart() error
    
    // GetStatus returns current process status
    GetStatus() ProcessStatus
    
    // GetPID returns process ID
    GetPID() int
}

// ProcessStatus represents process state
type ProcessStatus struct {
    State     ProcessState
    PID       int
    StartTime time.Time
    Uptime    time.Duration
    Memory    int64  // bytes
    CPU       float64 // percentage
}

// ProcessState enumerates process states
type ProcessState int

const (
    ProcessStateStarting ProcessState = iota
    ProcessStateRunning
    ProcessStateStopping
    ProcessStateStopped
    ProcessStateError
)
```

### Service Lifecycle
```go
// Service defines a manageable service
type Service interface {
    // Start initializes and starts the service
    Start() error
    
    // Stop gracefully shuts down the service
    Stop() error
    
    // Health returns service health status
    Health() Health
}

// Health represents service health
type Health struct {
    Status    HealthStatus
    Message   string
    Timestamp time.Time
    Details   map[string]interface{}
}

// HealthStatus enumerates health states
type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)
```

## Error Handling

### Error Types
```go
// TTSError represents TTS-specific errors
type TTSError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e *TTSError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *TTSError) Unwrap() error {
    return e.Cause
}

// ErrorCode identifies error types
type ErrorCode string

const (
    ErrorCodeEngineFailure   ErrorCode = "ENGINE_FAILURE"
    ErrorCodeAudioFailure    ErrorCode = "AUDIO_FAILURE"
    ErrorCodeQueueFull       ErrorCode = "QUEUE_FULL"
    ErrorCodeInvalidInput    ErrorCode = "INVALID_INPUT"
    ErrorCodeTimeout         ErrorCode = "TIMEOUT"
    ErrorCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
)
```

## Interface Testing

### Mock Generation
```go
//go:generate mockgen -source=engine.go -destination=mocks/mock_engine.go -package=mocks

// Use mockgen to generate mocks for interfaces
```

### Interface Compliance
```go
// Compile-time interface compliance checks
var (
    _ TTSEngine = (*PiperEngine)(nil)
    _ TTSEngine = (*GoogleTTSEngine)(nil)
    _ AudioPlayer = (*OtoPlayer)(nil)
    _ Queue[Sentence] = (*SentenceQueueImpl)(nil)
)
```

## Best Practices

### Interface Guidelines
1. **Keep interfaces small** - 3-5 methods maximum
2. **Return errors explicitly** - No panic in interface methods
3. **Use context for cancellation** - First parameter when needed
4. **Document behavior** - Clear godoc comments
5. **Version interfaces** - Use V2, V3 suffixes for breaking changes

### Naming Conventions
- **Interfaces**: Noun + "er" suffix or descriptive noun
- **Methods**: Verb or verb phrase
- **Constants**: PascalCase for exported, camelCase for internal
- **Errors**: Err prefix for sentinel errors

---

*Design clean, testable interfaces for robust TTS integration.*