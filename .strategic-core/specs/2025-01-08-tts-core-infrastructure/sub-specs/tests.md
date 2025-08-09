# Test Specification - TTS Core Infrastructure

## Test Strategy

### Testing Principles
- Test in isolation before integration
- Mock external dependencies (Piper, audio devices)
- Focus on timing and synchronization accuracy
- Ensure graceful failure handling
- Maintain fast test execution

### Test Coverage Goals
- Unit tests: 80% minimum coverage
- Integration tests: All major workflows
- E2E tests: Critical user journeys
- Performance tests: Timing-critical components

## Unit Test Scenarios

### Controller Tests

```go
// tts/controller_test.go

func TestControllerInitialization(t *testing.T) {
    tests := []struct {
        name      string
        config    Config
        wantError bool
    }{
        {
            name: "valid config with piper",
            config: Config{
                Enabled:    true,
                Engine:     "piper",
                PiperPath:  "/usr/bin/piper",
                PiperModel: "test-model",
            },
            wantError: false,
        },
        {
            name: "disabled tts",
            config: Config{
                Enabled: false,
            },
            wantError: false,
        },
        {
            name: "missing piper path",
            config: Config{
                Enabled: true,
                Engine:  "piper",
            },
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            controller, err := New(tt.config)
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, controller)
            }
        })
    }
}

func TestControllerPlayback(t *testing.T) {
    controller := NewMockController()
    content := "First sentence. Second sentence. Third sentence."
    
    // Start playback
    err := controller.Start(content)
    assert.NoError(t, err)
    assert.Equal(t, StatePlaying, controller.GetState().State)
    
    // Pause
    err = controller.Pause()
    assert.NoError(t, err)
    assert.Equal(t, StatePaused, controller.GetState().State)
    
    // Resume
    err = controller.Resume()
    assert.NoError(t, err)
    assert.Equal(t, StatePlaying, controller.GetState().State)
    
    // Stop
    err = controller.Stop()
    assert.NoError(t, err)
    assert.Equal(t, StateIdle, controller.GetState().State)
}

func TestControllerNavigation(t *testing.T) {
    controller := NewMockController()
    content := "One. Two. Three. Four. Five."
    
    err := controller.Start(content)
    require.NoError(t, err)
    
    // Initial position
    assert.Equal(t, 0, controller.GetState().Sentence)
    
    // Next sentence
    err = controller.NextSentence()
    assert.NoError(t, err)
    assert.Equal(t, 1, controller.GetState().Sentence)
    
    // Previous sentence
    err = controller.PrevSentence()
    assert.NoError(t, err)
    assert.Equal(t, 0, controller.GetState().Sentence)
    
    // Boundary test - can't go before first
    err = controller.PrevSentence()
    assert.Error(t, err)
    assert.Equal(t, 0, controller.GetState().Sentence)
    
    // Jump to last
    for i := 0; i < 4; i++ {
        controller.NextSentence()
    }
    assert.Equal(t, 4, controller.GetState().Sentence)
    
    // Boundary test - can't go past last
    err = controller.NextSentence()
    assert.Error(t, err)
}
```

### Sentence Parser Tests

```go
// tts/sentence/parser_test.go

func TestSentenceParsing(t *testing.T) {
    parser := NewParser()
    
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
            name:  "with markdown formatting",
            input: "**Bold text.** _Italic text._ `code`.",
            expected: []string{
                "Bold text.",
                "Italic text.",
                "code.",
            },
        },
        {
            name:  "multiline sentences",
            input: "This is a long sentence that\nspans multiple lines.",
            expected: []string{
                "This is a long sentence that spans multiple lines.",
            },
        },
        {
            name:  "abbreviations",
            input: "Dr. Smith works at U.S.A. Inc. He is great.",
            expected: []string{
                "Dr. Smith works at U.S.A. Inc.",
                "He is great.",
            },
        },
        {
            name:  "quotes and parentheses",
            input: `She said "Hello!" (quietly). Then left.`,
            expected: []string{
                `She said "Hello!" (quietly).`,
                "Then left.",
            },
        },
        {
            name:  "code blocks excluded",
            input: "Text before.\n```\ncode block\n```\nText after.",
            expected: []string{
                "Text before.",
                "Text after.",
            },
        },
        {
            name:  "bullet points",
            input: "List:\n- Item one.\n- Item two.\n\nDone.",
            expected: []string{
                "List:",
                "Item one.",
                "Item two.",
                "Done.",
            },
        },
    }
    
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

func TestDurationEstimation(t *testing.T) {
    parser := NewParser()
    
    tests := []struct {
        text             string
        expectedDuration time.Duration
        tolerance        time.Duration
    }{
        {
            text:             "Short sentence.",
            expectedDuration: 500 * time.Millisecond,
            tolerance:        200 * time.Millisecond,
        },
        {
            text:             "This is a medium length sentence with several words.",
            expectedDuration: 3 * time.Second,
            tolerance:        500 * time.Millisecond,
        },
        {
            text:             strings.Repeat("word ", 30) + ".",
            expectedDuration: 12 * time.Second,
            tolerance:        1 * time.Second,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.text[:min(20, len(tt.text))], func(t *testing.T) {
            duration := parser.estimateDuration(tt.text)
            diff := abs(duration - tt.expectedDuration)
            assert.LessOrEqual(t, diff, tt.tolerance)
        })
    }
}
```

### Synchronization Tests

```go
// tts/sync/manager_test.go

func TestSynchronization(t *testing.T) {
    sentences := []Sentence{
        {Index: 0, Text: "First.", Duration: 1 * time.Second},
        {Index: 1, Text: "Second.", Duration: 1 * time.Second},
        {Index: 2, Text: "Third.", Duration: 1 * time.Second},
    }
    
    player := NewMockPlayer()
    manager := NewManager(100 * time.Millisecond)
    
    currentIndex := -1
    manager.OnSentenceChange(func(index int) {
        currentIndex = index
    })
    
    manager.Start(sentences, player)
    
    // Test synchronization at various time points
    testCases := []struct {
        position time.Duration
        expected int
    }{
        {0 * time.Millisecond, 0},
        {500 * time.Millisecond, 0},
        {1000 * time.Millisecond, 1},
        {1500 * time.Millisecond, 1},
        {2000 * time.Millisecond, 2},
        {2500 * time.Millisecond, 2},
    }
    
    for _, tc := range testCases {
        player.SetPosition(tc.position)
        time.Sleep(150 * time.Millisecond) // Wait for update
        assert.Equal(t, tc.expected, currentIndex,
            "At position %v, expected sentence %d", tc.position, tc.expected)
    }
    
    manager.Stop()
}

func TestDriftCorrection(t *testing.T) {
    manager := NewManager(100 * time.Millisecond)
    manager.driftThreshold = 500 * time.Millisecond
    
    // Test drift detection
    tests := []struct {
        actual   time.Duration
        expected time.Duration
        hasDrift bool
    }{
        {5000 * time.Millisecond, 5100 * time.Millisecond, false}, // 100ms diff
        {5000 * time.Millisecond, 5400 * time.Millisecond, false}, // 400ms diff
        {5000 * time.Millisecond, 5600 * time.Millisecond, true},  // 600ms diff
        {5000 * time.Millisecond, 4300 * time.Millisecond, true},  // 700ms diff
    }
    
    for _, tt := range tests {
        drift := manager.calculateDrift(tt.actual, tt.expected)
        needsCorrection := abs(drift) > manager.driftThreshold
        assert.Equal(t, tt.hasDrift, needsCorrection)
    }
}
```

### Audio Player Tests

```go
// tts/audio/player_test.go

func TestAudioPlayback(t *testing.T) {
    player := NewMockPlayer()
    
    audio := &Audio{
        Data:     generateTestAudio(2 * time.Second),
        Duration: 2 * time.Second,
        Format:   FormatPCM16,
    }
    
    // Start playback
    err := player.Play(audio)
    assert.NoError(t, err)
    assert.True(t, player.IsPlaying())
    
    // Check position updates
    time.Sleep(500 * time.Millisecond)
    pos := player.GetPosition()
    assert.Greater(t, pos, 400*time.Millisecond)
    assert.Less(t, pos, 600*time.Millisecond)
    
    // Pause
    err = player.Pause()
    assert.NoError(t, err)
    pausedPos := player.GetPosition()
    
    time.Sleep(500 * time.Millisecond)
    assert.Equal(t, pausedPos, player.GetPosition()) // Position shouldn't change
    
    // Resume
    err = player.Resume()
    assert.NoError(t, err)
    
    time.Sleep(500 * time.Millisecond)
    assert.Greater(t, player.GetPosition(), pausedPos)
    
    // Stop
    err = player.Stop()
    assert.NoError(t, err)
    assert.False(t, player.IsPlaying())
    assert.Equal(t, time.Duration(0), player.GetPosition())
}

func TestAudioBuffering(t *testing.T) {
    buffer := NewAudioBuffer(3)
    
    // Add audio chunks
    for i := 0; i < 5; i++ {
        audio := &Audio{
            Data: []byte(fmt.Sprintf("audio-%d", i)),
        }
        buffer.Add(audio)
    }
    
    // Buffer should only keep last 3
    assert.Equal(t, 3, buffer.Size())
    
    // Get should return in order
    audio, ok := buffer.Get()
    assert.True(t, ok)
    assert.Equal(t, []byte("audio-2"), audio.Data)
    
    audio, ok = buffer.Get()
    assert.True(t, ok)
    assert.Equal(t, []byte("audio-3"), audio.Data)
}
```

## Integration Test Cases

### Piper Integration Tests

```go
// tts/engines/piper/piper_integration_test.go
//go:build integration

func TestPiperEngineIntegration(t *testing.T) {
    if !isPiperAvailable() {
        t.Skip("Piper not available")
    }
    
    engine, err := NewPiperEngine("piper", "en_US-lessac-medium")
    require.NoError(t, err)
    defer engine.Shutdown()
    
    // Test audio generation
    text := "Hello, this is a test."
    audio, err := engine.GenerateAudio(text)
    assert.NoError(t, err)
    assert.NotNil(t, audio)
    assert.Greater(t, len(audio.Data), 0)
    assert.Equal(t, FormatPCM16, audio.Format)
    assert.Equal(t, 22050, audio.SampleRate)
    
    // Test multiple generations
    for i := 0; i < 5; i++ {
        text := fmt.Sprintf("Sentence number %d.", i)
        audio, err := engine.GenerateAudio(text)
        assert.NoError(t, err)
        assert.NotNil(t, audio)
    }
    
    // Test error handling
    _, err = engine.GenerateAudio("")
    assert.Error(t, err)
    
    // Test long text
    longText := strings.Repeat("This is a long sentence. ", 20)
    audio, err = engine.GenerateAudio(longText)
    assert.NoError(t, err)
    assert.Greater(t, audio.Duration, 10*time.Second)
}

func TestPiperProcessManagement(t *testing.T) {
    if !isPiperAvailable() {
        t.Skip("Piper not available")
    }
    
    engine, err := NewPiperEngine("piper", "en_US-lessac-medium")
    require.NoError(t, err)
    
    // Verify process is running
    assert.True(t, engine.IsAvailable())
    
    // Test restart after crash
    engine.cmd.Process.Kill() // Simulate crash
    time.Sleep(100 * time.Millisecond)
    
    // Should auto-restart on next use
    _, err = engine.GenerateAudio("Test after crash.")
    assert.NoError(t, err)
    
    // Clean shutdown
    err = engine.Shutdown()
    assert.NoError(t, err)
    assert.False(t, engine.IsAvailable())
}
```

### Full TTS Flow Integration

```go
// tts/integration_test.go
//go:build integration

func TestFullTTSFlow(t *testing.T) {
    // Create controller with mock engine, real components
    config := Config{
        Enabled:    true,
        Engine:     "mock",
        BufferSize: 3,
        UpdateRate: 100 * time.Millisecond,
    }
    
    controller, err := New(config)
    require.NoError(t, err)
    defer controller.Shutdown()
    
    // Test content
    content := `
# Test Document

This is the first paragraph. It contains multiple sentences. Each one should be detected.

This is the second paragraph. It also has sentences. They should play in sequence.

## Section Two

Final paragraph here. Last sentence.
`
    
    // Start TTS
    err = controller.Start(content)
    assert.NoError(t, err)
    
    // Verify sentences were parsed
    state := controller.GetState()
    assert.Greater(t, state.TotalSentences, 5)
    assert.Equal(t, 0, state.Sentence)
    assert.Equal(t, StatePlaying, state.State)
    
    // Test navigation during playback
    time.Sleep(200 * time.Millisecond)
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
    
    // Test stop
    err = controller.Stop()
    assert.NoError(t, err)
    assert.Equal(t, StateIdle, controller.GetState().State)
    assert.Equal(t, 0, controller.GetState().Sentence)
}
```

## End-to-End Test Flows

### E2E Test Scenarios

```go
// e2e/tts_e2e_test.go
//go:build e2e

func TestE2EBasicPlayback(t *testing.T) {
    // Skip if no real TTS available
    if !isPiperAvailable() && !isGoogleTTSConfigured() {
        t.Skip("No TTS engine available for E2E test")
    }
    
    // Create temporary markdown file
    content := `
# E2E Test Document

This is a test of the TTS system. It should play this sentence.

Then it should play this one. And finally this one.
`
    
    tmpFile := createTempMarkdown(t, content)
    defer os.Remove(tmpFile)
    
    // Run Glow with TTS
    app := startGlowWithTTS(t, tmpFile)
    defer app.Shutdown()
    
    // Enable TTS
    app.SendKey('T')
    waitForTTSEnabled(t, app)
    
    // Start playback
    app.SendKey(' ')
    waitForPlaybackStarted(t, app)
    
    // Verify highlighting appears
    highlight := app.GetHighlightedText()
    assert.Contains(t, highlight, "This is a test")
    
    // Navigate to next sentence
    app.SendKey('→')
    time.Sleep(500 * time.Millisecond)
    
    highlight = app.GetHighlightedText()
    assert.Contains(t, highlight, "It should play this sentence")
    
    // Stop playback
    app.SendKey('S')
    waitForPlaybackStopped(t, app)
}

func TestE2EErrorRecovery(t *testing.T) {
    app := startGlowWithTTS(t, "test.md")
    defer app.Shutdown()
    
    // Try to enable TTS when engine not available
    simulateEngineFailure()
    
    app.SendKey('T')
    
    // Should show error message
    output := app.GetOutput()
    assert.Contains(t, output, "TTS unavailable")
    
    // Should still be able to use Glow normally
    app.SendKey('j') // Navigate down
    assert.NotPanics(t, func() {
        app.SendKey('q') // Quit
    })
}
```

## Performance Test Criteria

### Benchmarks

```go
// tts/bench_test.go

func BenchmarkSentenceParsing(b *testing.B) {
    parser := NewParser()
    content := loadTestMarkdown("large.md") // 10KB file
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = parser.Parse(content)
    }
}

func BenchmarkAudioGeneration(b *testing.B) {
    engine := NewMockEngine()
    text := "This is a benchmark test sentence."
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = engine.GenerateAudio(text)
    }
}

func BenchmarkSynchronizationUpdate(b *testing.B) {
    manager := NewManager(100 * time.Millisecond)
    sentences := generateTestSentences(100)
    player := NewMockPlayer()
    
    manager.Start(sentences, player)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        player.SetPosition(time.Duration(i%100) * time.Second)
        manager.update(player)
    }
}
```

### Performance Requirements

- Sentence parsing: <100ms for 10KB document
- Audio generation: <2s for 100-word sentence
- Synchronization update: <1ms per update
- Memory usage: <50MB for TTS components
- CPU usage: <10% during idle playback

## Test Data Requirements

### Test Fixtures

```
tts/testdata/
├── markdown/
│   ├── simple.md          # Basic sentences
│   ├── complex.md         # Markdown formatting
│   ├── large.md          # 10KB document
│   └── edge_cases.md     # Special characters
├── audio/
│   ├── test_pcm16.raw    # PCM16 sample
│   ├── test_float32.raw  # Float32 sample
│   └── silence.raw       # Silence for testing
└── config/
    ├── valid.yaml        # Valid configuration
    ├── invalid.yaml      # Invalid configuration
    └── minimal.yaml      # Minimal configuration
```

### Mock Data Generators

```go
// tts/testing/generators.go

func GenerateTestSentences(count int) []Sentence {
    sentences := make([]Sentence, count)
    for i := 0; i < count; i++ {
        sentences[i] = Sentence{
            Index:    i,
            Text:     fmt.Sprintf("Test sentence number %d.", i),
            Duration: time.Second,
        }
    }
    return sentences
}

func GenerateTestAudio(duration time.Duration) []byte {
    sampleRate := 22050
    samples := int(duration.Seconds() * float64(sampleRate))
    audio := make([]byte, samples*2) // 16-bit
    
    // Generate simple sine wave
    for i := 0; i < samples; i++ {
        value := int16(math.Sin(2*math.Pi*440*float64(i)/float64(sampleRate)) * 16384)
        binary.LittleEndian.PutUint16(audio[i*2:], uint16(value))
    }
    
    return audio
}

func GenerateTestMarkdown() string {
    return `
# Test Document

First paragraph with multiple sentences. Each sentence should be detected. This is the third sentence.

## Section Two

- Bullet point one.
- Bullet point two.
- Bullet point three.

### Subsection

Final paragraph. Last sentence here.
`
}
```

## Test Execution Plan

### Phase 1: Unit Tests
1. Run all unit tests with mocks
2. Verify 80% code coverage
3. Fix any failing tests

### Phase 2: Integration Tests
1. Test with real Piper if available
2. Test audio playback on platform
3. Test Bubble Tea integration

### Phase 3: E2E Tests
1. Full flow with real markdown
2. Error recovery scenarios
3. Performance validation

### Phase 4: Platform Testing
1. Linux (Ubuntu, Fedora, Arch)
2. macOS (Intel and Apple Silicon)
3. Windows (10 and 11)

## Continuous Integration

### GitHub Actions Configuration

```yaml
name: TTS Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run Unit Tests
        run: |
          go test -v -race -cover ./tts/...
      
  integration-tests:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Install Piper
        run: |
          ./scripts/install-piper-ci.sh
      - name: Run Integration Tests
        run: |
          go test -v -tags=integration ./tts/...
```