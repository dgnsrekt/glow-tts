# Technology Stack - Glow-TTS

## Programming Language

### Go (1.23.6)
- **Primary Language**: Entire codebase written in Go
- **Toolchain**: Go 1.24.1
- **Module System**: Go modules for dependency management
- **Rationale**: Performance, cross-platform compilation, strong concurrency support

## Core Frameworks

### Bubble Tea (v1.3.6)
- **Purpose**: Terminal User Interface framework
- **Usage**: Powers the interactive TUI mode, file browser, and pager
- **Key Features**: Event-driven architecture, composable components
- **Documentation**: https://github.com/charmbracelet/bubbletea

### Glamour (v0.10.0)
- **Purpose**: Markdown rendering engine for terminals
- **Usage**: Converts markdown to ANSI-styled terminal output
- **Key Features**: Syntax highlighting, emoji support, customizable styles
- **Documentation**: https://github.com/charmbracelet/glamour

### Cobra (v1.9.1)
- **Purpose**: CLI framework for command parsing
- **Usage**: Handles command-line arguments and subcommands
- **Key Features**: Auto-completion, help generation, nested commands
- **Documentation**: https://github.com/spf13/cobra

### Viper (v1.20.1)
- **Purpose**: Configuration management
- **Usage**: Handles YAML configuration files and environment variables
- **Key Features**: Multiple config formats, live watching, defaults
- **Documentation**: https://github.com/spf13/viper

## UI & Styling Libraries

### Lipgloss (v1.1.1)
- **Purpose**: Terminal styling and layout
- **Usage**: Styles UI components with colors, borders, padding
- **Key Features**: Declarative styling, responsive layouts
- **Documentation**: https://github.com/charmbracelet/lipgloss

### Bubbles (v0.21.0)
- **Purpose**: Pre-built TUI components
- **Usage**: Provides text inputs, spinners, progress bars
- **Key Features**: Ready-to-use components for Bubble Tea
- **Documentation**: https://github.com/charmbracelet/bubbles

## Rendering & Parsing

### Chroma (v2.14.0)
- **Purpose**: Syntax highlighting
- **Usage**: Highlights code blocks in markdown
- **Key Features**: 100+ language support, customizable themes
- **Documentation**: https://github.com/alecthomas/chroma

### Goldmark (v1.7.8)
- **Purpose**: Markdown parsing
- **Usage**: Parses markdown into AST for rendering
- **Key Features**: CommonMark compliant, extensible
- **Documentation**: https://github.com/yuin/goldmark

## Terminal & System Libraries

### Termenv (v0.16.0)
- **Purpose**: Terminal environment detection
- **Usage**: Detects terminal capabilities and colors
- **Key Features**: Color profile detection, ANSI support
- **Documentation**: https://github.com/muesli/termenv

### x/term (v0.33.0)
- **Purpose**: Terminal handling
- **Usage**: Terminal size detection, raw mode
- **Key Features**: Cross-platform terminal manipulation
- **Documentation**: golang.org/x/term

### x/sys (v0.34.0)
- **Purpose**: System calls and OS interface
- **Usage**: Low-level system operations
- **Key Features**: Platform-specific implementations
- **Documentation**: golang.org/x/sys

## Utility Libraries

### Reflow (v0.3.0)
- **Purpose**: Text reflow and word wrapping
- **Usage**: Wraps text to specified widths
- **Key Features**: ANSI-aware wrapping, padding, indentation
- **Documentation**: https://github.com/muesli/reflow

### go-humanize (v1.0.1)
- **Purpose**: Human-readable formatting
- **Usage**: Formats file sizes, times, numbers
- **Key Features**: Bytes to human format, time ago
- **Documentation**: https://github.com/dustin/go-humanize

### Clipboard (v0.1.4)
- **Purpose**: System clipboard access
- **Usage**: Copy/paste operations
- **Key Features**: Cross-platform clipboard support
- **Documentation**: https://github.com/atotto/clipboard

## Development Tools

### Task (Taskfile.yaml)
- **Purpose**: Task automation
- **Usage**: Build, test, and development tasks
- **Key Features**: Cross-platform task runner
- **Documentation**: https://taskfile.dev

### Docker
- **Purpose**: Containerization
- **Usage**: Building and distributing container images
- **Configuration**: Dockerfile present

## Planned TTS Dependencies

### Audio Libraries (To Be Added)
- **Beep/Oto**: Cross-platform audio playback in Go
- **Purpose**: Handle audio output for TTS
- **Requirements**: Platform audio drivers

### TTS Engines (External)

#### Piper TTS (Local)
- **Type**: External binary
- **Purpose**: Local text-to-speech generation
- **Requirements**: Piper binary installation
- **Benefits**: No internet required, privacy

#### Google TTS (Cloud)
- **Type**: REST API
- **Purpose**: Cloud-based text-to-speech
- **Requirements**: Internet connection, API key
- **Benefits**: High quality voices, multiple languages

## Platform Support

### Operating Systems
- Linux (Primary target)
- macOS (Full support)
- Windows (Full support)
- FreeBSD (Supported)
- OpenBSD (Supported)

### Terminal Emulators
- Terminal.app (macOS)
- iTerm2 (macOS)
- Windows Terminal
- GNOME Terminal
- Konsole
- Alacritty
- Any ANSI-compatible terminal

## Build & Distribution

### Build System
- **Go Build**: Standard Go compilation
- **Cross-Compilation**: Support for multiple platforms
- **Release**: Binaries for all major platforms

### Package Managers
- Homebrew (macOS/Linux)
- APT (Debian/Ubuntu)
- YUM (Fedora/RHEL)
- Chocolatey (Windows)
- Scoop (Windows)
- Snap (Ubuntu)
- AUR (Arch Linux)

## Version Control & CI/CD

### Version Control
- **Git**: Source control
- **GitHub**: Primary repository hosting
- **Branching**: Main/master branch development

### Continuous Integration
- **GitHub Actions**: Build and test automation
- **Testing**: Unit tests with Go testing package
- **Linting**: Go standard linting tools

## Documentation

### Documentation Tools
- **Markdown**: All documentation in markdown format
- **Man Pages**: Generated with roff
- **Inline Comments**: Go doc comments

## License & Compliance

### License
- **MIT License**: Open source, permissive license
- **Compatibility**: All dependencies MIT/BSD/Apache compatible
- **Attribution**: Proper attribution for Charm.sh and contributors