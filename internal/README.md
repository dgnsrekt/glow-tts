# TTS Internal Packages

This directory contains the internal implementation of the Text-to-Speech (TTS) functionality for Glow.

## Package Structure

- **`tts/`** - Core TTS orchestration and controller
  - `engines/` - TTS engine implementations (Piper, Google TTS)
  
- **`audio/`** - Cross-platform audio playback using oto/v3
  
- **`queue/`** - Sentence processing queue with lookahead buffering
  
- **`cache/`** - Two-level caching system (memory + disk)

## Key Design Principles

1. **No Automatic Activation** - TTS only runs when `--tts` flag is used
2. **Explicit Engine Selection** - User must choose between Piper or Google TTS
3. **Command Pattern** - All async operations use Bubble Tea commands (no direct goroutines)
4. **Memory Safety** - Audio data references kept alive during playback
5. **Race Condition Prevention** - Complete timeout protection for subprocesses

## Critical Implementation Notes

### Subprocess Management
- NEVER use `StdinPipe()` - always use `strings.NewReader()`
- Always add timeout protection
- Handle graceful shutdown before force kill

### Bubble Tea Integration
- NEVER use goroutines directly in Bubble Tea programs
- All async operations must be Commands
- State changes only in Update()

### Audio Memory Management
- Keep audio data references alive during playback
- OTO streams data, doesn't load everything at once
- Prevent garbage collection of playing audio

## Dependencies

- `github.com/ebitengine/oto/v3` - Cross-platform audio playback
- External: Piper binary for offline TTS
- External: Google Cloud TTS API for online TTS