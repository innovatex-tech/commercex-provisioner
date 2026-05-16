package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// ─── Palette (Claude/Modern inspired) ─────────────────────────────────────────

const (
	colorBg      = lipgloss.Color("#0F172A") // Slate 900
	colorSurface = lipgloss.Color("#1E293B") // Slate 800
	colorBorder  = lipgloss.Color("#334155") // Slate 700
	colorText    = lipgloss.Color("#F8FAFC") // Slate 50
	colorTextDim = lipgloss.Color("#64748B") // Slate 500
	colorPrimary = lipgloss.Color("#C084FC") // Purple 400
	colorAccent  = lipgloss.Color("#22D3EE") // Cyan 400
	colorSuccess = lipgloss.Color("#4ADE80") // Green 400
	colorDanger  = lipgloss.Color("#FB7185") // Rose 400
	colorWarning = lipgloss.Color("#FBBF24") // Amber 400
	colorMuted   = lipgloss.Color("#475569") // Slate 600
)

// ─── Shared Base Styles ───────────────────────────────────────────────────────

var (
	boldStyle = lipgloss.NewStyle().Bold(true)
	dimStyle  = lipgloss.NewStyle().Foreground(colorTextDim)

	// App Chrome
	headerStyle = lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorAccent).
			Padding(0, 1).
			Bold(true).
			MarginBottom(1)

	headerTitleStyle = lipgloss.NewStyle().
				Foreground(colorBg).
				Bold(true)

	headerVersionStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Faint(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 1).
			MarginTop(1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	// Notifications
	notifySuccessStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true).
				Padding(0, 1)

	notifyErrorStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true).
				Padding(0, 1)

	// Panels & Boxes
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorBorder).
			PaddingLeft(2)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(colorPrimary).
				PaddingLeft(2)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// Table Styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorBorder)

	tableSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText).
				Background(colorSurface)

	// Fields
	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Width(14)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(colorText)

	fieldURLStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Underline(true)

	// Status Badges
	statusRunningStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	statusStoppedStyle = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	statusBuildingStyle = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	statusUnknownStyle  = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().Foreground(colorAccent)

	// Dialogs
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDanger).
			Padding(1, 2)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true).
				MarginBottom(1)

	// Logs View
	logsHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	logsStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	// Empty State
	emptyStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Italic(true).
			Align(lipgloss.Center)

	// Wizard Specific
	cursorStyle        = lipgloss.NewStyle().Foreground(colorAccent)
	helpStyle          = lipgloss.NewStyle().Foreground(colorTextDim)
	stepIndicatorStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).MarginRight(1)
	labelStyle         = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).MarginRight(1)
	containerStyle     = lipgloss.NewStyle().PaddingLeft(2).Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(colorSurface)

	// Progress Bar
	progressBarWidth = 60
	progressStyle    = lipgloss.NewStyle().Foreground(colorAccent)
	progressFull     = lipgloss.Color("#22D3EE") // Cyan
	progressEmpty    = lipgloss.Color("#334155") // Slate 700

	stepTextStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Italic(true).
			MarginBottom(1)
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func GetTitleStyle() lipgloss.Style   { return headerStyle }
func GetErrorStyle() lipgloss.Style   { return notifyErrorStyle }
func GetSuccessStyle() lipgloss.Style { return notifySuccessStyle }

func statusBadge(status string) string {
	switch status {
	case "running", "healthy":
		return statusRunningStyle.Render("● " + status)
	case "exited", "dead", "not found":
		return statusStoppedStyle.Render("● " + status)
	case "restarting", "starting", "building":
		return statusBuildingStyle.Render("◐ " + status)
	default:
		if status == "" {
			return statusUnknownStyle.Render("○ unknown")
		}
		return statusUnknownStyle.Render("○ " + status)
	}
}
