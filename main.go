// Package main provides the entry point for the Glow CLI application.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/internal/tts"
	"github.com/charmbracelet/glow/v2/ui"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	gap "github.com/muesli/go-app-paths"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	// Version as provided by goreleaser.
	Version = ""
	// CommitSHA as provided by goreleaser.
	CommitSHA = ""

	readmeNames      = []string{"README.md", "README", "Readme.md", "Readme", "readme.md", "readme"}
	configFile       string
	pager            bool
	tui              bool
	style            string
	width            uint
	showAllFiles     bool
	showLineNumbers  bool
	preserveNewLines bool
	mouse            bool
	ttsEngine        string

	rootCmd = &cobra.Command{
		Use:   "glow [SOURCE|DIR]",
		Short: "Render markdown on the CLI, with pizzazz!",
		Long: paragraph(
			fmt.Sprintf("\nRender markdown on the CLI, %s!", keyword("with pizzazz")),
		),
		SilenceErrors:    false,
		SilenceUsage:     true,
		TraverseChildren: true,
		Args:             cobra.MaximumNArgs(1),
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return validateOptions(cmd)
		},
		RunE: execute,
	}
)

// source provides a readable markdown source.
type source struct {
	reader io.ReadCloser
	URL    string
}

// sourceFromArg parses an argument and creates a readable source for it.
func sourceFromArg(arg string) (*source, error) {
	// from stdin
	if arg == "-" {
		return &source{reader: os.Stdin}, nil
	}

	// a GitHub or GitLab URL (even without the protocol):
	src, err := readmeURL(arg)
	if src != nil && err == nil {
		// if there's an error, try next methods...
		return src, nil
	}

	// HTTP(S) URLs:
	if u, err := url.ParseRequestURI(arg); err == nil && strings.Contains(arg, "://") { //nolint:nestif
		if u.Scheme != "" {
			if u.Scheme != "http" && u.Scheme != "https" {
				return nil, fmt.Errorf("%s is not a supported protocol", u.Scheme)
			}
			// consumer of the source is responsible for closing the ReadCloser.
			resp, err := http.Get(u.String()) //nolint: noctx,bodyclose
			if err != nil {
				return nil, fmt.Errorf("unable to get url: %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
			return &source{resp.Body, u.String()}, nil
		}
	}

	// a directory:
	if len(arg) == 0 {
		// use the current working dir if no argument was supplied
		arg = "."
	}
	st, err := os.Stat(arg)
	if err == nil && st.IsDir() { //nolint:nestif
		var src *source
		_ = filepath.Walk(arg, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			for _, v := range readmeNames {
				if strings.EqualFold(filepath.Base(path), v) {
					r, err := os.Open(path)
					if err != nil {
						continue
					}

					u, _ := filepath.Abs(path)
					src = &source{r, u}

					// abort filepath.Walk
					return errors.New("source found")
				}
			}
			return nil
		})

		if src != nil {
			return src, nil
		}

		return nil, errors.New("missing markdown source")
	}

	r, err := os.Open(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to open file: %w", err)
	}
	u, err := filepath.Abs(arg)
	if err != nil {
		return nil, fmt.Errorf("unable to get absolute path: %w", err)
	}
	return &source{r, u}, nil
}

// validateStyle checks if the style is a default style, if not, checks that
// the custom style exists.
func validateStyle(style string) error {
	if style != "auto" && styles.DefaultStyles[style] == nil {
		style = utils.ExpandPath(style)
		if _, err := os.Stat(style); errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("specified style does not exist: %s", style)
		} else if err != nil {
			return fmt.Errorf("unable to stat file: %w", err)
		}
	}
	return nil
}

func validateOptions(cmd *cobra.Command) error {
	// grab config values from Viper
	width = viper.GetUint("width")
	mouse = viper.GetBool("mouse")
	pager = viper.GetBool("pager")
	tui = viper.GetBool("tui")
	showAllFiles = viper.GetBool("all")
	preserveNewLines = viper.GetBool("preserveNewLines")
	showLineNumbers = viper.GetBool("showLineNumbers")
	// TTS engine selection: CLI flag takes precedence over config file
	ttsEngine = viper.GetString("tts")
	if ttsEngine == "" {
		ttsEngine = viper.GetString("tts.engine")
	}

	// TTS validation and mode enforcement
	if ttsEngine != "" {
		// Ensure TTS defaults are applied when using CLI flag
		if viper.GetInt("tts.cache.max_size") == 0 {
			viper.Set("tts.cache.max_size", 100)
		}
		if viper.GetString("tts.cache.dir") == "" {
			// Keep empty to use default in TTS initialization
		}
		if viper.GetFloat64("tts.piper.speed") == 0 {
			viper.Set("tts.piper.speed", 1.0)
		}
		if viper.GetString("tts.gtts.language") == "" {
			viper.Set("tts.gtts.language", "en")
		}
		// TTS requires TUI mode
		if pager {
			return errors.New("cannot use pager mode with TTS (TTS requires TUI mode)")
		}

		// Force TUI mode when TTS is enabled
		tui = true

		// Validate TTS engine selection
		_, err := tts.ValidateEngineSelection(ttsEngine, tts.Config{})
		if err != nil {
			return fmt.Errorf("TTS validation failed: %w", err)
		}

		// Validate TTS configuration values
		if err := validateTTSConfig(); err != nil {
			return fmt.Errorf("TTS config validation failed: %w", err)
		}
	}

	if pager && tui {
		return errors.New("cannot use both pager and tui")
	}

	// validate the glamour style
	style = viper.GetString("style")
	if err := validateStyle(style); err != nil {
		return err
	}

	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	// We want to use a special no-TTY style, when stdout is not a terminal
	// and there was no specific style passed by arg
	if !isTerminal && !cmd.Flags().Changed("style") {
		style = "notty"
	}

	// Detect terminal width
	if !cmd.Flags().Changed("width") { //nolint:nestif
		if isTerminal && width == 0 {
			w, _, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				width = uint(w) //nolint:gosec
			}

			if width > 120 {
				width = 120
			}
		}
		if width == 0 {
			width = 80
		}
	}
	return nil
}

// validateTTSConfig validates TTS configuration values
func validateTTSConfig() error {
	// Validate cache size
	maxCacheSize := viper.GetInt("tts.cache.max_size")
	if maxCacheSize < 1 || maxCacheSize > 10000 {
		return fmt.Errorf("TTS cache max_size must be between 1 and 10000 MB, got %d", maxCacheSize)
	}

	// Validate Piper speed
	piperSpeed := viper.GetFloat64("tts.piper.speed")
	if piperSpeed < 0.1 || piperSpeed > 3.0 {
		return fmt.Errorf("TTS piper speed must be between 0.1 and 3.0, got %.2f", piperSpeed)
	}

	// Validate gTTS language code (basic validation - not exhaustive)
	gttsLang := viper.GetString("tts.gtts.language")
	if len(gttsLang) < 2 || len(gttsLang) > 5 {
		return fmt.Errorf("TTS gtts language code must be 2-5 characters, got %q", gttsLang)
	}

	// Validate cache directory if specified
	cacheDir := viper.GetString("tts.cache.dir")
	if cacheDir != "" {
		// Expand path
		cacheDir = utils.ExpandPath(cacheDir)
		// Check if parent directory exists or can be created
		parentDir := filepath.Dir(cacheDir)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			// Try to create parent directory
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return fmt.Errorf("TTS cache directory parent %q cannot be created: %w", parentDir, err)
			}
		}
	}

	// Validate Piper model path if specified
	piperModel := viper.GetString("tts.piper.model")
	if piperModel != "" {
		piperModel = utils.ExpandPath(piperModel)
		if _, err := os.Stat(piperModel); os.IsNotExist(err) {
			return fmt.Errorf("TTS piper model file does not exist: %s", piperModel)
		}
	}

	return nil
}

func stdinIsPipe() (bool, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false, fmt.Errorf("unable to open file: %w", err)
	}
	if stat.Mode()&os.ModeCharDevice == 0 || stat.Size() > 0 {
		return true, nil
	}
	return false, nil
}

func execute(cmd *cobra.Command, args []string) error {
	// if stdin is a pipe then use stdin for input. note that you can also
	// explicitly use a - to read from stdin.
	if yes, err := stdinIsPipe(); err != nil {
		return err
	} else if yes {
		src := &source{reader: os.Stdin}
		defer src.reader.Close() //nolint:errcheck
		return executeCLI(cmd, src, os.Stdout)
	}

	switch len(args) {
	// TUI running on cwd
	case 0:
		return runTUI("", "")

	// TUI with possible dir argument
	case 1:
		// Validate that the argument is a directory. If it's not treat it as
		// an argument to the non-TUI version of Glow (via fallthrough).
		info, err := os.Stat(args[0])
		if err == nil && info.IsDir() {
			p, err := filepath.Abs(args[0])
			if err == nil {
				return runTUI(p, "")
			}
		}
		fallthrough

	// CLI
	default:
		for _, arg := range args {
			if err := executeArg(cmd, arg, os.Stdout); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeArg(cmd *cobra.Command, arg string, w io.Writer) error {
	// create an io.Reader from the markdown source in cli-args
	src, err := sourceFromArg(arg)
	if err != nil {
		return err
	}
	defer src.reader.Close() //nolint:errcheck
	return executeCLI(cmd, src, w)
}

func executeCLI(cmd *cobra.Command, src *source, w io.Writer) error {
	b, err := io.ReadAll(src.reader)
	if err != nil {
		return fmt.Errorf("unable to read from reader: %w", err)
	}

	b = utils.RemoveFrontmatter(b)

	// render
	var baseURL string
	u, err := url.ParseRequestURI(src.URL)
	if err == nil {
		u.Path = filepath.Dir(u.Path)
		baseURL = u.String() + "/"
	}

	isCode := !utils.IsMarkdownFile(src.URL)

	// initialize glamour
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)), //nolint:gosec
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return fmt.Errorf("unable to create renderer: %w", err)
	}

	content := string(b)
	ext := filepath.Ext(src.URL)
	if isCode {
		content = utils.WrapCodeBlock(string(b), ext)
	}

	out, err := r.Render(content)
	if err != nil {
		return fmt.Errorf("unable to render markdown: %w", err)
	}

	// display
	switch {
	case pager || cmd.Flags().Changed("pager"):
		pagerCmd := os.Getenv("PAGER")
		if pagerCmd == "" {
			pagerCmd = "less -r"
		}

		pa := strings.Split(pagerCmd, " ")
		c := exec.Command(pa[0], pa[1:]...) //nolint:gosec
		c.Stdin = strings.NewReader(out)
		c.Stdout = os.Stdout
		if err := c.Run(); err != nil {
			return fmt.Errorf("unable to run command: %w", err)
		}
		return nil
	case tui || cmd.Flags().Changed("tui"):
		path := ""
		if !isURL(src.URL) {
			path = src.URL
		}
		return runTUI(path, content)
	default:
		if _, err = fmt.Fprint(w, out); err != nil {
			return fmt.Errorf("unable to write to writer: %w", err)
		}
		return nil
	}
}

func runTUI(path string, content string) error {
	// Read environment to get debugging stuff
	cfg, err := env.ParseAs[ui.Config]()
	if err != nil {
		return fmt.Errorf("error parsing config: %v", err)
	}

	// use style set in env, or auto if unset
	if err := validateStyle(cfg.GlamourStyle); err != nil {
		cfg.GlamourStyle = style
	}

	cfg.Path = path
	cfg.ShowAllFiles = showAllFiles
	cfg.ShowLineNumbers = showLineNumbers
	cfg.GlamourMaxWidth = width
	cfg.EnableMouse = mouse
	cfg.PreserveNewLines = preserveNewLines

	// TTS configuration
	cfg.TTSEngine = ttsEngine
	cfg.TTSEnabled = ttsEngine != ""
	cfg.TTSCacheDir = viper.GetString("tts.cache.dir")
	cfg.TTSMaxCacheSize = viper.GetInt("tts.cache.max_size")
	cfg.TTSPiperModel = viper.GetString("tts.piper.model")
	cfg.TTSPiperSpeed = viper.GetFloat64("tts.piper.speed")
	cfg.TTSGTTSLang = viper.GetString("tts.gtts.language")
	cfg.TTSGTTSSlow = viper.GetBool("tts.gtts.slow")

	// Run Bubble Tea program
	if _, err := ui.NewProgram(cfg, content).Run(); err != nil {
		return fmt.Errorf("unable to run tui program: %w", err)
	}

	return nil
}

func main() {
	closer, err := setupLog()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		_ = closer()
		os.Exit(1)
	}
	_ = closer()
}

func init() {
	tryLoadConfigFromDefaultPlaces()
	if len(CommitSHA) >= 7 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		Version = "unknown (built from source)"
	}
	rootCmd.Version = Version
	rootCmd.InitDefaultCompletionCmd()

	// "Glow Classic" cli arguments
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", fmt.Sprintf("config file (default %s)", viper.GetViper().ConfigFileUsed()))
	rootCmd.Flags().BoolVarP(&pager, "pager", "p", false, "display with pager")
	rootCmd.Flags().BoolVarP(&tui, "tui", "t", false, "display with tui")
	rootCmd.Flags().StringVarP(&style, "style", "s", styles.AutoStyle, "style name or JSON path")
	rootCmd.Flags().UintVarP(&width, "width", "w", 0, "word-wrap at width (set to 0 to disable)")
	rootCmd.Flags().BoolVarP(&showAllFiles, "all", "a", false, "show system files and directories (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&showLineNumbers, "line-numbers", "l", false, "show line numbers (TUI-mode only)")
	rootCmd.Flags().BoolVarP(&preserveNewLines, "preserve-new-lines", "n", false, "preserve newlines in the output")
	rootCmd.Flags().BoolVarP(&mouse, "mouse", "m", false, "enable mouse wheel (TUI-mode only)")
	_ = rootCmd.Flags().MarkHidden("mouse")
	rootCmd.Flags().StringVar(&ttsEngine, "tts", "", "enable TTS with specified engine (piper/gtts, forces TUI mode)")

	// Config bindings
	_ = viper.BindPFlag("pager", rootCmd.Flags().Lookup("pager"))
	_ = viper.BindPFlag("tui", rootCmd.Flags().Lookup("tui"))
	_ = viper.BindPFlag("style", rootCmd.Flags().Lookup("style"))
	_ = viper.BindPFlag("width", rootCmd.Flags().Lookup("width"))
	_ = viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))
	_ = viper.BindPFlag("mouse", rootCmd.Flags().Lookup("mouse"))
	_ = viper.BindPFlag("preserveNewLines", rootCmd.Flags().Lookup("preserve-new-lines"))
	_ = viper.BindPFlag("showLineNumbers", rootCmd.Flags().Lookup("line-numbers"))
	_ = viper.BindPFlag("all", rootCmd.Flags().Lookup("all"))
	_ = viper.BindPFlag("tts", rootCmd.Flags().Lookup("tts"))

	viper.SetDefault("style", styles.AutoStyle)
	viper.SetDefault("width", 0)
	viper.SetDefault("all", true)

	// TTS defaults
	viper.SetDefault("tts.engine", "")
	viper.SetDefault("tts.cache.dir", "")
	viper.SetDefault("tts.cache.max_size", 100)
	viper.SetDefault("tts.piper.model", "")
	viper.SetDefault("tts.piper.speed", 1.0)
	viper.SetDefault("tts.gtts.language", "en")
	viper.SetDefault("tts.gtts.slow", false)

	rootCmd.AddCommand(configCmd, manCmd)
}

func tryLoadConfigFromDefaultPlaces() {
	scope := gap.NewScope(gap.User, "glow")
	dirs, err := scope.ConfigDirs()
	if err != nil {
		fmt.Println("Could not load find configuration directory.")
		os.Exit(1)
	}

	if c := os.Getenv("XDG_CONFIG_HOME"); c != "" {
		dirs = append([]string{filepath.Join(c, "glow")}, dirs...)
	}

	if c := os.Getenv("GLOW_CONFIG_HOME"); c != "" {
		dirs = append([]string{c}, dirs...)
	}

	for _, v := range dirs {
		viper.AddConfigPath(v)
	}

	viper.SetConfigName("glow")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("glow")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warn("Could not parse configuration file", "err", err)
		}
	}

	if used := viper.ConfigFileUsed(); used != "" {
		log.Debug("Using configuration file", "path", viper.ConfigFileUsed())
		return
	}

	if viper.ConfigFileUsed() == "" {
		configFile = filepath.Join(dirs[0], "glow.yml")
	}
	if err := ensureConfigFile(); err != nil {
		log.Error("Could not create default configuration", "error", err)
	}
}
