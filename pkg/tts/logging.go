package tts

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
)

// MetricsLogger tracks and logs TTS performance metrics
type MetricsLogger struct {
	enabled bool
	logger  *log.Logger
}

// Metrics holds performance metrics
type Metrics struct {
	Engine            string
	Text              string
	TextLength        int
	SynthesisStart    time.Time
	SynthesisEnd      time.Time
	SynthesisDuration time.Duration
	AudioBytes        int
	CacheHit          bool
	ErrorOccurred     bool
	ErrorMessage      string
}

var (
	// Global metrics logger
	metricsLogger *MetricsLogger
	
	// Performance tracking
	synthesisMetrics []Metrics
)

// InitializeLogging sets up TTS-specific logging
func InitializeLogging(debugMode bool, traceMode bool) error {
	// Set log level based on flags
	if traceMode {
		log.SetLevel(log.DebugLevel) // Trace maps to Debug in charmbracelet/log
		log.Debug("TTS logging initialized", "level", "TRACE")
	} else if debugMode {
		log.SetLevel(log.DebugLevel)
		log.Debug("TTS logging initialized", "level", "DEBUG")
	} else {
		log.SetLevel(log.InfoLevel)
	}
	
	// Initialize metrics logger
	metricsLogger = &MetricsLogger{
		enabled: debugMode || traceMode,
		logger:  log.Default(),
	}
	
	// Create debug log file if in debug mode
	if debugMode || traceMode {
		if err := setupDebugLogFile(); err != nil {
			log.Warn("Failed to setup debug log file", "error", err)
		}
	}
	
	return nil
}

// setupDebugLogFile creates a debug log file
func setupDebugLogFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	logDir := filepath.Join(home, ".glow")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	
	logPath := filepath.Join(logDir, "tts-debug.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	// Note: We don't close the file here as it needs to remain open for logging
	log.Debug("TTS debug log file created", "path", logPath)
	
	// Create a file logger
	fileLogger := log.NewWithOptions(file, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Level:           log.DebugLevel,
	})
	
	// Store for later use
	metricsLogger.logger = fileLogger
	
	return nil
}

// StartSynthesis starts tracking synthesis metrics
func StartSynthesis(engine, text string) *Metrics {
	m := &Metrics{
		Engine:         engine,
		Text:           text,
		TextLength:     len(text),
		SynthesisStart: time.Now(),
	}
	
	if metricsLogger != nil && metricsLogger.enabled {
		metricsLogger.logger.Debug("Synthesis started",
			"engine", engine,
			"textLength", len(text),
			"timestamp", m.SynthesisStart.Format(time.RFC3339))
	}
	
	return m
}

// EndSynthesis completes tracking synthesis metrics
func (m *Metrics) EndSynthesis(audioBytes int, cacheHit bool, err error) {
	m.SynthesisEnd = time.Now()
	m.SynthesisDuration = m.SynthesisEnd.Sub(m.SynthesisStart)
	m.AudioBytes = audioBytes
	m.CacheHit = cacheHit
	
	if err != nil {
		m.ErrorOccurred = true
		m.ErrorMessage = err.Error()
	}
	
	// Store metrics
	synthesisMetrics = append(synthesisMetrics, *m)
	
	// Log metrics
	if metricsLogger != nil && metricsLogger.enabled {
		if m.ErrorOccurred {
			metricsLogger.logger.Error("Synthesis failed",
				"engine", m.Engine,
				"duration", m.SynthesisDuration,
				"error", m.ErrorMessage)
		} else {
			metricsLogger.logger.Info("Synthesis completed",
				"engine", m.Engine,
				"textLength", m.TextLength,
				"audioBytes", m.AudioBytes,
				"duration", m.SynthesisDuration,
				"cacheHit", m.CacheHit,
				"bytesPerSecond", calculateBytesPerSecond(m.AudioBytes, m.SynthesisDuration))
		}
	}
}

// calculateBytesPerSecond calculates synthesis throughput
func calculateBytesPerSecond(bytes int, duration time.Duration) string {
	if duration == 0 {
		return "N/A"
	}
	bps := float64(bytes) / duration.Seconds()
	return fmt.Sprintf("%.2f bytes/sec", bps)
}

// LogCacheHit logs a cache hit event
func LogCacheHit(key string, size int) {
	if metricsLogger != nil && metricsLogger.enabled {
		metricsLogger.logger.Debug("Cache hit",
			"key", key,
			"size", size)
	}
}

// LogCacheMiss logs a cache miss event
func LogCacheMiss(key string) {
	if metricsLogger != nil && metricsLogger.enabled {
		metricsLogger.logger.Debug("Cache miss",
			"key", key)
	}
}

// LogEngineSelection logs engine selection
func LogEngineSelection(engine string, reason string) {
	log.Info("TTS engine selected",
		"engine", engine,
		"reason", reason)
}

// LogSubprocessExecution logs subprocess execution
func LogSubprocessExecution(command string, args []string, duration time.Duration, err error) {
	if metricsLogger != nil && metricsLogger.enabled {
		if err != nil {
			metricsLogger.logger.Error("Subprocess failed",
				"command", command,
				"args", args,
				"duration", duration,
				"error", err)
		} else {
			metricsLogger.logger.Debug("Subprocess executed",
				"command", command,
				"args", args,
				"duration", duration)
		}
	}
}

// LogPlaybackEvent logs playback events
func LogPlaybackEvent(event string, details map[string]interface{}) {
	if metricsLogger != nil && metricsLogger.enabled {
		metricsLogger.logger.Debug("Playback event",
			"event", event,
			"details", details)
	}
}

// GetSynthesisStats returns synthesis statistics
func GetSynthesisStats() string {
	if len(synthesisMetrics) == 0 {
		return "No synthesis metrics available"
	}
	
	var totalDuration time.Duration
	var totalBytes int
	var cacheHits int
	var errors int
	
	for _, m := range synthesisMetrics {
		totalDuration += m.SynthesisDuration
		totalBytes += m.AudioBytes
		if m.CacheHit {
			cacheHits++
		}
		if m.ErrorOccurred {
			errors++
		}
	}
	
	avgDuration := totalDuration / time.Duration(len(synthesisMetrics))
	cacheHitRate := float64(cacheHits) / float64(len(synthesisMetrics)) * 100
	
	return fmt.Sprintf(
		"Synthesis Stats:\n"+
		"  Total: %d\n"+
		"  Avg Duration: %v\n"+
		"  Total Bytes: %d\n"+
		"  Cache Hit Rate: %.1f%%\n"+
		"  Errors: %d",
		len(synthesisMetrics),
		avgDuration,
		totalBytes,
		cacheHitRate,
		errors,
	)
}

// EnableDebugLogging enables debug logging at runtime
func EnableDebugLogging() {
	log.SetLevel(log.DebugLevel)
	if metricsLogger != nil {
		metricsLogger.enabled = true
	}
	log.Debug("Debug logging enabled")
}

// DisableDebugLogging disables debug logging at runtime
func DisableDebugLogging() {
	log.SetLevel(log.InfoLevel)
	if metricsLogger != nil {
		metricsLogger.enabled = false
	}
}