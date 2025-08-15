# GitHub Actions Workflows

This directory contains the CI/CD workflows for the glow-tts project.

## Workflows Overview

### 1. Build & Test (`build.yml`)
- **Trigger**: Push to main/master/develop branches, PRs to main/master
- **Purpose**: Quick build verification and smoke tests
- **Features**:
  - Multi-platform builds (Linux, macOS, Windows)
  - Security vulnerability scanning with govulncheck
  - Quick smoke tests with mock audio
  - Uses `CI=true` and `GLOW_TTS_MOCK_AUDIO=true` for fast execution

### 2. Test Suite (`test.yml`)
- **Trigger**: Push, PR, manual dispatch
- **Purpose**: Comprehensive testing with separate unit and integration tests
- **Features**:
  - **Unit Tests**: Run on every push/PR with mock audio
  - **Integration Tests**: Only run when manually triggered or PR has `integration-tests` label
  - **Platform Detection Tests**: Verify platform-specific code
  - **TTS Engine Tests**: Test all TTS engine implementations
  - **Benchmarks**: Performance benchmarking on main branch pushes
  - Code coverage reporting to Codecov

### 3. Pull Request Checks (`pr.yml`)
- **Trigger**: Pull request events
- **Purpose**: Fast feedback for PR authors
- **Features**:
  - Code formatting and linting checks
  - Quick unit tests with coverage
  - Parallel platform build verification
  - Documentation checks
  - Combined status reporting

### 4. Nightly Tests (`nightly.yml`)
- **Trigger**: Daily at 2 AM UTC, manual dispatch
- **Purpose**: Comprehensive testing and regression detection
- **Features**:
  - Full test suite across multiple Go versions
  - Memory leak and race condition testing
  - Fuzz testing for robustness
  - Dependency auditing
  - Performance regression testing
  - Automated report generation

## Environment Variables

All workflows use these key environment variables:

- `CI=true`: Indicates CI environment for test detection
- `GLOW_TTS_MOCK_AUDIO=true`: Forces mock audio usage instead of hardware
- `GO_VERSION=stable`: Go version to use (can be overridden)

## Running Integration Tests

Integration tests that require real audio hardware are disabled by default. To run them:

### Option 1: Manual Workflow Dispatch
1. Go to Actions tab
2. Select "Test Suite" workflow
3. Click "Run workflow"
4. Set "Run integration tests" to "true"

### Option 2: PR Label
Add the `integration-tests` label to your PR

## Platform-Specific Behavior

The workflows handle platform differences automatically:

- **Linux**: Uses mock audio in CI (no audio devices available)
- **macOS**: Uses mock audio in CI (CoreAudio initialization issues)
- **Windows**: Uses mock audio in CI (WASAPI not available)

## Test Categories

### Unit Tests (Fast)
- Run with `-short` flag
- Use mock audio contexts
- Complete in < 30 seconds
- Run on every push/PR

### Integration Tests (Slow)
- Require real audio hardware
- May take several minutes
- Run only when explicitly requested
- May fail in CI environments

### Platform Tests
- Verify OS detection
- Check audio subsystem identification
- Test platform-specific retry logic
- Always use mock audio in CI

## Optimizations

1. **Parallel Execution**: All platform builds run in parallel
2. **Caching**: Go modules and build cache are preserved
3. **Fail-Fast**: Disabled to see all platform results
4. **Mock Audio**: Eliminates audio hardware dependencies
5. **Short Tests**: PR checks use `-short` flag for speed

## Troubleshooting

### Tests Pass Locally but Fail in CI
- Check if `CI=true` environment variable is set locally
- Verify mock audio is being used: `GLOW_TTS_MOCK_AUDIO=true`
- Run with same flags as CI: `go test -v -short -race ./...`

### Audio-Related Test Failures
- Ensure platform detection is working correctly
- Check that mock audio context is being selected
- Verify no tests are trying to use real audio hardware

### Performance Issues
- Use `-short` flag for quick tests
- Separate unit and integration tests
- Enable test caching with `-count=1`

## Adding New Workflows

When adding new workflows, ensure:
1. Set `CI=true` environment variable
2. Set `GLOW_TTS_MOCK_AUDIO=true` for audio tests
3. Use matrix strategy for multi-platform testing
4. Add appropriate caching for dependencies
5. Document the workflow purpose and triggers