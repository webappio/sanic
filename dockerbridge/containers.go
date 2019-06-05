package dockerbridge

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

//CheckRunning checks that a container has been created, and is currently running
//returns an error if, e.g., docker is not installed
func CheckRunning(containerIdentifier string) (bool, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", containerIdentifier)
	out := &bytes.Buffer{}
	cmd.Stdout = out
	err := cmd.Start()
	if err != nil {
		return false, fmt.Errorf("could not connect to docker, is it installed and running? %s", err.Error())
	}
	cmd.Wait() //ignore error
	return strings.TrimSpace(out.String()) == "running", nil
}

func GetIPAddress(containerIdentifier string) (string, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", containerIdentifier)
	out := &bytes.Buffer{}
	cmd.Stdout = out
	err := cmd.Run()
	if err != nil || strings.TrimSpace(out.String()) == "" {
		return "", fmt.Errorf("could not find %s daemon's IP Address, are you using custom networking on your docker daemon? %s", containerIdentifier, err.Error())
	}
	return strings.TrimSpace(out.String()), nil
}

func ForceRemove(containerIdentifier string) error {
	cmd := exec.Command("docker", "rm", "--force", containerIdentifier)
	return cmd.Run()
}