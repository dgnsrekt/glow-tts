package tts

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewSubprocessManager(t *testing.T) {
	// Test with default timeout
	sm := NewSubprocessManager(0)
	if sm.defaultTimeout != 5*time.Second {
		t.Errorf("Expected default timeout 5s, got %v", sm.defaultTimeout)
	}

	// Test with custom timeout
	sm = NewSubprocessManager(10 * time.Second)
	if sm.defaultTimeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", sm.defaultTimeout)
	}
}

func TestExecuteWithStdin(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	tests := []struct {
		name        string
		input       string
		command     string
		args        []string
		expectError bool
		checkOutput func([]byte) bool
	}{
		{
			name:        "echo with stdin",
			input:       "hello world",
			command:     "cat",
			args:        []string{},
			expectError: false,
			checkOutput: func(output []byte) bool {
				return string(output) == "hello world"
			},
		},
		{
			name:        "word count",
			input:       "one two three four five",
			command:     "wc",
			args:        []string{"-w"},
			expectError: false,
			checkOutput: func(output []byte) bool {
				result := strings.TrimSpace(string(output))
				return result == "5"
			},
		},
		{
			name:        "nonexistent command",
			input:       "test",
			command:     "nonexistent_command_xyz",
			args:        []string{},
			expectError: true,
			checkOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests requiring Unix commands on Windows
			if runtime.GOOS == "windows" && (tt.command == "cat" || tt.command == "wc") {
				t.Skip("Skipping Unix command test on Windows")
			}

			ctx := context.Background()
			output, err := sm.ExecuteWithStdin(ctx, tt.input, tt.command, tt.args...)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got: %s", string(output))
				}
			}
		})
	}
}

func TestExecute(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		checkOutput func([]byte) bool
	}{
		{
			name:        "echo command",
			command:     "echo",
			args:        []string{"hello"},
			expectError: false,
			checkOutput: func(output []byte) bool {
				return strings.TrimSpace(string(output)) == "hello"
			},
		},
		{
			name:        "nonexistent command",
			command:     "nonexistent_command_xyz",
			args:        []string{},
			expectError: true,
			checkOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, err := sm.Execute(ctx, tt.command, tt.args...)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got: %s", string(output))
				}
			}
		})
	}
}

func TestTimeoutHandling(t *testing.T) {
	sm := NewSubprocessManager(100 * time.Millisecond)

	// Test command that exceeds timeout
	ctx := context.Background()
	_, err := sm.Execute(ctx, "sleep", "1")

	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running command in background
	done := make(chan error)
	go func() {
		_, err := sm.Execute(ctx, "sleep", "5")
		done <- err
	}()

	// Cancel after a short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for result
	err := <-done
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if !strings.Contains(err.Error(), "cancel") {
		t.Errorf("Expected cancellation error, got: %v", err)
	}
}

func TestCheckBinary(t *testing.T) {
	tests := []struct {
		name        string
		binary      string
		expectError bool
	}{
		{
			name:        "existing binary",
			binary:      "echo",
			expectError: false,
		},
		{
			name:        "nonexistent binary",
			binary:      "nonexistent_binary_xyz",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckBinary(tt.binary)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExecuteSafe(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	tests := []struct {
		name        string
		opts        SafeProcessOptions
		expectError bool
		checkOutput func([]byte) bool
	}{
		{
			name: "safe execution with input",
			opts: SafeProcessOptions{
				Input:   "test input",
				Command: "cat",
				Args:    []string{},
				Timeout: 1 * time.Second,
			},
			expectError: false,
			checkOutput: func(output []byte) bool {
				return string(output) == "test input"
			},
		},
		{
			name: "safe execution without input",
			opts: SafeProcessOptions{
				Command: "echo",
				Args:    []string{"hello"},
				Timeout: 1 * time.Second,
			},
			expectError: false,
			checkOutput: func(output []byte) bool {
				return strings.TrimSpace(string(output)) == "hello"
			},
		},
		{
			name: "nonexistent binary",
			opts: SafeProcessOptions{
				Command: "nonexistent_binary_xyz",
				Args:    []string{},
			},
			expectError: true,
			checkOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip cat test on Windows
			if runtime.GOOS == "windows" && tt.opts.Command == "cat" {
				t.Skip("Skipping cat test on Windows")
			}

			ctx := context.Background()
			output, err := sm.ExecuteSafe(ctx, tt.opts)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tt.checkOutput != nil && !tt.checkOutput(output) {
					t.Errorf("Output check failed. Got: %s", string(output))
				}
			}
		})
	}
}

func TestStreamProcess(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	// Test streaming output
	ctx := context.Background()
	reader, err := sm.StreamProcess(ctx, "test input", "cat")

	// Skip on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping cat test on Windows")
	}

	if err != nil {
		t.Fatalf("StreamProcess failed: %v", err)
	}

	// Read output
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		t.Errorf("Read failed: %v", err)
	}

	output := string(buf[:n])
	if output != "test input" {
		t.Errorf("Expected 'test input', got '%s'", output)
	}

	// Close reader
	if err := reader.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRacePrevention(t *testing.T) {
	// This test verifies that stdin is properly set before process starts
	// by running multiple concurrent subprocess calls
	sm := NewSubprocessManager(5 * time.Second)

	// Skip on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping cat test on Windows")
	}

	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			input := strings.Repeat("A", id+1)
			ctx := context.Background()
			output, err := sm.ExecuteWithStdin(ctx, input, "cat")
			if err != nil {
				errors <- err
				return
			}
			results <- string(output)
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("Goroutine failed: %v", err)
		case result := <-results:
			// Verify that each result has the expected length
			expectedLen := len(result)
			if expectedLen < 1 || expectedLen > numGoroutines {
				t.Errorf("Unexpected result length: %d", expectedLen)
			}
			// Verify all characters are 'A'
			for _, c := range result {
				if c != 'A' {
					t.Errorf("Unexpected character in result: %c", c)
					break
				}
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Test timeout")
		}
	}
}

func TestProcessReaderCleanup(t *testing.T) {
	sm := NewSubprocessManager(5 * time.Second)

	// Skip on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix command test on Windows")
	}

	ctx := context.Background()
	reader, err := sm.StreamProcess(ctx, "", "sleep", "0.1")
	if err != nil {
		t.Fatalf("StreamProcess failed: %v", err)
	}

	// Close should wait for process to finish
	start := time.Now()
	err = reader.Close()
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Should have waited approximately 0.1 seconds
	if elapsed < 50*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("Close didn't wait properly for process. Elapsed: %v", elapsed)
	}
}