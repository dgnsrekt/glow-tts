# Glow TTS Setup Guide

This guide will help you set up Text-to-Speech (TTS) capabilities in Glow. Glow supports two TTS engines: **Piper** (offline) and **Google TTS** (online).

## Quick Start

```bash
# Check which TTS dependencies are installed
glow --check-deps

# Generate a TTS configuration file
glow --generate-tts-config

# Start Glow with TTS enabled (auto-detects available engine)
glow --tts piper README.md  # Use Piper engine
glow --tts gtts README.md   # Use Google TTS engine
```

## Supported TTS Engines

### Piper TTS (Recommended for Offline Use)
- **Pros**: Works offline, low latency, privacy-focused, customizable voices
- **Cons**: Requires voice model downloads, larger disk space usage
- **Best for**: Users who want offline capability and consistent performance

### Google TTS (Recommended for Quick Setup)
- **Pros**: Easy setup, multiple languages, no model downloads needed
- **Cons**: Requires internet connection, potential latency, rate limits
- **Best for**: Users who want minimal setup and have reliable internet

## Installation

### Installing Piper TTS

#### Linux (Ubuntu/Debian)
```bash
# Download the latest Piper release
wget https://github.com/rhasspy/piper/releases/latest/download/piper_linux_x86_64.tar.gz
tar -xzf piper_linux_x86_64.tar.gz
sudo mv piper /usr/local/bin/

# Create voices directory
mkdir -p ~/.local/share/piper-voices

# Download a voice model (example: US English, Amy)
cd ~/.local/share/piper-voices
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/amy/medium/en_US-amy-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/amy/medium/en_US-amy-medium.onnx.json
```

#### Linux (ARM/Raspberry Pi)
```bash
# Download ARM version
wget https://github.com/rhasspy/piper/releases/latest/download/piper_linux_aarch64.tar.gz
tar -xzf piper_linux_aarch64.tar.gz
sudo mv piper /usr/local/bin/

# Follow same voice download steps as above
```

#### macOS
```bash
# Using Homebrew (if available)
brew install piper

# Or manual installation
wget https://github.com/rhasspy/piper/releases/latest/download/piper_macos_x64.tar.gz
tar -xzf piper_macos_x64.tar.gz
sudo mv piper /usr/local/bin/

# Create voices directory and download models (same as Linux)
mkdir -p ~/.local/share/piper-voices
```

#### Windows
1. Download the Windows release from [Piper Releases](https://github.com/rhasspy/piper/releases)
2. Extract to `C:\Program Files\piper\`
3. Add `C:\Program Files\piper\` to your PATH
4. Create `%USERPROFILE%\.local\share\piper-voices\` directory
5. Download voice models to this directory

### Installing Google TTS

Google TTS requires Python and ffmpeg:

#### All Platforms
```bash
# Install gtts-cli
pip install gtts

# Or using pipx (recommended)
pipx install gtts

# Verify installation
gtts-cli --help
```

#### Installing ffmpeg

**Linux:**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg

# Fedora
sudo dnf install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

**macOS:**
```bash
brew install ffmpeg
```

**Windows:**
1. Download from [ffmpeg.org](https://ffmpeg.org/download.html)
2. Extract to `C:\ffmpeg`
3. Add `C:\ffmpeg\bin` to PATH

## Voice Models for Piper

### Recommended English Voices

| Voice | Quality | Size | Download |
|-------|---------|------|----------|
| Amy | Medium | 63MB | [Download](https://huggingface.co/rhasspy/piper-voices/tree/main/en/en_US/amy/medium) |
| Danny | Low | 18MB | [Download](https://huggingface.co/rhasspy/piper-voices/tree/main/en/en_US/danny/low) |
| Ryan | High | 188MB | [Download](https://huggingface.co/rhasspy/piper-voices/tree/main/en/en_US/ryan/high) |
| Libritts | Medium | 64MB | [Download](https://huggingface.co/rhasspy/piper-voices/tree/main/en/en_US/libritts/high) |

### Installing Voice Models

1. Download both the `.onnx` and `.onnx.json` files
2. Place them in `~/.local/share/piper-voices/`
3. Glow will automatically detect available models

### Other Languages

Piper supports 30+ languages. Browse available models at:
- [Piper Voices on Hugging Face](https://huggingface.co/rhasspy/piper-voices/tree/main)

## Configuration

### Configuration File Location

Glow looks for TTS configuration in these locations (in order):
1. `./.glow/glow-tts.yml` (project-specific)
2. `~/.config/glow/glow-tts.yml` (user-wide)

### Basic Configuration

Generate an example configuration:
```bash
glow --generate-tts-config
```

Edit `~/.config/glow/glow-tts.yml`:

```yaml
# Default TTS engine (piper or gtts)
default_engine: piper

# Engine-specific settings
engines:
  piper:
    # Path to specific voice model (optional)
    model_path: ~/.local/share/piper-voices/en_US-amy-medium.onnx
    # Speaking speed (0.5 to 2.0)
    default_speed: 1.0
    
  gtts:
    # Language code (en, es, fr, de, etc.)
    language: en
    # Speaking speed (0.5 to 2.0)
    default_speed: 1.0

# Cache settings
cache:
  # Enable caching of synthesized audio
  enabled: true
  # Maximum memory cache size (bytes)
  max_memory_size: 52428800  # 50MB
  # Maximum disk cache size (bytes)
  max_disk_size: 524288000   # 500MB
  # Cache directory
  directory: ~/.cache/glow/tts

# Playback settings
playback:
  # Audio buffer size (samples)
  buffer_size: 512
  # Lookahead sentences for preprocessing
  lookahead_sentences: 3
```

### Environment Variables

You can also configure TTS using environment variables:

```bash
# Set default TTS engine
export GLOW_TTS_ENGINE=piper

# Set Piper model path
export GLOW_TTS_PIPER_MODEL=/path/to/model.onnx

# Set speaking speed
export GLOW_TTS_SPEED=1.2

# Set cache directory
export GLOW_TTS_CACHE_DIR=~/.cache/glow/tts
```

## Keyboard Shortcuts

When TTS is enabled in TUI mode:

| Key | Action |
|-----|--------|
| `Space` | Play/Pause |
| `s` | Stop playback |
| `→` / `n` | Next sentence |
| `←` / `p` | Previous sentence |
| `↑` | Increase speed |
| `↓` | Decrease speed |
| `r` | Reset to beginning |
| `1`-`5` | Set speed (0.5x to 2.0x) |

## Verifying Your Setup

Run the dependency check:
```bash
glow --check-deps
```

You should see output like:
```
Checking all TTS dependencies...

✓ piper: /usr/local/bin/piper
✓ piper_models: Found 2 models in ~/.local/share/piper-voices
✓ gtts-cli: /home/user/.local/bin/gtts-cli
✓ ffmpeg: /usr/bin/ffmpeg

Summary:
✓ Both Piper and Google TTS engines are available
```

## Testing TTS

Test with a simple markdown file:
```bash
echo "# Hello World\n\nThis is a test of text to speech." | glow --tts piper -
```

Or test with an existing file:
```bash
glow --tts piper README.md
```

## Performance Tuning

### Cache Optimization
- Enable caching to avoid re-synthesizing repeated text
- Increase cache size for large documents
- Clear cache periodically: `rm -rf ~/.cache/glow/tts`

### Speed vs Quality Trade-offs
- Lower quality Piper models synthesize faster
- Google TTS may have network latency
- Adjust lookahead sentences based on your system's performance

### Memory Usage
- Large documents are automatically chunked to prevent memory issues
- Reduce `lookahead_sentences` if experiencing high memory usage
- Use disk cache instead of memory cache for low-memory systems

## Troubleshooting

See [TTS Troubleshooting Guide](TTS_TROUBLESHOOTING.md) for common issues and solutions.

## Next Steps

- Try different voice models to find your preference
- Customize the configuration file for your needs
- Enable debug logging with `--debug` to diagnose issues
- Report issues at [GitHub Issues](https://github.com/charmbracelet/glow/issues)