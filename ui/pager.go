package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/v2/internal/tts"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/fsnotify/fsnotify"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
)

const (
	statusBarHeight = 1
	lineNumberWidth = 4
)

var (
	pagerHelpHeight int

	mintGreen = lipgloss.AdaptiveColor{Light: "#89F0CB", Dark: "#89F0CB"}
	darkGreen = lipgloss.AdaptiveColor{Light: "#1C8760", Dark: "#1C8760"}

	lineNumberFg = lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}

	statusBarNoteFg = lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}
	statusBarBg     = lipgloss.AdaptiveColor{Light: "#E6E6E6", Dark: "#242424"}

	statusBarScrollPosStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#949494", Dark: "#5A5A5A"}).
				Background(statusBarBg).
				Render

	statusBarNoteStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(statusBarBg).
				Render

	statusBarHelpStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(lipgloss.AdaptiveColor{Light: "#DCDCDC", Dark: "#323232"}).
				Render

	statusBarMessageStyle = lipgloss.NewStyle().
				Foreground(mintGreen).
				Background(darkGreen).
				Render

	statusBarMessageScrollPosStyle = lipgloss.NewStyle().
					Foreground(mintGreen).
					Background(darkGreen).
					Render

	statusBarMessageHelpStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#B6FFE4")).
					Background(green).
					Render

	helpViewStyle = lipgloss.NewStyle().
			Foreground(statusBarNoteFg).
			Background(lipgloss.AdaptiveColor{Light: "#f2f2f2", Dark: "#1B1B1B"}).
			Render

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lineNumberFg).
			Render
)

type (
	contentRenderedMsg string
	reloadMsg          struct{}
)

type pagerState int

const (
	pagerStateBrowse pagerState = iota
	pagerStateStatusMessage
)

type pagerModel struct {
	common   *commonModel
	viewport viewport.Model
	state    pagerState
	showHelp bool
	showTTSHelp bool // Toggle between standard and TTS help views

	statusMessage      string
	statusMessageTimer *time.Timer

	// Current document being rendered, sans-glamour rendering. We cache
	// it here so we can re-render it on resize.
	currentDocument markdown

	watcher *fsnotify.Watcher

	// TTS fields - only active when TTS is enabled
	ttsController   *tts.TTSController
	ttsEnabled      bool
	ttsState        string
	ttsError        string
	ttsProgress     float64
	ttsCurrent      int
	ttsTotal        int
	ttsInitializing bool
}

func newPagerModel(common *commonModel) pagerModel {
	// Init viewport
	vp := viewport.New(0, 0)
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	m := pagerModel{
		common:   common,
		state:    pagerStateBrowse,
		viewport: vp,
		// TTS fields
		ttsEnabled: common.cfg.TTSEnabled,
		ttsState:   "inactive",
	}
	m.initWatcher()
	return m
}

func (m *pagerModel) setSize(w, h int) {
	m.viewport.Width = w
	m.viewport.Height = h - statusBarHeight

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		m.viewport.Height -= (statusBarHeight + pagerHelpHeight)
	}
}

func (m *pagerModel) setContent(s string) {
	m.viewport.SetContent(s)
}

func (m *pagerModel) toggleHelp() {
	m.showHelp = !m.showHelp
	// Reset to standard help when toggling help off/on
	if !m.showHelp {
		m.showTTSHelp = false
	}
	m.setSize(m.common.width, m.common.height)
	if m.viewport.PastBottom() {
		m.viewport.GotoBottom()
	}
}

type pagerStatusMessage struct {
	message string
	isError bool
}

// Perform stuff that needs to happen after a successful markdown stash. Note
// that the returned command should be sent back the through the pager
// update function.
func (m *pagerModel) showStatusMessage(msg pagerStatusMessage) tea.Cmd {
	// Show a success message to the user
	m.state = pagerStateStatusMessage
	m.statusMessage = msg.message
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.statusMessageTimer = time.NewTimer(statusMessageTimeout)

	return waitForStatusMessageTimeout(pagerContext, m.statusMessageTimer)
}

func (m *pagerModel) unload() {
	log.Debug("unload")
	if m.showHelp {
		m.toggleHelp()
	}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	// Clean up TTS controller if active
	if m.ttsController != nil {
		_ = m.ttsController.Stop()
		m.ttsController = nil
		m.ttsState = "inactive"
	}
	m.state = pagerStateBrowse
	m.viewport.SetContent("")
	m.viewport.YOffset = 0
	m.unwatchFile()
}

func (m pagerModel) update(msg tea.Msg) (pagerModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", keyEsc:
			if m.state != pagerStateBrowse {
				m.state = pagerStateBrowse
				return m, nil
			}
		case "home", "g":
			m.viewport.GotoTop()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		case "end", "G":
			m.viewport.GotoBottom()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

		case "d":
			m.viewport.HalfViewDown()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

		case "u":
			m.viewport.HalfViewUp()
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}

		case "e":
			lineno := int(math.RoundToEven(float64(m.viewport.TotalLineCount()) * m.viewport.ScrollPercent()))
			if m.viewport.AtTop() {
				lineno = 0
			}
			log.Info(
				"opening editor",
				"file", m.currentDocument.localPath,
				"line", fmt.Sprintf("%d/%d", lineno, m.viewport.TotalLineCount()),
			)
			return m, openEditor(m.currentDocument.localPath, lineno)

		case "c":
			// Copy using OSC 52
			termenv.Copy(m.currentDocument.Body)
			// Copy using native system clipboard
			_ = clipboard.WriteAll(m.currentDocument.Body)
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"Copied contents", false}))

		case "r":
			return m, loadLocalMarkdown(&m.currentDocument)

		case "?":
			m.toggleHelp()
			// Reset to standard help when opening
			m.showTTSHelp = false
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
		
		case "t":
			// Toggle between standard and TTS help when help is shown and TTS is enabled
			if m.showHelp && m.ttsEnabled {
				m.showTTSHelp = !m.showTTSHelp
				if m.viewport.HighPerformanceRendering {
					cmds = append(cmds, viewport.Sync(m.viewport))
				}
			}

		// TTS Controls (only active when TTS is enabled)
		case " ": // Space key for play/pause
			if m.ttsEnabled {
				if m.ttsController == nil && !m.ttsInitializing {
					// Initialize TTS first
					m.ttsInitializing = true
					m.ttsState = "initializing"
					cmds = append(cmds, initTTSCmd(m.common.cfg.TTSEngine, m.common.cfg))
				} else if m.ttsController != nil {
					if m.ttsState == "playing" {
						cmds = append(cmds, ttsPauseCmd(m.ttsController))
					} else if m.ttsState == "paused" || m.ttsState == "ready" {
						if m.currentDocument.Body != "" {
							cmds = append(cmds, ttsPlayCmd(m.ttsController, m.currentDocument.Body))
						}
					}
				}
			}
		case "n": // Next sentence (only when TTS active)
			if m.ttsEnabled && m.ttsController != nil {
				cmds = append(cmds, ttsNextCmd(m.ttsController))
			}
		case "p": // Previous sentence (only when TTS active)
			if m.ttsEnabled && m.ttsController != nil && msg.String() == "p" && !strings.Contains(msg.String(), "ctrl") {
				cmds = append(cmds, ttsPrevCmd(m.ttsController))
			}
		case "s": // Stop TTS (only when TTS active)
			if m.ttsEnabled && m.ttsController != nil {
				cmds = append(cmds, ttsStopCmd(m.ttsController))
			}
		case "1", "2", "3", "4", "5": // Speed control (only when TTS active)
			if m.ttsEnabled && m.ttsController != nil {
				speedMap := map[string]float64{
					"1": 0.5, "2": 0.75, "3": 1.0, "4": 1.25, "5": 1.5,
				}
				if speed, exists := speedMap[msg.String()]; exists {
					cmds = append(cmds, ttsSetSpeedCmd(m.ttsController, speed))
				}
			}
		}

	// Glow has rendered the content
	case contentRenderedMsg:
		log.Info("content rendered", "state", m.state)

		m.setContent(string(msg))
		if m.viewport.HighPerformanceRendering {
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
		cmds = append(cmds, m.watchFile)

	// The file was changed on disk and we're reloading it
	case reloadMsg:
		return m, loadLocalMarkdown(&m.currentDocument)

	// We've finished editing the document, potentially making changes. Let's
	// retrieve the latest version of the document so that we display
	// up-to-date contents.
	case editorFinishedMsg:
		return m, loadLocalMarkdown(&m.currentDocument)

	// We've received terminal dimensions, either for the first time or
	// after a resize
	case tea.WindowSizeMsg:
		return m, renderWithGlamour(m, m.currentDocument.Body)

	case statusMessageTimeoutMsg:
		m.state = pagerStateBrowse

	// TTS Message Handlers
	case TTSInitDoneMsg:
		m.ttsInitializing = false
		if msg.Err != nil {
			m.ttsError = msg.Err.Error()
			m.ttsState = "error"
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS initialization failed: " + msg.Err.Error(), true}))
		} else {
			m.ttsController = msg.Controller
			m.ttsState = "ready"
			m.ttsError = ""
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS ready - Press space to start", false}))
			// Schedule periodic status updates
			cmds = append(cmds, ttsStatusCmd(m.ttsController))
		}

	case TTSPlayDoneMsg:
		if msg.Err != nil {
			m.ttsError = msg.Err.Error()
			m.ttsState = "error"
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS play failed: " + msg.Err.Error(), true}))
		} else {
			m.ttsState = "playing"
			m.ttsError = ""
		}

	case TTSPauseDoneMsg:
		if msg.Err != nil {
			m.ttsError = msg.Err.Error()
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS pause failed: " + msg.Err.Error(), true}))
		} else {
			m.ttsState = "paused"
		}

	case TTSStopDoneMsg:
		if msg.Err != nil {
			m.ttsError = msg.Err.Error()
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS stop failed: " + msg.Err.Error(), true}))
		} else {
			m.ttsState = "ready"
		}

	case TTSNextDoneMsg:
		if msg.Err != nil {
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS next failed", true}))
		}

	case TTSPrevDoneMsg:
		if msg.Err != nil {
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS previous failed", true}))
		}

	case TTSSpeedDoneMsg:
		if msg.Err != nil {
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS speed change failed", true}))
		} else {
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS speed updated", false}))
		}

	case TTSStatusMsg:
		if m.ttsEnabled {
			m.ttsState = msg.State
			m.ttsCurrent = msg.Current
			m.ttsTotal = msg.Total
			m.ttsProgress = msg.Progress
			if msg.Error != "" {
				m.ttsError = msg.Error
			}
			// Schedule next status update if still active
			if msg.State == "playing" || msg.State == "processing" {
				cmds = append(cmds, tea.Tick(time.Millisecond*100, func(time.Time) tea.Msg {
					return ttsStatusCmd(m.ttsController)()
				}))
			}
		}

	case TTSErrorMsg:
		m.ttsError = msg.Err.Error()
		m.ttsState = "error"
		cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{"TTS error: " + msg.Err.Error(), true}))
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m pagerModel) View() string {
	var b strings.Builder
	fmt.Fprint(&b, m.viewport.View()+"\n")

	// Footer
	m.statusBarView(&b)

	if m.showHelp {
		fmt.Fprint(&b, "\n"+m.helpView())
	}

	return b.String()
}

func (m pagerModel) statusBarView(b *strings.Builder) {
	const (
		minPercent               float64 = 0.0
		maxPercent               float64 = 1.0
		percentToStringMagnitude float64 = 100.0
	)

	showStatusMessage := m.state == pagerStateStatusMessage

	// Logo
	logo := glowLogoView()

	// Scroll percent
	percent := math.Max(minPercent, math.Min(maxPercent, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*percentToStringMagnitude)
	if showStatusMessage {
		scrollPercent = statusBarMessageScrollPosStyle(scrollPercent)
	} else {
		scrollPercent = statusBarScrollPosStyle(scrollPercent)
	}

	// "Help" note
	var helpNote string
	if showStatusMessage {
		helpNote = statusBarMessageHelpStyle(" ? Help ")
	} else {
		helpNote = statusBarHelpStyle(" ? Help ")
	}

	// Note with TTS status
	var note string
	if showStatusMessage {
		note = m.statusMessage
	} else if m.ttsEnabled {
		// Show TTS status when TTS is active
		switch m.ttsState {
		case "inactive":
			note = m.currentDocument.Note + " | TTS: Press space to start"
		case "initializing":
			note = m.currentDocument.Note + " | TTS: Initializing..."
		case "ready":
			note = m.currentDocument.Note + " | TTS: Ready"
		case "playing":
			if m.ttsTotal > 0 {
				note = fmt.Sprintf("%s | TTS: Playing (%d/%d) %.0f%%",
					m.currentDocument.Note, m.ttsCurrent, m.ttsTotal, m.ttsProgress*100)
			} else {
				note = m.currentDocument.Note + " | TTS: Playing"
			}
		case "paused":
			if m.ttsTotal > 0 {
				note = fmt.Sprintf("%s | TTS: Paused (%d/%d) %.0f%%",
					m.currentDocument.Note, m.ttsCurrent, m.ttsTotal, m.ttsProgress*100)
			} else {
				note = m.currentDocument.Note + " | TTS: Paused"
			}
		case "processing":
			note = m.currentDocument.Note + " | TTS: Processing..."
		case "error":
			if m.ttsError != "" {
				note = m.currentDocument.Note + " | TTS Error: " + m.ttsError
			} else {
				note = m.currentDocument.Note + " | TTS: Error"
			}
		default:
			note = m.currentDocument.Note + " | TTS: " + m.ttsState
		}
	} else {
		note = m.currentDocument.Note
	}
	note = truncate.StringWithTail(" "+note+" ", uint(max(0, //nolint:gosec
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)), ellipsis)
	if showStatusMessage {
		note = statusBarMessageStyle(note)
	} else {
		note = statusBarNoteStyle(note)
	}

	// Empty space
	padding := max(0,
		m.common.width-
			ansi.PrintableRuneWidth(logo)-
			ansi.PrintableRuneWidth(note)-
			ansi.PrintableRuneWidth(scrollPercent)-
			ansi.PrintableRuneWidth(helpNote),
	)
	emptySpace := strings.Repeat(" ", padding)
	if showStatusMessage {
		emptySpace = statusBarMessageStyle(emptySpace)
	} else {
		emptySpace = statusBarNoteStyle(emptySpace)
	}

	fmt.Fprintf(b, "%s%s%s%s%s",
		logo,
		note,
		emptySpace,
		scrollPercent,
		helpNote,
	)
}

func (m pagerModel) helpView() (s string) {
	// Show TTS help if toggled, otherwise show standard help
	if m.showTTSHelp && m.ttsEnabled {
		return m.ttsHelpView()
	}
	return m.standardHelpView()
}

// standardHelpView shows the original help layout
func (m pagerModel) standardHelpView() (s string) {
	col1 := []string{
		"g/home  go to top",
		"G/end   go to bottom",
		"c       copy contents",
		"e       edit this document",
		"r       reload this document",
		"esc     back to files",
		"q       quit",
	}

	s += "\n"
	s += "k/↑      up                  " + col1[0] + "\n"
	s += "j/↓      down                " + col1[1] + "\n"
	s += "b/pgup   page up             " + col1[2] + "\n"
	s += "f/pgdn   page down           " + col1[3] + "\n"
	s += "u        ½ page up           " + col1[4] + "\n"
	s += "d        ½ page down         "

	if len(col1) > 5 {
		s += col1[5]
	}
	
	// Add toggle hint if TTS is enabled
	if m.ttsEnabled {
		s += "\n\nt        toggle TTS help"
	}

	s = indent(s, 2)

	// Fill up empty cells with spaces for background coloring
	if m.common.width > 0 {
		lines := strings.Split(s, "\n")
		for i := 0; i < len(lines); i++ {
			l := runewidth.StringWidth(lines[i])
			n := max(m.common.width-l, 0)
			lines[i] += strings.Repeat(" ", n)
		}

		s = strings.Join(lines, "\n")
	}

	return helpViewStyle(s)
}

// ttsHelpView shows TTS-specific help
func (m pagerModel) ttsHelpView() (s string) {
	s += "\n"
	s += "### TTS Controls ###\n"
	s += "\n"
	s += "space    play/pause TTS\n"
	s += "n        next sentence\n"
	s += "p        previous sentence\n"
	s += "s        stop TTS\n"
	s += "1-5      speed control (0.5x to 1.5x)\n"
	s += "\n"
	s += "### Navigation (while playing) ###\n"
	s += "\n"
	s += "k/↑      scroll up\n"
	s += "j/↓      scroll down\n"
	s += "g/home   go to top\n"
	s += "G/end    go to bottom\n"
	s += "\n"
	s += "t        toggle standard help\n"
	s += "?        close help\n"
	s += "q        quit\n"

	s = indent(s, 2)

	// Fill up empty cells with spaces for background coloring
	if m.common.width > 0 {
		lines := strings.Split(s, "\n")
		for i := 0; i < len(lines); i++ {
			l := runewidth.StringWidth(lines[i])
			n := max(m.common.width-l, 0)
			lines[i] += strings.Repeat(" ", n)
		}

		s = strings.Join(lines, "\n")
	}

	return helpViewStyle(s)
}

// COMMANDS

func renderWithGlamour(m pagerModel, md string) tea.Cmd {
	return func() tea.Msg {
		s, err := glamourRender(m, md)
		if err != nil {
			log.Error("error rendering with Glamour", "error", err)
			return errMsg{err}
		}
		return contentRenderedMsg(s)
	}
}

// This is where the magic happens.
func glamourRender(m pagerModel, markdown string) (string, error) {
	trunc := lipgloss.NewStyle().MaxWidth(m.viewport.Width - lineNumberWidth).Render

	if !config.GlamourEnabled {
		return markdown, nil
	}

	isCode := !utils.IsMarkdownFile(m.currentDocument.Note)
	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), m.viewport.Width)) //nolint:gosec
	if isCode {
		width = 0
	}

	options := []glamour.TermRendererOption{
		utils.GlamourStyle(m.common.cfg.GlamourStyle, isCode),
		glamour.WithWordWrap(width),
	}

	if m.common.cfg.PreserveNewLines {
		options = append(options, glamour.WithPreservedNewLines())
	}
	r, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return "", fmt.Errorf("error creating glamour renderer: %w", err)
	}

	if isCode {
		markdown = utils.WrapCodeBlock(markdown, filepath.Ext(m.currentDocument.Note))
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", fmt.Errorf("error rendering markdown: %w", err)
	}

	if isCode {
		out = strings.TrimSpace(out)
	}

	// trim lines
	lines := strings.Split(out, "\n")

	var content strings.Builder
	for i, s := range lines {
		if isCode || m.common.cfg.ShowLineNumbers {
			content.WriteString(lineNumberStyle(fmt.Sprintf("%"+fmt.Sprint(lineNumberWidth)+"d", i+1)))
			content.WriteString(trunc(s))
		} else {
			content.WriteString(s)
		}

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content.WriteRune('\n')
		}
	}

	return content.String(), nil
}

func (m *pagerModel) initWatcher() {
	var err error
	m.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Error("error creating fsnotify watcher", "error", err)
	}
}

func (m *pagerModel) watchFile() tea.Msg {
	dir := m.localDir()

	if err := m.watcher.Add(dir); err != nil {
		log.Error("error adding dir to fsnotify watcher", "error", err)
		return nil
	}

	log.Info("fsnotify watching dir", "dir", dir)

	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok || event.Name != m.currentDocument.localPath {
				continue
			}

			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			log.Debug("fsnotify event", "file", event.Name, "event", event.Op)
			return reloadMsg{}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				continue
			}
			log.Debug("fsnotify error", "dir", dir, "error", err)
		}
	}
}

func (m *pagerModel) unwatchFile() {
	dir := m.localDir()

	err := m.watcher.Remove(dir)
	if err == nil {
		log.Debug("fsnotify dir unwatched", "dir", dir)
	} else {
		log.Error("fsnotify fail to unwatch dir", "dir", dir, "error", err)
	}
}

func (m *pagerModel) localDir() string {
	return filepath.Dir(m.currentDocument.localPath)
}
