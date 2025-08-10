# Glow-TTS Mission

## Project Vision

Glow-TTS is an enhanced fork of the popular Glow markdown reader that adds sophisticated Text-to-Speech (TTS) functionality through a background service architecture, bringing accessible and seamless audio narration to terminal-based markdown reading.

## Target Audience

### Primary Users
- **Terminal Power Users**: Developers and system administrators who prefer CLI tools
- **Accessibility-Focused Users**: People who benefit from audio narration of documentation
- **Documentation Readers**: Users who regularly consume technical documentation in markdown format
- **Multitaskers**: Users who want to listen to documentation while performing other tasks

### Secondary Users
- **DevOps Engineers**: Reading system documentation and runbooks
- **Open Source Contributors**: Reviewing project documentation
- **Technical Writers**: Testing documentation accessibility

## Core Purpose

Glow-TTS solves the problem of consuming lengthy technical documentation in the terminal by:
- Adding natural-sounding text-to-speech capabilities to markdown reading
- Enabling hands-free documentation consumption
- Improving accessibility for visually impaired users
- Supporting multiple TTS engines (Piper for offline, Google TTS for cloud)
- Maintaining the simplicity and elegance of the original Glow experience

## Core Values

1. **Accessibility First**: Making terminal-based documentation accessible to all users
2. **Performance**: Non-blocking TTS processing that doesn't compromise UI responsiveness
3. **Flexibility**: Support for multiple TTS engines with graceful fallback
4. **Simplicity**: Intuitive CLI flags and seamless TUI integration
5. **Reliability**: Robust error handling and process management

## Success Metrics

- **Adoption Rate**: Number of users adopting TTS features
- **Performance**: TTS initialization under 3 seconds, navigation response under 200ms
- **Reliability**: 99.9% uptime for TTS service during sessions
- **User Satisfaction**: Positive feedback on accessibility improvements
- **Engine Coverage**: Support for both offline (Piper) and online (Google TTS) options

## Key Differentiators

1. **Terminal-Native**: First-class TTS support in a terminal markdown reader
2. **Background Service Architecture**: Non-blocking audio processing
3. **Intelligent Queue System**: Look-ahead preprocessing for smooth playback
4. **Multi-Engine Support**: Flexibility between offline and cloud TTS engines
5. **Sentence-Level Control**: Precise navigation and synchronization
6. **Automatic Fallback**: Graceful degradation when preferred engine unavailable

## Project Scope

### In Scope
- TTS integration via `--tts` CLI flag
- Support for Piper (offline) and Google TTS (cloud) engines
- Background service for audio processing
- Queue-based sentence processing
- TUI mode enforcement with TTS
- Sentence-level navigation controls
- Audio caching system
- Process management and single-instance enforcement

### Out of Scope
- GUI interface
- Support for non-markdown formats
- Real-time voice changing
- Custom voice training
- Multi-language automatic detection
- Mobile application development

## Long-term Goals

1. **Expand Engine Support**: Add more TTS engines (Azure, AWS Polly, local AI models)
2. **Enhanced Accessibility**: Screen reader integration, keyboard shortcuts
3. **Voice Customization**: User-configurable voice parameters
4. **Smart Reading**: Skip code blocks, read only prose, smart emphasis
5. **Cross-Platform Excellence**: Optimize for all major operating systems