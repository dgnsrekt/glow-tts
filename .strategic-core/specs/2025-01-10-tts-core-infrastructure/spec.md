# TTS Core Infrastructure Specification

## Feature Overview

Add comprehensive Text-to-Speech (TTS) functionality to Glow, enabling users to listen to markdown documentation through synthesized speech. The implementation will support multiple TTS engines, background processing, and seamless UI integration.

## User Stories

### As a Developer
- I want to listen to documentation while coding so I can multitask effectively
- I want offline TTS capability so I can work without internet connectivity
- I want to control playback speed so I can adjust to my preference

### As a Visually Impaired User
- I want audio narration of markdown content so I can access documentation
- I want clear sentence-by-sentence navigation so I can follow along easily
- I want keyboard shortcuts for all TTS controls so I can operate without a mouse

### As a Documentation Reader
- I want automatic preprocessing of upcoming content so playback is smooth
- I want to skip code blocks when listening so I hear only prose content
- I want synchronized highlighting so I can see what's being read

## Acceptance Criteria

### Functional Requirements
- [ ] TTS is ONLY available when `--tts [engine]` flag is used
- [ ] Without `--tts` flag, all TTS code is completely inactive
- [ ] Supports Piper engine for offline synthesis
- [ ] Supports Google TTS via gTTS (no API key required)  
- [ ] Requires explicit engine selection (no automatic fallback)
- [ ] Manual start required (no auto-play on document open)
- [ ] Sentence-level navigation (next/previous)
- [ ] Playback controls (play/pause/stop)
- [ ] Speed adjustment (0.5x to 2.0x)
- [ ] TUI mode is enforced when TTS flag is used
- [ ] Audio caching for repeated content
- [ ] Space key only controls TTS when `--tts` flag is used

### Non-Functional Requirements
- [ ] TTS initialization completes within 3 seconds
- [ ] Audio starts playing within 200ms of request
- [ ] Memory usage stays under 75MB with TTS active
- [ ] No UI blocking during synthesis
- [ ] Clear error messages when engine unavailable
- [ ] Single TTS instance enforcement

## Success Metrics

- **Performance**: 95% of synthesis requests complete within 200ms
- **Reliability**: 99.9% uptime during active sessions
- **Cache Hit Rate**: >80% for repeated sentences
- **User Satisfaction**: Positive feedback on accessibility
- **Memory Efficiency**: <75MB total with TTS active

## Dependencies

### External Dependencies
- **Piper**: ONNX runtime and voice models
- **gTTS (Google TTS)**: Python gtts package for free TTS (no API key required)
- **ffmpeg**: For MP3 to PCM conversion (required for gTTS)
- **Audio Library**: oto/v3 for cross-platform playback

### Internal Dependencies
- **Bubble Tea**: UI framework for TTS controls
- **Glamour**: Markdown parsing for text extraction
- **Viper**: Configuration management

## Constraints

### Technical Constraints
- Must maintain existing Glow functionality
- Cannot block UI during audio processing
- Must work on Linux, macOS, and Windows
- Audio format: PCM 16-bit mono

### Resource Constraints
- Maximum 100MB cache size per session
- Maximum 5000 characters per synthesis request
- Single background process for TTS

### Security Constraints
- API keys must be stored securely
- No logging of synthesized text content
- Cache files must have restricted permissions (0600)

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌──────────────┐
│     CLI     │────▶│     TUI      │────▶│ TTS Control  │
└─────────────┘     └──────────────┘     └──────────────┘
                            │                     │
                            ▼                     ▼
                    ┌──────────────┐     ┌──────────────┐
                    │   Markdown   │     │   Sentence   │
                    │    Parser    │────▶│    Queue     │
                    └──────────────┘     └──────────────┘
                                                 │
                            ┌────────────────────┴────────┐
                            ▼                             ▼
                    ┌──────────────┐            ┌──────────────┐
                    │ Piper Engine │            │ Google TTS   │
                    └──────────────┘            └──────────────┘
                            │                             │
                            └──────────┬──────────────────┘
                                       ▼
                            ┌──────────────────┐
                            │   Audio Cache    │
                            └──────────────────┘
                                       │
                                       ▼
                            ┌──────────────────┐
                            │  Audio Player    │
                            └──────────────────┘
```

## Risk Assessment

### High Risk
- **Audio Device Conflicts**: Multiple applications competing for audio
  - *Mitigation*: Implement device locking and graceful fallback
- **Memory Leaks**: Long-running sessions accumulating memory
  - *Mitigation*: Implement cache eviction and resource cleanup

### Medium Risk
- **Engine Availability**: Piper models not installed
  - *Mitigation*: Clear installation instructions and auto-download option
- **Network Latency**: Slow Google TTS responses
  - *Mitigation*: Aggressive caching and preprocessing

### Low Risk
- **Platform Differences**: Audio implementation varies by OS
  - *Mitigation*: Use cross-platform audio library (oto)

## Timeline Estimate

- **Phase 1** (Core Infrastructure): 2-3 days
- **Phase 2** (Engine Integration): 2-3 days
- **Phase 3** (UI Integration): 1-2 days
- **Phase 4** (Testing & Polish): 2-3 days
- **Total**: 7-11 days