package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/innovatex-tech/commercex-provisioner/internal/core"
)

type wizardStep int

const (
	stepClient wizardStep = iota
	stepDatabase
	stepAdmin
	stepRemote
	stepReview
)

type WizardModel struct {
	step      wizardStep
	inputs    []textinput.Model
	quitting  bool
	confirmed bool

	width  int
	height int
}

func NewWizard() WizardModel {
	m := WizardModel{
		step:   stepClient,
		inputs: make([]textinput.Model, 11),
	}

	for i := range m.inputs {
		t := textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 64

		switch i {
		case 0: t.Placeholder = "howlsan-store"
		case 1: t.Placeholder = "howlsan.com"
		case 2: t.Placeholder = "Howlsan"
		case 3: t.Placeholder = "howlsanDB"
		case 4: t.Placeholder = "howlsanUser"
		case 5: t.Placeholder = "••••••••"; t.EchoMode = textinput.EchoPassword; t.EchoCharacter = '•'
		case 6: t.Placeholder = "admin"
		case 7: t.Placeholder = "••••••••"; t.EchoMode = textinput.EchoPassword; t.EchoCharacter = '•'
		case 8: t.Placeholder = "root@45.147.46.225 (optional)"
		case 9: t.Placeholder = "••••••••"; t.EchoMode = textinput.EchoPassword; t.EchoCharacter = '•'
		case 10: t.Placeholder = "~/.ssh/id_rsa"
		}
		m.inputs[i] = t
	}

	m.inputs[0].Focus()
	return m
}

func (m WizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if m.step == stepReview {
				m.confirmed = true
				return m, tea.Quit
			}
			m.nextInput()
			return m, nil

		case "tab":
			m.nextInput()
			return m, nil

		case "shift+tab", "up":
			m.prevInput()
			return m, nil
		}
	}

	// Update active input
	var cmd tea.Cmd
	idx := m.activeInputIdx()
	if idx >= 0 && idx < len(m.inputs) {
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
	}

	return m, cmd
}

func (m *WizardModel) nextInput() {
	idx := m.activeInputIdx()
	val := strings.TrimSpace(m.inputs[idx].Value())

	// Required fields check (steps 1-3)
	if m.step < stepRemote && val == "" {
		return
	}

	m.inputs[idx].Blur()

	switch m.step {
	case stepClient:
		if idx < 2 {
			idx++
		} else {
			m.step = stepDatabase
			idx = 3
		}
	case stepDatabase:
		if idx < 5 {
			idx++
		} else {
			m.step = stepAdmin
			idx = 6
		}
	case stepAdmin:
		if idx < 7 {
			idx++
		} else {
			m.step = stepRemote
			idx = 8
		}
	case stepRemote:
		if idx < 10 {
			idx++
		} else {
			m.step = stepReview
			return
		}
	}

	m.inputs[idx].Focus()
}

func (m *WizardModel) prevInput() {
	idx := m.activeInputIdx()
	m.inputs[idx].Blur()

	switch m.step {
	case stepClient:
		if idx > 0 { idx-- }
	case stepDatabase:
		if idx > 3 { idx-- } else { m.step = stepClient; idx = 2 }
	case stepAdmin:
		if idx > 6 { idx-- } else { m.step = stepDatabase; idx = 5 }
	case stepRemote:
		if idx > 8 { idx-- } else { m.step = stepAdmin; idx = 7 }
	case stepReview:
		m.step = stepRemote
		idx = 10
	}

	m.inputs[idx].Focus()
}

func (m WizardModel) activeInputIdx() int {
	for i, t := range m.inputs {
		if t.Focused() {
			return i
		}
	}
	return 0
}

func (m WizardModel) View() string {
	if m.width == 0 { return "" }

	var s strings.Builder

	// Header
	s.WriteString(headerStyle.Render(" INNOVATEX ") + " PROVISIONER\n\n")

	var body string
	switch m.step {
	case stepClient:
		body = m.renderForm("1. CLIENT IDENTITY", 0, 2)
	case stepDatabase:
		body = m.renderForm("2. DATABASE CONFIG", 3, 5)
	case stepAdmin:
		body = m.renderForm("3. ADMIN ACCOUNT", 6, 7)
	case stepRemote:
		body = m.renderForm("4. REMOTE TARGET", 8, 10)
	case stepReview:
		body = m.renderReview()
	}

	s.WriteString(containerStyle.Render(body))
	s.WriteString("\n" + footerStyle.Render("tab: continue • esc: cancel • ctrl+c: quit"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s.String())
}

func (m WizardModel) renderForm(title string, start, end int) string {
	var s strings.Builder
	s.WriteString(boldStyle.Foreground(colorPrimary).Render(title) + "\n\n")

	labels := []string{
		"Client ID", "Domain", "Brand Name",
		"DB Name", "DB User", "DB Password",
		"Admin User", "Admin Password",
		"Remote Host", "SSH Password", "SSH Key Path",
	}

	for i := start; i <= end; i++ {
		label := labels[i]
		if m.inputs[i].Focused() {
			s.WriteString(stepIndicatorStyle.Render("▸") + labelStyle.Render(label) + "\n")
			s.WriteString("  " + m.inputs[i].View() + "\n\n")
		} else {
			val := m.inputs[i].Value()
			if val == "" {
				val = m.inputs[i].Placeholder
			} else if m.inputs[i].EchoMode == textinput.EchoPassword {
				val = "••••••••"
			}
			s.WriteString("  " + dimStyle.Render(label) + "\n")
			s.WriteString("  " + dimStyle.Render(val) + "\n\n")
		}
	}

	return s.String()
}

func (m WizardModel) renderReview() string {
	var s strings.Builder
	s.WriteString(boldStyle.Foreground(colorSuccess).Render("READY TO PROVISION") + "\n\n")
	
	s.WriteString(fmt.Sprintf("  %-12s %s\n", dimStyle.Render("Client"), boldStyle.Render(m.inputs[0].Value())))
	s.WriteString(fmt.Sprintf("  %-12s %s\n", dimStyle.Render("Domain"), m.inputs[1].Value()))
	
	server := m.inputs[8].Value()
	if server == "" {
		s.WriteString(fmt.Sprintf("  %-12s %s\n", dimStyle.Render("Target"), "Local Machine"))
	} else {
		s.WriteString(fmt.Sprintf("  %-12s %s\n", dimStyle.Render("Target"), lipgloss.NewStyle().Foreground(colorAccent).Render(server)))
	}

	s.WriteString("\n" + boldStyle.Foreground(colorAccent).Render("Press ENTER to begin deployment"))
	return s.String()
}

func (m WizardModel) GetRequest() (core.CreateRequest, bool) {
	if m.quitting || !m.confirmed {
		return core.CreateRequest{}, false
	}

	server := m.inputs[8].Value()
	var user, host string
	if server != "" {
		parts := strings.Split(server, "@")
		if len(parts) == 2 {
			user, host = parts[0], parts[1]
		} else {
			user, host = "root", server
		}
	}

	return core.CreateRequest{
		ClientID:      m.inputs[0].Value(),
		Domain:        m.inputs[1].Value(),
		BrandName:     m.inputs[2].Value(),
		DBName:        m.inputs[3].Value(),
		DBUsername:    m.inputs[4].Value(),
		DBPassword:    m.inputs[5].Value(),
		AdminUsername: m.inputs[6].Value(),
		AdminPassword: m.inputs[7].Value(),
		ServerHost:    host,
		ServerUser:    user,
		SSHPassword:   m.inputs[9].Value(),
		SSHKeyPath:    m.inputs[10].Value(),
	}, true
}
