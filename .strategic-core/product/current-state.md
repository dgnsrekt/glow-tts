# Glow-TTS Current State

## Project Overview

Glow-TTS is a fork of Charmbracelet's Glow markdown reader (v2) that aims to add Text-to-Speech functionality. The project is currently in the planning/ideation phase on the master branch, with an experimental TTS implementation on a separate feature branch (marked as dead/experimental).

## Existing Features (Master Branch)

### Core Glow Functionality
- **Markdown Rendering**: Full markdown rendering in the terminal with glamour
- **TUI Mode**: Interactive file browser and markdown viewer
- **CLI Mode**: Direct markdown file rendering from command line
- **File Discovery**: Automatic markdown file discovery in directories and git repos
- **Remote Content**: Support for fetching markdown from GitHub, GitLab, and HTTP URLs
- **Pager**: High-performance pager for long documents
- **Stash System**: Local markdown file management and organization
- **Styling**: Multiple color schemes (dark/light/custom JSON)
- **Configuration**: YAML-based configuration file support

### UI Components
- **File Browser** (`ui/stash.go`): Browse and search local markdown files
- **Pager** (`ui/pager.go`): Scroll through rendered markdown
- **Editor Integration** (`ui/editor.go`): Open files in external editors
- **Markdown Processor** (`ui/markdown.go`): Handle markdown rendering
- **Help System** (`ui/stashhelp.go`): Interactive help screens

### Platform Support
- **GitHub Integration** (`github.go`): Fetch README files from GitHub repos
- **GitLab Integration** (`gitlab.go`): Fetch README files from GitLab repos
- **Windows Console** (`console_windows.go`): Windows-specific console handling
- **Cross-platform**: Support for macOS, Linux, Windows, FreeBSD, OpenBSD

## Work in Progress

### TTS Planning (from idea.md PRD)
The project has a comprehensive Product Requirements Document outlining the planned TTS implementation:

- **CLI Integration**: `--tts [engine]` flag to activate TTS mode
- **Engine Support**: Planned support for Piper (offline) and Google TTS (cloud)
- **Architecture**: Background service with queue-based processing
- **Features**: Sentence-level navigation, audio caching, preprocessing
- **Process Management**: Single-instance enforcement, graceful shutdown

### Strategic Core Integration
- Strategic Core framework has been installed for project management
- Commands and standards are configured but not yet utilized
- Product documentation is being generated

## Known Issues

No issues are currently tracked in the master branch codebase. The experimental TTS branch is considered non-functional and abandoned.

## Dependencies

### Core Dependencies
- **Charmbracelet Suite**: bubbletea, bubbles, glamour, lipgloss, log
- **Markdown**: glamour for rendering, goldmark for parsing
- **CLI**: cobra for commands, viper for configuration
- **Utilities**: fsnotify, fuzzy search, clipboard support

### Development Tools
- **Task Runner**: Taskfile for common development tasks
- **Testing**: Go standard testing framework
- **Linting**: golangci-lint for code quality
- **Docker**: Dockerfile for containerized builds

## Current Architecture

### Application Structure
```
glow (main.go)
├── CLI Commands (cobra)
├── Configuration (viper)
├── UI System (bubbletea)
│   ├── Stash (file browser)
│   ├── Pager (document viewer)
│   └── Editor (external editor)
├── Remote Fetchers
│   ├── GitHub
│   ├── GitLab
│   └── HTTP
└── Utilities
    └── Logging
```

### State Management
- Two main states: `stateShowStash` (file listing) and `stateShowDocument` (viewing)
- Message-based updates using Bubble Tea framework
- Async file discovery with channels

## Test Coverage

- Basic tests exist for URL parsing (`url_test.go`)
- Main functionality tests (`glow_test.go`)
- No TTS-specific tests in master branch

## Documentation

- Comprehensive README with installation and usage instructions
- No API documentation (internal use only)
- Strategic Core documentation framework installed but not populated