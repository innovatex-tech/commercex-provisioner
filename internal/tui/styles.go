package tui

import "github.com/charmbracelet/lipgloss"

// ─── Palette ─────────────────────────────────────────────────────────────────

const (
	colorPrimary   = lipgloss.Color("#7C3AED") // violet-600
	colorAccent    = lipgloss.Color("#A78BFA") // violet-400
	colorSuccess   = lipgloss.Color("#22C55E") // green-500
	colorWarning   = lipgloss.Color("#F59E0B") // amber-500
	colorDanger    = lipgloss.Color("#EF4444") // red-500
	colorMuted     = lipgloss.Color("#6B7280") // gray-500
	colorSubtle    = lipgloss.Color("#374151") // gray-700
	colorBg        = lipgloss.Color("#111827") // gray-900
	colorSurface   = lipgloss.Color("#1F2937") // gray-800
	colorBorder    = lipgloss.Color("#374151") // gray-700
	colorText      = lipgloss.Color("#F9FAFB") // gray-50
	colorTextDim   = lipgloss.Color("#9CA3AF") // gray-400
)

// ─── Base Styles ─────────────────────────────────────────────────────────────

var (
	// App chrome
	appStyle = lipgloss.NewStyle().
			Background(colorBg)

	// Header bar
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 2)

	headerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText)

	headerVersionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#DDD6FE")).
				Faint(true)

	// Footer / help bar
	footerStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Background(colorSurface).
			Padding(0, 2)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(colorTextDim)

	// Section panels
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Background(colorSurface).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Background(colorSurface).
				Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	// Detail panel fields
	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Width(14)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(colorText)

	fieldURLStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Underline(true)

	// Table header
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorBorder)

	// Table selected row
	tableSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorText).
				Background(colorPrimary)

	// Status badges
	statusRunningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSuccess)

	statusStoppedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDanger)

	statusBuildingStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWarning)

	statusUnknownStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorMuted)

	// Confirm dialog
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDanger).
			Background(colorSurface).
			Padding(1, 3).
			Align(lipgloss.Center)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDanger).
				MarginBottom(1)

	// Logs view
	logsHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Background(colorSurface).
			Padding(0, 1)

	logsStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Background(colorBg)

	// Notification / flash message
	notifySuccessStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSuccess).
				Background(colorSurface).
				Padding(0, 2)

	notifyErrorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDanger).
				Background(colorSurface).
				Padding(0, 2)

	notifyInfoStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Background(colorSurface).
			Padding(0, 2)

	// Spinner
	spinnerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary)

	// Empty state
	emptyStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Align(lipgloss.Center)

	// Wizard specific
	cursorStyle = lipgloss.NewStyle().Foreground(colorAccent)
	helpStyle   = lipgloss.NewStyle().Foreground(colorTextDim)
)

// ─── Status Badge Helpers ─────────────────────────────────────────────────────

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
