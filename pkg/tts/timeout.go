package tts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
)

// TimeoutConfig holds timeout configuration for subprocess execution
type TimeoutConfig struct {
	// Maximum time to wait for subprocess completion
	Timeout time.Duration
	
	// Time to wait after SIGINT before sending SIGKILL
	GracePeriod time.Duration
	
	// Whether to use context cancellation
	UseContext bool
}

// DefaultTimeoutConfig returns sensible default timeout settings
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:     10 * time.Second,
		GracePeriod: 500 * time.Millisecond,
		UseContext:  true,
	}
}

// TimeoutExecutor manages subprocess execution with timeout protection
type TimeoutExecutor struct {
	config TimeoutConfig
}

// NewTimeoutExecutor creates a new timeout executor with the given config
func NewTimeoutExecutor(config TimeoutConfig) *TimeoutExecutor {
	return &TimeoutExecutor{
		config: config,
	}
}

// RunWithTimeout executes a command with timeout protection
// It implements a graceful shutdown sequence: SIGINT -> wait -> SIGKILL
func (te *TimeoutExecutor) RunWithTimeout(cmd *exec.Cmd) error {
	if te.config.UseContext {
		return te.runWithContext(cmd)
	}
	return te.runWithTimer(cmd)
}

// runWithContext uses context cancellation for timeout management
func (te *TimeoutExecutor) runWithContext(cmd *exec.Cmd) error {
	ctx, cancel := context.WithTimeout(context.Background(), te.config.Timeout)
	defer cancel()
	
	// Track execution time
	startTime := time.Now()
	
	// Create a new command with context
	ctxCmd := exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	ctxCmd.Env = cmd.Env
	ctxCmd.Dir = cmd.Dir
	ctxCmd.Stdin = cmd.Stdin
	ctxCmd.Stdout = cmd.Stdout
	ctxCmd.Stderr = cmd.Stderr
	
	// Start the command
	if err := ctxCmd.Start(); err != nil {
		LogSubprocessExecution(cmd.Path, cmd.Args[1:], time.Since(startTime), err)
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Wait for completion
	err := ctxCmd.Wait()
	duration := time.Since(startTime)
	
	// Log the execution
	LogSubprocessExecution(cmd.Path, cmd.Args[1:], duration, err)
	
	// Check if context was cancelled (timeout)
	if ctx.Err() == context.DeadlineExceeded {
		log.Warn("Command timed out", 
			"command", cmd.Path,
			"timeout", te.config.Timeout)
		
		// Try graceful shutdown first
		if ctxCmd.Process != nil {
			te.gracefulShutdown(ctxCmd.Process)
		}
		
		return fmt.Errorf("command timed out after %v", te.config.Timeout)
	}
	
	return err
}

// runWithTimer uses a timer-based approach for timeout management
func (te *TimeoutExecutor) runWithTimer(cmd *exec.Cmd) error {
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Create channels for completion and timeout
	done := make(chan error, 1)
	timeout := time.NewTimer(te.config.Timeout)
	defer timeout.Stop()
	
	// Wait for command completion in goroutine
	go func() {
		done <- cmd.Wait()
	}()
	
	// Wait for either completion or timeout
	select {
	case err := <-done:
		// Command completed normally
		return err
		
	case <-timeout.C:
		// Timeout occurred
		log.Warn("Command timed out",
			"command", cmd.Path,
			"timeout", te.config.Timeout)
		
		// Try graceful shutdown
		if cmd.Process != nil {
			te.gracefulShutdown(cmd.Process)
		}
		
		// Kill the process if still running
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.Error("Failed to kill process", "error", err)
			}
		}
		
		// Wait for the goroutine to finish
		<-done
		
		return fmt.Errorf("command timed out after %v", te.config.Timeout)
	}
}

// gracefulShutdown attempts to gracefully terminate a process
func (te *TimeoutExecutor) gracefulShutdown(proc *os.Process) {
	// Send interrupt signal first (platform-specific)
	if err := te.sendInterrupt(proc); err != nil {
		log.Debug("Failed to send interrupt signal", "error", err)
		return
	}
	
	// Wait for grace period
	graceDone := make(chan struct{})
	go func() {
		time.Sleep(te.config.GracePeriod)
		close(graceDone)
	}()
	
	// Wait for grace period to expire
	<-graceDone
	// Grace period expired, process may still be running
	log.Debug("Grace period expired, process may need forceful termination")
}

// sendInterrupt sends an interrupt signal to the process (platform-specific)
func (te *TimeoutExecutor) sendInterrupt(proc *os.Process) error {
	if runtime.GOOS == "windows" {
		// Windows doesn't have SIGINT, use Kill directly
		return proc.Kill()
	}
	
	// Unix-like systems: send SIGINT
	return proc.Signal(syscall.SIGINT)
}

// RunCommand is a convenience function that runs a command with default timeout
func RunCommandWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	config := DefaultTimeoutConfig()
	config.Timeout = timeout
	executor := NewTimeoutExecutor(config)
	return executor.RunWithTimeout(cmd)
}

// RunCommandString builds and runs a command from a string with timeout
func RunCommandStringWithTimeout(command string, args []string, timeout time.Duration) ([]byte, error) {
	cmd := exec.Command(command, args...)
	
	// Capture output
	output, err := cmd.Output()
	if err != nil {
		// Try with timeout protection
		config := DefaultTimeoutConfig()
		config.Timeout = timeout
		executor := NewTimeoutExecutor(config)
		
		cmd = exec.Command(command, args...)
		if err := executor.RunWithTimeout(cmd); err != nil {
			return nil, err
		}
	}
	
	return output, nil
}