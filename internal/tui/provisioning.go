package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/innovatex-tech/commercex-provisioner/internal/core"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

type ProvisioningModel struct {
	progress    progress.Model
	provisioner *core.Provisioner
	request     core.CreateRequest
	
	currentStep string
	percent     float64
	err         error
	done        bool
	client      *registry.Client

	updates chan core.ProgressStep
	width   int
	height  int
}

type StepMsg core.ProgressStep

type DoneMsg struct {
	Client *registry.Client
	Err    error
}

func NewProvisioningModel(p *core.Provisioner, req core.CreateRequest) ProvisioningModel {
	pg := progress.New(
		progress.WithGradient(string(colorAccent), string(colorPrimary)),
		progress.WithoutPercentage(),
		progress.WithWidth(progressBarWidth),
	)

	return ProvisioningModel{
		progress:    pg,
		provisioner: p,
		request:     req,
		currentStep: "Initializing...",
		updates:     make(chan core.ProgressStep, 10),
	}
}

func (m ProvisioningModel) Init() tea.Cmd {
	return tea.Batch(
		m.startProvisioning(),
		m.waitForUpdate(),
	)
}

func (m ProvisioningModel) startProvisioning() tea.Cmd {
	return func() tea.Msg {
		client, err := m.provisioner.Create(&m.request, func(step core.ProgressStep) {
			m.updates <- step
		})
		return DoneMsg{Client: client, Err: err}
	}
}

func (m ProvisioningModel) waitForUpdate() tea.Cmd {
	return func() tea.Msg {
		return StepMsg(<-m.updates)
	}
}

func (m ProvisioningModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case StepMsg:
		m.currentStep = msg.Step
		m.percent = msg.Percent
		cmd := m.progress.SetPercent(msg.Percent)
		return m, tea.Batch(cmd, m.waitForUpdate())

	case DoneMsg:
		m.done = true
		m.client = msg.Client
		m.err = msg.Err
		return m, nil

	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		m.progress = newModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		if m.done || m.err != nil {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ProvisioningModel) View() string {
	if m.width == 0 { return "" }

	var s strings.Builder

	if m.err != nil {
		s.WriteString(notifyErrorStyle.Render("PROVISIONING FAILED") + "\n\n")
		s.WriteString(dimStyle.Render(m.err.Error()) + "\n\n")
		s.WriteString(footerStyle.Render("Press any key to exit"))
	} else if m.done {
		s.WriteString(boldStyle.Foreground(colorSuccess).Render("✅ DEPLOYMENT SUCCESSFUL") + "\n\n")
		
		target := m.client.Domain
		if m.client.IsRemote {
			target = m.client.ServerHost
		}

		s.WriteString(containerStyle.Render(
			fmt.Sprintf("%s %s\n%s %s\n\n%s %s\n%s %s",
				labelStyle.Render("Client ID:"), m.client.ID,
				labelStyle.Render("Domain:   "), m.client.Domain,
				labelStyle.Render("Store:    "), lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("http://%s:%d", target, m.client.StorefrontPort)),
				labelStyle.Render("Admin:    "), lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("http://%s:%d/admin", target, m.client.AppPort)),
			),
		) + "\n\n")

		s.WriteString(boldStyle.Foreground(colorAccent).Render("💡 Next Step: Run 'innovatex dashboard'") + "\n")
		s.WriteString(footerStyle.Render("Press any key to exit"))
	} else {
		s.WriteString(headerStyle.Render(" PROVISIONING ") + " " + m.request.ClientID + "\n\n")
		s.WriteString(stepTextStyle.Render(m.currentStep) + "\n")
		s.WriteString(m.progress.View() + "\n\n")
		s.WriteString(footerStyle.Render("Please wait, configuring your commerce engine..."))
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s.String())
}
