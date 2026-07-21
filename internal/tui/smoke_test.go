package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

func TestDashboardRenders(t *testing.T) {
	tmp := t.TempDir()
	reg := registry.NewStore(tmp + "/registry.json")

	m := NewDashboard(tmp+"/clients", reg)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updated.(Model).View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("dashboard view is empty")
	}
}

func TestWizardRenders(t *testing.T) {
	m := NewWizard()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	view := updated.(WizardModel).View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("wizard view is empty")
	}
}

func TestDashboardProgramStarts(t *testing.T) {
	if os.Getenv("INNOVATEX_TUI_INTERACTIVE") == "" {
		t.Skip("set INNOVATEX_TUI_INTERACTIVE=1 to run interactive dashboard test")
	}

	tmp := t.TempDir()
	reg := registry.NewStore(tmp + "/registry.json")
	p := tea.NewProgram(
		NewDashboard(tmp+"/clients", reg),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		t.Fatalf("dashboard program failed: %v", err)
	}
}
