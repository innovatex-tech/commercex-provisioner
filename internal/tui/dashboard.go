package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

const (
	appVersion   = "1.0.0"
	pollInterval = 5 * time.Second
)

type viewType int

const (
	listView    viewType = iota
	detailView
	logsView
	confirmView
)

type confirmAction int

const (
	confirmDelete confirmAction = iota
	confirmStop
)

type Model struct {
	workDir string
	reg     *registry.Store

	clients      []*registry.Client
	dockerStatus map[string]ClientStatus

	activeView viewType
	cursor     int
	viewport   viewport.Model
	spinner    spinner.Model

	selected      *registry.Client
	confirmAct    confirmAction
	notification  string
	notifyIsError bool
	notifyExpiry  time.Time
	loading       bool
	logContent    string

	width  int
	height int
}

func NewDashboard(workDir string, reg *registry.Store) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = spinnerStyle

	return Model{
		workDir:      workDir,
		reg:          reg,
		dockerStatus: make(map[string]ClientStatus),
		spinner:      sp,
		activeView:   listView,
	}
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadClients(),
		m.spinner.Tick,
		tickCmd(),
	)
}

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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncCursor()
		return m, nil

	case struct {
		clients []*registry.Client
		ids     []string
	}:
		m.clients = msg.clients
		m.syncCursor()
		if len(msg.clients) > 0 {
			cmds = append(cmds, refreshStatusCmd(msg.clients))
		}
		return m, tea.Batch(cmds...)

	case statusRefreshedMsg:
		m.dockerStatus = msg.statuses
		m.loading = false
		return m, nil

	case tickMsg:
		cmds = append(cmds, tickCmd())
		if len(m.clients) > 0 {
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(m.clients))
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case actionDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.notify(fmt.Sprintf("  %s failed: %v", msg.action, msg.err), true)
		} else {
			switch msg.action {
			case "start":
				m.notify(fmt.Sprintf("  %s started", msg.clientID), false)
			case "stop":
				m.notify(fmt.Sprintf("  %s stopped", msg.clientID), false)
			case "delete":
				m.notify(fmt.Sprintf("  %s deleted", msg.clientID), false)
				m.removeClientFromList(msg.clientID)
				m.activeView = listView
				m.selected = nil
			}
		}
		if len(m.clients) > 0 {
			cmds = append(cmds, refreshStatusCmd(m.clients))
		}
		return m, tea.Batch(cmds...)

	case logLineMsg:
		m.logContent = msg.line
		m.rebuildViewport()
		m.viewport.GotoBottom()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward non-key events to viewport (for logs scrolling)
	switch m.activeView {
	case logsView, detailView:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── Key Handling ─────────────────────────────────────────────────────────────

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if key.Matches(msg, keys.Quit) {
		return m, tea.Quit
	}

	switch m.activeView {

	case listView:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			m.syncSelected()

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.clients)-1 {
				m.cursor++
			}
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
				if cs.AllStopped() {
					m.loading = true
					m.notify(fmt.Sprintf("Starting %s...", m.selected.ID), false)
					cmds = append(cmds, startClientCmd(m.workDir, m.selected) )
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
				m.logContent = ""
				m.rebuildViewport()
				cmds = append(cmds, streamLogsCmd(m.workDir, m.selected, ""))
			}

		case key.Matches(msg, keys.Refresh):
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(m.clients))
		}

	case detailView:
		switch {
		case key.Matches(msg, keys.Back):
			m.activeView = listView

		case key.Matches(msg, keys.Toggle):
			if m.selected != nil {
				cs := m.dockerStatus[m.selected.ID]
				if cs.AllStopped() {
					m.loading = true
					m.notify(fmt.Sprintf("Starting %s...", m.selected.ID), false)
					cmds = append(cmds, startClientCmd(m.workDir, m.selected))
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
				m.logContent = ""
				m.rebuildViewport()
				cmds = append(cmds, streamLogsCmd(m.workDir, m.selected, ""))
			}

		case key.Matches(msg, keys.Refresh):
			m.loading = true
			cmds = append(cmds, refreshStatusCmd(m.clients))

		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case logsView:
		switch {
		case key.Matches(msg, keys.Back):
			m.activeView = listView
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case confirmView:
		switch {
		case key.Matches(msg, keys.Confirm):
			if m.selected != nil {
				switch m.confirmAct {
				case confirmDelete:
					m.loading = true
					m.activeView = listView
					cmds = append(cmds, deleteClientCmd(m.workDir, m.selected))
					m.reg.Delete(m.selected.ID)
				case confirmStop:
					m.loading = true
					m.activeView = listView
					m.notify(fmt.Sprintf("Stopping %s...", m.selected.ID), false)
					cmds = append(cmds, stopClientCmd(m.workDir, m.selected))
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
		return ""
	}

	if m.width < minWidth || m.height < minHeight {
		return tooSmallMessage(m.width, m.height)
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

// ─── Header ───────────────────────────────────────────────────────────────────

func (m Model) renderHeader() string {
	title := headerStyle.Render(" InnovateX ")

	sp := ""
	if m.loading {
		sp = " " + m.spinner.View()
	}

	left := title + sp
	right := headerVersionStyle.Render("v" + appVersion)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		left,
		strings.Repeat(" ", gap),
		right,
	)

	return lipgloss.NewStyle().
		Width(m.width).
		Background(bgBase).
		Render(row)
}

// ─── Footer ───────────────────────────────────────────────────────────────────

func (m Model) renderFooter() string {
	help := helpString(m.activeView, m.selected != nil)
	sep := hLine(m.width)

	content := footerStyle.Width(m.width).Render("  " + help)
	return lipgloss.JoinVertical(lipgloss.Left, sep, content)
}

// ─── Status Bar ───────────────────────────────────────────────────────────────

func (m Model) renderStatusBar() string {
	if m.notification != "" && time.Now().Before(m.notifyExpiry) {
		if m.notifyIsError {
			return statusBarErrStyle.Width(m.width).Render(m.notification)
		}
		return statusBarOKStyle.Width(m.width).Render(m.notification)
	}
	return statusBarStyle.Width(m.width).Render("")
}

// ─── Body ─────────────────────────────────────────────────────────────────────

func (m Model) renderBody() string {
	bodyH := m.height - 5
	if bodyH < 1 {
		bodyH = 1
	}

	switch m.activeView {
	case listView, detailView:
		return m.renderSplitView(bodyH)
	case logsView:
		return m.renderLogsView(bodyH)
	case confirmView:
		return m.renderConfirmView(bodyH)
	}
	return ""
}

// ─── Split View ───────────────────────────────────────────────────────────────

func (m Model) renderSplitView(height int) string {
	tableW := m.width * 38 / 100
	if tableW < 42 {
		tableW = 42
	}
	detailW := m.width - tableW - 1
	if detailW < 40 {
		detailW = 40
	}

	left := m.renderClientList(tableW, height)
	right := m.renderDetailPanel(detailW, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// ─── Client List Panel ────────────────────────────────────────────────────────

func (m Model) renderClientList(width, height int) string {
	title := panelTitleStyle.Render("  CLIENTS")

	if len(m.clients) == 0 {
		msg := emptyStyle.Width(width - 4).Height(height - 4).Render(
			"No clients yet\n\ninnovatex create",
		)
		return panelStyle.Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, msg),
		)
	}

	// Manual table: no bubbles table, no viewport, just cursor math
	idW := 16
	statusW := 12
	portsW := width - idW - statusW - 10
	if portsW < 12 {
		portsW = 12
	}

	// Header row
	headerRow := tableHeaderStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(idW).Render("ID"),
			lipgloss.NewStyle().Width(statusW).Render("Status"),
			lipgloss.NewStyle().Width(portsW).Render("Ports"),
		),
	)

	// Visible rows: height minus title line, header, and borders
	visibleH := height - 4
	if visibleH < 1 {
		visibleH = 1
	}

	// Scroll window: keep cursor visible
	start := 0
	if len(m.clients) > visibleH {
		start = m.cursor - visibleH/2
		if start < 0 {
			start = 0
		}
		if start+visibleH > len(m.clients) {
			start = len(m.clients) - visibleH
		}
		if start < 0 {
			start = 0
		}
	}

	end := start + visibleH
	if end > len(m.clients) {
		end = len(m.clients)
	}

	var rows []string
	for i := start; i < end; i++ {
		c := m.clients[i]
		cs := m.dockerStatus[c.ID]
		statusCell := overallStatusBadge(cs)
		ports := fmt.Sprintf("%d/%d/%d", c.AppPort, c.StorefrontPort, c.PostgresPort)

		idCol := lipgloss.NewStyle().Width(idW).Render(c.ID)
		statusCol := lipgloss.NewStyle().Width(statusW).Render(statusCell)
		portsCol := lipgloss.NewStyle().Width(portsW).Render(ports)
		line := lipgloss.JoinHorizontal(lipgloss.Top, idCol, statusCol, portsCol)

		if i == m.cursor {
			rows = append(rows, tableSelectedStyle.Width(width-4).Render(line))
		} else {
			rows = append(rows, lipgloss.NewStyle().Width(width-4).Render(line))
		}
	}

	// Pad remaining space
	for len(rows) < visibleH {
		rows = append(rows, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{headerRow}, rows...)...)

	focused := m.activeView == listView
	var panel lipgloss.Style
	if focused {
		panel = panelFocusedStyle
	} else {
		panel = panelStyle
	}

	return panel.Width(width).Height(height).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, content),
	)
}

// ─── Detail Panel ─────────────────────────────────────────────────────────────

func (m Model) renderDetailPanel(width, height int) string {
	if m.selected == nil {
		msg := emptyStyle.Width(width - 4).Height(height - 4).Render(
			"Select a client\nthen press enter",
		)
		return panelStyle.Width(width).Height(height).Render(msg)
	}

	c := m.selected
	cs := m.dockerStatus[c.ID]

	domainDisplay := c.Domain
	if c.Domain == "localhost" || strings.HasSuffix(c.Domain, ".local") {
		domainDisplay = "localhost"
	}

	var lines []string

	lines = append(lines, panelTitleStyle.Render("  "+c.ID))
	lines = append(lines, "")

	lines = append(lines, fieldLabelStyle.Render("  Brand")+
		fieldValueStyle.Render(c.BrandName))
	lines = append(lines, fieldLabelStyle.Render("  Created")+
		fieldValueStyle.Render(c.CreatedAt.Format("Jan 02, 2006  15:04")))
	lines = append(lines, "")

	lines = append(lines, sectionHeaderStyle.Render("  URLS"))
	lines = append(lines, "")
	lines = append(lines, fieldLabelStyle.Render("  Storefront")+
		fieldURLStyle.Render(fmt.Sprintf("http://%s:%d", domainDisplay, c.StorefrontPort)))
	lines = append(lines, fieldLabelStyle.Render("  CommerceX")+
		fieldURLStyle.Render(fmt.Sprintf("http://%s:%d", domainDisplay, c.AppPort)))
	lines = append(lines, fieldLabelStyle.Render("  Database")+
		fieldValueStyle.Render(fmt.Sprintf("%s:%d", domainDisplay, c.PostgresPort)))
	lines = append(lines, "")

	lines = append(lines, sectionHeaderStyle.Render("  CREDENTIALS"))
	lines = append(lines, "")
	lines = append(lines, fieldLabelStyle.Render("  Admin User")+
		fieldValueStyle.Render(c.AdminUsername))
	lines = append(lines, fieldLabelStyle.Render("  Admin Pass")+
		fieldValueStyle.Render(c.AdminPassword))
	lines = append(lines, fieldLabelStyle.Render("  DB Name")+
		fieldValueStyle.Render(c.DBName))
	lines = append(lines, fieldLabelStyle.Render("  DB User")+
		fieldValueStyle.Render(c.DBUsername))
	lines = append(lines, "")

	lines = append(lines, sectionHeaderStyle.Render("  CONTAINERS"))
	lines = append(lines, "")

	services := []struct{ key, label string }{
		{"commercex-server", "  Server"},
		{"commercex-worker", "  Worker"},
		{"postgres", "  Postgres"},
		{"storefront", "  Storefront"},
	}

	for _, svc := range services {
		state := "not found"
		if len(cs.Containers) > 0 {
			state = cs.ServiceState(svc.key)
		}
		lines = append(lines, containerLabelStyle.Render(svc.label)+statusBadge(state))
	}

	content := strings.Join(lines, "\n")

	focused := m.activeView == detailView
	var panel lipgloss.Style
	if focused {
		panel = panelFocusedStyle
	} else {
		panel = panelStyle
	}

	return panel.Width(width).Height(height).Render(content)
}

// ─── Logs View ────────────────────────────────────────────────────────────────

func (m Model) renderLogsView(height int) string {
	title := ""
	if m.selected != nil {
		title = logsHeaderStyle.Width(m.width).Render(
			fmt.Sprintf("  Logs  —  %s     [esc] back", m.selected.ID),
		)
	}

	logArea := logsStyle.Width(m.width).Height(height - 1).Render(m.viewport.View())
	return lipgloss.JoinVertical(lipgloss.Left, title, logArea)
}

// ─── Confirm Dialog ───────────────────────────────────────────────────────────

func (m Model) renderConfirmView(height int) string {
	if m.selected == nil {
		return ""
	}

	var title, body string
	switch m.confirmAct {
	case confirmDelete:
		title = dialogTitleStyle.Render("  Delete Client")
		body = fmt.Sprintf(
			"This will permanently remove:\n\n"+
				"  All containers for  %s\n"+
				"  Work directory files\n"+
				"  Registry entry\n\n"+
				"  Client ID:  %s\n\n"+
				"  [y] yes       [n] cancel",
			m.selected.ID, m.selected.ID,
		)
	case confirmStop:
		title = dialogTitleStyle.Render("  Stop Client")
		body = fmt.Sprintf(
			"Stop all containers for\n\n"+
				"  %s\n\n"+
				"  [y] yes       [n] cancel",
			m.selected.ID,
		)
	}

	dialog := dialogStyle.Render(title + "\n\n" + body)

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

	return strings.Repeat("\n", padTop) + strings.Repeat(" ", padLeft) + dialog
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (m *Model) rebuildViewport() {
	vpW := m.width - 2
	vpH := m.height - 6

	if m.activeView == logsView {
		m.viewport = viewport.New(vpW, vpH)
		m.viewport.SetContent(m.logContent)
		m.viewport.GotoBottom()
	} else if m.activeView == detailView {
		m.viewport = viewport.New(vpW, vpH)
	}
}

func (m *Model) syncCursor() {
	if m.cursor < 0 {
		m.cursor = 0
	}
	if len(m.clients) > 0 && m.cursor >= len(m.clients) {
		m.cursor = len(m.clients) - 1
	}
	m.syncSelected()
}

func (m *Model) syncSelected() {
	if m.cursor >= 0 && m.cursor < len(m.clients) {
		m.selected = m.clients[m.cursor]
	} else {
		m.selected = nil
	}
}

func (m *Model) clientIDs() []string {
	ids := make([]string, len(m.clients))
	for i, c := range m.clients {
		ids[i] = c.ID
	}
	return ids
}

func (m *Model) removeClientFromList(clientID string) {
	newList := make([]*registry.Client, 0, len(m.clients))
	for _, c := range m.clients {
		if c.ID != clientID {
			newList = append(newList, c)
		}
	}
	m.clients = newList
	m.syncCursor()
}

func (m *Model) notify(msg string, isError bool) {
	m.notification = msg
	m.notifyIsError = isError
	m.notifyExpiry = time.Now().Add(4 * time.Second)
}
