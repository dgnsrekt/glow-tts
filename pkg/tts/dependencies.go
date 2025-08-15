package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Name         string
	Required     bool
	Installed    bool
	Version      string
	Path         string
	Error        error
	Instructions string
}

// DependencyChecker interface for checking dependencies
type DependencyChecker interface {
	Check() DependencyStatus
	GetInstructions() string
}

// SystemDependencies holds all dependency checkers
type SystemDependencies struct {
	Checkers map[string]DependencyChecker
	Results  map[string]DependencyStatus
}

// NewSystemDependencies creates a new dependency checker system
func NewSystemDependencies() *SystemDependencies {
	return &SystemDependencies{
		Checkers: make(map[string]DependencyChecker),
		Results:  make(map[string]DependencyStatus),
	}
}

// AddChecker adds a dependency checker
func (sd *SystemDependencies) AddChecker(name string, checker DependencyChecker) {
	sd.Checkers[name] = checker
}

// CheckAll checks all registered dependencies
func (sd *SystemDependencies) CheckAll() error {
	var hasErrors bool
	
	for name, checker := range sd.Checkers {
		status := checker.Check()
		sd.Results[name] = status
		
		if status.Required && !status.Installed {
			hasErrors = true
			log.Error("Missing required dependency", 
				"name", status.Name,
				"instructions", status.Instructions)
		} else if status.Installed {
			log.Debug("Dependency found", 
				"name", status.Name,
				"version", status.Version,
				"path", status.Path)
		}
	}
	
	if hasErrors {
		return fmt.Errorf("missing required dependencies")
	}
	
	return nil
}

// PrintReport prints a formatted dependency report
func (sd *SystemDependencies) PrintReport() string {
	var report strings.Builder
	
	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)
	
	// Status styles
	installedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))
	
	missingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
	
	optionalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))
	
	report.WriteString(titleStyle.Render("TTS Dependency Check Report"))
	report.WriteString("\n\n")
	
	// Group by engine
	report.WriteString("Piper TTS Engine:\n")
	if status, ok := sd.Results["piper"]; ok {
		if status.Installed {
			report.WriteString(installedStyle.Render("  ✓ piper: "))
			report.WriteString(fmt.Sprintf("%s\n", status.Path))
		} else {
			report.WriteString(missingStyle.Render("  ✗ piper: "))
			report.WriteString("Not installed\n")
			report.WriteString(fmt.Sprintf("    %s\n", status.Instructions))
		}
	}
	
	if status, ok := sd.Results["piper_models"]; ok {
		if status.Installed {
			report.WriteString(installedStyle.Render("  ✓ ONNX models: "))
			report.WriteString(fmt.Sprintf("%s\n", status.Path))
		} else {
			report.WriteString(missingStyle.Render("  ✗ ONNX models: "))
			report.WriteString("Not found\n")
			report.WriteString(fmt.Sprintf("    %s\n", status.Instructions))
		}
	}
	
	report.WriteString("\nGoogle TTS Engine:\n")
	if status, ok := sd.Results["gtts-cli"]; ok {
		if status.Installed {
			report.WriteString(installedStyle.Render("  ✓ gtts-cli: "))
			report.WriteString(fmt.Sprintf("%s %s\n", status.Path, status.Version))
		} else {
			report.WriteString(optionalStyle.Render("  ○ gtts-cli: "))
			report.WriteString("Not installed (optional)\n")
			report.WriteString(fmt.Sprintf("    %s\n", status.Instructions))
		}
	}
	
	if status, ok := sd.Results["ffmpeg"]; ok {
		if status.Installed {
			report.WriteString(installedStyle.Render("  ✓ ffmpeg: "))
			report.WriteString(fmt.Sprintf("%s %s\n", status.Path, status.Version))
		} else {
			report.WriteString(optionalStyle.Render("  ○ ffmpeg: "))
			report.WriteString("Not installed (optional)\n")
			report.WriteString(fmt.Sprintf("    %s\n", status.Instructions))
		}
	}
	
	return report.String()
}

// PiperChecker checks for piper binary
type PiperChecker struct{}

func (pc *PiperChecker) Check() DependencyStatus {
	status := DependencyStatus{
		Name:     "piper",
		Required: false, // Optional since we have GTTS
	}
	
	// Check common locations
	paths := []string{
		"piper",
		"/usr/local/bin/piper",
		"/usr/bin/piper",
		"/opt/piper/piper",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "piper"),
	}
	
	for _, path := range paths {
		if fullPath, err := exec.LookPath(path); err == nil {
			// Found piper - it doesn't have a --version flag, so we'll test with --help
			// If --help works, we know piper is properly installed
			cmd := exec.Command(fullPath, "--help")
			if output, err := cmd.CombinedOutput(); err == nil || strings.Contains(string(output), "piper") {
				status.Installed = true
				status.Path = fullPath
				// Since piper doesn't have a version flag, mark as installed without version
				status.Version = "installed"
				return status
			}
		}
	}
	
	status.Instructions = pc.GetInstructions()
	return status
}

func (pc *PiperChecker) GetInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install with: brew install piper-tts\n    Or download from: https://github.com/rhasspy/piper/releases"
	case "linux":
		return "Download from: https://github.com/rhasspy/piper/releases\n    Extract and add to PATH"
	case "windows":
		return "Download from: https://github.com/rhasspy/piper/releases\n    Extract and add to PATH"
	default:
		return "Download from: https://github.com/rhasspy/piper/releases"
	}
}

// PiperModelsChecker checks for ONNX models
type PiperModelsChecker struct{}

func (pmc *PiperModelsChecker) Check() DependencyStatus {
	status := DependencyStatus{
		Name:     "piper_models",
		Required: false, // Optional since we have GTTS
	}
	
	// Check common model locations
	modelPaths := []string{
		"/usr/share/piper-voices",
		"/usr/local/share/piper-voices",
		filepath.Join(os.Getenv("HOME"), ".local", "share", "piper-voices"),
		filepath.Join(os.Getenv("HOME"), ".config", "piper", "voices"),
		filepath.Join(os.Getenv("HOME"), "piper-voices"),
	}
	
	for _, basePath := range modelPaths {
		// Look for any .onnx files
		pattern := filepath.Join(basePath, "**", "*.onnx")
		if matches, err := filepath.Glob(pattern); err == nil && len(matches) > 0 {
			status.Installed = true
			status.Path = basePath
			status.Version = fmt.Sprintf("%d models found", len(matches))
			return status
		}
		
		// Also check direct subdirectories
		if entries, err := os.ReadDir(basePath); err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".onnx") {
					status.Installed = true
					status.Path = basePath
					return status
				}
			}
		}
	}
	
	status.Instructions = pmc.GetInstructions()
	return status
}

func (pmc *PiperModelsChecker) GetInstructions() string {
	return "Download models from: https://github.com/rhasspy/piper/blob/master/VOICES.md\n" +
		"    Example: wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/amy/medium/en_US-amy-medium.onnx\n" +
		"    Place in: ~/.local/share/piper-voices/"
}

// GTTSChecker checks for gtts-cli
type GTTSChecker struct{}

func (gc *GTTSChecker) Check() DependencyStatus {
	status := DependencyStatus{
		Name:     "gtts-cli",
		Required: false, // Optional since we have Piper
	}
	
	// Check for gtts-cli
	paths := []string{
		"gtts-cli",
		"/usr/local/bin/gtts-cli",
		"/usr/bin/gtts-cli",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "gtts-cli"),
	}
	
	for _, path := range paths {
		// Check if file exists
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// Verify it's gtts-cli
			cmd := exec.Command(path, "--version")
			if output, err := cmd.CombinedOutput(); err == nil {
				status.Installed = true
				status.Path = path
				status.Version = strings.TrimSpace(string(output))
				return status
			}
		} else if _, err := exec.LookPath(path); err == nil {
			// Found in PATH
			cmd := exec.Command(path, "--version")
			if output, err := cmd.CombinedOutput(); err == nil {
				status.Installed = true
				status.Path = path
				status.Version = strings.TrimSpace(string(output))
				return status
			}
		}
	}
	
	status.Instructions = gc.GetInstructions()
	return status
}

func (gc *GTTSChecker) GetInstructions() string {
	return "Install with pip:\n" +
		"    pip install gtts\n" +
		"    Or: pipx install gtts"
}

// FFmpegChecker checks for ffmpeg
type FFmpegChecker struct{}

func (fc *FFmpegChecker) Check() DependencyStatus {
	status := DependencyStatus{
		Name:     "ffmpeg",
		Required: false, // Optional, only needed for GTTS
	}
	
	// Check for ffmpeg
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		// Get version
		cmd := exec.Command(path, "-version")
		if output, err := cmd.CombinedOutput(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				// Extract version from first line
				parts := strings.Fields(lines[0])
				if len(parts) >= 3 {
					status.Version = parts[2]
				}
			}
			status.Installed = true
			status.Path = path
			return status
		}
	}
	
	status.Instructions = fc.GetInstructions()
	return status
}

func (fc *FFmpegChecker) GetInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install with: brew install ffmpeg"
	case "linux":
		distro := detectLinuxDistro()
		if strings.Contains(distro, "debian") || strings.Contains(distro, "ubuntu") {
			return "Install with: sudo apt-get install ffmpeg"
		} else if strings.Contains(distro, "fedora") || strings.Contains(distro, "rhel") {
			return "Install with: sudo dnf install ffmpeg"
		} else if strings.Contains(distro, "arch") {
			return "Install with: sudo pacman -S ffmpeg"
		}
		return "Install with your package manager: ffmpeg"
	case "windows":
		return "Download from: https://ffmpeg.org/download.html\n    Extract and add to PATH"
	default:
		return "Install ffmpeg from: https://ffmpeg.org/download.html"
	}
}

// detectLinuxDistro attempts to detect the Linux distribution
func detectLinuxDistro() string {
	// Try to read /etc/os-release
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "ubuntu") {
			return "ubuntu"
		} else if strings.Contains(content, "debian") {
			return "debian"
		} else if strings.Contains(content, "fedora") {
			return "fedora"
		} else if strings.Contains(content, "arch") {
			return "arch"
		} else if strings.Contains(content, "rhel") || strings.Contains(content, "centos") {
			return "rhel"
		}
	}
	return "unknown"
}

// CheckSystemDependencies performs a full dependency check
func CheckSystemDependencies(engine string) (*SystemDependencies, error) {
	deps := NewSystemDependencies()
	
	switch engine {
	case "piper":
		// Piper engine dependencies
		deps.AddChecker("piper", &PiperChecker{})
		deps.AddChecker("piper_models", &PiperModelsChecker{})
		
	case "gtts":
		// Google TTS dependencies
		deps.AddChecker("gtts-cli", &GTTSChecker{})
		deps.AddChecker("ffmpeg", &FFmpegChecker{})
		
	case "":
		// Check all dependencies
		deps.AddChecker("piper", &PiperChecker{})
		deps.AddChecker("piper_models", &PiperModelsChecker{})
		deps.AddChecker("gtts-cli", &GTTSChecker{})
		deps.AddChecker("ffmpeg", &FFmpegChecker{})
		
	default:
		return nil, fmt.Errorf("unknown engine: %s", engine)
	}
	
	// Check all dependencies
	err := deps.CheckAll()
	
	return deps, err
}

// ValidateEngineAvailability checks if a specific engine can be used
func ValidateEngineAvailability(engine string) error {
	deps, err := CheckSystemDependencies(engine)
	if err != nil {
		// Check if we have the minimum requirements
		switch engine {
		case "piper":
			if !deps.Results["piper"].Installed {
				return fmt.Errorf("piper not installed: %s", deps.Results["piper"].Instructions)
			}
			if !deps.Results["piper_models"].Installed {
				return fmt.Errorf("no piper models found: %s", deps.Results["piper_models"].Instructions)
			}
		case "gtts":
			if !deps.Results["gtts-cli"].Installed {
				return fmt.Errorf("gtts-cli not installed: %s", deps.Results["gtts-cli"].Instructions)
			}
			if !deps.Results["ffmpeg"].Installed {
				return fmt.Errorf("ffmpeg not installed: %s", deps.Results["ffmpeg"].Instructions)
			}
		}
	}
	return nil
}