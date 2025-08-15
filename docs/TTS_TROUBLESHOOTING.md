# Glow TTS Troubleshooting Guide

This guide helps resolve common issues with Text-to-Speech functionality in Glow.

## Common Issues

### 1. "TTS engine not found" Error

**Symptoms:**
- Error: `piper binary not found`
- Error: `gtts-cli not found`

**Solutions:**

Check if the binary is in your PATH:
```bash
which piper
which gtts-cli
```

If not found, verify installation:
```bash
# Check common installation locations
ls -la /usr/local/bin/piper
ls -la ~/.local/bin/gtts-cli

# Run dependency check
glow --check-deps
```

Add to PATH if needed:
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
export PATH="/usr/local/bin:$PATH"
```

### 2. "No voice models found" (Piper)

**Symptoms:**
- Error: `No ONNX voice models found`
- TTS shows "Not Ready" status

**Solutions:**

Check voice model directory:
```bash
ls -la ~/.local/share/piper-voices/
```

Download a voice model:
```bash
mkdir -p ~/.local/share/piper-voices
cd ~/.local/share/piper-voices
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/amy/medium/en_US-amy-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/amy/medium/en_US-amy-medium.onnx.json
```

Specify model path in config:
```yaml
# ~/.config/glow/glow-tts.yml
engines:
  piper:
    model_path: /full/path/to/model.onnx
```

### 3. "ffmpeg not found" (Google TTS)

**Symptoms:**
- Error: `ffmpeg not found`
- Google TTS fails to convert audio

**Solutions:**

Install ffmpeg:
```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Check installation
ffmpeg -version
```

### 4. No Audio Output

**Symptoms:**
- TTS appears to be working but no sound
- Status shows "Playing" but silent

**Solutions:**

Check audio system:
```bash
# Linux - check ALSA
aplay -l
speaker-test

# Linux - check PulseAudio
pactl info
pavucontrol  # GUI mixer

# macOS
system_profiler SPAudioDataType
```

Test audio playback:
```bash
# Generate test audio with Piper
echo "Test" | piper --model ~/.local/share/piper-voices/en_US-amy-medium.onnx --output_file test.wav
aplay test.wav  # Linux
afplay test.wav # macOS

# Test with Google TTS
gtts-cli "Test" --output test.mp3
ffplay test.mp3
```

Enable debug logging:
```bash
glow --tts piper --debug README.md
```

### 5. Poor Audio Quality or Distortion

**Symptoms:**
- Crackling or distorted audio
- Audio cuts in and out
- Speed sounds wrong

**Solutions:**

Adjust buffer size in config:
```yaml
playback:
  buffer_size: 1024  # Increase from default 512
```

Check CPU usage:
```bash
top -p $(pgrep glow)
```

Reduce processing load:
```yaml
playback:
  lookahead_sentences: 1  # Reduce from 3
```

Clear cache if corrupted:
```bash
rm -rf ~/.cache/glow/tts
```

### 6. High Memory Usage

**Symptoms:**
- Glow using excessive RAM
- System becomes unresponsive
- Out of memory errors

**Solutions:**

Limit cache size:
```yaml
cache:
  max_memory_size: 10485760  # 10MB instead of 50MB
  enabled: false  # Or disable entirely
```

Use lower quality voice models:
```bash
# Use "low" quality models instead of "high"
wget .../en_US-danny-low.onnx  # 18MB
# Instead of
wget .../en_US-ryan-high.onnx  # 188MB
```

### 7. Google TTS Network Issues

**Symptoms:**
- Error: `gtts-cli timed out`
- Error: `No internet connection`
- Slow synthesis

**Solutions:**

Check internet connection:
```bash
ping -c 1 google.com
curl -I https://translate.google.com
```

Increase timeout in environment:
```bash
export GLOW_TTS_TIMEOUT=30  # Increase from default 10 seconds
```

Use Piper for offline mode:
```bash
glow --tts piper README.md
```

### 8. TTS Not Starting in TUI Mode

**Symptoms:**
- TTS controls not appearing
- Engine shows "Not Initialized"

**Solutions:**

Force TUI mode with TTS:
```bash
glow --tts piper --tui README.md
```

Check for conflicting options:
```bash
# Don't use --pager with --tts
glow --tts piper README.md  # Good
glow --tts piper --pager README.md  # Won't work
```

### 9. Subprocess Timeout Errors

**Symptoms:**
- Error: `command timed out after 10s`
- Synthesis hangs indefinitely

**Solutions:**

Check if subprocess is hanging:
```bash
ps aux | grep piper
ps aux | grep gtts-cli
```

Kill hung processes:
```bash
pkill -f piper
pkill -f gtts-cli
```

Increase timeout in config:
```yaml
advanced:
  subprocess_timeout: 30  # Increase from 10 seconds
```

### 10. Cache Not Working

**Symptoms:**
- Same text re-synthesized every time
- Cache directory empty
- Slow repeated playback

**Solutions:**

Check cache directory permissions:
```bash
ls -la ~/.cache/glow/
chmod 755 ~/.cache/glow
chmod 755 ~/.cache/glow/tts
```

Enable cache in config:
```yaml
cache:
  enabled: true
  directory: ~/.cache/glow/tts
```

Verify cache is being written:
```bash
glow --tts piper --debug README.md 2>&1 | grep -i cache
ls -la ~/.cache/glow/tts/
```

## Debug Logging

Enable detailed logging to diagnose issues:

```bash
# Basic debug logging
glow --tts piper --debug README.md

# Trace-level logging (very verbose)
glow --tts piper --trace README.md

# Save debug output
glow --tts piper --debug README.md 2> debug.log
```

Check log file:
```bash
tail -f ~/.cache/glow/glow.log
```

## Platform-Specific Issues

### Linux

**Audio System Conflicts:**
```bash
# Switch between ALSA and PulseAudio
export GLOW_AUDIO_BACKEND=alsa  # or pulseaudio
```

**Permission Issues:**
```bash
# Add user to audio group
sudo usermod -a -G audio $USER
# Logout and login again
```

### macOS

**Security Warnings:**
```bash
# If Piper is blocked by Gatekeeper
xattr -d com.apple.quarantine /usr/local/bin/piper
```

**Audio Permission:**
- System Preferences → Security & Privacy → Microphone
- Allow Terminal/iTerm access

### Windows

**Path Issues:**
- Use forward slashes in config files
- Or escape backslashes: `C:\\path\\to\\model.onnx`

**Terminal Encoding:**
```powershell
# Set UTF-8 encoding
chcp 65001
```

## Performance Profiling

Profile TTS performance:

```bash
# Time synthesis
time echo "Test sentence" | piper --model model.onnx --output_raw > /dev/null

# Monitor resource usage
glow --tts piper --debug README.md &
top -p $!
```

## Getting Help

If these solutions don't resolve your issue:

1. **Collect Debug Information:**
   ```bash
   glow --version
   glow --check-deps
   glow --tts piper --debug README.md 2> debug.log
   ```

2. **System Information:**
   ```bash
   uname -a
   python --version
   pip show gtts
   piper --version
   ffmpeg -version
   ```

3. **Report Issue:**
   - [GitHub Issues](https://github.com/dgnsrekt/glow-tts/issues)
   - Include debug.log and system information
   - Describe steps to reproduce

## Quick Fixes Checklist

- [ ] Run `glow --check-deps` to verify dependencies
- [ ] Check PATH includes binary locations
- [ ] Verify internet connection (for Google TTS)
- [ ] Download at least one voice model (for Piper)
- [ ] Check audio system is working
- [ ] Clear cache if issues persist
- [ ] Try with `--debug` flag for detailed output
- [ ] Test with simple text first: `echo "test" | glow --tts piper -`
- [ ] Disable cache temporarily to isolate issues
- [ ] Try alternative engine if one fails