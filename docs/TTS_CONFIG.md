# TTS Configuration Guide

## Overview

Glow-TTS supports comprehensive configuration through both configuration files and environment variables. This guide covers all available TTS configuration options.

## Configuration File

The configuration file is located at `~/.config/glow/config.yml` (on Linux/macOS) or `%APPDATA%\glow\config.yml` (on Windows).

### Full Configuration Example

```yaml
# TTS (Text-to-Speech) Configuration
tts:
  # Engine selection: piper (offline) or gtts (online)
  # Leave empty to disable TTS
  engine: "piper"
  
  # Cache settings
  cache:
    # Cache directory for audio files
    # Default: ~/.cache/glow-tts
    dir: "~/.cache/glow-tts"
    
    # Maximum cache size in MB (1-10000)
    # Default: 100
    max_size: 100
  
  # Piper engine settings (offline TTS)
  piper:
    # Path to Piper ONNX model file
    # Leave empty for auto-detection
    model: "~/.local/share/piper/models/en_US-amy-medium.onnx"
    
    # Speech speed (0.1-3.0, where 1.0 is normal)
    # Default: 1.0
    speed: 1.0
  
  # Google TTS settings (online TTS)
  gtts:
    # Language code (e.g., en, es, fr, de, ja)
    # Default: en
    language: "en"
    
    # Use slower speech rate
    # Default: false
    slow: false
```

## Environment Variables

All configuration options can be overridden using environment variables. This is useful for scripts, Docker containers, or temporary configuration changes.

### Environment Variable Format

Environment variables follow the pattern: `GLOW_<SECTION>_<KEY>`

- Dots (.) in config paths become underscores (_)
- Hyphens (-) also become underscores (_)
- All uppercase

### Available Environment Variables

| Environment Variable | Config Path | Description | Default |
|---------------------|-------------|-------------|---------|
| `GLOW_TTS_ENGINE` | `tts.engine` | TTS engine (piper/gtts) | "" (disabled) |
| `GLOW_TTS_CACHE_DIR` | `tts.cache.dir` | Cache directory | "~/.cache/glow-tts" |
| `GLOW_TTS_CACHE_MAX_SIZE` | `tts.cache.max_size` | Max cache size (MB) | 100 |
| `GLOW_TTS_PIPER_MODEL` | `tts.piper.model` | Piper model path | "" (auto-detect) |
| `GLOW_TTS_PIPER_SPEED` | `tts.piper.speed` | Piper speed (0.1-3.0) | 1.0 |
| `GLOW_TTS_GTTS_LANGUAGE` | `tts.gtts.language` | gTTS language code | "en" |
| `GLOW_TTS_GTTS_SLOW` | `tts.gtts.slow` | gTTS slow speech | false |

### Examples

```bash
# Enable Piper TTS with custom model
export GLOW_TTS_ENGINE=piper
export GLOW_TTS_PIPER_MODEL=~/.local/share/piper/models/en_GB-alan-medium.onnx
glow README.md

# Use Google TTS with Spanish language
export GLOW_TTS_ENGINE=gtts
export GLOW_TTS_GTTS_LANGUAGE=es
glow --tts document.md

# Increase cache size for large document processing
export GLOW_TTS_CACHE_MAX_SIZE=500
glow --tts large-book.md

# Set custom cache directory
export GLOW_TTS_CACHE_DIR=/tmp/glow-tts-cache
glow --tts document.md
```

## Configuration Validation

Glow validates TTS configuration on startup:

- **Cache Size**: Must be between 1 and 10000 MB
- **Piper Speed**: Must be between 0.1 and 3.0
- **Language Code**: Must be 2-5 characters
- **Model Path**: File must exist if specified
- **Cache Directory**: Parent directory must be writable

## Migration from Older Versions

If you're upgrading from an older version of Glow without TTS support, your existing configuration will continue to work. TTS features are disabled by default and only activate when:

1. The `--tts` flag is used on the command line
2. The `tts.engine` configuration is set

## Troubleshooting

### TTS Not Working

1. **Check engine selection**: Ensure `tts.engine` is set to either "piper" or "gtts"
2. **Verify dependencies**: 
   - For Piper: `piper` binary must be in PATH
   - For gTTS: `gtts-cli` and `ffmpeg` must be installed
3. **Check permissions**: Ensure cache directory is writable
4. **Review logs**: Run with `--debug` flag for detailed output

### Environment Variables Not Working

1. **Check prefix**: All variables must start with `GLOW_`
2. **Use underscores**: Replace dots and hyphens with underscores
3. **Verify export**: Use `export` command in bash/zsh
4. **Check precedence**: CLI flags override env vars which override config file

### Cache Issues

1. **Clear cache**: Delete cache directory if corrupted
2. **Check disk space**: Ensure sufficient space for cache
3. **Permissions**: Cache directory must be writable
4. **Size limits**: Reduce `max_size` if disk space is limited

## Best Practices

1. **Use config file for permanent settings**: Store your preferred configuration in the config file
2. **Use environment variables for temporary changes**: Override specific settings without modifying the config file
3. **Choose appropriate cache size**: Balance between performance and disk usage
4. **Set language correctly**: Use proper ISO language codes for gTTS
5. **Test configuration**: Use `glow config` to edit and validate your configuration

## See Also

- [Glow README](../README.md) - Main documentation
- [TTS Implementation](../internal/tts/README.md) - Technical details
- [Piper Documentation](https://github.com/rhasspy/piper) - Piper TTS engine
- [gTTS Documentation](https://gtts.readthedocs.io/) - Google TTS library