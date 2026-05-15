package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type DockerDeployer struct {
	workDir string
}

func NewDockerDeployer(workDir string) *DockerDeployer {
	return &DockerDeployer{workDir: workDir}
}

func (d *DockerDeployer) Deploy(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker", "compose", "up", "-d", "--build")
	cmd.Dir = clientDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("deploy failed: %s", string(output))
	}

	fmt.Println(string(output))
	return nil
}

func (d *DockerDeployer) Stop(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker", "compose", "stop")
	cmd.Dir = clientDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop failed: %s", string(output))
	}

	return nil
}

func (d *DockerDeployer) Remove(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = clientDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove failed: %s", string(output))
	}

	return nil
}

// Start resumes stopped containers without rebuilding.
func (d *DockerDeployer) Start(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker", "compose", "start")
	cmd.Dir = clientDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start failed: %s", string(output))
	}

	return nil
}

// Logs streams docker compose logs for a client to stdout.
// service is optional — if empty, all services are shown.
func (d *DockerDeployer) Logs(clientID, service string, tail int, follow bool) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	args := []string{"compose", "logs", fmt.Sprintf("--tail=%d", tail), "--no-color"}
	if follow {
		args = append(args, "-f")
	}
	if service != "" {
		args = append(args, service)
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = clientDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ContainerStatus holds the live state of a single container.
type ContainerStatus struct {
	Name   string
	State  string
	Status string
}

// Status returns a map of service-name → ContainerStatus for all containers
// belonging to the given clientID.
func (d *DockerDeployer) Status(clientID string) (map[string]ContainerStatus, error) {
	type entry struct {
		Names  string `json:"Names"`
		State  string `json:"State"`
		Status string `json:"Status"`
	}

	cmd := exec.Command("docker", "ps", "-a",
		"--filter", fmt.Sprintf("name=%s", clientID),
		"--format", `{"Names":"{{.Names}}","State":"{{.State}}","Status":"{{.Status}}"}`,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %v", err)
	}

	result := make(map[string]ContainerStatus)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		name := strings.TrimPrefix(e.Names, "/")
		result[name] = ContainerStatus{
			Name:   name,
			State:  e.State,
			Status: e.Status,
		}
	}

	return result, nil
}
