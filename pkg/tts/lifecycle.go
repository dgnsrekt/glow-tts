package tts

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
)

// LifecycleManager coordinates graceful shutdown of TTS components
type LifecycleManager struct {
	mu          sync.Mutex
	components  []LifecycleComponent
	shutdownCh  chan struct{}
	done        chan struct{}
	wg          sync.WaitGroup
	isShutdown  bool
	forceKillTimeout time.Duration
}

// LifecycleComponent represents a component that needs cleanup on shutdown
type LifecycleComponent interface {
	// Name returns the component name for logging
	Name() string
	
	// Shutdown performs graceful shutdown
	Shutdown(ctx context.Context) error
	
	// ForceStop performs immediate termination if graceful shutdown fails
	ForceStop() error
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		components:  make([]LifecycleComponent, 0),
		shutdownCh:  make(chan struct{}),
		done:        make(chan struct{}),
		forceKillTimeout: 5 * time.Second,
	}
}

// Register adds a component to lifecycle management
func (lm *LifecycleManager) Register(component LifecycleComponent) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if lm.isShutdown {
		log.Warn("Cannot register component during shutdown", "component", component.Name())
		return
	}
	
	lm.components = append(lm.components, component)
	log.Debug("Registered lifecycle component", "name", component.Name())
}

// Start begins monitoring for shutdown signals
func (lm *LifecycleManager) Start() {
	lm.wg.Add(1)
	go lm.monitorSignals()
}

// monitorSignals watches for system signals
func (lm *LifecycleManager) monitorSignals() {
	defer lm.wg.Done()
	
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case sig := <-sigCh:
		log.Info("Received shutdown signal", "signal", sig)
		lm.Shutdown()
	case <-lm.shutdownCh:
		log.Debug("Shutdown initiated programmatically")
	}
}

// Shutdown performs graceful shutdown of all components
func (lm *LifecycleManager) Shutdown() error {
	lm.mu.Lock()
	if lm.isShutdown {
		lm.mu.Unlock()
		return nil
	}
	lm.isShutdown = true
	lm.mu.Unlock()
	
	log.Info("Starting graceful shutdown")
	close(lm.shutdownCh)
	
	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), lm.forceKillTimeout)
	defer cancel()
	
	// Shutdown components in reverse order of registration
	var shutdownErrors []error
	for i := len(lm.components) - 1; i >= 0; i-- {
		component := lm.components[i]
		log.Debug("Shutting down component", "name", component.Name())
		
		// Try graceful shutdown
		if err := component.Shutdown(ctx); err != nil {
			log.Warn("Component graceful shutdown failed", 
				"name", component.Name(),
				"error", err)
			
			// Force stop if graceful shutdown failed
			if forceErr := component.ForceStop(); forceErr != nil {
				log.Error("Component force stop failed",
					"name", component.Name(),
					"error", forceErr)
				shutdownErrors = append(shutdownErrors, forceErr)
			}
		}
	}
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		lm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Info("Graceful shutdown complete")
	case <-time.After(2 * time.Second):
		log.Warn("Timeout waiting for goroutines to finish")
	}
	
	close(lm.done)
	
	if len(shutdownErrors) > 0 {
		return fmt.Errorf("shutdown completed with %d errors", len(shutdownErrors))
	}
	
	return nil
}

// Wait blocks until shutdown is complete
func (lm *LifecycleManager) Wait() {
	<-lm.done
}

// EngineLifecycle wraps a TTS engine with lifecycle management
type EngineLifecycle struct {
	engine TTSEngine
	name   string
}

// NewEngineLifecycle creates a lifecycle wrapper for a TTS engine
func NewEngineLifecycle(engine TTSEngine, name string) *EngineLifecycle {
	return &EngineLifecycle{
		engine: engine,
		name:   name,
	}
}

// Name returns the engine name
func (el *EngineLifecycle) Name() string {
	return fmt.Sprintf("TTS Engine: %s", el.name)
}

// Shutdown performs graceful engine shutdown
func (el *EngineLifecycle) Shutdown(ctx context.Context) error {
	// Check if engine implements cleanup interface
	if cleaner, ok := el.engine.(interface{ Cleanup() error }); ok {
		return cleaner.Cleanup()
	}
	return nil
}

// ForceStop performs immediate engine termination
func (el *EngineLifecycle) ForceStop() error {
	// Most engines don't need force stop, but we could kill subprocesses here
	return nil
}

// QueueLifecycle wraps the audio queue with lifecycle management
type QueueLifecycle struct {
	queue *TTSAudioQueue
}

// NewQueueLifecycle creates a lifecycle wrapper for the audio queue
func NewQueueLifecycle(queue *TTSAudioQueue) *QueueLifecycle {
	return &QueueLifecycle{queue: queue}
}

// Name returns the component name
func (ql *QueueLifecycle) Name() string {
	return "TTS Audio Queue"
}

// Shutdown performs graceful queue shutdown
func (ql *QueueLifecycle) Shutdown(ctx context.Context) error {
	if ql.queue == nil {
		return nil
	}
	
	// Stop the queue
	ql.queue.Stop()
	
	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		// Queue should handle its own worker shutdown
		time.Sleep(100 * time.Millisecond) // Give workers time to finish
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("queue shutdown timed out")
	}
}

// ForceStop performs immediate queue termination
func (ql *QueueLifecycle) ForceStop() error {
	if ql.queue != nil {
		ql.queue.Stop()
	}
	return nil
}

// PlayerLifecycle wraps the audio player with lifecycle management
type PlayerLifecycle struct {
	player *AudioPlayer
}

// NewPlayerLifecycle creates a lifecycle wrapper for the audio player
func NewPlayerLifecycle(player *AudioPlayer) *PlayerLifecycle {
	return &PlayerLifecycle{player: player}
}

// Name returns the component name
func (pl *PlayerLifecycle) Name() string {
	return "Audio Player"
}

// Shutdown performs graceful player shutdown
func (pl *PlayerLifecycle) Shutdown(ctx context.Context) error {
	player := GetGlobalAudioPlayer()
	if player != nil {
		player.Stop()
		player.Close()
	}
	return nil
}

// ForceStop performs immediate player termination
func (pl *PlayerLifecycle) ForceStop() error {
	// Same as graceful for audio player
	return pl.Shutdown(context.Background())
}

// CacheLifecycle wraps the cache with lifecycle management
type CacheLifecycle struct {
	cache    *Cache
	flushOnShutdown bool
}

// NewCacheLifecycle creates a lifecycle wrapper for the cache
func NewCacheLifecycle(cache *Cache, flushOnShutdown bool) *CacheLifecycle {
	return &CacheLifecycle{
		cache: cache,
		flushOnShutdown: flushOnShutdown,
	}
}

// Name returns the component name
func (cl *CacheLifecycle) Name() string {
	return "TTS Cache"
}

// Shutdown performs graceful cache shutdown
func (cl *CacheLifecycle) Shutdown(ctx context.Context) error {
	if cl.cache == nil {
		return nil
	}
	
	if cl.flushOnShutdown {
		log.Debug("Flushing cache on shutdown")
		// The cache Flush method persists memory cache to disk
		if flusher, ok := interface{}(cl.cache).(interface{ Flush() error }); ok {
			return flusher.Flush()
		}
	}
	
	return nil
}

// ForceStop performs immediate cache termination
func (cl *CacheLifecycle) ForceStop() error {
	// Cache doesn't need force stop
	return nil
}

// SubprocessLifecycle manages external process cleanup
type SubprocessLifecycle struct {
	processes map[string]*os.Process
	mu        sync.RWMutex
}

// NewSubprocessLifecycle creates a lifecycle manager for subprocesses
func NewSubprocessLifecycle() *SubprocessLifecycle {
	return &SubprocessLifecycle{
		processes: make(map[string]*os.Process),
	}
}

// Register adds a process to track
func (sl *SubprocessLifecycle) Register(name string, proc *os.Process) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.processes[name] = proc
	log.Debug("Registered subprocess", "name", name, "pid", proc.Pid)
}

// Unregister removes a process from tracking
func (sl *SubprocessLifecycle) Unregister(name string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	delete(sl.processes, name)
}

// Name returns the component name
func (sl *SubprocessLifecycle) Name() string {
	return "Subprocess Manager"
}

// Shutdown performs graceful subprocess shutdown
func (sl *SubprocessLifecycle) Shutdown(ctx context.Context) error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	for name, proc := range sl.processes {
		log.Debug("Terminating subprocess", "name", name, "pid", proc.Pid)
		
		// Send SIGTERM for graceful shutdown
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			log.Warn("Failed to send SIGTERM", "name", name, "error", err)
		}
	}
	
	// Wait a bit for processes to exit
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

// ForceStop performs immediate subprocess termination
func (sl *SubprocessLifecycle) ForceStop() error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	
	var errors []error
	for name, proc := range sl.processes {
		log.Debug("Force killing subprocess", "name", name, "pid", proc.Pid)
		
		if err := proc.Kill(); err != nil {
			log.Error("Failed to kill process", "name", name, "error", err)
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to kill %d processes", len(errors))
	}
	
	return nil
}

// ResourceMonitor tracks resource usage and detects leaks (debug mode only)
type ResourceMonitor struct {
	enabled      bool
	goroutineCount int
	memoryUsage  uint64
	ticker       *time.Ticker
	done         chan struct{}
}

// NewResourceMonitor creates a resource monitor
func NewResourceMonitor(enabled bool) *ResourceMonitor {
	return &ResourceMonitor{
		enabled: enabled,
		done:    make(chan struct{}),
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() {
	if !rm.enabled {
		return
	}
	
	rm.ticker = time.NewTicker(10 * time.Second)
	go rm.monitor()
}

// monitor tracks resource usage
func (rm *ResourceMonitor) monitor() {
	for {
		select {
		case <-rm.ticker.C:
			rm.checkResources()
		case <-rm.done:
			return
		}
	}
}

// checkResources logs current resource usage
func (rm *ResourceMonitor) checkResources() {
	// This would use runtime package to get actual metrics
	// For now, just a placeholder
	log.Debug("Resource check", 
		"goroutines", "N/A",
		"memory", "N/A")
}

// Name returns the component name
func (rm *ResourceMonitor) Name() string {
	return "Resource Monitor"
}

// Shutdown performs graceful monitor shutdown
func (rm *ResourceMonitor) Shutdown(ctx context.Context) error {
	if rm.ticker != nil {
		rm.ticker.Stop()
	}
	close(rm.done)
	return nil
}

// ForceStop performs immediate monitor termination
func (rm *ResourceMonitor) ForceStop() error {
	return rm.Shutdown(context.Background())
}

// Global lifecycle manager instance
var globalLifecycle *LifecycleManager
var lifecycleOnce sync.Once

// GetLifecycleManager returns the global lifecycle manager
func GetLifecycleManager() *LifecycleManager {
	lifecycleOnce.Do(func() {
		globalLifecycle = NewLifecycleManager()
		globalLifecycle.Start()
	})
	return globalLifecycle
}

// RegisterForCleanup registers a component for cleanup on shutdown
func RegisterForCleanup(component LifecycleComponent) {
	GetLifecycleManager().Register(component)
}

// InitiateShutdown starts the shutdown process
func InitiateShutdown() error {
	return GetLifecycleManager().Shutdown()
}

// WaitForShutdown blocks until shutdown is complete
func WaitForShutdown() {
	GetLifecycleManager().Wait()
}