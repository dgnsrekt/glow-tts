# TTS Testing Guide

## Overview

The TTS package uses a flexible testing strategy that allows tests to run both locally with real audio hardware and in CI environments without audio dependencies.

## Audio Context Strategy

Instead of using complex build tags, we use an **audio context interface** that automatically detects the environment and provides the appropriate implementation:

- **Production Context**: Uses real audio hardware via the `oto` library
- **Mock Context**: Simulates audio operations without requiring hardware

## Running Tests

### Local Development (with audio hardware)

Run all tests with real audio:
```bash
go test ./pkg/tts/...
```

### CI Environment (without audio hardware)

The tests automatically detect CI environments and use mock audio:
```bash
CI=true go test ./pkg/tts/...
```

### Force Mock Audio

You can force mock audio usage even in local development:
```bash
MOCK_AUDIO=true go test ./pkg/tts/...
```

## Environment Detection

The system automatically detects CI environments by checking for:
- `CI=true`
- `GITHUB_ACTIONS=true`
- `GITLAB_CI=true`
- Other common CI environment variables

When detected, tests automatically use the mock audio context.

## Test Categories

### Unit Tests (Audio-Independent)
These tests work with both real and mock audio contexts:
- `cache_test.go` - Cache functionality tests
- `parser_test.go` - Text parsing and sentence splitting
- `speed_test.go` - Audio speed adjustment calculations
- `subprocess_test.go` - External process management

### Integration Tests (Audio-Aware)
These tests adapt to the available audio context:
- `player_test.go` - Audio playback tests (uses mock in CI)
- `queue_test.go` - Audio queue management (uses mock in CI)
- `controller_test.go` - TTS controller tests (uses mock in CI)
- `audio_context_test.go` - Audio context factory tests

## Mock Audio Features

The mock audio context provides:
- Simulated playback timing
- Operation counting (PlayCount, PauseCount, etc.)
- State tracking for test assertions
- No actual audio output
- Deterministic behavior for testing

## Testing in GitHub Actions

The CI workflows automatically set `CI=true`, causing all tests to use mock audio. This eliminates issues with:
- Missing ALSA libraries on Linux
- CoreAudio on macOS runners
- WASAPI on Windows runners

## Example Test Helper

Tests can use the helper function to set up the appropriate context:

```go
func TestMyFeature(t *testing.T) {
    ctx := setupTestAudioContext(t)
    defer cleanupTestAudioContext(t)
    
    // Your test code here
    // Will use mock in CI, real audio locally
}
```

## Running Specific Test Suites

```bash
# Run only cache tests
go test -v ./pkg/tts -run TestCache

# Run with race detector
go test -race ./pkg/tts/...

# Run with coverage
go test -cover ./pkg/tts/...

# Run in CI mode locally for testing
CI=true go test -v ./pkg/tts/...
```

## Debugging Test Failures

If tests fail in CI but pass locally:
1. Run locally with `CI=true` to reproduce the CI environment
2. Check debug logs for audio context selection
3. Verify mock context behavior matches expectations

## Performance Considerations

- Mock audio context has minimal overhead
- Tests run faster in CI without actual audio processing
- No audio hardware initialization delays