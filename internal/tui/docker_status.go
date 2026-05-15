package tui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ─── Types ────────────────────────────────────────────────────────────────────

// ContainerInfo holds the parsed status of a single container.
type ContainerInfo struct {
	Name   string
	State  string // running, exited, restarting, etc.
	Status string // human: "Up 2 hours", "Exited (1) 5 minutes ago"
}

// ClientStatus holds the live Docker state for all containers of one client.
type ClientStatus struct {
	ClientID   string
	Containers map[string]ContainerInfo // key = service name
	Error      error
}

// AllRunning returns true if all 4 expected containers are running/healthy.
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

// AllStopped returns true if no containers are running.
func (cs ClientStatus) AllStopped() bool {
	for _, c := range cs.Containers {
		if c.State == "running" || c.State == "restarting" {
			return false
		}
	}
	return true
}

// HasBuilding returns true if any container is restarting or in a transient state.
func (cs ClientStatus) HasBuilding() bool {
	for _, c := range cs.Containers {
		if c.State == "restarting" || c.State == "starting" {
			return true
		}
	}
	return false
}

// ServiceState returns the state string for a given service name,
// checking both exact and prefixed names (e.g. "postgres_clientID").
func (cs ClientStatus) ServiceState(service string) string {
	// Direct match first
	if c, ok := cs.Containers[service]; ok {
		return c.State
	}
	// Fuzzy match by prefix (docker compose names services as service_clientID)
	for name, c := range cs.Containers {
		if strings.HasPrefix(name, service) {
			return c.State
		}
	}
	return "not found"
}

// ─── Docker Inspect Types ─────────────────────────────────────────────────────

type dockerPsEntry struct {
	Names  string `json:"Names"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

// ─── Polling ──────────────────────────────────────────────────────────────────

// FetchClientStatus queries `docker ps -a` and returns the status of all
// containers belonging to the given clientID.
func FetchClientStatus(clientID string) ClientStatus {
	cs := ClientStatus{
		ClientID:   clientID,
		Containers: make(map[string]ContainerInfo),
	}

	// Filter containers whose names contain the clientID
	filter := fmt.Sprintf("name=%s", clientID)
	cmd := exec.Command("docker", "ps", "-a",
		"--filter", filter,
		"--format", `{"Names":"{{.Names}}","State":"{{.State}}","Status":"{{.Status}}"}`,
	)

	out, err := cmd.Output()
	if err != nil {
		cs.Error = fmt.Errorf("docker ps failed: %v", err)
		return cs
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry dockerPsEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Normalize container name to service key
		// Docker Compose names containers as: {service}_{clientID} or {clientID}-{service}-1
		name := strings.TrimPrefix(entry.Names, "/")
		serviceKey := extractServiceKey(name, clientID)

		cs.Containers[serviceKey] = ContainerInfo{
			Name:   name,
			State:  entry.State,
			Status: entry.Status,
		}
	}

	return cs
}

// FetchAllStatuses fetches Docker status for a slice of clientIDs concurrently.
func FetchAllStatuses(clientIDs []string) map[string]ClientStatus {
	result := make(map[string]ClientStatus)
	ch := make(chan ClientStatus, len(clientIDs))

	for _, id := range clientIDs {
		go func(cid string) {
			ch <- FetchClientStatus(cid)
		}(id)
	}

	for range clientIDs {
		cs := <-ch
		result[cs.ClientID] = cs
	}

	return result
}

// extractServiceKey normalizes a full container name to a short service key.
// e.g. "commercex_server_mystore" → "commercex-server"
//
//	"mystore-postgres_db-1"       → "postgres"
func extractServiceKey(containerName, clientID string) string {
	// Remove clientID suffix/prefix variations
	name := containerName
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
	// Normalize underscores to dashes
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
