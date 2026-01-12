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
	TabWorkflows
	TabDocs
	TabConventions
)

func (t Tab) String() string {
	return []string{"Status", "Sessions", "Workflows", "Docs", "Conventions"}[t]
}

// Panel represents which panel is focused
type Panel int

const (
	PanelList Panel = iota
	PanelDetail
)

// Model is the main TUI model
type Model struct {
	// Config
	projectRoot string
	dbPath      string

	// State
	currentTab   Tab
	focusedPanel Panel
	showDetail   bool
	width        int
	height       int
	ready        bool
	lastRefresh  time.Time
	err          error
	showHelp     bool

	// Cursor state per tab
	cursors map[Tab]int

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
		cursors:     make(map[Tab]int),
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

// getListSize returns the list size for the current tab
func (m Model) getListSize() int {
	switch m.currentTab {
	case TabSessions:
		return len(m.sessions)
	case TabWorkflows:
		return len(m.pipelines)
	case TabDocs:
		return len(m.documents)
	case TabConventions:
		return len(m.conventions)
	default:
		return 0
	}
}

// moveCursor moves the cursor up or down
func (m *Model) moveCursor(delta int) {
	size := m.getListSize()
	if size == 0 {
		return
	}

	cursor := m.cursors[m.currentTab]
	cursor += delta

	// Clamp cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= size {
		cursor = size - 1
	}

	m.cursors[m.currentTab] = cursor
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle help overlay first
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = true
		case "1":
			m.currentTab = TabStatus
		case "2":
			m.currentTab = TabSessions
		case "3":
			m.currentTab = TabWorkflows
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

		// List navigation
		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)
		case "g", "home":
			m.cursors[m.currentTab] = 0
		case "G", "end":
			size := m.getListSize()
			if size > 0 {
				m.cursors[m.currentTab] = size - 1
			}

		// Panel navigation (only for tabs with lists)
		case "enter", "l":
			if m.currentTab != TabStatus && m.getListSize() > 0 {
				m.showDetail = true
				m.focusedPanel = PanelDetail
			}
		case "h", "esc":
			if m.showDetail {
				m.showDetail = false
				m.focusedPanel = PanelList
			}
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

		// Clamp cursors after data refresh
		for tab := TabSessions; tab <= TabConventions; tab++ {
			var size int
			switch tab {
			case TabSessions:
				size = len(m.sessions)
			case TabWorkflows:
				size = len(m.pipelines)
			case TabDocs:
				size = len(m.documents)
			case TabConventions:
				size = len(m.conventions)
			}
			if m.cursors[tab] >= size && size > 0 {
				m.cursors[tab] = size - 1
			}
		}

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

	// Show help overlay if active
	if m.showHelp {
		return m.renderHelpOverlay()
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content - split view for list tabs when detail is shown
	if m.showDetail && m.currentTab != TabStatus {
		b.WriteString(m.renderSplitView())
	} else {
		switch m.currentTab {
		case TabStatus:
			b.WriteString(m.renderStatusTab())
		case TabSessions:
			b.WriteString(m.renderSessionsTab())
		case TabWorkflows:
			b.WriteString(m.renderWorkflowsTab())
		case TabDocs:
			b.WriteString(m.renderDocsTab())
		case TabConventions:
			b.WriteString(m.renderConventionsTab())
		}
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
	var help string
	if m.showDetail {
		help = "  [j/k] Navigate  [h/Esc] Close detail  [l/Enter] Open detail  [?] Help  [q] Quit"
	} else {
		help = "  [j/k] Navigate  [l/Enter] Detail  [1-5] Tabs  [?] Help  [q] Quit"
	}
	return helpStyle.Render(help)
}

func (m Model) renderHelpOverlay() string {
	var b strings.Builder

	// Title
	b.WriteString(helpTitleStyle.Render("📖 PAL Kit TUI Help"))
	b.WriteString("\n\n")

	// Navigation section
	b.WriteString(helpSectionStyle.Render("Navigation"))
	b.WriteString("\n")
	helpItems := []struct{ key, desc string }{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"g / Home", "Go to first item"},
		{"G / End", "Go to last item"},
		{"l / Enter", "Open detail view"},
		{"h / Esc", "Close detail view"},
	}
	for _, item := range helpItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-10s", item.key)),
			helpDescStyle.Render(item.desc)))
	}

	// Tabs section
	b.WriteString("\n")
	b.WriteString(helpSectionStyle.Render("Tabs"))
	b.WriteString("\n")
	tabItems := []struct{ key, desc string }{
		{"1", "Status overview"},
		{"2", "Sessions list"},
		{"3", "Workflows list"},
		{"4", "Documents list"},
		{"5", "Conventions list"},
		{"Tab", "Next tab"},
		{"Shift+Tab", "Previous tab"},
	}
	for _, item := range tabItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-10s", item.key)),
			helpDescStyle.Render(item.desc)))
	}

	// General section
	b.WriteString("\n")
	b.WriteString(helpSectionStyle.Render("General"))
	b.WriteString("\n")
	generalItems := []struct{ key, desc string }{
		{"r", "Refresh data"},
		{"?", "Toggle help"},
		{"q / Ctrl+C", "Quit"},
	}
	for _, item := range generalItems {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Render(fmt.Sprintf("%-10s", item.key)),
			helpDescStyle.Render(item.desc)))
	}

	// Close hint
	b.WriteString("\n")
	b.WriteString(statusMutedStyle.Render("Press ? or Esc to close"))

	// Center the overlay
	content := overlayStyle.Render(b.String())

	// Calculate position for centering
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	padLeft := (m.width - contentWidth) / 2
	padTop := (m.height - contentHeight) / 2

	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	return lipgloss.NewStyle().
		PaddingLeft(padLeft).
		PaddingTop(padTop).
		Render(content)
}

func (m Model) renderSplitView() string {
	// Calculate panel widths (40% list, 60% detail)
	availableWidth := m.width
	if availableWidth < 80 {
		availableWidth = 80
	}

	listWidth := availableWidth * 35 / 100
	detailWidth := availableWidth * 55 / 100

	// Ensure minimum widths
	if listWidth < 25 {
		listWidth = 25
	}
	if detailWidth < 30 {
		detailWidth = 30
	}

	// Get list content
	var listContent string
	switch m.currentTab {
	case TabSessions:
		listContent = m.renderSessionsTab()
	case TabWorkflows:
		listContent = m.renderWorkflowsTab()
	case TabDocs:
		listContent = m.renderDocsTab()
	case TabConventions:
		listContent = m.renderConventionsTab()
	}

	// Get detail content
	detailContent := m.renderDetailPanel()

	// Apply panel styles based on focus
	var listPanel, detailPanel string
	if m.focusedPanel == PanelList {
		listPanel = listPanelStyle.Width(listWidth).Render(listContent)
		detailPanel = detailPanelInactiveStyle.Width(detailWidth).Render(detailContent)
	} else {
		listPanel = listPanelInactiveStyle.Width(listWidth).Render(listContent)
		detailPanel = detailPanelStyle.Width(detailWidth).Render(detailContent)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)
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

	// Workflows summary
	activeWorkflows := 0
	for _, p := range m.pipelines {
		if p.Status == "running" {
			activeWorkflows++
		}
	}

	workflowBox := boxStyle.Width(35).Render(
		titleStyle.Render("🔄 Workflows") + "\n" +
			fmt.Sprintf("Running: %s\n", statusActiveStyle.Render(fmt.Sprintf("%d", activeWorkflows))) +
			fmt.Sprintf("Total:   %d", len(m.pipelines)),
	)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, sessionBox, "  ", workflowBox))
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

	cursor := m.cursors[TabSessions]
	b.WriteString(titleStyle.Render("📍 Sessions"))
	b.WriteString(statusMutedStyle.Render(fmt.Sprintf(" (%d/%d)", cursor+1, len(m.sessions))))
	b.WriteString("\n\n")

	if len(m.sessions) == 0 {
		b.WriteString(statusMutedStyle.Render("  No sessions"))
		return b.String()
	}

	for i, s := range m.sessions {
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

		line := fmt.Sprintf("%s %s%s", icon, title, duration)
		if i == cursor {
			b.WriteString(cursorStyle.Render(">") + " " + selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalItemStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderWorkflowsTab() string {
	var b strings.Builder

	cursor := m.cursors[TabWorkflows]
	b.WriteString(titleStyle.Render("🔄 Workflows"))
	b.WriteString(statusMutedStyle.Render(fmt.Sprintf(" (%d/%d)", cursor+1, len(m.pipelines))))
	b.WriteString("\n\n")

	if len(m.pipelines) == 0 {
		b.WriteString(statusMutedStyle.Render("  No workflows"))
		return b.String()
	}

	for i, p := range m.pipelines {
		icon := StatusIcon(p.Status)
		line := fmt.Sprintf("%s %s (%s)", icon, p.Name, p.Status)
		if i == cursor {
			b.WriteString(cursorStyle.Render(">") + " " + selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalItemStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

func (m Model) renderDocsTab() string {
	var b strings.Builder

	cursor := m.cursors[TabDocs]
	b.WriteString(titleStyle.Render("📚 Documents"))
	b.WriteString(statusMutedStyle.Render(fmt.Sprintf(" (%d/%d)", cursor+1, len(m.documents))))
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

	// Flat index for cursor
	idx := 0
	for _, dt := range typeOrder {
		docList := byType[dt]
		if len(docList) == 0 {
			continue
		}

		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %s (%d)", dt, len(docList))))
		b.WriteString("\n")

		for _, d := range docList {
			icon := StatusIcon(string(d.Status))
			line := fmt.Sprintf("%s %s", icon, d.RelativePath)
			if idx == cursor {
				b.WriteString("  " + cursorStyle.Render(">") + " " + selectedItemStyle.Render(line) + "\n")
			} else {
				b.WriteString("    " + line + "\n")
			}
			idx++
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderConventionsTab() string {
	var b strings.Builder

	cursor := m.cursors[TabConventions]
	b.WriteString(titleStyle.Render("📋 Conventions"))
	b.WriteString(statusMutedStyle.Render(fmt.Sprintf(" (%d/%d)", cursor+1, len(m.conventions))))
	b.WriteString("\n\n")

	if len(m.conventions) == 0 {
		b.WriteString(statusMutedStyle.Render("  No conventions"))
		b.WriteString("\n\n")
		b.WriteString(subtitleStyle.Render("  Run: pal conv init"))
		return b.String()
	}

	for i, c := range m.conventions {
		status := "disabled"
		if c.Enabled {
			status = "enabled"
		}
		icon := StatusIcon(status)

		rules := statusMutedStyle.Render(fmt.Sprintf("(%d rules)", len(c.Rules)))
		line := fmt.Sprintf("%s %s %s", icon, c.Name, rules)
		if i == cursor {
			b.WriteString(cursorStyle.Render(">") + " " + selectedItemStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalItemStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

// Detail view renderers

func (m Model) renderSessionDetail() string {
	var b strings.Builder

	cursor := m.cursors[TabSessions]
	if cursor >= len(m.sessions) {
		return statusMutedStyle.Render("No session selected")
	}

	s := m.sessions[cursor]

	// Title
	title := s.ID
	if s.Title.Valid {
		title = s.Title.String
	}
	b.WriteString(detailTitleStyle.Render("📍 " + title))
	b.WriteString("\n\n")

	// Details
	details := []struct{ label, value string }{
		{"ID", s.ID},
		{"Status", s.Status},
		{"Started", s.StartedAt.Format("2006-01-02 15:04:05")},
	}

	if s.EndedAt.Valid {
		details = append(details, struct{ label, value string }{
			"Ended", s.EndedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	if s.Status == "active" {
		elapsed := time.Since(s.StartedAt).Round(time.Second)
		details = append(details, struct{ label, value string }{
			"Duration", elapsed.String(),
		})
	}

	for _, d := range details {
		b.WriteString(detailLabelStyle.Render(d.label+":") + " ")
		if d.label == "Status" {
			b.WriteString(StatusIcon(d.value) + " " + d.value)
		} else {
			b.WriteString(detailValueStyle.Render(d.value))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderWorkflowDetail() string {
	var b strings.Builder

	cursor := m.cursors[TabWorkflows]
	if cursor >= len(m.pipelines) {
		return statusMutedStyle.Render("No workflow selected")
	}

	p := m.pipelines[cursor]

	// Title
	b.WriteString(detailTitleStyle.Render("🔄 " + p.Name))
	b.WriteString("\n\n")

	// Details
	details := []struct{ label, value string }{
		{"ID", p.ID},
		{"Status", p.Status},
		{"Created", p.CreatedAt.Format("2006-01-02 15:04:05")},
	}

	if p.SessionID.Valid {
		details = append(details, struct{ label, value string }{
			"Session", p.SessionID.String,
		})
	}

	if p.StartedAt.Valid {
		details = append(details, struct{ label, value string }{
			"Started", p.StartedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	if p.CompletedAt.Valid {
		details = append(details, struct{ label, value string }{
			"Completed", p.CompletedAt.Time.Format("2006-01-02 15:04:05"),
		})
	}

	for _, d := range details {
		b.WriteString(detailLabelStyle.Render(d.label+":") + " ")
		if d.label == "Status" {
			b.WriteString(StatusIcon(d.value) + " " + d.value)
		} else {
			b.WriteString(detailValueStyle.Render(d.value))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderDocDetail() string {
	var b strings.Builder

	cursor := m.cursors[TabDocs]
	if cursor >= len(m.documents) {
		return statusMutedStyle.Render("No document selected")
	}

	d := m.documents[cursor]

	// Title
	b.WriteString(detailTitleStyle.Render("📄 " + d.RelativePath))
	b.WriteString("\n\n")

	// Details
	details := []struct{ label, value string }{
		{"Type", string(d.Type)},
		{"Status", string(d.Status)},
		{"Size", fmt.Sprintf("%d bytes", d.Size)},
	}

	for _, detail := range details {
		b.WriteString(detailLabelStyle.Render(detail.label+":") + " ")
		if detail.label == "Status" {
			b.WriteString(StatusIcon(detail.value) + " " + detail.value)
		} else {
			b.WriteString(detailValueStyle.Render(detail.value))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderConventionDetail() string {
	var b strings.Builder

	cursor := m.cursors[TabConventions]
	if cursor >= len(m.conventions) {
		return statusMutedStyle.Render("No convention selected")
	}

	c := m.conventions[cursor]

	// Title
	b.WriteString(detailTitleStyle.Render("📋 " + c.Name))
	b.WriteString("\n\n")

	// Details
	status := "disabled"
	if c.Enabled {
		status = "enabled"
	}

	details := []struct{ label, value string }{
		{"Status", status},
		{"Rules", fmt.Sprintf("%d rules", len(c.Rules))},
	}

	if c.Description != "" {
		details = append(details, struct{ label, value string }{
			"Description", c.Description,
		})
	}

	for _, d := range details {
		b.WriteString(detailLabelStyle.Render(d.label+":") + " ")
		if d.label == "Status" {
			b.WriteString(StatusIcon(d.value) + " " + d.value)
		} else {
			b.WriteString(detailValueStyle.Render(d.value))
		}
		b.WriteString("\n")
	}

	// Rules list
	if len(c.Rules) > 0 {
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render("Rules:"))
		b.WriteString("\n")
		for i, rule := range c.Rules {
			if i >= 5 {
				b.WriteString(statusMutedStyle.Render(fmt.Sprintf("  ... and %d more", len(c.Rules)-5)))
				break
			}
			b.WriteString(fmt.Sprintf("  • %s\n", rule.ID))
		}
	}

	return b.String()
}

func (m Model) renderDetailPanel() string {
	switch m.currentTab {
	case TabSessions:
		return m.renderSessionDetail()
	case TabWorkflows:
		return m.renderWorkflowDetail()
	case TabDocs:
		return m.renderDocDetail()
	case TabConventions:
		return m.renderConventionDetail()
	default:
		return ""
	}
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
