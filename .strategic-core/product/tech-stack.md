# Glow-TTS Technology Stack

## Programming Languages

### Go (Primary)
- **Version**: 1.23.6+ (toolchain 1.24.1)
- **Purpose**: Core application, all business logic
- **Standards**: Go modules for dependency management

## Core Frameworks & Libraries

### UI/TUI Framework
- **Bubble Tea** (v1.3.6): Terminal UI framework for interactive interfaces
- **Bubbles** (v0.21.0): Pre-built Bubble Tea components
- **Lipgloss** (v1.1.1): Terminal styling and layout
- **Termenv** (v0.16.0): Terminal environment detection and styling

### Markdown Processing
- **Glamour** (v0.10.0): Markdown rendering with terminal styling
- **Goldmark** (v1.7.8): Extensible markdown parser
- **Goldmark-emoji** (v1.0.5): Emoji support in markdown
- **Chroma** (v2.14.0): Syntax highlighting

### CLI Framework
- **Cobra** (v1.9.1): Command-line interface creation
- **Viper** (v1.20.1): Configuration management
- **Mango-cobra** (v1.2.0): Man page generation for Cobra

## Dependencies

### System Integration
- **fsnotify** (v1.9.0): File system notifications
- **go-homedir** (v1.1.0): Cross-platform home directory detection
- **go-app-paths** (v0.2.2): Application data paths
- **clipboard** (v0.1.4): System clipboard integration

### Terminal Utilities
- **go-runewidth** (v0.0.16): Unicode width calculations
- **terminfo** (xo/terminfo): Terminal capability detection
- **ansi** (muesli/ansi): ANSI escape sequence handling
- **coninput** (erikgeiser): Console input handling

### Git Integration
- **gitcha** (v0.3.0): Git repository scanning for markdown files
- **go-gitignore**: Gitignore file parsing

### Network & HTTP
- **Standard library** (net/http): HTTP client for remote markdown fetching
- **No external HTTP framework**: Uses Go's built-in HTTP capabilities

### Utilities
- **fuzzy** (v0.1.1): Fuzzy string searching
- **reflow** (v0.3.0): Text reflowing and formatting
- **go-humanize** (v1.0.1): Human-readable formatting
- **env** (v11.3.1): Environment variable parsing

### Logging
- **charmbracelet/log** (v0.4.2): Structured logging with style

## Development Tools

### Build & Task Management
- **Taskfile**: Task automation (lint, test, log commands)
- **Make**: Not used (replaced by Taskfile)

### Testing
- **Go standard testing**: Built-in testing framework
- **No external testing frameworks**: Uses standard `go test`

### Code Quality
- **golangci-lint**: Comprehensive Go linting
- **GitHub Actions**: CI/CD pipeline

### Containerization
- **Docker**: Containerized builds supported
- **Multi-stage builds**: Optimized container images

## Platform Support

### Operating Systems
- macOS (including ARM)
- Linux (all major distributions)
- Windows
- FreeBSD
- OpenBSD
- Android (via Termux)

### Package Managers
- Homebrew (macOS/Linux)
- APT (Debian/Ubuntu)
- YUM/DNF (Fedora/RHEL)
- Pacman (Arch)
- Scoop/Chocolatey/Winget (Windows)
- Snap (Ubuntu)
- Various others

## Configuration

### Format
- **YAML**: Configuration file format (`glow.yml`)
- **Environment Variables**: Via caarlos0/env library

### Storage
- **XDG Base Directory**: Linux configuration locations
- **Application Support**: macOS configuration
- **AppData**: Windows configuration

## Planned TTS Stack (from PRD)

### TTS Engines (Planned)
- **Piper**: Offline TTS engine (ONNX models)
- **Google TTS**: Cloud-based TTS with API key support

### Audio Processing (Planned)
- **Audio buffering**: In-memory audio caching
- **File caching**: Temporary directory for audio files

### IPC (Planned)
- **Go channels**: Inter-process communication
- **Named pipes**: Alternative IPC mechanism

## Version Control

### Git
- **Integration**: Built-in git repository detection
- **Remote Support**: GitHub and GitLab README fetching

## Security Considerations

### Current Implementation
- **HTTPS**: For remote markdown fetching
- **No authentication**: Public content only
- **Local file access**: User permission based

### Planned (from PRD)
- **API Key Management**: Secure storage for Google TTS
- **File Permissions**: Restricted cache file access (0600)
- **Process Isolation**: Minimal privilege for TTS service

## Performance Characteristics

### Current
- **Memory Usage**: Lightweight (~10-30MB typical)
- **Startup Time**: Near instant (<100ms)
- **Rendering**: Hardware-accelerated when available

### Target (from PRD)
- **TTS Init**: <3 seconds
- **Navigation**: <200ms response time
- **Memory**: <75MB with TTS service
- **Cache Size**: <100MB per session