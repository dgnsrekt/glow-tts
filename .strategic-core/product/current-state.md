# Current State - Glow-TTS

## Implementation Status

### ✅ Completed Features

1. **Core TTS Infrastructure**
   - TTSController created and integrated with pager UI
   - State machine for TTS operations (Idle, Ready, Playing, Paused, Stopped)
   - Keyboard shortcuts implemented (t=toggle, space=play/pause, s=stop)
   - Content loading mechanism from markdown documents
   - Debug logging system to `/tmp/glow_tts_debug.log`

2. **Mock TTS Engine**
   - Fully functional mock engine for testing
   - Generates 440Hz tone (A4 note) for audio verification
   - Proper timing simulation for sentences
   - Successfully integrated with audio player

3. **Audio Playback System**
   - Oto audio library integrated
   - Cross-platform audio output working
   - PCM audio format support (16-bit, 44100Hz)
   - Mock engine produces audible output

4. **Fallback System**
   - FallbackEngine wrapper implemented
   - Automatic switching from primary to secondary engine
   - Configurable failure threshold
   - Thread-safe operation with sync.RWMutex

### 🚧 Partially Working Features

1. **Piper TTS Integration**
   - Piper binary is installed and working
   - Model available (en_US-amy-medium.onnx)
   - Process management code written but failing

2. **Sentence Highlighting**
   - ApplyTTSHighlighting function implemented
   - Shows "🔊 TTS: Sentence X" indicator
   - Not visible in actual UI rendering

### ❌ Broken Features

1. **Piper Process Stability**
   - Process starts but dies immediately
   - Health check fails after 500ms
   - Falls back to mock engine every time

2. **UI Rendering Issues**  
   - Sentence highlighting indicator not showing
   - TTS status may not update properly

3. **Command Line Interface**
   - Requires -t flag to keep TUI open
   - Without flag, exits immediately

## Known Bugs

### 🔴 Critical Bugs

#### Bug #1: Piper Process Dies Immediately
**Location**: tts/engines/piper/piper_v2.go:281
**Root Cause**: Process.Signal(nil) health check unreliable

#### Bug #2: Sentence Highlighting Not Visible
**Location**: ui/pager.go:365 and ui/highlighting.go:141
**Root Cause**: Content rendered by Glamour before highlighting

#### Bug #3: TUI Exits Without -t Flag
**Location**: main.go or ui/ui.go
**Root Cause**: Auto-detection of TUI mode not working
EOF < /dev/null