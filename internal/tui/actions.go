package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── Message Types ────────────────────────────────────────────────────────────

// statusRefreshedMsg is sent when Docker status polling completes.
type statusRefreshedMsg struct {
	statuses map[string]ClientStatus
}

// tickMsg fires on the polling interval.
type tickMsg time.Time

// actionDoneMsg is sent when a start/stop/delete action completes.
type actionDoneMsg struct {
	clientID string
	action   string
	err      error
}

// logLineMsg is a single line appended to the log view.
type logLineMsg struct {
	line string
}

// logsReadyMsg signals the log stream has started.
type logsReadyMsg struct {
	clientID string
}

// ─── Tick Command ─────────────────────────────────────────────────────────────

// tickCmd returns a command that fires after the polling interval.
func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ─── Status Refresh Command ───────────────────────────────────────────────────

// refreshStatusCmd fetches Docker status for all clientIDs asynchronously.
func refreshStatusCmd(clientIDs []string) tea.Cmd {
	return func() tea.Msg {
		statuses := FetchAllStatuses(clientIDs)
		return statusRefreshedMsg{statuses: statuses}
	}
}

// ─── Docker Action Commands ───────────────────────────────────────────────────

// startClientCmd runs `docker compose start` for the given client.
func startClientCmd(workDir, clientID string) tea.Cmd {
	return func() tea.Msg {
		clientDir := filepath.Join(workDir, clientID)
		cmd := exec.Command("docker", "compose", "start")
		cmd.Dir = clientDir
		_, err := cmd.CombinedOutput()
		return actionDoneMsg{clientID: clientID, action: "start", err: err}
	}
}

// stopClientCmd runs `docker compose stop` for the given client.
func stopClientCmd(workDir, clientID string) tea.Cmd {
	return func() tea.Msg {
		clientDir := filepath.Join(workDir, clientID)
		cmd := exec.Command("docker", "compose", "stop")
		cmd.Dir = clientDir
		_, err := cmd.CombinedOutput()
		return actionDoneMsg{clientID: clientID, action: "stop", err: err}
	}
}

// deleteClientCmd runs `docker compose down -v` + removes the client directory.
func deleteClientCmd(workDir, clientID string) tea.Cmd {
	return func() tea.Msg {
		clientDir := filepath.Join(workDir, clientID)

		// Stop and remove containers
		cmd := exec.Command("docker", "compose", "down", "-v")
		cmd.Dir = clientDir
		cmd.CombinedOutput() // best-effort

		// Remove directory
		os.RemoveAll(clientDir) // best-effort

		return actionDoneMsg{clientID: clientID, action: "delete", err: nil}
	}
}

// streamLogsCmd streams docker compose logs to the TUI viewport.
// It sends individual logLineMsg messages for each line.
func streamLogsCmd(p *tea.Program, workDir, clientID, service string) tea.Cmd {
	return func() tea.Msg {
		clientDir := filepath.Join(workDir, clientID)

		args := []string{"compose", "logs", "--tail=200", "--no-color"}
		if service != "" {
			args = append(args, service)
		}

		cmd := exec.Command("docker", args...)
		cmd.Dir = clientDir

		out, err := cmd.Output()
		if err != nil {
			return logLineMsg{line: fmt.Sprintf("Error fetching logs: %v\n", err)}
		}

		return logLineMsg{line: string(out)}
	}
}
