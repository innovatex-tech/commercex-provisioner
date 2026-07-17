package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// ─── Semantic Color Slots ─────────────────────────────────────────────────────
// Base16-inspired: 8 monotones + accent slots.
// Map by function, never by appearance. Terminal theme controls the rest.

const (
	// Backgrounds (dark, warm)
	bgBase     = lipgloss.Color("#0D1117") // near-black
	bgSurface  = lipgloss.Color("#161B22") // elevated panels
	bgOverlay  = lipgloss.Color("#1C2128") // highest layer

	// Foreground text (warm grays — NOT saturated, Gemini-style)
	fgMuted    = lipgloss.Color("#6E7681") // dimmest — timestamps, hints
	fgDefault  = lipgloss.Color("#9198A1") // body text — easy on eyes
	fgMid      = lipgloss.Color("#C9D1D9") // emphasis — headings, labels
	fgBright   = lipgloss.Color("#E6EDF3") // brightest — active selection
	fgPure     = lipgloss.Color("#F0F6FC") // white — only for pure highlights

	// Accent (teal — used sparingly for focus, links, borders)
	accentPrimary = lipgloss.Color("#3FBAA0") // teal — focus ring, links
	accentDim     = lipgloss.Color("#2D8B78") // darker teal — selected bg

	// Status (semantic, low-saturation for eye comfort)
	statusOK    = lipgloss.Color("#3FB950") // green — running
	statusWarn  = lipgloss.Color("#D29922") // amber — building, partial
	statusErr   = lipgloss.Color("#F85149") // red — stopped, error
	statusInfo  = lipgloss.Color("#58A6FF") // blue — info
	statusMuted = lipgloss.Color("#6E7681") // gray — unknown

	// Borders (subtle)
	borderDefault = lipgloss.Color("#30363D") // very subtle
	borderFocus   = lipgloss.Color("#3FBAA0") // teal accent
	borderDanger  = lipgloss.Color("#F85149") // red
)

// ─── Layout Constants ─────────────────────────────────────────────────────────

const (
	minWidth  = 80
	minHeight = 24
)

// ─── Separator ────────────────────────────────────────────────────────────────

func hLine(w int) string {
	return lipgloss.NewStyle().
		Foreground(borderDefault).
		Width(w).
		Render("─")
}

func hLineFocus(w int) string {
	return lipgloss.NewStyle().
		Foreground(borderFocus).
		Width(w).
		Render("─")
}

// ─── Header ───────────────────────────────────────────────────────────────────

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgBright).
			Background(bgBase).
			Padding(0, 1)

	headerVersionStyle = lipgloss.NewStyle().
				Foreground(fgMuted)
)

// ─── Footer ───────────────────────────────────────────────────────────────────

var (
	footerStyle = lipgloss.NewStyle().
			Foreground(fgMuted).
			Background(bgSurface).
			Padding(0, 1)

	footerKeybindStyle = lipgloss.NewStyle().
				Foreground(accentPrimary).
				MarginRight(1)
)

// ─── Panels ───────────────────────────────────────────────────────────────────

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderDefault).
			Background(bgSurface).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderFocus).
				Background(bgSurface).
				Padding(0, 1)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentPrimary).
			Padding(0, 1)
)

// ─── Detail Fields ────────────────────────────────────────────────────────────

var (
	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(fgMuted).
			Width(14)

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(fgMid)

	fieldURLStyle = lipgloss.NewStyle().
			Foreground(accentPrimary).
			Underline(true)

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(accentDim).
				Bold(true)

	containerLabelStyle = lipgloss.NewStyle().
				Foreground(fgDefault).
				Width(16)
)

// ─── Table ────────────────────────────────────────────────────────────────────

var (
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(fgBright).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(borderDefault)

	tableSelectedStyle = lipgloss.NewStyle().
				Foreground(fgPure).
				Background(accentDim)
)

// ─── Status Badges ────────────────────────────────────────────────────────────

var (
	statusOKStyle    = lipgloss.NewStyle().Foreground(statusOK)
	statusErrStyle   = lipgloss.NewStyle().Foreground(statusErr)
	statusWarnStyle  = lipgloss.NewStyle().Foreground(statusWarn)
	statusMutedStyle = lipgloss.NewStyle().Foreground(statusMuted)
)

func statusBadge(status string) string {
	switch status {
	case "running", "healthy":
		return statusOKStyle.Render("● running")
	case "exited", "dead":
		return statusErrStyle.Render("● stopped")
	case "not found":
		return statusMutedStyle.Render("○ not found")
	case "restarting", "starting", "building":
		return statusWarnStyle.Render("◐ building")
	default:
		if status == "" {
			return statusMutedStyle.Render("○ unknown")
		}
		return statusMutedStyle.Render("○ " + status)
	}
}

func overallStatus(cs ClientStatus) string {
	if cs.AllRunning() {
		return "running"
	}
	if cs.AllStopped() {
		return "stopped"
	}
	if cs.HasBuilding() {
		return "building"
	}
	return "partial"
}

func overallStatusBadge(cs ClientStatus) string {
	switch overallStatus(cs) {
	case "running":
		return statusOKStyle.Render("● running")
	case "stopped":
		return statusErrStyle.Render("● stopped")
	case "building":
		return statusWarnStyle.Render("◐ building")
	case "partial":
		return statusWarnStyle.Render("◐ partial")
	default:
		return statusMutedStyle.Render("○ unknown")
	}
}

// ─── Dialog ───────────────────────────────────────────────────────────────────

var (
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(borderDanger).
			Background(bgSurface).
			Padding(1, 3).
			Width(50).
			Align(lipgloss.Center)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(statusErr).
				MarginBottom(1)
)

// ─── Logs ─────────────────────────────────────────────────────────────────────

var (
	logsHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgBright).
			Background(bgSurface).
			Padding(0, 1)

	logsStyle = lipgloss.NewStyle().
			Foreground(fgDefault).
			Background(bgBase)
)

// ─── Status Bar (replaces notification bar) ───────────────────────────────────

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(fgMuted).
			Background(bgSurface).
			Padding(0, 1)

	statusBarOKStyle = lipgloss.NewStyle().
				Foreground(statusOK).
				Background(bgSurface).
				Padding(0, 1)

	statusBarErrStyle = lipgloss.NewStyle().
				Foreground(statusErr).
				Background(bgSurface).
				Padding(0, 1)
)

// ─── Misc ─────────────────────────────────────────────────────────────────────

var (
	spinnerStyle = lipgloss.NewStyle().
			Foreground(accentPrimary)

	emptyStyle = lipgloss.NewStyle().
			Foreground(fgMuted).
			Italic(true).
			Align(lipgloss.Center)
)

// ─── Too-Small Gate ───────────────────────────────────────────────────────────

func tooSmallMessage(w, h int) string {
	msg := fmt.Sprintf("Terminal too small\n\nNeed 80x24 minimum\nCurrent: %dx%d", w, h)
	content := lipgloss.NewStyle().
		Foreground(fgMuted).
		Align(lipgloss.Center).
		Width(w).
		Render(msg)
	// Vertically center by padding top
	lines := (h - 4) / 2
	if lines < 0 {
		lines = 0
	}
	return content
}
