package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

const (
	appVersion     = "1.0.0"
	pollInterval   = 5 * time.Second
	headerHeight   = 3
	footerHeight   = 3
	minTableWidth  = 48
	detailMinWidth = 44
)

// ─── View Types ───────────────────────────────────────────────────────────────

type viewType int

const (
	listView    viewType = iota
	detailView           // client detail + container status
	logsView             // log stream
	confirmView          // delete confirm dialog
)

// ─── Confirm Actions ──────────────────────────────────────────────────────────

type confirmAction int

const (
	confirmDelete confirmAction = iota
	confirmStop
)

// ─── Model ────────────────────────────────────────────────────────────────────

type Model struct {
	// Config
	workDir string
	reg     *registry.Store

	// Data
	clients      []*registry.Client
	dockerStatus map[string]ClientStatus

	// Active view
	activeView viewType

	// Components
	table    table.Model
	viewport viewport.Model
	spinner  spinner.Model

	// State
	selected      *registry.Client
	confirmAct    confirmAction
	notification  string
	notifyIsError bool
	notifyExpiry  time.Time
	loading       bool
	logContent    string

	// Terminal dimensions
	width  int
	height int
}

// NewDashboard creates a new dashboard model connected to the registry.
func NewDashboard(workDir string, reg *registry.Store) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	m := Model{
		workDir:      workDir,
		reg:          reg,
		dockerStatus: make(map[string]ClientStatus),
		spinner:      sp,
		activeView:   listView,
	}
	return m
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadClients(),
		m.spinner.Tick,
		tickCmd(),
	)
}

// loadClients reads from the registry and sets up the table.
func (m Model) loadClients() tea.Cmd {
	return func() tea.Msg {
		clients, err := m.reg.List()
		if err != nil {
			return actionDoneMsg{err: err, action: "load"}
		}
		ids := make([]string, len(clients))
		for i, c := range clients {
			ids[i] = c.ID
		}
		return struct {
			clients []*registry.Client
			ids     []string
		}{clients, ids}
	}
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ── Terminal resize ──────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildTable()
		m.rebuildViewport()
		return m, nil

	// ── Clients loaded from registry ─────────────────────────────────────────
	case struct {
		clients []*registry.Client
		ids     []string
	}:
		m.clients = msg.clients
		m.rebuildTable()
		if len(msg.ids) > 0 {
			cmds = append(cmds, refreshStatusCmd(msg.ids))
		}
		return m, tea.Batch(cmds...)

	// ── Docker status poll result ─────────────────────────────────────────────
	case statusRefreshedMsg:
		m.dockerStatus = msg.statuses
		m.loading = false
		m.rebuildTable()
		return m, nil

	// ── Periodic tick → trigger refresh ──────────────────────────────────────
	case tickMsg:
		ids := m.clientIDs()
		cmds = append(cmds, tickCmd())
		if len(ids) > 0 {
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(ids))
		}
		return m, tea.Batch(cmds...)

	// ── Spinner tick ──────────────────────────────────────────────────────────
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// ── Async action result ───────────────────────────────────────────────────
	case actionDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.notify(fmt.Sprintf("✗ %s failed: %v", msg.action, msg.err), true)
		} else {
			switch msg.action {
			case "start":
				m.notify(fmt.Sprintf("✓ %s started", msg.clientID), false)
			case "stop":
				m.notify(fmt.Sprintf("✓ %s stopped", msg.clientID), false)
			case "delete":
				m.notify(fmt.Sprintf("✓ %s deleted", msg.clientID), false)
				m.removeClientFromList(msg.clientID)
				m.activeView = listView
				m.selected = nil
			}
		}
		// Refresh after action
		ids := m.clientIDs()
		if len(ids) > 0 {
			cmds = append(cmds, refreshStatusCmd(ids))
		}
		return m, tea.Batch(cmds...)

	// ── Log content received ──────────────────────────────────────────────────
	case logLineMsg:
		m.logContent = msg.line
		m.rebuildViewport()
		m.viewport.GotoBottom()
		return m, nil

	// ── Keyboard input ────────────────────────────────────────────────────────
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// ── Delegate to active component ──────────────────────────────────────────
	switch m.activeView {
	case listView:
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	case logsView, detailView:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKey routes keyboard events for each view.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Global quit
	if key.Matches(msg, keys.Quit) {
		return m, tea.Quit
	}

	switch m.activeView {

	// ─── List View ────────────────────────────────────────────────────────────
	case listView:
		switch {
		case key.Matches(msg, keys.Up), key.Matches(msg, keys.Down):
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)
			m.syncSelected()

		case key.Matches(msg, keys.Enter):
			m.syncSelected()
			if m.selected != nil {
				m.activeView = detailView
				m.rebuildViewport()
			}

		case key.Matches(msg, keys.Toggle):
			m.syncSelected()
			if m.selected != nil {
				cs := m.dockerStatus[m.selected.ID]
				if cs.AllStopped() || len(cs.Containers) == 0 {
					m.loading = true
					m.notify(fmt.Sprintf("Starting %s...", m.selected.ID), false)
					cmds = append(cmds, startClientCmd(m.workDir, m.selected.ID))
				} else {
					m.activeView = confirmView
					m.confirmAct = confirmStop
				}
			}

		case key.Matches(msg, keys.Delete):
			m.syncSelected()
			if m.selected != nil {
				m.activeView = confirmView
				m.confirmAct = confirmDelete
			}

		case key.Matches(msg, keys.Logs):
			m.syncSelected()
			if m.selected != nil {
				m.activeView = logsView
				m.logContent = "Fetching logs...\n"
				m.rebuildViewport()
				cmds = append(cmds, streamLogsCmd(nil, m.workDir, m.selected.ID, ""))
			}

		case key.Matches(msg, keys.Refresh):
			ids := m.clientIDs()
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(ids))
		}

	// ─── Detail View ──────────────────────────────────────────────────────────
	case detailView:
		switch {
		case key.Matches(msg, keys.Back):
			m.activeView = listView

		case key.Matches(msg, keys.Toggle):
			if m.selected != nil {
				cs := m.dockerStatus[m.selected.ID]
				if cs.AllStopped() || len(cs.Containers) == 0 {
					m.loading = true
					m.notify(fmt.Sprintf("Starting %s...", m.selected.ID), false)
					cmds = append(cmds, startClientCmd(m.workDir, m.selected.ID))
				} else {
					m.activeView = confirmView
					m.confirmAct = confirmStop
				}
			}

		case key.Matches(msg, keys.Delete):
			if m.selected != nil {
				m.activeView = confirmView
				m.confirmAct = confirmDelete
			}

		case key.Matches(msg, keys.Logs):
			if m.selected != nil {
				m.activeView = logsView
				m.logContent = "Fetching logs...\n"
				m.rebuildViewport()
				cmds = append(cmds, streamLogsCmd(nil, m.workDir, m.selected.ID, ""))
			}

		case key.Matches(msg, keys.Refresh):
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(m.clientIDs()))

		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	// ─── Logs View ────────────────────────────────────────────────────────────
	case logsView:
		switch {
		case key.Matches(msg, keys.Back):
			m.activeView = listView
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	// ─── Confirm Dialog ───────────────────────────────────────────────────────
	case confirmView:
		switch {
		case key.Matches(msg, keys.Confirm):
			if m.selected != nil {
				switch m.confirmAct {
				case confirmDelete:
					m.loading = true
					m.activeView = listView
					cmds = append(cmds, deleteClientCmd(m.workDir, m.selected.ID))
					// Also remove from registry
					m.reg.Delete(m.selected.ID)
				case confirmStop:
					m.loading = true
					m.activeView = listView
					m.notify(fmt.Sprintf("Stopping %s...", m.selected.ID), false)
					cmds = append(cmds, stopClientCmd(m.workDir, m.selected.ID))
				}
			}
		case key.Matches(msg, keys.Cancel):
			m.activeView = listView
		}
	}

	return m, tea.Batch(cmds...)
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	body := m.renderBody()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		footer,
	)
}

// ─── Render Sections ──────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	title := headerTitleStyle.Render("⚡ InnovateX Dashboard")
	version := headerVersionStyle.Render("v" + appVersion)

	spinner := ""
	if m.loading {
		spinner = "  " + m.spinner.View()
	}

	left := title + spinner
	right := version + headerVersionStyle.Render("  [q]uit")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 0
	}

	row := headerStyle.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)

	// Notification bar
	if m.notification != "" && time.Now().Before(m.notifyExpiry) {
		var notif string
		if m.notifyIsError {
			notif = notifyErrorStyle.Width(m.width).Render(m.notification)
		} else {
			notif = notifySuccessStyle.Width(m.width).Render(m.notification)
		}
		return lipgloss.JoinVertical(lipgloss.Left, row, notif)
	}

	return row
}

func (m Model) renderFooter() string {
	help := helpString(m.activeView, m.selected != nil)
	return footerStyle.Width(m.width).Render(help)
}

func (m Model) renderBody() string {
	bodyHeight := m.height - headerHeight - footerHeight
	if m.notification != "" && time.Now().Before(m.notifyExpiry) {
		bodyHeight--
	}
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	switch m.activeView {
	case listView, detailView:
		return m.renderSplitView(bodyHeight)
	case logsView:
		return m.renderLogsView(bodyHeight)
	case confirmView:
		return m.renderConfirmView(bodyHeight)
	}
	return ""
}

func (m Model) renderSplitView(height int) string {
	// Divide width: 45% list | 55% detail
	tableW := m.width * 45 / 100
	if tableW < minTableWidth {
		tableW = minTableWidth
	}
	detailW := m.width - tableW - 2
	if detailW < detailMinWidth {
		detailW = detailMinWidth
	}

	leftPanel := m.renderClientList(tableW, height)
	rightPanel := m.renderDetailPanel(detailW, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (m Model) renderClientList(width, height int) string {
	title := panelTitleStyle.Render("  CLIENTS")

	if len(m.clients) == 0 {
		empty := emptyStyle.Width(width - 4).Height(height - 4).Render(
			"No clients provisioned yet.\n\nRun:\n  innovatex create",
		)
		return panelFocusedStyle.Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, empty),
		)
	}

	tbl := m.table.View()
	content := lipgloss.JoinVertical(lipgloss.Left, title, tbl)

	style := panelFocusedStyle
	if m.activeView == detailView {
		style = panelStyle
	}

	return style.Width(width).Height(height).Render(content)
}

func (m Model) renderDetailPanel(width, height int) string {
	if m.selected == nil {
		hint := emptyStyle.Width(width - 4).Height(height - 4).Render(
			"Select a client with ↑↓\nthen press enter",
		)
		return panelStyle.Width(width).Height(height).Render(hint)
	}

	c := m.selected
	cs := m.dockerStatus[c.ID]

	domainDisplay := c.Domain
	if c.Domain == "localhost" || strings.HasSuffix(c.Domain, ".local") {
		domainDisplay = "localhost"
	}

	// ── Header ──
	title := panelTitleStyle.Render("  " + c.ID)
	brand := fieldLabelStyle.Render("Brand") + fieldValueStyle.Render(c.BrandName)
	created := fieldLabelStyle.Render("Created") + fieldValueStyle.Render(c.CreatedAt.Format("2006-01-02 15:04"))

	// ── URLs ──
	sectionURLs := fieldLabelStyle.Foreground(colorAccent).Bold(true).Render("\n  URLs")
	sfURL := fieldLabelStyle.Render("  Storefront") + fieldURLStyle.Render(fmt.Sprintf("http://%s:%d", domainDisplay, c.StorefrontPort))
	apiURL := fieldLabelStyle.Render("  CommerceX") + fieldURLStyle.Render(fmt.Sprintf("http://%s:%d", domainDisplay, c.AppPort))
	pgURL := fieldLabelStyle.Render("  PostgreSQL") + fieldValueStyle.Render(fmt.Sprintf("%s:%d", domainDisplay, c.PostgresPort))

	// ── Credentials ──
	sectionCreds := fieldLabelStyle.Foreground(colorAccent).Bold(true).Render("\n  Credentials")
	adminUser := fieldLabelStyle.Render("  Admin") + fieldValueStyle.Render(c.AdminUsername+" / "+c.AdminPassword)
	dbName := fieldLabelStyle.Render("  Database") + fieldValueStyle.Render(c.DBName)
	dbUser := fieldLabelStyle.Render("  DB User") + fieldValueStyle.Render(c.DBUsername)

	// ── Container Status ──
	sectionContainers := fieldLabelStyle.Foreground(colorAccent).Bold(true).Render("\n  Containers")

	services := []struct{ key, label string }{
		{"commercex-server", "  server "},
		{"commercex-worker", "  worker "},
		{"postgres", "  postgres"},
		{"storefront", "  storefront"},
	}

	containerLines := []string{}
	for _, svc := range services {
		state := "not found"
		if len(cs.Containers) > 0 {
			state = cs.ServiceState(svc.key)
		}
		label := fieldLabelStyle.Render(svc.label)
		badge := statusBadge(state)
		containerLines = append(containerLines, label+badge)
	}

	rows := []string{
		title,
		brand,
		created,
		sectionURLs,
		sfURL,
		apiURL,
		pgURL,
		sectionCreds,
		adminUser,
		dbName,
		dbUser,
		sectionContainers,
	}
	rows = append(rows, containerLines...)

	content := strings.Join(rows, "\n")

	style := panelStyle
	if m.activeView == detailView {
		style = panelFocusedStyle
	}
	return style.Width(width).Height(height).Render(content)
}

func (m Model) renderLogsView(height int) string {
	title := ""
	if m.selected != nil {
		title = logsHeaderStyle.Width(m.width).Render(
			fmt.Sprintf("  Logs — %s  [esc] back", m.selected.ID),
		)
	}

	logArea := logsStyle.Width(m.width).Height(height - 2).Render(m.viewport.View())
	return lipgloss.JoinVertical(lipgloss.Left, title, logArea)
}

func (m Model) renderConfirmView(height int) string {
	if m.selected == nil {
		return ""
	}

	var title, body string
	switch m.confirmAct {
	case confirmDelete:
		title = dialogTitleStyle.Render("⚠  Delete Client")
		body = fmt.Sprintf(
			"This will permanently remove:\n\n"+
				"  • All containers for  %s\n"+
				"  • Work directory files\n"+
				"  • Registry entry\n\n"+
				"  Client ID:  %s\n\n"+
				"Continue? [y] yes  [n/esc] cancel",
			m.selected.ID, m.selected.ID,
		)
	case confirmStop:
		title = dialogTitleStyle.Render("Stop Client")
		body = fmt.Sprintf(
			"Stop all containers for  %s?\n\n"+
				"[y] yes  [n/esc] cancel",
			m.selected.ID,
		)
	}

	dialog := dialogStyle.Render(title + "\n\n" + body)

	// Center the dialog
	dw := lipgloss.Width(dialog)
	dh := lipgloss.Height(dialog)
	padLeft := (m.width - dw) / 2
	padTop := (height - dh) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	top := strings.Repeat("\n", padTop)
	left := strings.Repeat(" ", padLeft)
	return top + left + dialog
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// rebuildTable reconstructs the table model from current client + status data.
func (m *Model) rebuildTable() {
	tableW := m.width * 45 / 100
	if tableW < minTableWidth {
		tableW = minTableWidth
	}
	// Column widths
	idW := 16
	statusW := 14
	portsW := tableW - idW - statusW - 6
	if portsW < 10 {
		portsW = 10
	}

	cols := []table.Column{
		{Title: "ID", Width: idW},
		{Title: "Status", Width: statusW},
		{Title: "Ports", Width: portsW},
	}

	rows := []table.Row{}
	for _, c := range m.clients {
		cs := m.dockerStatus[c.ID]
		status := overallStatus(cs)
		statusCell := statusBadge(status)
		ports := fmt.Sprintf("%d/%d/%d", c.AppPort, c.StorefrontPort, c.PostgresPort)
		rows = append(rows, table.Row{c.ID, statusCell, ports})
	}

	tableHeight := m.height - headerHeight - footerHeight - 4
	if tableHeight < 1 {
		tableHeight = 1
	}

	ts := table.DefaultStyles()
	ts.Header = tableHeaderStyle
	ts.Selected = tableSelectedStyle

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
		table.WithStyles(ts),
	)

	// Preserve cursor position
	oldCursor := 0
	if m.table.Cursor() < len(rows) {
		oldCursor = m.table.Cursor()
	}
	for i := 0; i < oldCursor; i++ {
		t.MoveDown(1)
	}

	m.table = t
	m.syncSelected()
}

// rebuildViewport updates the viewport content based on active view.
func (m *Model) rebuildViewport() {
	vpW := m.width
	vpH := m.height - headerHeight - footerHeight - 2

	if m.activeView == logsView {
		m.viewport = viewport.New(vpW, vpH)
		m.viewport.SetContent(m.logContent)
		m.viewport.GotoBottom()
	} else if m.activeView == detailView {
		// Detail panel viewport isn't used as scrollable directly — detail is static text
		m.viewport = viewport.New(vpW, vpH)
	}
}

// syncSelected updates m.selected based on the current table cursor.
func (m *Model) syncSelected() {
	cursor := m.table.Cursor()
	if cursor >= 0 && cursor < len(m.clients) {
		m.selected = m.clients[cursor]
	} else {
		m.selected = nil
	}
}

// clientIDs returns a slice of all client IDs.
func (m *Model) clientIDs() []string {
	ids := make([]string, len(m.clients))
	for i, c := range m.clients {
		ids[i] = c.ID
	}
	return ids
}

// removeClientFromList removes a client from the in-memory list.
func (m *Model) removeClientFromList(clientID string) {
	newList := make([]*registry.Client, 0, len(m.clients))
	for _, c := range m.clients {
		if c.ID != clientID {
			newList = append(newList, c)
		}
	}
	m.clients = newList
	m.rebuildTable()
}

// notify sets a timed notification message.
func (m *Model) notify(msg string, isError bool) {
	m.notification = msg
	m.notifyIsError = isError
	m.notifyExpiry = time.Now().Add(4 * time.Second)
}
