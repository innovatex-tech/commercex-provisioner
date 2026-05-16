package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/innovatex-tech/commercex-provisioner/internal/deploy"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

// ─── Message Types ────────────────────────────────────────────────────────────

type statusRefreshedMsg struct {
	statuses map[string]ClientStatus
}

type tickMsg time.Time

type actionDoneMsg struct {
	clientID string
	action   string
	err      error
}

type logLineMsg struct {
	line string
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func refreshStatusCmd(clients []*registry.Client) tea.Cmd {
	return func() tea.Msg {
		statuses := FetchAllStatuses(clients)
		return statusRefreshedMsg{statuses: statuses}
	}
}

func startClientCmd(workDir string, c *registry.Client) tea.Cmd {
	return func() tea.Msg {
		var err error
		if c.IsRemote {
			ssh, errSSH := deploy.NewSSHOrchestrator(c.ServerHost, c.ServerUser, c.SSHPassword, c.SSHKeyPath)
			if errSSH != nil {
				return actionDoneMsg{clientID: c.ID, action: "start", err: errSSH}
			}
			remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", c.ID)
			_, err = ssh.RunCommand(fmt.Sprintf("cd %s && docker compose start", remotePath))
			ssh.Close()
		} else {
			clientDir := filepath.Join(workDir, c.ID)
			cmd := exec.Command("docker", "compose", "start")
			cmd.Dir = clientDir
			_, err = cmd.CombinedOutput()
		}
		return actionDoneMsg{clientID: c.ID, action: "start", err: err}
	}
}

func stopClientCmd(workDir string, c *registry.Client) tea.Cmd {
	return func() tea.Msg {
		var err error
		if c.IsRemote {
			ssh, errSSH := deploy.NewSSHOrchestrator(c.ServerHost, c.ServerUser, c.SSHPassword, c.SSHKeyPath)
			if errSSH != nil {
				return actionDoneMsg{clientID: c.ID, action: "stop", err: errSSH}
			}
			remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", c.ID)
			_, err = ssh.RunCommand(fmt.Sprintf("cd %s && docker compose stop", remotePath))
			ssh.Close()
		} else {
			clientDir := filepath.Join(workDir, c.ID)
			cmd := exec.Command("docker", "compose", "stop")
			cmd.Dir = clientDir
			_, err = cmd.CombinedOutput()
		}
		return actionDoneMsg{clientID: c.ID, action: "stop", err: err}
	}
}

func deleteClientCmd(workDir string, c *registry.Client) tea.Cmd {
	return func() tea.Msg {
		if c.IsRemote {
			ssh, errSSH := deploy.NewSSHOrchestrator(c.ServerHost, c.ServerUser, c.SSHPassword, c.SSHKeyPath)
			if errSSH == nil {
				remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", c.ID)
				ssh.RunCommand(fmt.Sprintf("cd %s && docker compose down -v", remotePath))
				ssh.RunCommand(fmt.Sprintf("rm -rf %s", remotePath))
				ssh.Close()
			}
		}

		// Always clean up local staging if it exists
		clientDir := filepath.Join(workDir, c.ID)
		if _, err := os.Stat(clientDir); err == nil {
			cmd := exec.Command("docker", "compose", "down", "-v")
			cmd.Dir = clientDir
			cmd.CombinedOutput()
			os.RemoveAll(clientDir)
		}

		return actionDoneMsg{clientID: c.ID, action: "delete", err: nil}
	}
}

func streamLogsCmd(workDir string, c *registry.Client, service string) tea.Cmd {
	return func() tea.Msg {
		var out string
		var err error

		args := []string{"compose", "logs", "--tail=200", "--no-color"}
		if service != "" {
			args = append(args, service)
		}

		if c.IsRemote {
			ssh, errSSH := deploy.NewSSHOrchestrator(c.ServerHost, c.ServerUser, c.SSHPassword, c.SSHKeyPath)
			if errSSH != nil {
				return logLineMsg{line: fmt.Sprintf("Error connecting to remote: %v", errSSH)}
			}
			remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", c.ID)
			cmdStr := fmt.Sprintf("cd %s && docker %s", remotePath, strings.Join(args, " "))
			out, err = ssh.RunCommand(cmdStr)
			ssh.Close()
		} else {
			clientDir := filepath.Join(workDir, c.ID)
			cmd := exec.Command("docker", args...)
			cmd.Dir = clientDir
			outBytes, errLocal := cmd.Output()
			out = string(outBytes)
			err = errLocal
		}

		if err != nil {
			return logLineMsg{line: fmt.Sprintf("Error fetching logs: %v\n%s", err, out)}
		}

		return logLineMsg{line: out}
	}
}
