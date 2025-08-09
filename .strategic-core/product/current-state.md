# Current State - Glow-TTS

## Project Status

This is a fork of the Glow markdown reader (v2) that is in the initial planning phase for adding TTS functionality. The base Glow functionality is fully operational.

## Existing Features (Inherited from Glow)

### Core Functionality
- **Markdown Rendering**: Full markdown-to-terminal rendering using Glamour
- **TUI Mode**: Interactive file browser and reader using Bubble Tea framework
- **CLI Mode**: Direct markdown file rendering from command line
- **Pager Support**: Built-in pager for navigating long documents
- **Syntax Highlighting**: Code block syntax highlighting via Chroma
- **Style Support**: Multiple rendering styles (dark, light, custom JSON)
- **Git Integration**: Automatic discovery of markdown files in git repositories

### User Interface Components
- **File Browser** (`ui/stash.go`): Browse and select markdown files
- **Markdown Viewer** (`ui/pager.go`): Read markdown with keyboard navigation
- **Configuration Editor** (`ui/editor.go`): Edit configuration in-terminal
- **Key Bindings** (`ui/keys.go`): Comprehensive keyboard shortcuts
- **Styling System** (`ui/styles.go`): Theme and style management

### Platform Support
- **Cross-Platform**: Linux, macOS, Windows, FreeBSD, OpenBSD
- **Terminal Compatibility**: Wide terminal emulator support
- **Mouse Support**: Optional mouse wheel scrolling in TUI mode
- **Configuration**: YAML-based configuration system

### Network Features
- **GitHub Integration** (`github.go`): Fetch README from GitHub repos
- **GitLab Integration** (`gitlab.go`): Fetch README from GitLab repos
- **HTTP Support**: Render markdown from URLs

## Work in Progress

### TTS Feature Development
Currently, no TTS functionality has been implemented. The project has:
- Comprehensive PRD (`idea_prd.md`) outlining TTS requirements
- Strategic Core configuration initialized
- Base Glow codebase ready for extension

### Planned TTS Components (Not Yet Implemented)
Based on the PRD and refined standards, the following components need to be developed in the isolated `tts/` directory:
- `tts/controller.go` - Main TTS orchestrator
- `tts/engines/piper/` - Piper TTS integration
- `tts/engines/google/` - Google TTS integration  
- `tts/audio/` - Cross-platform audio playback
- `tts/sentence/` - Sentence parsing and tracking
- `tts/sync/` - Audio-visual synchronization
- `ui/tts_status.go` - Minimal status display (only new UI file)
- Minimal extensions to `ui/pager.go` for highlighting
- Keyboard shortcuts integrated into existing handlers

## Known Issues

### Current Limitations (Base Glow)
- No accessibility features for screen readers
- No audio support of any kind
- Limited to visual-only content consumption

### Technical Debt
- No TTS-related code exists yet
- Will need to carefully integrate with Bubble Tea event loop
- Audio synchronization with terminal rendering will require careful timing

## Project Structure

### Source Organization
```
/
├── main.go                 # Entry point and CLI setup
├── config_cmd.go          # Configuration command
├── man_cmd.go            # Manual page generation
├── github.go             # GitHub integration
├── gitlab.go             # GitLab integration
├── style.go              # Style handling
├── log.go                # Logging utilities
├── url.go                # URL parsing and handling
├── ui/                   # Terminal UI components
│   ├── ui.go            # Main UI controller
│   ├── pager.go         # Document viewer
│   ├── stash.go         # File browser
│   ├── markdown.go      # Markdown rendering
│   ├── keys.go          # Keyboard handling
│   └── styles.go        # Visual styling
└── utils/               # Utility functions
```

### Configuration Files
- `go.mod`: Go module dependencies
- `Taskfile.yaml`: Task automation
- `Dockerfile`: Container build configuration
- `.strategic-core/`: Strategic Core project management

## Dependencies

### Core Libraries
- **Bubble Tea** (v1.3.6): Terminal UI framework
- **Glamour** (v0.10.0): Markdown rendering engine
- **Lipgloss** (v1.1.1): Terminal styling
- **Cobra** (v1.9.1): CLI framework
- **Viper** (v1.20.1): Configuration management

### Supporting Libraries
- **Chroma** (v2.14.0): Syntax highlighting
- **Goldmark** (v1.7.8): Markdown parsing
- **Termenv** (v0.16.0): Terminal environment detection

## Testing

Current test coverage includes:
- Basic Glow functionality tests (`glow_test.go`)
- URL parsing tests (`url_test.go`)

No TTS-related tests exist yet as the functionality hasn't been implemented.

## Development Environment

- **Language**: Go 1.23.6 (toolchain 1.24.1)
- **Build System**: Standard Go build tools
- **Task Runner**: Taskfile for automation
- **Version Control**: Git
- **Documentation**: Markdown-based (README, PRD)

## Next Steps

Following the refined architecture standards:

1. Create `tts/` directory structure with clean interfaces
2. Implement TTS controller in `tts/controller.go`
3. Add engine implementations in `tts/engines/` (Piper first)
4. Develop audio playback in `tts/audio/`
5. Implement sentence parsing in `tts/sentence/`
6. Add synchronization manager in `tts/sync/`
7. Create minimal UI integration:
   - Add `ui/tts_status.go` for status display only
   - Extend `ui/pager.go` minimally for highlighting
8. Add comprehensive testing in `tts/*_test.go` files
9. Update documentation with TTS usage instructions