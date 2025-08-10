# Subprocess Handling Standards

> Best practices for spawning and managing subprocesses in Go
> Critical patterns to avoid race conditions and ensure reliability

## The Stdin Race Condition

### Critical Warning ⚠️

**Never use `StdinPipe()` with programs that read stdin immediately on startup.**

This pattern creates a race condition:

```go
// ❌ BROKEN - Race Condition
cmd := exec.Command("program", args...)
cmd.Start()                    // Program starts reading NOW
stdin, _ := cmd.StdinPipe()    // Too late!
stdin.Write(data)               // May write to nothing
```

### Why This Happens

1. **Timing Issue**: Process starts before pipe is ready
2. **Program Behavior**: Some programs (like Piper, ffmpeg) read stdin immediately
3. **Non-deterministic**: Works sometimes, fails unpredictably
4. **Platform Dependent**: Different timing on different OS

## Correct Patterns

### Pattern 1: Pre-configured stdin (Recommended)

```go
// ✅ CORRECT - No race possible
func RunWithInput(program string, input string) ([]byte, error) {
    cmd := exec.Command(program, args...)
    
    // Set stdin BEFORE starting
    cmd.Stdin = strings.NewReader(input)
    
    // Capture output
    var stdout bytes.Buffer
    cmd.Stdout = &stdout
    
    // Run synchronously
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("%s failed: %w", program, err)
    }
    
    return stdout.Bytes(), nil
}
```

### Pattern 2: Pre-filled pipe (For streaming)

```go
// ✅ CORRECT - Data ready before start
func StreamWithInput(program string, input []byte) (<-chan []byte, error) {
    cmd := exec.Command(program, args...)
    
    // Create pipe with data
    reader, writer := io.Pipe()
    
    // Fill pipe BEFORE starting
    go func() {
        writer.Write(input)
        writer.Close()
    }()
    
    cmd.Stdin = reader
    
    // Start process
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    // ... handle output streaming
}
```

### Pattern 3: File-based I/O (When necessary)

```go
// ✅ CORRECT - No pipes at all
func RunWithFiles(program string, input string) ([]byte, error) {
    // Create temp input file
    inputFile, err := os.CreateTemp("", "input-*.txt")
    if err != nil {
        return nil, err
    }
    defer os.Remove(inputFile.Name())
    
    inputFile.WriteString(input)
    inputFile.Close()
    
    // Run with file arguments
    cmd := exec.Command(program, 
        "--input", inputFile.Name(),
        "--output", "-")  // stdout
    
    return cmd.Output()
}
```

## Process Lifecycle Management

### Starting Processes

```go
// Synchronous execution (preferred for one-shot operations)
func RunSync(program string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, program, args...)
    
    // Pre-configure everything
    cmd.Stdin = strings.NewReader(input)
    cmd.Stdout = &outputBuffer
    cmd.Stderr = &errorBuffer
    
    // Run and wait
    return cmd.Run()
}

// Asynchronous execution (when needed)
func RunAsync(program string) (*exec.Cmd, error) {
    cmd := exec.Command(program, args...)
    
    // Configure BEFORE starting
    cmd.Stdin = strings.NewReader(input)
    
    // Start asynchronously
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    // Caller must call cmd.Wait()
    return cmd, nil
}
```

### Process Termination

```go
func StopProcess(cmd *exec.Cmd) error {
    if cmd.Process == nil {
        return nil
    }
    
    // Try graceful shutdown first
    cmd.Process.Signal(os.Interrupt)
    
    // Wait with timeout
    done := make(chan error, 1)
    go func() {
        done <- cmd.Wait()
    }()
    
    select {
    case err := <-done:
        return err
    case <-time.After(5 * time.Second):
        // Force kill if not responsive
        cmd.Process.Kill()
        return fmt.Errorf("process killed after timeout")
    }
}
```

## Error Handling

### Comprehensive Error Checking

```go
func RunWithErrorHandling(program string, input string) ([]byte, error) {
    // Check program exists
    path, err := exec.LookPath(program)
    if err != nil {
        return nil, fmt.Errorf("program not found: %w", err)
    }
    
    cmd := exec.Command(path, args...)
    cmd.Stdin = strings.NewReader(input)
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    err = cmd.Run()
    
    // Check exit code
    if exitErr, ok := err.(*exec.ExitError); ok {
        return nil, fmt.Errorf("exit code %d: %s", 
            exitErr.ExitCode(), stderr.String())
    }
    
    if err != nil {
        return nil, fmt.Errorf("execution failed: %w", err)
    }
    
    // Validate output
    if stdout.Len() == 0 {
        return nil, fmt.Errorf("no output produced")
    }
    
    return stdout.Bytes(), nil
}
```

### Timeout Handling

```go
func RunWithTimeout(program string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, program, args...)
    cmd.Stdin = strings.NewReader(input)
    
    err := cmd.Run()
    
    // Check if timeout occurred
    if ctx.Err() == context.DeadlineExceeded {
        return fmt.Errorf("process timed out after %v", timeout)
    }
    
    return err
}
```

## Resource Management

### Preventing Resource Leaks

```go
type ProcessManager struct {
    processes map[string]*exec.Cmd
    mu        sync.Mutex
}

func (pm *ProcessManager) Start(name, program string) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    // Check if already running
    if cmd, exists := pm.processes[name]; exists {
        if cmd.Process != nil {
            return fmt.Errorf("process %s already running", name)
        }
    }
    
    cmd := exec.Command(program, args...)
    cmd.Stdin = strings.NewReader(input)
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    pm.processes[name] = cmd
    
    // Monitor for completion
    go func() {
        cmd.Wait()
        pm.mu.Lock()
        delete(pm.processes, name)
        pm.mu.Unlock()
    }()
    
    return nil
}

func (pm *ProcessManager) StopAll() {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    for name, cmd := range pm.processes {
        if cmd.Process != nil {
            cmd.Process.Kill()
        }
        delete(pm.processes, name)
    }
}
```

## Testing Subprocess Code

### Unit Testing with Mocks

```go
// Interface for testing
type CommandRunner interface {
    Run(program string, args []string, input string) ([]byte, error)
}

// Real implementation
type RealRunner struct{}

func (r RealRunner) Run(program string, args []string, input string) ([]byte, error) {
    cmd := exec.Command(program, args...)
    cmd.Stdin = strings.NewReader(input)
    return cmd.Output()
}

// Mock for testing
type MockRunner struct {
    Output []byte
    Error  error
}

func (m MockRunner) Run(program string, args []string, input string) ([]byte, error) {
    return m.Output, m.Error
}
```

### Integration Testing

```go
func TestSubprocessIntegration(t *testing.T) {
    // Skip if binary not available
    if _, err := exec.LookPath("piper"); err != nil {
        t.Skip("piper not installed")
    }
    
    // Test with real subprocess
    output, err := RunWithInput("piper", "test input")
    assert.NoError(t, err)
    assert.NotEmpty(t, output)
    
    // Test timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, "sleep", "1")
    err = cmd.Run()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "signal: killed")
}
```

### Race Condition Testing

```go
func TestNoRaceCondition(t *testing.T) {
    // Run multiple times to catch race
    for i := 0; i < 100; i++ {
        t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
            t.Parallel()
            
            output, err := RunWithInput("echo", "test")
            assert.NoError(t, err)
            assert.Equal(t, "test\n", string(output))
        })
    }
}
```

## Common Subprocess Programs

### Program-Specific Patterns

#### Piper (TTS)
```go
// Piper reads stdin immediately - use pre-configured stdin
cmd := exec.Command("piper", "--model", model, "--output-raw")
cmd.Stdin = strings.NewReader(text)
```

#### FFmpeg
```go
// FFmpeg also reads stdin immediately for pipe input
cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "pipe:1")
cmd.Stdin = bytes.NewReader(audioData)
```

#### Shell Commands
```go
// For shell commands, use sh -c
cmd := exec.Command("sh", "-c", "echo $input | grep pattern")
cmd.Env = append(os.Environ(), "input="+userInput)
```

## Debugging Subprocess Issues

### Logging and Diagnostics

```go
func DebugSubprocess(program string, args []string) {
    cmd := exec.Command(program, args...)
    cmd.Stdin = strings.NewReader(input)
    
    // Capture all outputs
    var stdout, stderr bytes.Buffer
    cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
    cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
    
    log.Printf("Running: %s %v", program, args)
    start := time.Now()
    
    err := cmd.Run()
    
    log.Printf("Duration: %v", time.Since(start))
    log.Printf("Exit code: %v", cmd.ProcessState.ExitCode())
    log.Printf("Stdout size: %d bytes", stdout.Len())
    log.Printf("Stderr: %s", stderr.String())
    
    if err != nil {
        log.Printf("Error: %v", err)
    }
}
```

## Best Practices Summary

### DO:
- ✅ Set stdin before starting process
- ✅ Use `cmd.Run()` for synchronous operations
- ✅ Handle timeouts with context
- ✅ Capture both stdout and stderr
- ✅ Check process existence before running
- ✅ Clean up resources properly

### DON'T:
- ❌ Use `StdinPipe()` with immediate-reading programs
- ❌ Start process before configuring I/O
- ❌ Ignore stderr output
- ❌ Forget to handle process termination
- ❌ Leave zombie processes
- ❌ Assume process will exit cleanly

### REMEMBER:
- The stdin race condition is real and will bite you
- Test with multiple iterations to catch races
- Different programs have different stdin behavior
- Platform differences affect timing
- Simple solutions (pre-configured stdin) are often best

---

*Learn from the experimental branch: The stdin race condition caused weeks of debugging. Always use pre-configured stdin for subprocess communication.*