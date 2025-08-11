# TTS Core Infrastructure - Validation Guide

## Quick Validation Commands

### Primary Validation (Recommended)
```bash
task check
```
Runs: `go fmt`, `go vet`, `task test`

### Full Validation (When golangci-lint available)
```bash
task validate  
```
Runs: `go fmt`, `go vet`, `task lint`, `task test`

### Individual Components
```bash
task test    # Run all tests
task lint    # Full linting (if available)
go fmt ./... # Format all code
go vet ./... # Static analysis
```

## Validation Requirements

### ✅ **Must Pass Before Task Completion:**

1. **Code Formatting**: `go fmt ./...` - No style inconsistencies
2. **Static Analysis**: `go vet ./...` - No suspicious constructs  
3. **Test Suite**: `task test` - All tests pass, no regressions
4. **Build Status**: Code compiles without errors

### 🎯 **Benefits Demonstrated:**

- **Early Issue Detection**: Catches problems during development
- **Consistent Code Quality**: Maintains professional standards
- **Regression Prevention**: Validates existing functionality works
- **Faster Reviews**: Reduces back-and-forth in code reviews

## Usage Examples

### Task Implementation Workflow
```bash
# 1. Implement your changes
# 2. Run validation
task check

# 3. Address any issues found
# 4. Re-run validation until clean
task check

# 5. Mark task complete only after validation passes
```

### Common Issues and Solutions

#### Format Issues
```bash
# Problem: Code style inconsistencies
go fmt ./...  # Fixes automatically
```

#### Static Analysis Warnings  
```bash
# Problem: Suspicious constructs
go vet ./...  # Shows specific issues to fix manually
```

#### Test Failures
```bash
# Problem: Regression or new bugs
task test     # Shows failing tests
# Fix the issues and re-run
```

## Integration with Taskfile

The enhanced `Taskfile.yaml` provides flexible validation options:

- **`task check`**: Core validation (works everywhere)
- **`task validate`**: Full validation (requires golangci-lint)
- **`task test`**: Just run tests
- **`task lint`**: Just run linting (if available)

## Quality Gates

All remaining tasks (12-20) now enforce these quality gates:

- ✅ **Code Formatting**: Consistent style
- ✅ **Static Analysis**: No suspicious code  
- ✅ **Test Coverage**: All tests pass
- ✅ **Build Success**: Clean compilation

This ensures **professional development practices** and **early issue detection** throughout the TTS Core Infrastructure implementation.