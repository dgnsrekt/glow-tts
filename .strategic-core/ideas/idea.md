<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# **Glow-TTS Product Requirements Document (PRD) - Final Version**

**Project Name:** Glow-TTS
**Version:** 3.0 Final
**Date:** August 10, 2025
**Architecture:** Background Service with Queue-Based Processing and Enhanced Engine Management

***

## **1. Project Overview**

### **1.1 Project Summary**

Glow-TTS is a fork of the popular Glow markdown reader that adds sophisticated Text-to-Speech (TTS) functionality through a background service architecture. The system uses engine-specific CLI flags (`--tts piper` or `--tts gtts`) to activate a persistent background TTS service with intelligent queue-based sentence processing, preprocessing capabilities, and robust engine availability detection.

### **1.2 Core Philosophy**

The architecture separates TTS processing from the main UI thread while enforcing Terminal User Interface (TUI) mode for interactive reading experiences. The system provides intelligent engine selection, graceful fallback handling, and sophisticated process management to ensure reliable operation across diverse environments.

***

## **2. CLI Integration and Engine Selection**

### **2.1 TTS Mode Activation**

- **Mandatory TUI Mode:** The `--tts` flag **always forces Terminal User Interface mode**, equivalent to using Glow's existing `-t` flag
- **Engine Selection Syntax:**

```bash
glow --tts piper document.md      # Use Piper TTS engine
glow --tts gtts document.md       # Use Google TTS engine
```

- **Process Exclusivity:** System checks for existing `glow --tts` processes and prevents multiple TTS instances


### **2.2 Engine Availability Detection**

#### **2.2.1 Piper Engine Validation**

- **Binary Detection:** Check if `piper` executable exists in `$PATH` or configured location
- **Model Validation:** Verify presence of required voice models (`.onnx` files) in Piper model directory
- **Error Messages:**
    - **Binary Missing:** "Piper TTS not found. Install from https://github.com/rhasspy/piper and ensure binary is in PATH."
    - **Model Missing:** "Piper installed but no voice models found. Download models from Piper repository and configure location."


#### **2.2.2 Google TTS Engine Validation**

- **Environment Variable Check:** Look for `GLOW_GTTS_API_KEY` environment variable
- **API Key Behavior:**
    - **If Present:** Use authenticated Google TTS with full feature access
    - **If Absent:** Attempt fallback to unauthenticated Google TTS (if available)
- **Connectivity Check:** Verify internet connection and API endpoint accessibility
- **Error Messages:**
    - **No Internet:** "Google TTS unavailable. Check internet connection."
    - **No API Key:** "Google TTS running without API key. Export GLOW_GTTS_API_KEY for full features."


### **2.3 Process Instance Management**

- **Single Instance Enforcement:** Detect existing `glow --tts` processes using process detection
- **Conflict Resolution:** If another TTS instance is running, display message and revert to standard Glow functionality
- **Clean Shutdown:** Ensure proper cleanup of TTS service processes on application termination

***

## **3. System Architecture**

### **3.1 Background Service Design**

- **Service Spawning:** Main process creates child TTS service process during initialization
- **Process Separation:** TTS service runs independently to prevent UI blocking
- **Communication Protocol:** Inter-Process Communication (IPC) using Go channels or named pipes
- **Resource Management:** Service manages TTS engines, audio buffers, and system audio independently


### **3.2 Queue System Architecture**

#### **3.2.1 Advanced Queue Operations**

```go
type SentenceQueue interface {
    Enqueue(sentence string, position int) error
    Dequeue() (*Sentence, error)
    Interrupt() error                    // Stop current, preserve position
    SkipForward() error                 // Next sentence
    SkipBack() error                    // Previous sentence  
    Pause() error                       // Suspend processing
    Resume() error                      // Continue processing
    GetStatus() *QueueStatus            // Current state info
}
```


#### **3.2.2 Intelligent Preprocessing**

- **Look-ahead Processing:** Generate audio for next 2-3 sentences during current playback
- **Adaptive Buffering:** Monitor user navigation patterns to optimize preprocessing
- **Memory Management:** Automatic cleanup of unused audio buffers with configurable limits
- **Performance Optimization:** Balance preprocessing depth with memory usage


### **3.3 Local Caching Strategy**

- **Cache Location:** Use system temporary directory (`/tmp` on Unix-like systems)[^1][^2]
- **Cache Namespace:** Create subdirectory `/tmp/glow-tts-{pid}/` for process isolation
- **Cache Lifecycle:** Automatic cleanup on application exit and periodic cleanup of stale cache files
- **Cross-platform Support:** Use Go's `os.TempDir()` for platform-appropriate temp directory detection
- **Security:** Set appropriate file permissions (0600) for cached audio files

***

## **4. Functional Requirements**

### **4.1 Core TTS Integration**

- **REQ-001:** System shall recognize `--tts [engine]` CLI syntax and enforce TUI mode
- **REQ-002:** System shall validate selected TTS engine availability before service initialization
- **REQ-003:** System shall provide clear error messages and setup instructions for engine failures
- **REQ-004:** System shall prevent multiple TTS instances and gracefully handle conflicts


### **4.2 Engine Management**

- **REQ-005:** Piper integration shall verify binary existence and voice model availability
- **REQ-006:** Google TTS shall check for `GLOW_GTTS_API_KEY` environment variable
- **REQ-007:** System shall attempt unauthenticated Google TTS fallback when API key unavailable
- **REQ-008:** Engine failures shall trigger informative user guidance for resolution


### **4.3 Background Service Operations**

- **REQ-009:** Service shall initialize within 3 seconds of CLI invocation
- **REQ-010:** Service shall maintain persistent connection with main application
- **REQ-011:** Service shall handle graceful shutdown on main application termination
- **REQ-012:** Service shall provide health monitoring and automatic recovery capabilities


### **4.4 Queue System Functionality**

- **REQ-013:** Queue shall support real-time interrupt for immediate playback changes
- **REQ-014:** System shall preprocess next 2-3 sentences during current sentence playback
- **REQ-015:** Navigation commands (forward/back) shall respond within 200ms
- **REQ-016:** Queue shall maintain sentence position tracking for synchronized highlighting


### **4.5 Caching and Performance**

- **REQ-017:** System shall cache generated audio in platform-appropriate temporary directory
- **REQ-018:** Cache files shall use process-specific naming to prevent conflicts
- **REQ-019:** Automatic cache cleanup shall occur on application exit
- **REQ-020:** Cache storage shall not exceed 100MB per session with automatic pruning

***

## **5. User Experience Specifications**

### **5.1 Startup Flow**

1. **CLI Parsing:** User runs `glow --tts [engine] document.md`
2. **Engine Validation:** System checks selected engine availability
3. **Process Check:** Verify no existing TTS instances are running
4. **TUI Activation:** Force TUI mode with TTS enhancement indicators
5. **Service Initialization:** Background service starts with loading indicators
6. **Ready State:** TTS controls become available with status confirmation

### **5.2 Error Handling Experience**

- **Engine Unavailable:** Clear error message with setup instructions and fallback options
- **Process Conflict:** Informative message about existing TTS instance with graceful fallback
- **API Failures:** Helpful guidance for API key configuration or connectivity issues
- **Recovery Options:** User-friendly prompts for switching engines or retrying initialization


### **5.3 Control Interface**

- **Status Indicators:** Real-time display of TTS engine status, queue position, and preprocessing activity
- **Loading Feedback:** Visual indicators during service warmup, engine switching, and preprocessing
- **Navigation Controls:** Responsive sentence-level navigation with immediate visual feedback
- **Error Recovery:** Clear options for troubleshooting and manual engine switching

***

## **6. Technical Implementation Details**

### **6.1 Process Management**

- **Instance Detection:** Use process enumeration to detect existing `glow --tts` instances
- **PID File Management:** Create lock files in temp directory for process coordination
- **Resource Cleanup:** Comprehensive cleanup of processes, files, and system resources on exit
- **Signal Handling:** Proper SIGTERM/SIGINT handling for graceful shutdown


### **6.2 Engine Integration Architecture**

```go
type TTSEngine interface {
    Initialize() error
    IsAvailable() bool
    Synthesize(text string) (io.Reader, error)
    GetVoices() ([]Voice, error)
    Cleanup() error
}
```


### **6.3 Cache Management System**

- **Directory Structure:** `/tmp/glow-tts-{pid}/{session-id}/` for organized cache storage
- **File Naming:** Hash-based naming for sentences to enable efficient lookup and deduplication
- **Cleanup Strategy:** Age-based cleanup (files older than 1 hour) plus size-based limits
- **Cross-platform Paths:** Platform-appropriate temp directory detection and permissions


### **6.4 Communication Protocol**

```go
type TTSCommand struct {
    Type     CommandType  // PLAY, PAUSE, SKIP_FORWARD, SKIP_BACK, STOP
    Position int          // Sentence position
    Data     string       // Additional command data
}

type TTSStatus struct {
    State       ServiceState  // LOADING, READY, PLAYING, PAUSED, BUFFERING
    Position    int          // Current sentence position  
    QueueSize   int          // Total sentences in queue
    BufferLevel float64     // Preprocessing buffer fill percentage
    Error       string      // Error message if applicable
}
```


***

## **7. Error Handling and Recovery**

### **7.1 Engine Failure Recovery**

- **Automatic Retry:** Retry failed TTS requests up to 3 times with exponential backoff
- **Engine Switching:** Option to switch between Piper and Google TTS during runtime
- **Graceful Degradation:** Continue operation with reduced functionality during engine failures
- **User Notification:** Clear status updates and recovery options during failures


### **7.2 Process Management Resilience**

- **Service Monitoring:** Health checks for background TTS service with automatic restart
- **Resource Management:** Memory and CPU monitoring with automatic throttling under load
- **Network Resilience:** Robust handling of network failures for Google TTS with local fallback
- **State Recovery:** Preserve playback position and queue state during brief service interruptions


### **7.3 Cache System Reliability**

- **Corruption Handling:** Automatic detection and cleanup of corrupted cache files
- **Disk Space Management:** Intelligent cache pruning when disk space becomes limited
- **Permission Handling:** Graceful fallback when temp directory permissions are restricted
- **Cleanup Verification:** Ensure complete cleanup even during unexpected termination

***

## **8. Performance Requirements**

### **8.1 Initialization Performance**

- **Engine Detection:** Complete availability checks within 1 second
- **Service Startup:** Background service ready within 3 seconds
- **First Sentence:** Audio playback begins within 2 seconds of play command
- **UI Responsiveness:** Main UI remains responsive (<100ms input latency) during all operations


### **8.2 Runtime Performance**

- **Navigation Latency:** Skip forward/back commands respond within 200ms
- **Preprocessing Efficiency:** Next sentence audio ready before current sentence completes
- **Memory Usage:** Service memory footprint stable under 75MB during normal operation
- **Cache Performance:** File system operations complete without noticeable UI delays


### **8.3 Resource Management**

- **CPU Usage:** TTS processing distributed to avoid blocking main application thread
- **Disk I/O:** Minimal impact on system performance during cache operations
- **Network Usage:** Efficient Google TTS API usage with request batching and caching
- **Audio Latency:** <150ms from command to audio output for preprocessed content

***

## **9. Security and Privacy**

### **9.1 Data Protection**

- **API Key Security:** Never log or expose `GLOW_GTTS_API_KEY` in error messages or debug output
- **Content Privacy:** Warn users that Google TTS sends document content to external servers
- **Local Cache Security:** Set restrictive permissions (0600) on cached audio files
- **Process Isolation:** TTS service runs with minimal necessary privileges


### **9.2 Network Security**

- **HTTPS Enforcement:** All Google TTS API communications use encrypted connections
- **Certificate Validation:** Proper SSL certificate verification for cloud TTS services
- **Network Timeout:** Reasonable timeout values to prevent hanging network requests
- **Error Information:** Sanitize network error messages to avoid information leakage

***

## **10. Testing and Quality Assurance**

### **10.1 Functional Testing**

- **Engine Detection:** Verify accurate detection of available/unavailable TTS engines
- **Process Management:** Test single-instance enforcement and conflict resolution
- **Queue Operations:** Validate all queue manipulations under various load conditions
- **Cache Management:** Confirm proper cache lifecycle and cleanup behavior


### **10.2 Performance Testing**

- **Load Testing:** Verify performance with large documents (>1MB) and extended sessions
- **Memory Testing:** Confirm no memory leaks during long TTS sessions (>2 hours)
- **Concurrency Testing:** Validate behavior with rapid navigation and command sequences
- **Resource Testing:** Ensure graceful handling of system resource constraints


### **10.3 Error Scenario Testing**

- **Engine Failures:** Test recovery from TTS engine crashes and network interruptions
- **Process Conflicts:** Verify proper handling of multiple instance attempts
- **Cache Failures:** Test behavior when temp directory is unavailable or full
- **API Limitations:** Validate graceful handling of Google TTS API rate limits and errors

***

## **11. Documentation and User Support**

### **11.1 User Documentation**

- **Installation Guide:** Complete setup instructions for Piper TTS, voice models, and API keys
- **Quick Start Tutorial:** Step-by-step first-use guide with screenshots and examples
- **Troubleshooting Guide:** Common error scenarios with clear resolution steps
- **Advanced Configuration:** Engine selection, voice customization, and performance tuning


### **11.2 Technical Documentation**

- **Architecture Overview:** Detailed system design with component interaction diagrams
- **API Reference:** Complete IPC protocol specification and message formats
- **Developer Guide:** Extension points, customization options, and contribution guidelines
- **Performance Tuning:** Optimization recommendations for various system configurations

***

This comprehensive PRD provides a robust foundation for implementing Glow-TTS with sophisticated engine management, reliable process control, and an exceptional user experience while maintaining the performance and simplicity that makes Glow popular among terminal users.

<div style="text-align: center">⁂</div>

[^1]: https://community.adobe.com/t5/audition-discussions/moving-media-cache/td-p/8965339

[^2]: https://stackoverflow.com/questions/3957998/is-it-ok-to-rely-on-the-writeability-of-tmp-folder

[^3]: https://forum.audacityteam.org/t/removing-any-cached-temp-audio-files/50468

[^4]: https://github.com/google/gops

[^5]: https://www.reddit.com/r/Python/comments/11po5eq/best_practices_for_caching_data_between_runs/

[^6]: https://learn.microsoft.com/en-us/answers/questions/2717984/find-temp-or-cached-file-for-sound-recorder

[^7]: https://stackoverflow.com/questions/52964022/how-to-detect-if-the-current-go-process-is-running-in-a-headless-non-gui-envir

[^8]: https://www.youtube.com/watch?v=b6cnkomw35E

[^9]: https://go.dev/doc/diagnostics

[^10]: https://livsycode.com/best-practices/filemanager-directories-documents-vs-application-support-vs-tmp-vs-caches/

[^11]: https://discussions.apple.com/thread/2244392

[^12]: https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html

[^13]: https://stackoverflow.com/questions/14366646/is-it-considered-good-acceptable-practice-to-save-a-file-in-the-temporary-direct

[^14]: https://community.adobe.com/t5/audition/best-locations-for-temp-cache-files/m-p/11224735

[^15]: https://go.dev/blog/race-detector

[^16]: https://answers.netlify.com/t/file-based-caching-in-tmp-directory/90920

[^17]: https://www.psafe.com/en/blog/whats-the-difference-between-cache-and-temporary-files/

[^18]: https://www.reddit.com/r/golang/comments/mk7zfm/which_design_pattern_for_process_monitoringcontrol/

[^19]: https://www.wipster.io/blog/how-to-clear-media-cache-files-in-premiere-pro

[^20]: https://support.izotope.com/hc/en-us/articles/6658275507985-How-to-clear-RX-s-Session-Data-Folder

