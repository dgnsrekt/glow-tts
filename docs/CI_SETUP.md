# CI/CD Setup and Configuration Guide

This document provides comprehensive information about the CI/CD setup for the glow-tts project, including build tags, environment variables, test categories, and troubleshooting.

## Table of Contents

- [Overview](#overview)
- [Build Tag System](#build-tag-system)
- [Environment Variables](#environment-variables)
- [Test Categories](#test-categories)
- [CI Workflows](#ci-workflows)
- [Platform-Specific Configuration](#platform-specific-configuration)
- [Local Development Setup](#local-development-setup)
- [Running Test Suites](#running-test-suites)
- [Troubleshooting](#troubleshooting)
- [Maintenance](#maintenance)

## Overview

The glow-tts CI/CD system is designed to:
- Run fast unit tests on every push/PR
- Perform comprehensive integration testing when needed
- Support multiple platforms (Linux, macOS, Windows)
- Handle audio hardware dependencies gracefully
- Provide static analysis without CGO issues

## Build Tag System

### Available Build Tags

| Tag | Purpose | Usage |
|-----|---------|-------|
| `nocgo` | Excludes CGO-dependent files for static analysis | `go build -tags=nocgo` |
| `integration` | Includes integration tests requiring hardware | `go test -tags=integration` |
| `ci` | Optimizes for CI environment (deprecated, use env vars) | `go test -tags=ci` |

### How Build Tags Work

Build tags control which files are included during compilation:

```go
//go:build !nocgo
// +build !nocgo

// This file is excluded when -tags=nocgo is used
```

```go
//go:build nocgo
// +build nocgo

// This file is only included when -tags=nocgo is used
```

### Files Using Build Tags

**CGO-dependent files (excluded with `nocgo`):**
- `pkg/tts/player.go` - Audio playback using oto library
- `pkg/tts/audio_context_production.go` - Production audio context

**Stub implementations (included with `nocgo`):**
- `pkg/tts/player_nocgo.go` - Stub audio player for static analysis
- `pkg/tts/audio_context_production_nocgo.go` - Stub audio context

## Environment Variables

### CI Detection Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CI` | Indicates CI environment | `CI=true` |
| `GITHUB_ACTIONS` | GitHub Actions specific | `GITHUB_ACTIONS=true` |
| `GLOW_TTS_MOCK_AUDIO` | Force mock audio usage | `GLOW_TTS_MOCK_AUDIO=true` |

### TTS Configuration Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `GLOW_TTS_CACHE_MAX_SIZE` | Maximum cache size in bytes | 100MB | `GLOW_TTS_CACHE_MAX_SIZE=50000000` |
| `GLOW_TTS_PIPER_SPEED` | Piper TTS speech speed | 1.0 | `GLOW_TTS_PIPER_SPEED=1.5` |
| `GLOW_TTS_PIPER_MODEL` | Piper voice model | en_US-amy-medium | `GLOW_TTS_PIPER_MODEL=en_GB-alan-medium` |
| `GLOW_LOG_LEVEL` | Log verbosity | info | `GLOW_LOG_LEVEL=debug` |

### Testing Environment Variables

| Variable | Description | Usage |
|----------|-------------|-------|
| `GO_TEST_TIMEOUT` | Test timeout duration | `GO_TEST_TIMEOUT=10m` |
| `GO_TEST_PARALLEL` | Number of parallel tests | `GO_TEST_PARALLEL=4` |
| `MOCK_AUDIO` | Use mock audio in tests | `MOCK_AUDIO=true` |

## Test Categories

### Unit Tests

Fast, isolated tests that don't require external dependencies:
- Run on every push/PR
- Use mock audio contexts
- Complete in < 30 seconds
- Located throughout the codebase

**Run unit tests:**
```bash
go test -short -race ./...
```

### Integration Tests

Tests requiring real hardware or external services:
- Audio playback tests
- TTS engine integration
- End-to-end workflows
- May take several minutes

**Run integration tests:**
```bash
# With real audio hardware
GLOW_TTS_MOCK_AUDIO=false go test -v ./...

# Specific integration tests
go test -v -run TestIntegration ./...
```

### Platform Tests

Tests specific to OS/platform detection:
- OS identification
- Audio subsystem detection
- Platform-specific retry logic

**Run platform tests:**
```bash
go test -v -run "TestPlatform|TestDetect" ./pkg/tts/
```

### Performance Tests

Benchmarks and performance measurements:
- Cache performance
- TTS processing speed
- Memory usage

**Run benchmarks:**
```bash
go test -bench=. -benchmem ./pkg/tts/...
```

## CI Workflows

### Primary Workflows

1. **build.yml** - Build & Test
   - Triggers: Push to main/master/develop, PRs
   - Quick build verification
   - Smoke tests with mock audio
   - Security scanning

2. **test.yml** - Comprehensive Testing
   - Unit tests on all platforms
   - Optional integration tests
   - Platform detection tests
   - Code coverage reporting

3. **pr.yml** - Pull Request Checks
   - Fast feedback for PR authors
   - Linting and formatting
   - Quick unit tests
   - Build verification

4. **static-analysis.yml** - Code Quality
   - Multiple linters (golangci-lint, staticcheck, etc.)
   - Security analysis (gosec)
   - Uses `nocgo` tag to avoid CGO issues

5. **nightly.yml** - Comprehensive Nightly Tests
   - Full test suite
   - Multiple Go versions
   - Memory and race testing
   - Fuzz testing

### Workflow Matrix Strategy

All workflows use matrix builds for multi-platform testing:

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
    go-version: [stable]
```

## Platform-Specific Configuration

### Linux

- **CI Environment**: Ubuntu latest
- **Audio System**: ALSA (mock in CI)
- **Dependencies**: build-essential
- **Common Issues**: Missing ALSA libraries

### macOS

- **CI Environment**: macOS latest
- **Audio System**: CoreAudio (mock in CI)
- **Dependencies**: Built-in
- **Common Issues**: CoreAudio race conditions

### Windows

- **CI Environment**: Windows latest
- **Audio System**: WASAPI (mock in CI)
- **Dependencies**: Built-in
- **Common Issues**: Path separator differences

## Local Development Setup

### Matching CI Environment

To replicate CI behavior locally:

```bash
# Set CI environment variables
export CI=true
export GLOW_TTS_MOCK_AUDIO=true

# Run tests as CI would
go test -v -short -race ./...
```

### Testing with Real Audio

For local development with audio hardware:

```bash
# Ensure audio dependencies are installed
# Linux: sudo apt-get install libasound2-dev
# macOS: Built-in
# Windows: Built-in

# Run with real audio
GLOW_TTS_MOCK_AUDIO=false go test -v ./pkg/tts/...
```

### Using Build Tags

```bash
# Static analysis mode (no CGO)
go build -tags=nocgo ./...
go vet -tags=nocgo ./...

# Integration testing
go test -tags=integration ./...
```

## Running Test Suites

### Quick Test Suite

For rapid feedback during development:

```bash
# Fast unit tests only
go test -short ./...

# With race detection
go test -short -race ./...

# Specific package
go test -short ./pkg/tts/engines/...
```

### Comprehensive Test Suite

For thorough testing before commits:

```bash
# All tests with coverage
go test -v -race -coverprofile=coverage.txt ./...

# View coverage report
go tool cover -html=coverage.txt
```

### Platform-Specific Tests

```bash
# Test platform detection
CI=true go test -v -run TestDetectPlatform ./pkg/tts/

# Test audio context factory
CI=true go test -v -run TestAudioContextFactory ./pkg/tts/

# Test with mock audio
GLOW_TTS_MOCK_AUDIO=true go test -v ./pkg/tts/...
```

### TTS Engine Tests

```bash
# Test all engines
go test -v ./pkg/tts/engines/...

# Test specific engine
go test -v -run TestPiper ./pkg/tts/engines/

# With debug output
GLOW_LOG_LEVEL=debug go test -v ./pkg/tts/engines/...
```

### Cache Tests

```bash
# Test cache functionality
go test -v ./pkg/tts/cache/...

# Test with custom cache size
GLOW_TTS_CACHE_MAX_SIZE=10000000 go test -v ./pkg/tts/cache/...
```

## Troubleshooting

### Common CI Failures and Solutions

#### 1. ALSA Configuration Errors (Linux)

**Error:**
```
cannot open audio device: no such device
ALSA lib confmisc.c:767:(parse_card) cannot find card '0'
```

**Solution:**
- CI automatically uses mock audio (`GLOW_TTS_MOCK_AUDIO=true`)
- For local testing, ensure ALSA is installed: `sudo apt-get install libasound2-dev`
- Or use mock audio: `GLOW_TTS_MOCK_AUDIO=true go test`

#### 2. CGO Import Errors in Static Analysis

**Error:**
```
could not import C (no metadata for C)
```

**Solution:**
- Use `nocgo` build tag: `go vet -tags=nocgo ./...`
- Static analysis workflow already configured with this tag
- See `static-analysis.yml` for proper configuration

#### 3. Compilation Error: Redundant Newline

**Error:**
```
fmt.Println("text\n") - redundant newline
```

**Solution:**
- Remove `\n` from `fmt.Println` statements
- Use separate `fmt.Println()` for blank lines
- Lint locally: `golangci-lint run`

#### 4. Race Condition Failures

**Error:**
```
WARNING: DATA RACE
```

**Solution:**
- Review mutex usage in failing code
- Ensure proper locking for shared state
- Test locally: `go test -race ./...`

#### 5. Mock Audio Not Being Used

**Symptoms:**
- Tests hang waiting for audio
- Audio device errors in CI

**Solution:**
```bash
# Ensure environment variables are set
export CI=true
export GLOW_TTS_MOCK_AUDIO=true

# Verify mock is being used
go test -v -run TestAudioContextFactory ./pkg/tts/
```

#### 6. Vulnerability Scan Failures

**Error:**
```
govulncheck: vulnerability found
```

**Solution:**
1. Run `govulncheck ./...` locally
2. Update vulnerable dependency: `go get -u <package>`
3. Run `go mod tidy`
4. Test and commit changes

#### 7. Platform-Specific Test Failures

**Issue:** Tests pass on one platform but fail on another

**Solution:**
- Check platform-specific code paths
- Use platform detection properly
- Test locally with platform simulation:
  ```bash
  GOOS=windows go test ./...
  GOOS=darwin go test ./...
  ```

### Debugging CI Issues

#### Enable Debug Logging

```bash
# In CI workflow
env:
  GLOW_LOG_LEVEL: debug
  
# Locally
GLOW_LOG_LEVEL=debug go test -v ./...
```

#### Run Specific Failing Test

```bash
# Identify failing test from CI logs
go test -v -run TestSpecificName ./pkg/path/

# With timeout for hanging tests
go test -v -timeout 30s -run TestSpecificName ./pkg/path/
```

#### Check Environment Detection

```bash
# Test CI detection
CI=true go run -tags=nocgo . --help

# Test platform detection
go test -v -run TestDetectPlatform ./pkg/tts/
```

## Failure Notifications

The CI system includes automated failure notifications to alert maintainers of issues.

### Notification Channels

Notifications can be sent through multiple channels:
1. **GitHub Issues** - Automatic issue creation for critical failures
2. **Discord** - Webhook-based notifications
3. **Slack** - Webhook-based notifications
4. **GitHub Actions Summary** - Always generated for failed runs

### Setting Up Notifications

#### Discord Notifications

1. Create a Discord webhook in your server
2. Add the webhook URL as a repository variable:
   - Go to Settings → Secrets and variables → Actions → Variables
   - Add `DISCORD_WEBHOOK_URL` with your webhook URL

#### Slack Notifications

1. Create a Slack incoming webhook
2. Add the webhook URL as a repository variable:
   - Go to Settings → Secrets and variables → Actions → Variables
   - Add `SLACK_WEBHOOK_URL` with your webhook URL

#### Automatic Issue Creation

For critical failures on the main branch, the system can automatically create GitHub issues:
- Issues are labeled with `ci-failure` and `automated`
- Duplicate issues are prevented (checks for similar open issues)
- Issues include failure details and troubleshooting steps

### Using the Notification System

The notification system is implemented as a reusable workflow. To add notifications to any workflow:

```yaml
jobs:
  your-job:
    # ... your job definition ...

  notify-failure:
    if: failure() && github.ref == 'refs/heads/main'
    needs: [your-job]
    uses: ./.github/workflows/notify.yml
    with:
      workflow_name: "Your Workflow Name"
      failure_type: "test|build|security|deployment"
      create_issue: true  # Optional, creates GitHub issue
```

### Notification Examples

#### Test Failure Notification
```yaml
notify-test-failure:
  if: failure()
  needs: [unit-tests]
  uses: ./.github/workflows/notify.yml
  with:
    workflow_name: "Unit Tests"
    failure_type: "test"
```

#### Security Scan Failure
```yaml
notify-security-failure:
  if: failure()
  needs: [security-scan]
  uses: ./.github/workflows/notify.yml
  with:
    workflow_name: "Security Scan"
    failure_type: "security"
    create_issue: true
```

## Maintenance

### Updating Dependencies

```bash
# Check for updates
go list -u -m all

# Update specific dependency
go get -u github.com/package/name@latest

# Update all dependencies
go get -u ./...

# Clean up
go mod tidy

# Verify
go test ./...
govulncheck ./...
```

### Adding New Workflows

1. Create workflow file in `.github/workflows/`
2. Set required environment variables:
   ```yaml
   env:
     CI: true
     GLOW_TTS_MOCK_AUDIO: true
   ```
3. Use build tags for static analysis:
   ```yaml
   run: go test -tags=nocgo ./...
   ```
4. Document in this file and workflow README

### Monitoring Workflow Performance

- Check Actions tab for execution times
- Review parallel job execution
- Optimize slow tests with `-short` flag
- Consider splitting large test suites

### Updating Build Tags

When adding new build tags:
1. Document in this file
2. Update relevant workflow files
3. Add examples in test documentation
4. Ensure stub implementations exist

## Best Practices

1. **Always use mock audio in CI** - Set `GLOW_TTS_MOCK_AUDIO=true`
2. **Run tests locally before pushing** - `go test -short -race ./...`
3. **Use appropriate build tags** - `nocgo` for static analysis
4. **Keep tests fast** - Use `-short` flag for unit tests
5. **Document platform-specific code** - Add comments explaining OS differences
6. **Handle missing dependencies gracefully** - Fall back to mocks when needed
7. **Monitor CI performance** - Keep workflows under 10 minutes

## Related Documentation

- [GitHub Actions Workflows README](.github/workflows/README.md)
- [Testing Guide](TESTING.md)
- [Platform Detection](../pkg/tts/platform.go)
- [Audio Context Factory](../pkg/tts/audio_context_factory.go)