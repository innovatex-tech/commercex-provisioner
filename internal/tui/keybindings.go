package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

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
		key.WithHelp("y", "yes"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "N", "esc"),
		key.WithHelp("n", "cancel"),
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

func helpString(view viewType, hasSelected bool) string {
	type keybind struct{ key, desc string }

	var bindings []keybind

	switch view {
	case listView:
		bindings = []keybind{
			{"j/k", "navigate"},
			{"enter", "details"},
		}
		if hasSelected {
			bindings = append(bindings,
				keybind{"s", "start/stop"},
				keybind{"l", "logs"},
				keybind{"d", "delete"},
			)
		}
		bindings = append(bindings,
			keybind{"r", "refresh"},
			keybind{"q", "quit"},
		)

	case detailView:
		bindings = []keybind{
			{"esc", "back"},
			{"s", "start/stop"},
			{"l", "logs"},
			{"d", "delete"},
			{"r", "refresh"},
			{"q", "quit"},
		}

	case logsView:
		bindings = []keybind{
			{"j/k", "scroll"},
			{"esc", "back"},
			{"q", "quit"},
		}

	case confirmView:
		bindings = []keybind{
			{"y", "yes"},
			{"n", "cancel"},
		}
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts, footerKeybindStyle.Render("["+b.key+"]"+b.desc))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
