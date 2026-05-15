package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keyboard shortcuts used in the dashboard.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Start    key.Binding
	Stop     key.Binding
	Toggle   key.Binding
	Delete   key.Binding
	Logs     key.Binding
	Refresh  key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

// keys is the global keybinding set.
var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start/stop"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "N", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("pgdn", "page down"),
	),
}

// ─── Footer Help Text ─────────────────────────────────────────────────────────

// helpString returns a formatted footer help bar string.
func helpString(view viewType, hasSelected bool) string {
	type helpItem struct{ key, desc string }

	var items []helpItem

	switch view {
	case listView:
		items = []helpItem{
			{"↑↓/jk", "navigate"},
			{"enter", "details"},
		}
		if hasSelected {
			items = append(items,
				helpItem{"s", "start/stop"},
				helpItem{"l", "logs"},
				helpItem{"d", "delete"},
			)
		}
		items = append(items,
			helpItem{"r", "refresh"},
			helpItem{"q", "quit"},
		)

	case detailView:
		items = []helpItem{
			{"esc", "back"},
			{"s", "start/stop"},
			{"l", "logs"},
			{"d", "delete"},
			{"r", "refresh"},
			{"q", "quit"},
		}

	case logsView:
		items = []helpItem{
			{"↑↓/pgup/pgdn", "scroll"},
			{"esc", "back"},
			{"q", "quit"},
		}

	case confirmView:
		items = []helpItem{
			{"y", "confirm"},
			{"n/esc", "cancel"},
		}
	}

	var parts []string
	for _, item := range items {
		k := footerKeyStyle.Render("[" + item.key + "]")
		d := footerDescStyle.Render(" " + item.desc)
		parts = append(parts, k+d)
	}

	sep := footerDescStyle.Render("  ")
	result := sep
	for i, p := range parts {
		result += p
		if i < len(parts)-1 {
			result += footerDescStyle.Render("  ·  ")
		}
	}
	return result
}
