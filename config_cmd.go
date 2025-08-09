package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/charmbracelet/x/editor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultConfig = `# style name or JSON path (default "auto")
style: "auto"
# mouse support (TUI-mode only)
mouse: false
# use pager to display markdown
pager: false
# word-wrap at width
width: 80
# show all files, including hidden and ignored.
all: false

# TTS (Text-to-Speech) configuration
tts:
  # Enable TTS functionality
  enabled: false
  # TTS engine: mock, piper, or google
  engine: "mock"
  # Sample rate for audio output
  sample_rate: 22050
  # Volume level (0.0 to 2.0)
  volume: 1.0
  
  # Playback settings
  auto_play: false
  pause_on_focus_loss: true
  buffer_size: 3
  buffer_ahead: true
  
  # Navigation settings
  wrap_navigation: true
  skip_code_blocks: false
  skip_urls: false
  
  # Visual settings
  highlight_enabled: true
  highlight_color: "yellow"
  show_progress: true
  
  # Piper TTS engine configuration
  piper:
    binary: "piper"
    model: "en_US-lessac-medium"
    # model_path: "/path/to/model.onnx"
    # config_path: "/path/to/model.onnx.json"
    # data_dir: "/usr/share/piper"
    output_raw: true
    speaker_id: 0
    length_scale: 1.0
    noise_scale: 0.667
    noise_w: 0.8
    sentence_silence: "200ms"
    phoneme_gap: "0ms"
    timeout: "30s"
  
  # Google TTS engine configuration
  google:
    # api_key: "your-api-key-here"
    language_code: "en-US"
    voice_name: "en-US-Standard-A"
    speaking_rate: 1.0
    pitch: 0.0
    volume_gain: 0.0
    timeout: "10s"
  
  # Mock TTS engine configuration (for testing)
  mock:
    generation_delay: "100ms"
    words_per_minute: 150
    failure_rate: 0.0
    simulate_latency: true
`

var configCmd = &cobra.Command{
	Use:     "config",
	Hidden:  false,
	Short:   "Edit the glow config file",
	Long:    paragraph(fmt.Sprintf("\n%s the glow config file. We’ll use EDITOR to determine which editor to use. If the config file doesn't exist, it will be created.", keyword("Edit"))),
	Example: paragraph("glow config\nglow config --config path/to/config.yml"),
	Args:    cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		if err := ensureConfigFile(); err != nil {
			return err
		}

		c, err := editor.Cmd("Glow", configFile)
		if err != nil {
			return fmt.Errorf("unable to set config file: %w", err)
		}
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}

		fmt.Println("Wrote config file to:", configFile)
		return nil
	},
}

func ensureConfigFile() error {
	if configFile == "" {
		configFile = viper.GetViper().ConfigFileUsed()
		if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil { //nolint:gosec
			return fmt.Errorf("could not write configuration file: %w", err)
		}
	}

	if ext := path.Ext(configFile); ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("'%s' is not a supported configuration type: use '%s' or '%s'", ext, ".yaml", ".yml")
	}

	if _, err := os.Stat(configFile); errors.Is(err, fs.ErrNotExist) {
		// File doesn't exist yet, create all necessary directories and
		// write the default config file
		if err := os.MkdirAll(filepath.Dir(configFile), 0o700); err != nil {
			return fmt.Errorf("unable create directory: %w", err)
		}

		f, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("unable to create config file: %w", err)
		}
		defer func() { _ = f.Close() }()

		if _, err := f.WriteString(defaultConfig); err != nil {
			return fmt.Errorf("unable to write config file: %w", err)
		}
	} else if err != nil { // some other error occurred
		return fmt.Errorf("unable to stat config file: %w", err)
	}
	return nil
}
