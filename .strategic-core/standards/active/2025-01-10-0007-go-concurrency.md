# Go Concurrency Standards

> Patterns and best practices for concurrent programming in Go
> Focus on goroutines, channels, synchronization, and process management

## Concurrency Principles

### Core Guidelines
1. **Don't communicate by sharing memory; share memory by communicating**
2. **Goroutines are cheap but not free** - Don't leak them
3. **Channels orchestrate; mutexes serialize**
4. **Make the zero value useful** - Especially for concurrent types
5. **Context is king** - Use it for cancellation and deadlines

## Goroutine Management

### Starting Goroutines
```go
// Always know how goroutines will end
type Worker struct {
    done chan struct{}
    wg   sync.WaitGroup
}

func (w *Worker) Start() {
    w.wg.Add(1)
    go func() {
        defer w.wg.Done()
        w.run()
    }()
}

func (w *Worker) run() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            w.doWork()
        case <-w.done:
            return
        }
    }
}

func (w *Worker) Stop() {
    close(w.done)
    w.wg.Wait()
}
```

### Goroutine Lifecycle
```go
// Pattern: Managed goroutine with context
func ProcessAudioStream(ctx context.Context, input <-chan []byte) error {
    g, ctx := errgroup.WithContext(ctx)
    
    // Producer
    g.Go(func() error {
        return produceAudio(ctx, input)
    })
    
    // Consumer
    g.Go(func() error {
        return consumeAudio(ctx)
    })
    
    // Wait for all goroutines
    return g.Wait()
}
```

### Preventing Goroutine Leaks
```go
// Bad: Potential goroutine leak
func bad() {
    ch := make(chan int)
    go func() {
        val := <-ch  // Blocks forever if ch is never written to
        fmt.Println(val)
    }()
    // Function returns, goroutine leaks
}

// Good: Goroutine can exit
func good(ctx context.Context) {
    ch := make(chan int)
    go func() {
        select {
        case val := <-ch:
            fmt.Println(val)
        case <-ctx.Done():
            return  // Exit on context cancellation
        }
    }()
}
```

## Channel Patterns

### Channel Direction
```go
// Use directional channels in function signatures
func producer() <-chan int {  // Read-only
    ch := make(chan int)
    go func() {
        defer close(ch)
        for i := 0; i < 10; i++ {
            ch <- i
        }
    }()
    return ch
}

func consumer(ch <-chan int) {  // Read-only
    for val := range ch {
        process(val)
    }
}

func processor(in <-chan int, out chan<- int) {  // Read and write
    for val := range in {
        out <- transform(val)
    }
}
```

### Channel Ownership
```go
// The goroutine that writes to a channel should close it
type AudioProcessor struct {
    input  chan []byte
    output chan []byte
}

func (p *AudioProcessor) Start() {
    p.output = make(chan []byte)
    go func() {
        defer close(p.output)  // Writer closes
        for data := range p.input {
            processed := p.process(data)
            p.output <- processed
        }
    }()
}
```

### Fan-In/Fan-Out
```go
// Fan-out: Distribute work
func fanOut(in <-chan int, workers int) []<-chan int {
    outs := make([]<-chan int, workers)
    for i := 0; i < workers; i++ {
        out := make(chan int)
        outs[i] = out
        
        go func() {
            defer close(out)
            for val := range in {
                out <- process(val)
            }
        }()
    }
    return outs
}

// Fan-in: Combine results
func fanIn(channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for val := range c {
                out <- val
            }
        }(ch)
    }
    
    go func() {
        wg.Wait()
        close(out)
    }()
    
    return out
}
```

### Pipeline Pattern
```go
// Pipeline stages for audio processing
func pipeline(ctx context.Context, input <-chan string) <-chan []byte {
    // Stage 1: Parse sentences
    sentences := parseSentences(ctx, input)
    
    // Stage 2: Synthesize audio
    audio := synthesizeAudio(ctx, sentences)
    
    // Stage 3: Process audio
    processed := processAudio(ctx, audio)
    
    return processed
}

func parseSentences(ctx context.Context, input <-chan string) <-chan Sentence {
    out := make(chan Sentence)
    go func() {
        defer close(out)
        for text := range input {
            select {
            case out <- Sentence{Text: text}:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}
```

## Synchronization Primitives

### Mutex Usage
```go
// Protect shared state with mutex
type SafeCache struct {
    mu    sync.RWMutex
    items map[string][]byte
}

func (c *SafeCache) Get(key string) ([]byte, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    val, ok := c.items[key]
    return val, ok
}

func (c *SafeCache) Set(key string, value []byte) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.items == nil {
        c.items = make(map[string][]byte)
    }
    c.items[key] = value
}

// For complex operations, use methods
func (c *SafeCache) UpdateIfExists(key string, updater func([]byte) []byte) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if val, ok := c.items[key]; ok {
        c.items[key] = updater(val)
        return true
    }
    return false
}
```

### Once for Initialization
```go
type TTSController struct {
    engine   TTSEngine
    initOnce sync.Once
    initErr  error
}

func (c *TTSController) Init() error {
    c.initOnce.Do(func() {
        c.engine, c.initErr = createEngine()
    })
    return c.initErr
}
```

### WaitGroup for Coordination
```go
func ProcessBatch(items []string) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(items))
    
    for _, item := range items {
        wg.Add(1)
        go func(text string) {
            defer wg.Done()
            if err := processItem(text); err != nil {
                errors <- err
            }
        }(item)
    }
    
    // Wait for all to complete
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        if err != nil {
            return err
        }
    }
    return nil
}
```

## Context Usage

### Context Propagation
```go
// Always pass context as first parameter
func (s *TTSService) ProcessDocument(ctx context.Context, doc string) error {
    // Create child context with timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    sentences := s.parseSentences(ctx, doc)
    
    for _, sentence := range sentences {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := s.processSentence(ctx, sentence); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

### Context Values
```go
// Define typed keys for context values
type contextKey string

const (
    requestIDKey contextKey = "requestID"
    userIDKey    contextKey = "userID"
)

func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
    if id, ok := ctx.Value(requestIDKey).(string); ok {
        return id
    }
    return ""
}
```

## Worker Pools

### Fixed Worker Pool
```go
type WorkerPool struct {
    workers   int
    tasks     chan Task
    results   chan Result
    done      chan struct{}
    wg        sync.WaitGroup
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers: workers,
        tasks:   make(chan Task, workers*2),
        results: make(chan Result, workers*2),
        done:    make(chan struct{}),
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    for {
        select {
        case task := <-p.tasks:
            result := p.process(task)
            select {
            case p.results <- result:
            case <-p.done:
                return
            }
        case <-p.done:
            return
        }
    }
}

func (p *WorkerPool) Stop() {
    close(p.done)
    p.wg.Wait()
}
```

### Dynamic Worker Pool
```go
type DynamicPool struct {
    maxWorkers int
    tasks      chan Task
    workers    int32
    mu         sync.Mutex
}

func (p *DynamicPool) Submit(task Task) {
    select {
    case p.tasks <- task:
        // Task queued
    default:
        // Queue full, spawn new worker if under limit
        if atomic.LoadInt32(&p.workers) < int32(p.maxWorkers) {
            p.spawnWorker()
            p.tasks <- task
        }
    }
}

func (p *DynamicPool) spawnWorker() {
    atomic.AddInt32(&p.workers, 1)
    go func() {
        defer atomic.AddInt32(&p.workers, -1)
        
        timeout := time.NewTimer(30 * time.Second)
        defer timeout.Stop()
        
        for {
            select {
            case task := <-p.tasks:
                p.process(task)
                timeout.Reset(30 * time.Second)
            case <-timeout.C:
                // Exit if idle for too long
                return
            }
        }
    }()
}
```

## Rate Limiting

### Token Bucket
```go
type RateLimiter struct {
    rate     int
    bucket   chan struct{}
    stop     chan struct{}
}

func NewRateLimiter(rate int) *RateLimiter {
    rl := &RateLimiter{
        rate:   rate,
        bucket: make(chan struct{}, rate),
        stop:   make(chan struct{}),
    }
    
    // Fill bucket initially
    for i := 0; i < rate; i++ {
        rl.bucket <- struct{}{}
    }
    
    // Refill bucket periodically
    go rl.refill()
    
    return rl
}

func (rl *RateLimiter) refill() {
    ticker := time.NewTicker(time.Second / time.Duration(rl.rate))
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            select {
            case rl.bucket <- struct{}{}:
            default:
                // Bucket full
            }
        case <-rl.stop:
            return
        }
    }
}

func (rl *RateLimiter) Allow() bool {
    select {
    case <-rl.bucket:
        return true
    default:
        return false
    }
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    select {
    case <-rl.bucket:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## Select Patterns

### Priority Select
```go
// Process high priority items first
func prioritySelect(high, low <-chan Item) Item {
    select {
    case item := <-high:
        return item
    default:
        select {
        case item := <-high:
            return item
        case item := <-low:
            return item
        }
    }
}
```

### Timeout Handling
```go
func processWithTimeout(ctx context.Context, data []byte) error {
    done := make(chan error, 1)
    
    go func() {
        done <- expensiveOperation(data)
    }()
    
    select {
    case err := <-done:
        return err
    case <-time.After(5 * time.Second):
        return errors.New("operation timed out")
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## Atomic Operations

### Atomic Counters
```go
type Stats struct {
    processed uint64
    errors    uint64
    inFlight  int32
}

func (s *Stats) IncrementProcessed() {
    atomic.AddUint64(&s.processed, 1)
}

func (s *Stats) IncrementErrors() {
    atomic.AddUint64(&s.errors, 1)
}

func (s *Stats) StartProcessing() {
    atomic.AddInt32(&s.inFlight, 1)
}

func (s *Stats) FinishProcessing() {
    atomic.AddInt32(&s.inFlight, -1)
}

func (s *Stats) GetStats() (processed, errors uint64, inFlight int32) {
    return atomic.LoadUint64(&s.processed),
           atomic.LoadUint64(&s.errors),
           atomic.LoadInt32(&s.inFlight)
}
```

### Atomic Value
```go
type Config struct {
    value atomic.Value // stores *ConfigData
}

type ConfigData struct {
    TTSEngine string
    Voice     string
    Speed     float64
}

func (c *Config) Load() *ConfigData {
    if v := c.value.Load(); v != nil {
        return v.(*ConfigData)
    }
    return &ConfigData{} // Return default
}

func (c *Config) Store(cfg *ConfigData) {
    c.value.Store(cfg)
}
```

## Process Management

### Subprocess Control
```go
type TTSProcess struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.ReadCloser
    mu     sync.Mutex
}

func (p *TTSProcess) Start(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.cmd = exec.CommandContext(ctx, "piper", "--model", "en_US")
    
    var err error
    p.stdin, err = p.cmd.StdinPipe()
    if err != nil {
        return err
    }
    
    p.stdout, err = p.cmd.StdoutPipe()
    if err != nil {
        return err
    }
    
    return p.cmd.Start()
}

func (p *TTSProcess) Stop() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.stdin != nil {
        p.stdin.Close()
    }
    
    if p.cmd != nil && p.cmd.Process != nil {
        // Graceful shutdown
        p.cmd.Process.Signal(os.Interrupt)
        
        done := make(chan error, 1)
        go func() {
            done <- p.cmd.Wait()
        }()
        
        select {
        case <-done:
            // Process exited
        case <-time.After(5 * time.Second):
            // Force kill
            p.cmd.Process.Kill()
        }
    }
    
    return nil
}
```

## Common Pitfalls

### Race Conditions
```go
// Bad: Race condition
type Counter struct {
    value int
}

func (c *Counter) Increment() {
    c.value++  // Not thread-safe!
}

// Good: Thread-safe
type SafeCounter struct {
    mu    sync.Mutex
    value int
}

func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}
```

### Deadlocks
```go
// Bad: Potential deadlock
func bad(ch1, ch2 chan int) {
    select {
    case v1 := <-ch1:
        ch2 <- v1  // Can block forever
    case v2 := <-ch2:
        ch1 <- v2  // Can block forever
    }
}

// Good: Non-blocking sends
func good(ch1, ch2 chan int) {
    select {
    case v1 := <-ch1:
        select {
        case ch2 <- v1:
        default:
            // Drop if can't send
        }
    case v2 := <-ch2:
        select {
        case ch1 <- v2:
        default:
            // Drop if can't send
        }
    }
}
```

## Testing Concurrent Code

### Race Detector
```bash
# Always test with race detector
go test -race ./...
```

### Concurrent Test Helpers
```go
func TestConcurrentCache(t *testing.T) {
    cache := NewSafeCache()
    
    // Run concurrent operations
    const goroutines = 100
    var wg sync.WaitGroup
    
    for i := 0; i < goroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            key := fmt.Sprintf("key-%d", id)
            value := []byte(fmt.Sprintf("value-%d", id))
            
            cache.Set(key, value)
            
            if got, ok := cache.Get(key); !ok || !bytes.Equal(got, value) {
                t.Errorf("Cache mismatch for %s", key)
            }
        }(i)
    }
    
    wg.Wait()
    
    if cache.Size() != goroutines {
        t.Errorf("Expected %d items, got %d", goroutines, cache.Size())
    }
}
```

---

*Master Go's concurrency primitives for robust, scalable TTS processing.*