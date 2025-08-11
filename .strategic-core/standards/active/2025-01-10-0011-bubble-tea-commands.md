# Bubble Tea Command Patterns

## Critical Rule

**⚠️ NEVER use goroutines directly in Bubble Tea programs!**

Bubble Tea has its own scheduler for managing concurrency. Bypassing this scheduler with direct goroutines causes:
- Race conditions with internal state management
- UI freezes and unresponsive behavior  
- Missed re-renders after state changes
- Unpredictable crashes

## The Command Pattern

### What is a Command?

A Command (`tea.Cmd`) is a function that returns a message:
```go
type Cmd func() Msg
```

Commands are how Bubble Tea handles:
- I/O operations
- Network requests
- File operations
- Long-running computations
- ANY async operation

### Why Commands?

1. **Managed Concurrency**: Bubble Tea schedules commands in its own goroutine pool
2. **Predictable State**: All state changes happen in Update()
3. **Automatic Re-rendering**: UI updates after every Update() call
4. **No Race Conditions**: Serial message processing ensures safety

## Correct Patterns

### Basic Command Usage

```go
// ✅ CORRECT - Return a command for async work
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case StartWorkMsg:
        // Don't do the work here, return a command
        return m, doWorkCmd(msg.Data)
        
    case WorkDoneMsg:
        // Handle the result
        m.result = msg.Result
        return m, nil
    }
    return m, nil
}

// Command does the actual work
func doWorkCmd(data string) tea.Cmd {
    return func() tea.Msg {
        // This runs in Bubble Tea's goroutine pool
        result := performWork(data)
        return WorkDoneMsg{Result: result}
    }
}
```

### Multiple Commands

```go
// ✅ CORRECT - Batch multiple commands
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case StartMultipleMsg:
        return m, tea.Batch(
            loadDataCmd(),
            startAnimationCmd(),
            checkNetworkCmd(),
        )
    }
    return m, nil
}
```

### Sequential Commands

```go
// ✅ CORRECT - Chain commands via messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case Step1DoneMsg:
        // First step complete, start second
        return m, step2Cmd(msg.Data)
        
    case Step2DoneMsg:
        // Second step complete, start third
        return m, step3Cmd(msg.Data)
    }
    return m, nil
}
```

### File I/O Commands

```go
// ✅ CORRECT - File operations in commands
func readFileCmd(path string) tea.Cmd {
    return func() tea.Msg {
        data, err := os.ReadFile(path)
        if err != nil {
            return ErrorMsg{err}
        }
        return FileReadMsg{Data: data}
    }
}
```

## Anti-Patterns to Avoid

### Never Use Goroutines Directly

```go
// ❌ WRONG - Breaks Bubble Tea completely
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case StartMsg:
        go func() {  // NEVER DO THIS!
            result := doWork()
            // How do we update the model? We can't!
            m.result = result  // Race condition!
        }()
        return m, nil
    }
    return m, nil
}
```

### Never Use Channels for Communication

```go
// ❌ WRONG - Channels bypass Bubble Tea's message system
type Model struct {
    resultChan chan string  // DON'T DO THIS
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    go func() {
        m.resultChan <- doWork()  // Wrong!
    }()
    return m, nil
}
```

### Never Block in Update

```go
// ❌ WRONG - Blocks the UI
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case LoadMsg:
        // This blocks the entire UI!
        data := fetchFromNetwork()  // DON'T DO THIS
        m.data = data
        return m, nil
    }
    return m, nil
}
```

### Never Mutate State Outside Update

```go
// ❌ WRONG - State mutation outside Update
func (m *Model) backgroundWork() {
    ticker := time.NewTicker(time.Second)
    go func() {
        for range ticker.C {
            m.counter++  // Race condition!
        }
    }()
}
```

## Command Best Practices

### 1. Keep Commands Pure

Commands should not have side effects on the model:
```go
// ✅ Good - Command only returns data
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        data := fetchData()
        return DataMsg{data}
    }
}
```

### 2. Handle Errors via Messages

```go
// ✅ Good - Errors as messages
func riskyCmd() tea.Cmd {
    return func() tea.Msg {
        result, err := riskyOperation()
        if err != nil {
            return ErrorMsg{err}
        }
        return SuccessMsg{result}
    }
}
```

### 3. Use Context for Cancellation

```go
// ✅ Good - Cancellable commands
func longRunningCmd(ctx context.Context) tea.Cmd {
    return func() tea.Msg {
        select {
        case <-ctx.Done():
            return CancelledMsg{}
        case result := <-doWork():
            return ResultMsg{result}
        }
    }
}
```

### 4. Small, Focused Commands

```go
// ✅ Good - Each command does one thing
func loadUserCmd(id string) tea.Cmd { ... }
func loadPostsCmd(userID string) tea.Cmd { ... }
func loadCommentsCmd(postID string) tea.Cmd { ... }

// Not one giant command that does everything
```

## Testing Commands

Commands are easy to test:
```go
func TestFetchCommand(t *testing.T) {
    cmd := fetchDataCmd("test")
    msg := cmd()  // Execute the command
    
    switch m := msg.(type) {
    case DataMsg:
        // Assert on m.Data
    case ErrorMsg:
        t.Fatalf("unexpected error: %v", m.Error)
    }
}
```

## TTS-Specific Command Examples

```go
// Synthesis command
func synthesizeCmd(engine TTSEngine, text string) tea.Cmd {
    return func() tea.Msg {
        audio, err := engine.Synthesize(context.Background(), text)
        if err != nil {
            return TTSErrorMsg{Error: err}
        }
        return AudioReadyMsg{Audio: audio, Text: text}
    }
}

// Playback command
func playAudioCmd(player AudioPlayer, audio []byte) tea.Cmd {
    return func() tea.Msg {
        err := player.Play(audio)
        if err != nil {
            return PlaybackErrorMsg{Error: err}
        }
        return PlaybackStartedMsg{}
    }
}

// Cache command
func loadFromCacheCmd(cache Cache, key string) tea.Cmd {
    return func() tea.Msg {
        if data, ok := cache.Get(key); ok {
            return CacheHitMsg{Key: key, Data: data}
        }
        return CacheMissMsg{Key: key}
    }
}
```

## Summary

1. **ALWAYS** use Commands for async operations
2. **NEVER** use goroutines directly
3. **NEVER** block in Update()
4. **ALWAYS** handle errors via messages
5. **KEEP** commands small and focused
6. **TEST** commands independently

Following these patterns ensures your Bubble Tea application remains responsive, maintainable, and bug-free.