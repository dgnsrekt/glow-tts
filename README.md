# Glow-TTS

[![Build & Test](https://github.com/dgnsrekt/glow-tts/actions/workflows/build.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/build.yml)
[![Test Suite](https://github.com/dgnsrekt/glow-tts/actions/workflows/test.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/test.yml)
[![Static Analysis](https://github.com/dgnsrekt/glow-tts/actions/workflows/static-analysis.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/static-analysis.yml)
[![PR Checks](https://github.com/dgnsrekt/glow-tts/actions/workflows/pr.yml/badge.svg)](https://github.com/dgnsrekt/glow-tts/actions/workflows/pr.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dgnsrekt/glow-tts)](https://goreportcard.com/report/github.com/dgnsrekt/glow-tts)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dgnsrekt/glow-tts)](go.mod)

A fork of [Glow](https://github.com/charmbracelet/glow) with experimental Text-to-Speech capabilities.

> **⚠️ DISCLAIMER**: This is an unofficial fork not affiliated with or maintained by the Charm team. See [FORK_NOTICE.md](FORK_NOTICE.md) for details.

## What is this?

Glow-TTS adds Text-to-Speech functionality to the excellent [Glow markdown reader](https://github.com/charmbracelet/glow). This experimental fork allows you to listen to your markdown documents using either:

- **Piper TTS** - Fast, offline, privacy-focused
- **Google TTS** - Online, easy setup, multiple languages

## Features

### TTS Capabilities
- Two TTS engines (Piper offline, Google online)
- Playback controls (play, pause, stop, skip)
- Speed control (0.5x to 2.0x)
- Audio caching for repeated content
- Sentence-by-sentence navigation
- Keyboard shortcuts in TUI mode

### Keyboard Controls

When TTS is enabled in TUI mode:

| Key | Action |
|-----|--------|
| `Space` | Play/Pause |
| `s` | Stop |
| `→` / `n` | Next sentence |
| `←` / `p` | Previous sentence |
| `↑` | Increase speed |
| `↓` | Decrease speed |

## Quick Start

```bash
# Check TTS dependencies
glow-tts --check-deps

# Use Piper TTS (offline)
glow-tts --tts piper README.md

# Use Google TTS (online)
glow-tts --tts gtts README.md

# Generate TTS config file
glow-tts --generate-tts-config
```

## Installation

### Base Installation

This fork maintains the same installation methods as the original Glow. Clone and build from source:

```bash
git clone https://github.com/dgnsrekt/glow-tts.git
cd glow-tts
go build -o glow-tts
```

### TTS Dependencies

#### For Piper TTS (Offline)
1. Download Piper from [releases](https://github.com/rhasspy/piper/releases)
2. Download voice models from [Hugging Face](https://huggingface.co/rhasspy/piper-voices)
3. See [TTS Setup Guide](docs/TTS_SETUP.md) for detailed instructions

#### For Google TTS (Online)
```bash
pip install gtts
# or
pipx install gtts
```

## Documentation

- [TTS Setup Guide](docs/TTS_SETUP.md) - Detailed installation instructions
- [TTS Troubleshooting](docs/TTS_TROUBLESHOOTING.md) - Common issues and solutions
- [Original Glow Docs](https://github.com/charmbracelet/glow#readme) - For base markdown features

## Configuration

Generate a TTS config file:
```bash
glow-tts --generate-tts-config
```

This creates `~/.config/glow/glow-tts.yml` with options for:
- Default TTS engine selection
- Voice preferences
- Cache settings
- Speed defaults

## Original Glow Features

This fork retains all original Glow functionality. For information about:
- Markdown rendering
- GitHub/GitLab integration  
- Stashing documents
- Configuration options

Please refer to the [original Glow documentation](https://github.com/charmbracelet/glow#readme).

## Important Notes

- **Experimental**: TTS features are experimental and may have bugs
- **Upstream Sync**: This fork attempts to stay synchronized with upstream Glow
- **No Support**: This is a personal project with no official support
- **Original Credit**: All base functionality credit goes to [Charm](https://charm.sh)

## License

MIT - See [LICENSE](LICENSE)

- Original Glow Copyright (c) Charm
- TTS modifications Copyright (c) 2024 Contributors

## Acknowledgments

- The [Charm](https://charm.sh) team for creating Glow
- [Piper](https://github.com/rhasspy/piper) for offline TTS
- [gTTS](https://github.com/pndurette/gTTS) for Google TTS interface