package deploy

import (
	"fmt"
	"os/exec"
)

type DockerDeployer struct {
	workDir string
}

func NewDockerDeployer(workDir string) *DockerDeployer {
	return &DockerDeployer{workDir: workDir}
}

func (d *DockerDeployer) Deploy(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker-compose", "up", "-d", "--build")
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

	cmd := exec.Command("docker-compose", "stop")
	cmd.Dir = clientDir

	return cmd.Run()
}

func (d *DockerDeployer) Remove(clientID string) error {
	clientDir := fmt.Sprintf("%s/%s", d.workDir, clientID)

	cmd := exec.Command("docker-compose", "down", "-v")
	cmd.Dir = clientDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove failed: %s", string(output))
	}

	return nil
}
