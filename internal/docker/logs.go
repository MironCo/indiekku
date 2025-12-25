package docker

import (
	"fmt"
	"os/exec"
)

// GetContainerLogs retrieves logs from a Docker container
func GetContainerLogs(containerName string, follow bool, tail string) (*exec.Cmd, error) {
	args := []string{"logs"}

	if follow {
		args = append(args, "-f")
	}

	if tail != "" {
		args = append(args, "--tail", tail)
	}

	args = append(args, containerName)

	cmd := exec.Command("docker", args...)
	return cmd, nil
}

// GetContainerLogsSince retrieves logs from a Docker container since a specific time
func GetContainerLogsSince(containerName string, since string) ([]byte, error) {
	cmd := exec.Command("docker", "logs", "--since", since, containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	return output, nil
}
