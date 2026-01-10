package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/n0roo/pal-kit/internal/convention"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/n0roo/pal-kit/internal/pipeline"
	"github.com/n0roo/pal-kit/internal/session"
)

// Tab represents a dashboard tab
type Tab int

const (
	TabStatus Tab = iota
	TabSessions
	TabPipelines
	TabDocs
	TabConventions
)

func (t Tab) String() string {
	return []string{"Status", "Sessions", "Pipelines", "Docs", "Conventions"}[t]
}

// Model is the main TUI model
type Model struct {
	// Config
	projectRoot string
	dbPath      string

	// State
	currentTab  Tab
	width       int
	height      int
	ready       bool
	lastRefresh time.Time
	err         error

	// Data
	sessions    []session.Session
	pipelines   []pipeline.Pipeline
	documents   []docs.Document
	conventions []*convention.Convention
	stats       map[string]interface{}

	// Components
	spinner spinner.Model
}

// tickMsg is sent periodically to refresh data
type tickMsg time.Time

// dataMsg carries refreshed data
type dataMsg struct {
	sessions    []session.Session
	pipelines   []pipeline.Pipeline
	documents   []docs.Document
	conventions []*convention.Convention
	stats       map[string]interface{}
	err         error
}

// NewModel creates a new TUI model
func NewModel(projectRoot, dbPath string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return Model{
		projectRoot: projectRoot,
		dbPath:      dbPath,
		currentTab:  TabStatus,
		spinner:     s,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.refreshData,
		tickEvery(5*time.Second),
	)
}

// tickEvery returns a command that ticks every duration
func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshData fetches fresh data
func (m Model) refreshData() tea.Msg {
	data := dataMsg{
		stats: make(map[string]interface{}),
	}

	// Sessions
	database, err := db.Open(m.dbPath)
	if err != nil {
		data.err = err
		return data
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	if sessions, err := sessionSvc.List(false, 10); err == nil {
		data.sessions = sessions
	}

	// Pipelines
	plSvc := pipeline.NewService(database)
	if pipelines, err := plSvc.List("", 10); err == nil {
		data.pipelines = pipelines
	}

	// Docs
	docsSvc := docs.NewService(m.projectRoot)
	if documents, err := docsSvc.List(); err == nil {
		data.documents = documents
	}

	// Conventions
	convSvc := convention.NewService(m.projectRoot)
	if conventions, err := convSvc.List(); err == nil {
		data.conventions = conventions
	}

	return data
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentTab = TabStatus
		case "2":
			m.currentTab = TabSessions
		case "3":
			m.currentTab = TabPipelines
		case "4":
			m.currentTab = TabDocs
		case "5":
			m.currentTab = TabConventions
		case "r":
			return m, m.refreshData
		case "tab":
			m.currentTab = Tab((int(m.currentTab) + 1) % 5)
		case "shift+tab":
			m.currentTab = Tab((int(m.currentTab) + 4) % 5)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tickMsg:
		return m, tea.Batch(
			m.refreshData,
			tickEvery(5*time.Second),
		)

	case dataMsg:
		m.sessions = msg.sessions
		m.pipelines = msg.pipelines
		m.documents = msg.documents
		m.conventions = msg.conventions
		m.stats = msg.stats
		m.err = msg.err
		m.lastRefresh = time.Now()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content
	switch m.currentTab {
	case TabStatus:
		b.WriteString(m.renderStatusTab())
	case TabSessions:
		b.WriteString(m.renderSessionsTab())
	case TabPipelines:
		b.WriteString(m.renderPipelinesTab())
	case TabDocs:
		b.WriteString(m.renderDocsTab())
	case TabConventions:
		b.WriteString(m.renderConventionsTab())
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m Model) renderHeader() string {
	title := "🚀 PAL Kit Dashboard"
	refresh := fmt.Sprintf("Last refresh: %s", m.lastRefresh.Format("15:04:05"))

	headerWidth := m.width
	if headerWidth < 60 {
		headerWidth = 60
	}

	left := lipgloss.NewStyle().Bold(true).Render(title)
	right := lipgloss.NewStyle().Foreground(mutedColor).Render(refresh)

	gap := headerWidth - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 0
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#2D3748")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Width(headerWidth).
		Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderTabs() string {
	var tabs []string
	for i := 0; i < 5; i++ {
		tab := Tab(i)
		style := tabStyle
		if tab == m.currentTab {
			style = activeTabStyle
		}
		tabs = append(tabs, style.Render(fmt.Sprintf("[%d]%s", i+1, tab.String())))
	}
	return strings.Join(tabs, " ")
}

func (m Model) renderFooter() string {
	help := "  [1-5] Switch tabs  [Tab] Next  [r] Refresh  [q] Quit"
	return helpStyle.Render(help)
}

func (m Model) renderStatusTab() string {
	var b strings.Builder

	// Sessions summary
	activeSessions := 0
	for _, s := range m.sessions {
		if s.Status == "active" {
			activeSessions++
		}
	}

	sessionBox := boxStyle.Width(35).Render(
		titleStyle.Render("📍 Sessions") + "\n" +
			fmt.Sprintf("Active: %s\n", statusActiveStyle.Render(fmt.Sprintf("%d", activeSessions))) +
			fmt.Sprintf("Total:  %d", len(m.sessions)),
	)

	// Pipelines summary
	activePipelines := 0
	for _, p := range m.pipelines {
		if p.Status == "running" {
			activePipelines++
		}
	}

	pipelineBox := boxStyle.Width(35).Render(
		titleStyle.Render("🔄 Pipelines") + "\n" +
			fmt.Sprintf("Running: %s\n", statusActiveStyle.Render(fmt.Sprintf("%d", activePipelines))) +
			fmt.Sprintf("Total:   %d", len(m.pipelines)),
	)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sessionBox, "  ", pipelineBox))
	b.WriteString("\n\n")

	// Documents summary
	validDocs := 0
	modifiedDocs := 0
	for _, d := range m.documents {
		switch d.Status {
		case docs.StatusValid:
			validDocs++
		case docs.StatusModified:
			modifiedDocs++
		}
	}

	docsBox := boxStyle.Width(35).Render(
		titleStyle.Render("📚 Documents") + "\n" +
			fmt.Sprintf("Valid:    %s\n", statusActiveStyle.Render(fmt.Sprintf("%d", validDocs))) +
			fmt.Sprintf("Modified: %s\n", statusPendingStyle.Render(fmt.Sprintf("%d", modifiedDocs))) +
			fmt.Sprintf("Total:    %d", len(m.documents)),
	)

	// Conventions summary
	enabledConv := 0
	for _, c := range m.conventions {
		if c.Enabled {
			enabledConv++
		}
	}

	convBox := boxStyle.Width(35).Render(
		titleStyle.Render("📋 Conventions") + "\n" +
			fmt.Sprintf("Enabled: %s\n", statusActiveStyle.Render(fmt.Sprintf("%d", enabledConv))) +
			fmt.Sprintf("Total:   %d", len(m.conventions)),
	)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, docsBox, "  ", convBox))

	return b.String()
}

func (m Model) renderSessionsTab() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📍 Sessions"))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(statusMutedStyle.Render("  No sessions"))
		return b.String()
	}

	for _, s := range m.sessions {
		icon := StatusIcon(s.Status)
		title := s.ID
		if s.Title.Valid {
			title = s.Title.String
		}

		duration := ""
		if s.Status == "active" {
			elapsed := time.Since(s.StartedAt).Round(time.Minute)
			duration = statusMutedStyle.Render(fmt.Sprintf(" (%s)", elapsed))
		}

		b.WriteString(fmt.Sprintf("  %s %s%s\n", icon, title, duration))
	}

	return b.String()
}

func (m Model) renderPipelinesTab() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🔄 Pipelines"))
	b.WriteString("\n\n")

	if len(m.pipelines) == 0 {
		b.WriteString(statusMutedStyle.Render("  No pipelines"))
		return b.String()
	}

	for _, p := range m.pipelines {
		icon := StatusIcon(p.Status)
		b.WriteString(fmt.Sprintf("  %s %s (%s)\n", icon, p.Name, p.Status))
	}

	return b.String()
}

func (m Model) renderDocsTab() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📚 Documents"))
	b.WriteString("\n\n")

	if len(m.documents) == 0 {
		b.WriteString(statusMutedStyle.Render("  No documents"))
		return b.String()
	}

	// Group by type
	byType := make(map[docs.DocType][]docs.Document)
	for _, d := range m.documents {
		byType[d.Type] = append(byType[d.Type], d)
	}

	typeOrder := []docs.DocType{
		docs.DocTypeClaude,
		docs.DocTypeAgent,
		docs.DocTypePort,
		docs.DocTypeConvention,
	}

	for _, dt := range typeOrder {
		docList := byType[dt]
		if len(docList) == 0 {
			continue
		}

		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %s (%d)", dt, len(docList))))
		b.WriteString("\n")

		for _, d := range docList {
			icon := StatusIcon(string(d.Status))
			b.WriteString(fmt.Sprintf("    %s %s\n", icon, d.RelativePath))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderConventionsTab() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📋 Conventions"))
	b.WriteString("\n\n")

	if len(m.conventions) == 0 {
		b.WriteString(statusMutedStyle.Render("  No conventions"))
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render("  Run: pal conv init"))
		return b.String()
	}

	for _, c := range m.conventions {
		status := "disabled"
		if c.Enabled {
			status = "enabled"
		}
		icon := StatusIcon(status)

		rules := statusMutedStyle.Render(fmt.Sprintf("(%d rules)", len(c.Rules)))
		b.WriteString(fmt.Sprintf("  %s %s %s\n", icon, c.Name, rules))
	}

	return b.String()
}

// Run starts the TUI
func Run(projectRoot, dbPath string) error {
	p := tea.NewProgram(
		NewModel(projectRoot, dbPath),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
