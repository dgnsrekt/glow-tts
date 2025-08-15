package tts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SubprocessManager handles safe subprocess execution for TTS engines.
// It prevents stdin race conditions by setting up stdin before process start.
type SubprocessManager struct {
	// mutex protects concurrent subprocess execution
	mu sync.Mutex

	// defaultTimeout for subprocess operations
	defaultTimeout time.Duration
}

// NewSubprocessManager creates a new subprocess manager.
func NewSubprocessManager(timeout time.Duration) *SubprocessManager {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &SubprocessManager{
		defaultTimeout: timeout,
	}
}

// ExecuteWithStdin executes a command with stdin input, preventing race conditions.
// This method sets up stdin before starting the process to avoid races.
func (sm *SubprocessManager) ExecuteWithStdin(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create context with timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sm.defaultTimeout)
		defer cancel()
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, name, args...)

	// CRITICAL: Set up stdin BEFORE starting the process to prevent race conditions
	// This is the key pattern that prevents stdin race issues
	cmd.Stdin = strings.NewReader(input)

	// Set up stdout and stderr buffers
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start and wait for the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Wait for process completion
	err := cmd.Wait()

	// Check for context cancellation
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("subprocess timed out after %v", sm.defaultTimeout)
		}
		return nil, fmt.Errorf("subprocess cancelled: %w", ctx.Err())
	}

	// Check for process errors
	if err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("subprocess failed: %w\nstderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("subprocess failed: %w", err)
	}

	return stdout.Bytes(), nil
}

// Execute runs a command without stdin input.
func (sm *SubprocessManager) Execute(ctx context.Context, name string, args ...string) ([]byte, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create context with timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sm.defaultTimeout)
		defer cancel()
	}

	// Create and run command
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()

	// Check for context cancellation
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("subprocess timed out after %v", sm.defaultTimeout)
		}
		return nil, fmt.Errorf("subprocess cancelled: %w", ctx.Err())
	}

	if err != nil {
		return nil, fmt.Errorf("subprocess failed: %w\noutput: %s", err, string(output))
	}

	return output, nil
}

// StreamProcess handles processes that produce streaming output.
// This is useful for TTS engines that output audio data progressively.
func (sm *SubprocessManager) StreamProcess(ctx context.Context, input string, name string, args ...string) (io.ReadCloser, error) {
	var cancel context.CancelFunc
	// Create context with timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		ctx, cancel = context.WithTimeout(ctx, sm.defaultTimeout)
		// Don't defer cancel here - it will be called by processReader.Close()
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, name, args...)

	// Set up stdin before starting (race prevention)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Return a wrapper that handles cleanup
	return &processReader{
		reader: stdout,
		cmd:    cmd,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// processReader wraps a process stdout with cleanup handling.
type processReader struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
	ctx    context.Context
	cancel context.CancelFunc // May be nil if context already had deadline
	once   sync.Once
}

// Read implements io.Reader.
func (pr *processReader) Read(p []byte) (n int, err error) {
	// Check context before reading
	if pr.ctx.Err() != nil {
		return 0, pr.ctx.Err()
	}
	return pr.reader.Read(p)
}

// Close implements io.Closer.
func (pr *processReader) Close() error {
	var err error
	pr.once.Do(func() {
		// Close the reader first
		if closeErr := pr.reader.Close(); closeErr != nil {
			err = closeErr
		}

		// Don't cancel context immediately - let process finish naturally if possible
		// Wait for process to finish or timeout
		done := make(chan error, 1)
		go func() {
			done <- pr.cmd.Wait()
		}()

		// Give the process reasonable time to complete
		waitTimeout := 5 * time.Second
		select {
		case waitErr := <-done:
			// Process finished naturally
			// Ignore "signal: killed" errors as they're expected when context is cancelled
			if err == nil && waitErr != nil {
				// Check if it's a killed signal error
				if !strings.Contains(waitErr.Error(), "signal: killed") &&
					!strings.Contains(waitErr.Error(), "context canceled") {
					err = waitErr
				}
			}
		case <-time.After(waitTimeout):
			// Timeout - cancel context and force kill
			if pr.cancel != nil {
				pr.cancel()
			}
			
			// Give it a moment to respond to context cancellation
			select {
			case waitErr := <-done:
				if err == nil && waitErr != nil {
					if !strings.Contains(waitErr.Error(), "signal: killed") &&
						!strings.Contains(waitErr.Error(), "context canceled") {
						err = waitErr
					}
				}
			case <-time.After(100 * time.Millisecond):
				// Force kill if still not dead
				if pr.cmd.Process != nil {
					if killErr := pr.cmd.Process.Kill(); killErr != nil && err == nil {
						err = killErr
					}
					// Wait for the process to actually exit after kill
					<-done
				}
			}
		}
		
		// Finally cancel the context if we haven't already
		if pr.cancel != nil {
			pr.cancel()
		}
	})
	return err
}

// CheckBinary checks if a binary exists in the system PATH.
func CheckBinary(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("binary '%s' not found in PATH: %w", name, err)
	}
	return nil
}

// SafeProcessOptions provides configuration for safe subprocess execution.
type SafeProcessOptions struct {
	// Input is the stdin data
	Input string

	// Command is the command to execute
	Command string

	// Args are the command arguments
	Args []string

	// Timeout overrides the default timeout
	Timeout time.Duration

	// Environment variables
	Env []string
}

// ExecuteSafe provides a high-level safe subprocess execution with all protections.
func (sm *SubprocessManager) ExecuteSafe(ctx context.Context, opts SafeProcessOptions) ([]byte, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = sm.defaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check binary exists first
	if err := CheckBinary(opts.Command); err != nil {
		return nil, err
	}

	// Execute with or without stdin
	if opts.Input != "" {
		return sm.ExecuteWithStdin(ctx, opts.Input, opts.Command, opts.Args...)
	}
	return sm.Execute(ctx, opts.Command, opts.Args...)
}