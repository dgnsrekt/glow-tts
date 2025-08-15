// Package ui provides the main UI for the glow application.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/glow/v2/pkg/tts"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/log"
	"github.com/muesli/gitcha"
	te "github.com/muesli/termenv"
)

const (
	statusMessageTimeout = time.Second * 3 // how long to show status messages like "stashed!"
	ellipsis             = "…"
)

var (
	config Config

	markdownExtensions = []string{
		"*.md", "*.mdown", "*.mkdn", "*.mkd", "*.markdown",
	}
)

// NewProgram returns a new Tea program.
func NewProgram(cfg Config, content string) *tea.Program {
	log.Debug(
		"Starting glow",
		"high_perf_pager",
		cfg.HighPerformancePager,
		"glamour",
		cfg.GlamourEnabled,
	)
	
	// Load TTS config if TTS is enabled
	if cfg.TTSEngine != "" {
		if ttsConfig, err := tts.LoadTTSConfig(); err == nil {
			// Apply config settings
			cfg.TTSEngine = ttsConfig.GetEngineOrDefault(cfg.TTSEngine)
			log.Debug("Loaded TTS config", "engine", cfg.TTSEngine)
		}
	}

	config = cfg
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.EnableMouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	m := newModel(cfg, content)
	return tea.NewProgram(m, opts...)
}

type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

type (
	initLocalFileSearchMsg struct {
		cwd string
		ch  chan gitcha.SearchResult
	}
)

type (
	foundLocalFileMsg       gitcha.SearchResult
	localFileSearchFinished struct{}
	statusMessageTimeoutMsg applicationContext
)

// applicationContext indicates the area of the application something applies
// to. Occasionally used as an argument to commands and messages.
type applicationContext int

const (
	stashContext applicationContext = iota
	pagerContext
)

// state is the top-level application state.
type state int

const (
	stateShowStash state = iota
	stateShowDocument
)

func (s state) String() string {
	return map[state]string{
		stateShowStash:    "showing file listing",
		stateShowDocument: "showing document",
	}[s]
}

// Common stuff we'll need to access in all models.
type commonModel struct {
	cfg    Config
	cwd    string
	width  int
	height int
}

type model struct {
	common   *commonModel
	state    state
	fatalErr error

	// Sub-models
	stash stashModel
	pager pagerModel

	// TTS state (nil if TTS is disabled)
	tts *TTSState

	// Channel that receives paths to local markdown files
	// (via the github.com/muesli/gitcha package)
	localFileFinder chan gitcha.SearchResult
}

// unloadDocument unloads a document from the pager. Note that while this
// method alters the model we also need to send along any commands returned.
func (m *model) unloadDocument() []tea.Cmd {
	m.state = stateShowStash
	m.stash.viewState = stashStateReady
	m.pager.unload()
	m.pager.showHelp = false

	var batch []tea.Cmd
	if m.pager.viewport.HighPerformanceRendering {
		batch = append(batch, tea.ClearScrollArea) //nolint:staticcheck
	}

	if !m.stash.shouldSpin() {
		batch = append(batch, m.stash.spinner.Tick)
	}
	return batch
}

func newModel(cfg Config, content string) tea.Model {
	initSections()

	if cfg.GlamourStyle == styles.AutoStyle {
		if te.HasDarkBackground() {
			cfg.GlamourStyle = styles.DarkStyle
		} else {
			cfg.GlamourStyle = styles.LightStyle
		}
	}

	common := commonModel{
		cfg: cfg,
	}

	m := model{
		common: &common,
		state:  stateShowStash,
		pager:  newPagerModel(&common),
		stash:  newStashModel(&common),
	}

	// Initialize TTS if enabled
	if cfg.TTSEngine != "" {
		log.Debug("creating TTS state", "engine", cfg.TTSEngine)
		m.tts = NewTTSState(cfg.TTSEngine)
		// Pass TTS state to pager for status display
		m.pager.tts = m.tts
		log.Debug("TTS state created", "enabled", m.tts.IsEnabled())
	}

	path := cfg.Path
	if path == "" && content != "" {
		m.state = stateShowDocument
		m.pager.currentDocument = markdown{Body: content}
		return m
	}

	if path == "" {
		path = "."
	}
	info, err := os.Stat(path)
	if err != nil {
		log.Error("unable to stat file", "file", path, "error", err)
		m.fatalErr = err
		return m
	}
	if info.IsDir() {
		m.state = stateShowStash
	} else {
		cwd, _ := os.Getwd()
		m.state = stateShowDocument
		m.pager.currentDocument = markdown{
			localPath: path,
			Note:      stripAbsolutePath(path, cwd),
			Modtime:   info.ModTime(),
		}
	}

	return m
}

func (m model) Init() tea.Cmd {
	log.Debug("Init() called", "state", m.state)
	cmds := []tea.Cmd{m.stash.spinner.Tick}

	// Initialize TTS if enabled
	if m.tts != nil && m.tts.IsEnabled() {
		log.Debug("TTS enabled, queuing initialization", 
			"isInitializing", m.tts.isInitializing, 
			"isInitialized", m.tts.isInitialized)
		cmds = append(cmds, initTTSCmd(m.tts.engine, m.tts))
		// Start ticker for UI updates during initialization
		cmds = append(cmds, ttsTick())
		// Start spinner for loading animation
		cmds = append(cmds, m.tts.loadingSpinner.Tick)
		log.Debug("queued commands", "count", len(cmds))
	}

	switch m.state {
	case stateShowStash:
		cmds = append(cmds, findLocalFiles(*m.common))
	case stateShowDocument:
		content, err := os.ReadFile(m.common.cfg.Path)
		if err != nil {
			log.Error("unable to read file", "file", m.common.cfg.Path, "error", err)
			return func() tea.Msg { return errMsg{err} }
		}
		body := string(utils.RemoveFrontmatter(content))
		log.Debug("initial document load", 
			"path", m.common.cfg.Path,
			"contentLength", len(content),
			"bodyLength", len(body))
		cmds = append(cmds, renderWithGlamour(m.pager, body))
		
		// Parse sentences for TTS if enabled
		if m.tts != nil && m.tts.IsEnabled() {
			cmds = append(cmds, parseSentencesCmd(body))
		}
	}

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If there's been an error, any key exits
	if m.fatalErr != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			return m, tea.Quit
		}
	}

	var cmds []tea.Cmd
	var skipChildUpdate bool  // Flag to skip passing message to children

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.state == stateShowDocument || m.stash.viewState == stashStateLoadingDocument {
				batch := m.unloadDocument()
				return m, tea.Batch(batch...)
			}
		case "r":
			var cmd tea.Cmd
			if m.state == stateShowStash {
				// pass through all keys if we're editing the filter
				if m.stash.filterState == filtering {
					m.stash, cmd = m.stash.update(msg)
					return m, cmd
				}
				m.stash.markdowns = nil
				return m, m.Init()
			}

		case "q":
			var cmd tea.Cmd

			switch m.state { //nolint:exhaustive
			case stateShowStash:
				// pass through all keys if we're editing the filter
				if m.stash.filterState == filtering {
					m.stash, cmd = m.stash.update(msg)
					return m, cmd
				}
			}

			// Perform TTS cleanup if initialized
			if m.tts != nil && m.tts.isInitialized {
				log.Debug("Performing TTS cleanup before quit")
				go func() {
					if err := tts.InitiateShutdown(); err != nil {
						log.Warn("TTS shutdown error", "error", err)
					}
				}()
			}

			return m, tea.Quit

		case "h", "delete":
			if m.state == stateShowDocument {
				cmds = append(cmds, m.unloadDocument()...)
				return m, tea.Batch(cmds...)
			}

		case "ctrl+z":
			return m, tea.Suspend

		// Ctrl+C always quits no matter where in the application you are.
		case "ctrl+c":
			// Perform TTS cleanup if initialized
			if m.tts != nil && m.tts.isInitialized {
				log.Debug("Performing TTS cleanup before quit")
				go func() {
					if err := tts.InitiateShutdown(); err != nil {
						log.Warn("TTS shutdown error", "error", err)
					}
				}()
			}
			return m, tea.Quit

		// TTS keyboard shortcuts
		case " ": // Space for play/pause
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				// Check if TTS is initialized
				if !m.tts.isInitialized {
					// Don't process play command if not initialized
					// The error will be shown in the status bar
					return m, nil
				}
				
				skipChildUpdate = true  // Don't pass space to pager
				
				// Clear any previous errors when starting a new action
				m.tts.lastError = nil
				
				if m.tts.isPlaying {
					return m, pauseTTSCmd(m.tts.controller)
				} else {
					// Get the raw markdown text from the pager (not the glamour-rendered version)
					documentText := m.pager.rawMarkdownText
					if documentText == "" {
						// Fallback to currentDocument.Body if rawMarkdownText not set yet
						documentText = m.pager.currentDocument.Body
						log.Debug("rawMarkdownText empty, using currentDocument.Body", 
							"bodyLength", len(documentText))
					} else {
						log.Debug("using rawMarkdownText", "length", len(documentText))
					}
					log.Debug("TTS play command", "textLength", len(documentText))
					
					// Set synthesizing state before playing
					m.tts.SetLoadingState(true, false, "Synthesizing audio...")
					
					// Start the spinner tick and play command
					return m, tea.Batch(
						m.tts.loadingSpinner.Tick,
						playTTSCmd(m.tts.controller, documentText),
					)
				}
			}

		case "left":
			// TTS: Previous sentence (when in document view with TTS)
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				m.tts.lastError = nil  // Clear previous errors
				return m, prevSentenceCmd(m.tts.controller, m.tts.currentSentenceIndex)
			}
			// Otherwise, handle normally (go back)
			if m.state == stateShowDocument {
				cmds = append(cmds, m.unloadDocument()...)
				return m, tea.Batch(cmds...)
			}

		case "right":
			// TTS: Next sentence (when in document view with TTS)
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				m.tts.lastError = nil  // Clear previous errors
				return m, nextSentenceCmd(m.tts.controller, m.tts.currentSentenceIndex, m.tts.totalSentences)
			}

		case "+", "=":
			// TTS: Increase speed
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				if cmd := m.tts.increaseSpeedCmd(); cmd != nil {
					return m, cmd
				}
			}

		case "-", "_":
			// TTS: Decrease speed
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				if cmd := m.tts.decreaseSpeedCmd(); cmd != nil {
					return m, cmd
				}
			}

		case "s", "S":
			// TTS: Stop
			if m.tts != nil && m.tts.IsEnabled() && m.state == stateShowDocument {
				return m, stopTTSCmd(m.tts.controller)
			}
		}

	// Window size is received when starting up and on every resize
	case tea.WindowSizeMsg:
		m.common.width = msg.Width
		m.common.height = msg.Height
		m.stash.setSize(msg.Width, msg.Height)
		m.pager.setSize(msg.Width, msg.Height)

	case initLocalFileSearchMsg:
		m.localFileFinder = msg.ch
		m.common.cwd = msg.cwd
		cmds = append(cmds, findNextLocalFile(m))

	case fetchedMarkdownMsg:
		// We've loaded a markdown file's contents for rendering
		log.Debug("fetchedMarkdownMsg received", 
			"bodyLength", len(msg.Body),
			"Note", msg.Note)
		m.pager.currentDocument = *msg
		body := string(utils.RemoveFrontmatter([]byte(msg.Body)))
		cmds = append(cmds, renderWithGlamour(m.pager, body))

	case contentRenderedMsg:
		m.state = stateShowDocument
		log.Debug("contentRenderedMsg received in main UI", 
			"rawMarkdownLength", len(msg.rawMarkdown))

	case localFileSearchFinished:
		// Always pass these messages to the stash so we can keep it updated
		// about network activity, even if the user isn't currently viewing
		// the stash.
		stashModel, cmd := m.stash.update(msg)
		m.stash = stashModel
		return m, cmd

	case foundLocalFileMsg:
		newMd := localFileToMarkdown(m.common.cwd, gitcha.SearchResult(msg))
		m.stash.addMarkdowns(newMd)
		if m.stash.filterApplied() {
			newMd.buildFilterValue()
		}
		if m.stash.shouldUpdateFilter() {
			cmds = append(cmds, filterMarkdowns(m.stash))
		}
		cmds = append(cmds, findNextLocalFile(m))

	case filteredMarkdownMsg:
		if m.state == stateShowDocument {
			newStashModel, cmd := m.stash.update(msg)
			m.stash = newStashModel
			cmds = append(cmds, cmd)
		}

	// TTS message handling
	case ttsInitMsg:
		log.Debug("received ttsInitMsg", "error", msg.err)
		if m.tts != nil {
			if msg.err != nil {
				log.Error("TTS initialization failed", "error", msg.err)
				m.tts.lastError = msg.err
				m.tts.isInitializing = false
				m.tts.isInitialized = false
			} else {
				log.Info("TTS initialization succeeded")
				m.tts.lastError = nil  // Clear any previous errors on successful init
				m.tts.isInitializing = false
				m.tts.isInitialized = true
				log.Debug("TTS state after init", 
					"isInitialized", m.tts.isInitialized, 
					"isInitializing", m.tts.isInitializing)
			}
			// Force a UI refresh by returning a no-op command
			// This ensures the view is re-rendered with the updated TTS state
			return m, tea.Batch(cmds...)
		}

	case spinner.TickMsg:
		// Forward spinner tick to TTS state if it's loading
		if m.tts != nil && (m.tts.isInitializing || m.tts.isSynthesizing || m.tts.isBuffering) {
			_, cmd := m.tts.Update(msg)
			return m, cmd
		}
		return m, nil
		
	case timer.TickMsg:
		// Forward timer tick to TTS state if playing
		if m.tts != nil && m.tts.isPlaying && !m.tts.isPaused {
			_, cmd := m.tts.Update(msg)
			return m, cmd
		}
		return m, nil
		
	case ttsTickMsg:
		// Update spinner and continue ticking during loading states
		if m.tts != nil {
			var spinnerCmd tea.Cmd
			if m.tts.isInitializing || m.tts.isSynthesizing || m.tts.isBuffering {
				_, spinnerCmd = m.tts.Update(msg)
				// Continue ticking for animation
				return m, tea.Batch(spinnerCmd, ttsTick())
			}
		}
		// Stop ticking when not loading
		return m, nil

	case ttsSentencesParsedMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
			} else {
				m.tts.sentences = msg.sentences
				m.tts.totalSentences = len(msg.sentences)
				m.tts.currentSentenceIndex = 0
			}
		}

	case ttsPlayMsg:
		if m.tts != nil {
			// Clear synthesizing state once play starts
			m.tts.SetLoadingState(false, false, "")
			
			if msg.err != nil {
				m.tts.lastError = msg.err
				// Clear the error after 3 seconds (longer for play errors)
				cmds = append(cmds, clearTTSErrorCmd(3*time.Second))
			} else {
				m.tts.lastError = nil  // Clear any previous errors
				m.tts.isPlaying = true
				m.tts.isPaused = false
				m.tts.isStopped = false
				// Record playback start time and start timer
				m.tts.playbackStart = time.Now()
				// Initialize and start timer
				cmds = append(cmds, m.tts.playbackTimer.Init())
				cmds = append(cmds, m.tts.playbackTimer.Start())
				cmds = append(cmds, monitorPlaybackCmd(m.tts.controller))
			}
		}

	case ttsPauseMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
			} else {
				m.tts.isPlaying = false
				m.tts.isPaused = true
				// Stop the timer to pause it
				cmds = append(cmds, m.tts.playbackTimer.Stop())
			}
		}

	case ttsStopMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
			} else {
				m.tts.isPlaying = false
				m.tts.isPaused = false
				m.tts.isStopped = true
				m.tts.currentSentenceIndex = 0
				// Stop and reset the timer
				cmds = append(cmds, m.tts.playbackTimer.Stop())
				m.tts.playbackStart = time.Time{}
			}
		}

	case ttsNextMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
				// Clear the error after 2 seconds
				cmds = append(cmds, clearTTSErrorCmd(2*time.Second))
			} else {
				m.tts.currentSentenceIndex = msg.sentenceIndex
			}
		}

	case ttsPrevMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
				// Clear the error after 2 seconds
				cmds = append(cmds, clearTTSErrorCmd(2*time.Second))
			} else {
				m.tts.currentSentenceIndex = msg.sentenceIndex
			}
		}
	
	case ttsClearErrorMsg:
		if m.tts != nil {
			m.tts.lastError = nil
		}

	case ttsSpeedChangeMsg:
		if m.tts != nil {
			if msg.err != nil {
				m.tts.lastError = msg.err
			} else {
				// Update the speed controller
				if m.tts.speedController != nil {
					m.tts.speedController.SetSpeed(msg.newSpeed)
				}
				// Also update the controller if it exists
				if m.tts.controller != nil {
					m.tts.controller.SetSpeed(msg.newSpeed)
				}
			}
		}
	
	case ttsPlaybackFinishedMsg:
		if m.tts != nil {
			log.Debug("TTS playback finished")
			m.tts.isPlaying = false
			m.tts.isPaused = false
			m.tts.isStopped = true
			m.tts.currentSentenceIndex = 0
			// Stop the timer
			cmds = append(cmds, m.tts.playbackTimer.Stop())
		}
	
	case ttsMonitorMsg:
		if m.tts != nil && m.tts.isPlaying {
			if msg.continueMonitoring {
				// Continue monitoring playback after a delay
				cmds = append(cmds, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
					// After delay, check playback status again
					return monitorPlaybackCmd(m.tts.controller)()
				}))
			}
		}
	}

	// Process children (unless we've consumed the message)
	// Special case: Always update pager for TTS messages to ensure UI refresh
	if _, isTTSMsg := msg.(ttsTickMsg); isTTSMsg && m.state == stateShowDocument {
		newPagerModel, cmd := m.pager.update(msg)
		m.pager = newPagerModel
		cmds = append(cmds, cmd)
	} else if !skipChildUpdate {
		switch m.state {
		case stateShowStash:
			newStashModel, cmd := m.stash.update(msg)
			m.stash = newStashModel
			cmds = append(cmds, cmd)

		case stateShowDocument:
			newPagerModel, cmd := m.pager.update(msg)
			m.pager = newPagerModel
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.fatalErr != nil {
		return errorView(m.fatalErr, true)
	}

	switch m.state { //nolint:exhaustive
	case stateShowDocument:
		return m.pager.View()
	default:
		return m.stash.view()
	}
}

func errorView(err error, fatal bool) string {
	exitMsg := "press any key to "
	if fatal {
		exitMsg += "exit"
	} else {
		exitMsg += "return"
	}
	s := fmt.Sprintf("%s\n\n%v\n\n%s",
		errorTitleStyle.Render("ERROR"),
		err,
		subtleStyle.Render(exitMsg),
	)
	return "\n" + indent(s, 3)
}

// COMMANDS

func findLocalFiles(m commonModel) tea.Cmd {
	return func() tea.Msg {
		log.Info("findLocalFiles")
		var (
			cwd = m.cfg.Path
			err error
		)

		if cwd == "" {
			cwd, err = os.Getwd()
		} else {
			var info os.FileInfo
			info, err = os.Stat(cwd)
			if err == nil && info.IsDir() {
				cwd, err = filepath.Abs(cwd)
			}
		}

		// Note that this is one error check for both cases above
		if err != nil {
			log.Error("error finding local files", "error", err)
			return errMsg{err}
		}

		log.Debug("local directory is", "cwd", cwd)

		// Switch between FindFiles and FindAllFiles to bypass .gitignore rules
		var ch chan gitcha.SearchResult
		if m.cfg.ShowAllFiles {
			ch, err = gitcha.FindAllFilesExcept(cwd, markdownExtensions, nil)
		} else {
			ch, err = gitcha.FindFilesExcept(cwd, markdownExtensions, ignorePatterns(m))
		}

		if err != nil {
			log.Error("error finding local files", "error", err)
			return errMsg{err}
		}

		return initLocalFileSearchMsg{ch: ch, cwd: cwd}
	}
}

func findNextLocalFile(m model) tea.Cmd {
	return func() tea.Msg {
		res, ok := <-m.localFileFinder

		if ok {
			// Okay now find the next one
			return foundLocalFileMsg(res)
		}
		// We're done
		log.Debug("local file search finished")
		return localFileSearchFinished{}
	}
}

func waitForStatusMessageTimeout(appCtx applicationContext, t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return statusMessageTimeoutMsg(appCtx)
	}
}

// ETC

// Convert a Gitcha result to an internal representation of a markdown
// document. Note that we could be doing things like checking if the file is
// a directory, but we trust that gitcha has already done that.
func localFileToMarkdown(cwd string, res gitcha.SearchResult) *markdown {
	return &markdown{
		localPath: res.Path,
		Note:      stripAbsolutePath(res.Path, cwd),
		Modtime:   res.Info.ModTime(),
	}
}

func stripAbsolutePath(fullPath, cwd string) string {
	fp, _ := filepath.EvalSymlinks(fullPath)
	cp, _ := filepath.EvalSymlinks(cwd)
	return strings.ReplaceAll(fp, cp+string(os.PathSeparator), "")
}

// Lightweight version of reflow's indent function.
func indent(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	l := strings.Split(s, "\n")
	b := strings.Builder{}
	i := strings.Repeat(" ", n)
	for _, v := range l {
		fmt.Fprintf(&b, "%s%s\n", i, v)
	}
	return b.String()
}
