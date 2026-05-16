package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/innovatex-tech/commercex-provisioner/internal/deploy"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// ClientStatus holds the live Docker state for all containers of one client.
type ClientStatus struct {
	ClientID   string
	Containers map[string]ContainerState // key = service name
	Error      error
}

// ContainerState holds the parsed status of a single container.
type ContainerState struct {
	ID     string
	State  string // running, exited, restarting, etc.
	Status string // human: "Up 2 hours", "Exited (1) 5 minutes ago"
}

// ─── Polling ──────────────────────────────────────────────────────────────────

// FetchAllStatuses gathers Docker info for all clients concurrently.
func FetchAllStatuses(clients []*registry.Client) map[string]ClientStatus {
	var wg sync.WaitGroup
	results := make(map[string]ClientStatus)
	var mu sync.Mutex

	for _, c := range clients {
		wg.Add(1)
		go func(client *registry.Client) {
			defer wg.Done()
			status := FetchStatus(client)
			mu.Lock()
			results[client.ID] = status
			mu.Unlock()
		}(c)
	}

	wg.Wait()
	return results
}

// FetchStatus queries docker ps for a single client (local or remote).
func FetchStatus(c *registry.Client) ClientStatus {
	var out string
	var err error

	format := `{"Names":"{{.Names}}","Status":"{{.Status}}","State":"{{.State}}"}`
	cmd := fmt.Sprintf("docker ps -a --filter \"name=%s\" --format '%s'", c.ID, format)

	if c.IsRemote {
		orchestrator, errSSH := deploy.NewSSHOrchestrator(c.ServerHost, c.ServerUser, c.SSHPassword, c.SSHKeyPath)
		if errSSH == nil {
			out, err = orchestrator.RunCommand(cmd)
			orchestrator.Close()
		} else {
			err = errSSH
		}
	} else {
		deployer := deploy.NewDockerDeployer("") // workDir not needed for raw ps
		statusMap, errLocal := deployer.Status(c.ID)
		if errLocal == nil {
			cs := ClientStatus{ClientID: c.ID, Containers: make(map[string]ContainerState)}
			for name, s := range statusMap {
				key := extractServiceKey(name, c.ID)
				cs.Containers[key] = ContainerState{
					ID:     name,
					Status: s.Status,
					State:  s.State,
				}
			}
			return cs
		}
		err = errLocal
	}

	status := ClientStatus{
		ClientID:   c.ID,
		Containers: make(map[string]ContainerState),
		Error:      err,
	}

	if err != nil {
		return status
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var raw struct {
			Names  string `json:"Names"`
			Status string `json:"Status"`
			State  string `json:"State"`
		}

		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		serviceKey := extractServiceKey(raw.Names, c.ID)
		status.Containers[serviceKey] = ContainerState{
			ID:     raw.Names,
			Status: raw.Status,
			State:  raw.State,
		}
	}

	return status
}

// extractServiceKey normalizes container names.
func extractServiceKey(containerName, clientID string) string {
	name := strings.TrimPrefix(containerName, "/")
	name = strings.ReplaceAll(name, "_"+clientID, "")
	name = strings.ReplaceAll(name, "-"+clientID, "")
	name = strings.TrimPrefix(name, clientID+"_")
	name = strings.TrimPrefix(name, clientID+"-")

	// Remove compose trailing index (e.g. "-1")
	if idx := strings.LastIndex(name, "-"); idx != -1 {
		suffix := name[idx+1:]
		if len(suffix) <= 2 && isDigits(suffix) {
			name = name[:idx]
		}
	}

	name = strings.ReplaceAll(name, "_", "-")
	return strings.Trim(name, "-_")
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ─── View Helpers ─────────────────────────────────────────────────────────────

func (cs ClientStatus) AllRunning() bool {
	services := []string{"commercex-server", "commercex-worker", "postgres", "storefront"}
	for _, svc := range services {
		c, ok := cs.Containers[svc]
		if !ok || (c.State != "running" && c.State != "healthy") {
			return false
		}
	}
	return true
}

func (cs ClientStatus) AllStopped() bool {
	if len(cs.Containers) == 0 {
		return true
	}
	for _, c := range cs.Containers {
		if c.State == "running" || c.State == "restarting" {
			return false
		}
	}
	return true
}

func (cs ClientStatus) HasBuilding() bool {
	for _, c := range cs.Containers {
		if c.State == "restarting" || c.State == "starting" {
			return true
		}
	}
	return false
}

func (cs ClientStatus) ServiceState(service string) string {
	if c, ok := cs.Containers[service]; ok {
		return c.State
	}
	// Fuzzy match
	for name, c := range cs.Containers {
		if strings.HasPrefix(name, service) {
			return c.State
		}
	}
	return "down"
}

func overallStatus(cs ClientStatus) string {
	if len(cs.Containers) == 0 {
		return "down"
	}

	running := 0
	for _, c := range cs.Containers {
		if c.State == "running" {
			running++
		}
	}

	if running == 0 {
		return "exited"
	}
	if running < len(cs.Containers) {
		return "partial"
	}
	return "running"
}
