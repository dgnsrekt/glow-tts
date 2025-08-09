# Testing Strategy - Glow-TTS

> Comprehensive testing approach for TTS features
> Maintains Glow's quality while testing audio/timing components

## Testing Philosophy

### Core Principles

1. **Non-Breaking**: TTS tests should not break existing Glow tests
2. **Isolation**: TTS tests can run independently
3. **Mockable**: External dependencies (TTS engines, audio) must be mockable
4. **Fast**: Unit tests should run in milliseconds
5. **Comprehensive**: Cover timing, synchronization, and edge cases

## Test Organization

### Directory Structure

```
/
├── glow_test.go           # Existing Glow tests (preserve)
├── tts/
│   ├── controller_test.go # Controller unit tests
│   ├── engine_test.go     # Engine interface tests
│   ├── audio_test.go      # Audio playback tests
│   ├── sentence_test.go   # Sentence parsing tests
│   ├── sync_test.go       # Synchronization tests
│   ├── integration_test.go # Full TTS flow tests
│   └── testdata/          # Test fixtures
│       ├── sample.md      # Sample markdown files
│       ├── audio/         # Test audio files
│       └── config/        # Test configurations
└── e2e/                   # End-to-end tests
    └── tts_e2e_test.go    # Full application tests
```

## Testing Levels

### 1. Unit Tests

Focus on individual components in isolation:

```go
// tts/sentence_test.go
package tts

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestSentenceParser(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []string
    }{
        {
            name:  "simple sentences",
            input: "Hello world. How are you? I'm fine!",
            expected: []string{
                "Hello world.",
                "How are you?",
                "I'm fine!",
            },
        },
        {
            name:  "with markdown",
            input: "**Bold text.** _Italic text._ [Link](url).",
            expected: []string{
                "Bold text.",
                "Italic text.",
                "Link.",
            },
        },
        {
            name:  "code blocks excluded",
            input: "Text before. ```code block``` Text after.",
            expected: []string{
                "Text before.",
                "Text after.",
            },
        },
    }
    
    parser := NewSentenceParser()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sentences := parser.Parse(tt.input)
            actual := make([]string, len(sentences))
            for i, s := range sentences {
                actual[i] = s.Text
            }
            assert.Equal(t, tt.expected, actual)
        })
    }
}
```

### 2. Integration Tests

Test component interactions:

```go
// tts/integration_test.go
package tts

func TestTTSControllerIntegration(t *testing.T) {
    // Use mock engine but real controller
    mockEngine := &MockEngine{
        generateFunc: func(text string) (Audio, error) {
            return Audio{
                Data:     []byte("mock audio"),
                Duration: time.Millisecond * 100,
            }, nil
        },
    }
    
    controller := &Controller{
        engine: mockEngine,
        player: NewMockPlayer(),
        parser: NewSentenceParser(),
    }
    
    // Test full flow
    content := "First sentence. Second sentence. Third sentence."
    err := controller.Start(content)
    assert.NoError(t, err)
    
    // Verify state progression
    assert.Equal(t, StatePlaying, controller.GetState().State)
    
    // Test navigation
    err = controller.NextSentence()
    assert.NoError(t, err)
    assert.Equal(t, 1, controller.GetState().Sentence)
    
    // Test pause/resume
    err = controller.Pause()
    assert.NoError(t, err)
    assert.Equal(t, StatePaused, controller.GetState().State)
    
    err = controller.Resume()
    assert.NoError(t, err)
    assert.Equal(t, StatePlaying, controller.GetState().State)
    
    // Cleanup
    controller.Stop()
}
```

### 3. End-to-End Tests

Test the complete application flow:

```go
// e2e/tts_e2e_test.go
//go:build e2e

package e2e

func TestTTSEndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }
    
    // Check prerequisites
    if !isPiperAvailable() {
        t.Skip("Piper TTS not available for E2E test")
    }
    
    // Create test markdown file
    testFile := createTestMarkdown(t, `
# Test Document

This is the first paragraph. It has multiple sentences.

This is the second paragraph. It also has sentences.
    `)
    defer os.Remove(testFile)
    
    // Run Glow with TTS
    app := teatest.NewTestModel(t, ui.NewModel())
    
    // Open file
    app.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
    app.Send(testFile)
    
    // Enable TTS
    app.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
    
    // Start playback
    app.Send(tea.KeyMsg{Type: tea.KeySpace})
    
    // Verify TTS is playing
    teatest.WaitFor(t, app, func(model tea.Model) bool {
        m := model.(*ui.Model)
        return m.TTSState.Playing
    }, teatest.WithTimeout(5*time.Second))
    
    // Test navigation
    app.Send(tea.KeyMsg{Type: tea.KeyRight})
    
    // Verify sentence changed
    teatest.WaitFor(t, app, func(model tea.Model) bool {
        m := model.(*ui.Model)
        return m.TTSState.Sentence == 1
    }, teatest.WithTimeout(2*time.Second))
}
```

## Mock Implementations

### Mock Engine

```go
// tts/testing/mocks.go
package testing

type MockEngine struct {
    generateFunc   func(string) (Audio, error)
    availableCalls int
    configureFunc  func(EngineOptions) error
}

func (m *MockEngine) GenerateAudio(text string) (Audio, error) {
    m.availableCalls++
    if m.generateFunc != nil {
        return m.generateFunc(text)
    }
    // Default mock audio
    return Audio{
        Data:     generateSilence(len(text) * 100), // 100ms per char
        Duration: time.Duration(len(text)) * 100 * time.Millisecond,
        Format:   FormatPCM16,
    }, nil
}

func (m *MockEngine) IsAvailable() bool {
    return true
}

func (m *MockEngine) Configure(opts EngineOptions) error {
    if m.configureFunc != nil {
        return m.configureFunc(opts)
    }
    return nil
}

func (m *MockEngine) GetCallCount() int {
    return m.availableCalls
}
```

### Mock Audio Player

```go
type MockPlayer struct {
    playing   bool
    position  time.Duration
    playFunc  func(Audio) error
    mu        sync.Mutex
}

func (m *MockPlayer) Play(audio Audio) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if m.playFunc != nil {
        return m.playFunc(audio)
    }
    
    m.playing = true
    
    // Simulate playback timing
    go func() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        for m.position < audio.Duration {
            <-ticker.C
            m.mu.Lock()
            if !m.playing {
                m.mu.Unlock()
                return
            }
            m.position += 100 * time.Millisecond
            m.mu.Unlock()
        }
    }()
    
    return nil
}

func (m *MockPlayer) GetPosition() time.Duration {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.position
}
```

## Timing and Synchronization Tests

### Testing Synchronization

```go
// tts/sync_test.go
func TestSynchronization(t *testing.T) {
    sentences := []Sentence{
        {Index: 0, Text: "First.", Duration: 1 * time.Second},
        {Index: 1, Text: "Second.", Duration: 1 * time.Second},
        {Index: 2, Text: "Third.", Duration: 1 * time.Second},
    }
    
    player := &MockPlayer{}
    sync := NewSynchronizer()
    
    currentSentence := -1
    sync.OnSentenceChange(func(index int) {
        currentSentence = index
    })
    
    sync.Start(sentences, player)
    
    // Simulate playback progress
    testCases := []struct {
        position time.Duration
        expected int
    }{
        {0, 0},                        // Start of first
        {500 * time.Millisecond, 0},   // Middle of first
        {1 * time.Second, 1},          // Start of second
        {1500 * time.Millisecond, 1},  // Middle of second
        {2 * time.Second, 2},          // Start of third
    }
    
    for _, tc := range testCases {
        player.position = tc.position
        sync.Update()
        assert.Equal(t, tc.expected, currentSentence,
            "At position %v, expected sentence %d", tc.position, tc.expected)
    }
}
```

### Testing Drift Correction

```go
func TestDriftCorrection(t *testing.T) {
    sync := NewSynchronizer()
    sync.driftThreshold = 500 * time.Millisecond
    
    // Simulate drift
    actualPosition := 5 * time.Second
    expectedPosition := 4500 * time.Millisecond
    
    needsCorrection := sync.checkDrift(actualPosition, expectedPosition)
    assert.True(t, needsCorrection, "Should detect drift > 500ms")
    
    // Test correction
    corrected := sync.correctDrift(actualPosition, expectedPosition)
    assert.Equal(t, expectedPosition, corrected)
}
```

## Performance Tests

### Benchmark Tests

```go
// tts/bench_test.go
func BenchmarkSentenceParsing(b *testing.B) {
    parser := NewSentenceParser()
    content := loadLargeMarkdown() // 10KB markdown file
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = parser.Parse(content)
    }
}

func BenchmarkAudioGeneration(b *testing.B) {
    engine := NewMockEngine()
    text := "This is a test sentence for benchmarking."
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = engine.GenerateAudio(text)
    }
}

func BenchmarkSynchronization(b *testing.B) {
    sync := NewSynchronizer()
    sentences := generateTestSentences(100) // 100 sentences
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sync.findCurrentSentence(time.Duration(i) * time.Second)
    }
}
```

## Test Data Management

### Fixtures

```go
// tts/testdata/fixtures.go
package testdata

var (
    SimpleMarkdown = `
# Simple Document

This is a paragraph. It has sentences.

## Section

Another paragraph here.
`

    ComplexMarkdown = `
# Complex Document

**Bold text.** _Italic text._ ` + "`code`" + `.

> Blockquote with text.

- List item one.
- List item two.

` + "```go" + `
func example() {
    // Code block
}
` + "```" + `

More text after code.
`
)

// Audio test data
func GenerateTestAudio(duration time.Duration) []byte {
    sampleRate := 16000
    samples := int(duration.Seconds() * float64(sampleRate))
    audio := make([]byte, samples*2) // 16-bit audio
    
    // Generate sine wave
    for i := 0; i < samples; i++ {
        value := int16(math.Sin(2*math.Pi*440*float64(i)/float64(sampleRate)) * 32767)
        binary.LittleEndian.PutUint16(audio[i*2:], uint16(value))
    }
    
    return audio
}
```

## Error Testing

### Error Scenarios

```go
// tts/error_test.go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        setupFunc   func(*Controller)
        operation   func(*Controller) error
        expectError bool
        checkState  func(*Controller) bool
    }{
        {
            name: "engine not available",
            setupFunc: func(c *Controller) {
                c.engine = &MockEngine{
                    generateFunc: func(string) (Audio, error) {
                        return Audio{}, ErrEngineNotAvailable
                    },
                }
            },
            operation: func(c *Controller) error {
                return c.Start("Test text")
            },
            expectError: true,
            checkState: func(c *Controller) bool {
                return c.GetState().State == StateError
            },
        },
        {
            name: "audio device failure",
            setupFunc: func(c *Controller) {
                c.player = &MockPlayer{
                    playFunc: func(Audio) error {
                        return ErrAudioDeviceNotFound
                    },
                }
            },
            operation: func(c *Controller) error {
                return c.Start("Test text")
            },
            expectError: true,
            checkState: func(c *Controller) bool {
                // Should fallback gracefully
                return c.GetState().State == StateReady
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            controller := NewTestController()
            tt.setupFunc(controller)
            
            err := tt.operation(controller)
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            if tt.checkState != nil {
                assert.True(t, tt.checkState(controller))
            }
        })
    }
}
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/tts-tests.yml
name: TTS Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Run TTS Unit Tests
        run: |
          go test -v -race -coverprofile=coverage.out ./tts/...
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
          flags: tts-unit

  integration-tests:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Install Piper (Linux/Mac)
        if: matrix.os != 'windows-latest'
        run: |
          # Install Piper for testing
          ./scripts/install-piper.sh
      
      - name: Run Integration Tests
        run: |
          go test -v -tags=integration ./tts/...
```

## Test Coverage Requirements

### Coverage Goals

- **Unit Tests**: 80% coverage minimum
- **Integration Tests**: Cover all major flows
- **E2E Tests**: Cover critical user journeys

### Coverage Report

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./tts/...
go tool cover -html=coverage.out -o coverage.html

# Coverage by package
go test -cover ./tts/...
```

## Test Helpers

### Common Test Utilities

```go
// tts/testing/helpers.go
package testing

// Create test controller with all mocks
func NewTestController(t *testing.T) *Controller {
    t.Helper()
    
    return &Controller{
        engine: NewMockEngine(),
        player: NewMockPlayer(),
        parser: NewMockParser(),
        sync:   NewMockSynchronizer(),
        config: DefaultTestConfig(),
    }
}

// Wait for condition with timeout
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration) {
    t.Helper()
    
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatal("Timeout waiting for condition")
}

// Generate test markdown
func GenerateMarkdown(sentences int) string {
    var builder strings.Builder
    builder.WriteString("# Test Document\n\n")
    
    for i := 0; i < sentences; i++ {
        builder.WriteString(fmt.Sprintf("Sentence number %d. ", i+1))
        if i%3 == 2 {
            builder.WriteString("\n\n")
        }
    }
    
    return builder.String()
}
```