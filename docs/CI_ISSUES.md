# CI/CD Pipeline Issues Documentation

## Overview
This document catalogs all issues found in the GitHub Actions CI/CD pipelines for the glow-tts project as of 2025-08-15.

## Issue Categories

### 1. Build Workflow Issues

#### 1.1 Compilation Errors
- **File**: `main.go:438`
- **Error**: `fmt.Println arg list ends with redundant newline`
- **Details**: Line 438 has `fmt.Println("Checking all TTS dependencies...\n")` where the `\n` is redundant since Println already adds a newline
- **Affects**: All platforms (Windows, Linux, macOS)

#### 1.2 Govulncheck Security Vulnerability
- **Vulnerability ID**: GO-2025-3787
- **Package**: `github.com/go-viper/mapstructure/v2`
- **Current Version**: v2.2.1
- **Fixed Version**: v2.3.0
- **Impact**: May leak sensitive information in logs when processing malformed data
- **Usage**: `pkg/tts/config.go` via viper for configuration parsing
- **Severity**: Non-critical but should be updated

#### 1.3 Ruleguard Static Analysis Issues
- **Error**: `could not import C (no metadata for C)`
- **Location**: `github.com/ebitengine/oto/v3@v3.3.3/driver_unix.go:22:8`
- **Impact**: Analysis skipped for all packages
- **Cause**: CGO dependency in audio library

### 2. Race Condition Issues

#### 2.1 Audio Context Initialization Race
- **Platform**: macOS
- **Location**: `oto/v3` library (`driver_darwin.go:161`)
- **Description**: Concurrent read/write during audio context initialization
- **Affected Test**: `TestAudioContextInitialization`

#### 2.2 Queue Processing Races
- **Location**: `pkg/tts/queue.go:261`
- **Method**: `processTextQueue()`
- **Issue**: Multiple goroutines accessing `TTSAudioQueue` fields without proper synchronization
- **Affected Tests**:
  - `TestPlaybackControls` - Position not advancing after resume
  - `TestAddText` - Race condition during text addition
  - `TestQueueNavigation` - Race during navigation operations
  - Multiple "Synthesis queue full" warnings

### 3. Test Failures by Platform

#### 3.1 macOS Test Failures
```
--- FAIL: TestPlaybackControls (0.10s)
    player_test.go:174: Position should advance after resume
    testing.go:1617: race detected during execution of test
--- FAIL: TestAddText (0.11s)
    testing.go:1617: race detected during execution of test
--- FAIL: TestQueueNavigation (0.20s)
    testing.go:1617: race detected during execution of test
```

#### 3.2 Linux/Ubuntu Test Failures
- **ALSA Configuration Errors**:
  ```
  ALSA lib conf.c:5204:(_snd_config_evaluate) function snd_func_card_inum returned error: No such file or directory
  ALSA lib confmisc.c:422:(snd_func_concat) error evaluating strings
  ALSA lib conf.c:5727:(snd_config_expand) Evaluate error: No such file or directory
  ```
- **Failed Tests**:
  - TestPlaybackControls
  - TestPlaybackCompletion
  - TestAddText
  - TestQueueNavigation
  - TestLookaheadBuffer
  - TestQueueMemoryManagement
  - TestQueueState
  - TestWaitForReady
  - TestQueueMetrics
  - TestConcurrentAccess
  - TestQueueClear
  - TestStreamProcess
  - TestProcessReaderCleanup

#### 3.3 Windows Test Failures
- Build fails due to compilation error before tests can run
- Same `fmt.Println` redundant newline issue

### 4. CI Environment Limitations

#### 4.1 Audio Hardware Absence
- **Issue**: CI runners don't have actual sound hardware
- **Impact**: ALSA can't find sound card configuration on Linux
- **Symptoms**: Configuration file errors, missing audio devices
- **Affected**: All audio-dependent tests

#### 4.2 Platform-Specific Audio Systems
- **Linux**: Requires ALSA, expects physical sound cards
- **macOS**: CoreAudio available but has race conditions in oto library
- **Windows**: WASAPI/DirectSound available but build fails before testing

### 5. External Dependency Issues

#### 5.1 Audio Library (ebitengine/oto/v3)
- **Version**: v3.3.3
- **Issues**:
  - CGO dependency breaks static analysis tools
  - Race conditions in the library itself (not our code)
  - Platform-specific implementations have different behaviors
  - Requires platform-specific audio system dependencies

#### 5.2 Configuration Library (spf13/viper with go-viper/mapstructure)
- **Current**: mapstructure v2.2.1
- **Required**: mapstructure v2.3.0 (security fix)
- **Usage**: Configuration file parsing for TTS settings

### 6. Workflow Status Summary

| Workflow | Status | Issues |
|----------|--------|--------|
| Lint | ✅ Passing | None |
| Build - Govulncheck | ❌ Failing | Security vulnerability warning |
| Build - Linux | ❌ Failing | Compilation error, ALSA issues |
| Build - macOS | ❌ Failing | Compilation error, race conditions |
| Build - Windows | ❌ Failing | Compilation error |
| Coverage | ❌ Failing | Can't run due to build failures |
| Semgrep | ✅ Passing | None |
| Ruleguard | ❌ Failing | CGO import issues |

## Priority Classification

### Critical (Blocks all CI):
1. Fix `fmt.Println` redundant newline in main.go:438

### High (Security/Stability):
1. Update mapstructure to v2.3.0 for security fix
2. Fix race conditions in TTS queue processing

### Medium (Test reliability):
1. Implement mock audio context for CI testing
2. Add build tags to skip audio-dependent tests in CI

### Low (Nice to have):
1. Fix ruleguard CGO issues (may require excluding oto from analysis)
2. Optimize test execution time

## Recommendations

1. **Immediate fixes**:
   - Remove redundant `\n` from fmt.Println statements
   - Update mapstructure dependency

2. **Testing strategy**:
   - Consider using build tags to separate unit tests from integration tests
   - Mock audio subsystem for CI environments
   - Run audio integration tests only in environments with audio hardware

3. **Race condition fixes**:
   - Add proper mutex protection to queue operations
   - Review concurrent access patterns in TTS subsystem
   - Consider using sync.Map or channels for thread-safe communication

4. **CI environment adaptation**:
   - Detect CI environment and skip hardware-dependent tests
   - Use dummy audio drivers for testing logic without hardware
   - Separate unit tests (logic) from integration tests (hardware)

## Next Steps

1. Fix compilation error (quick win)
2. Update dependencies for security
3. Address race conditions with proper synchronization
4. Implement CI-friendly test strategy
5. Document which tests require actual hardware

---
*Generated: 2025-08-15*
*Status: Issues documented, awaiting implementation decisions*