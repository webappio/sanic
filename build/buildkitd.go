package build

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func daemonRunning() (bool, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Status}}", "sanic-buildkitd")
	out := &bytes.Buffer{}
	cmd.Stdout = out
	err := cmd.Start()
	if err != nil {
		return false, err
	}
	cmd.Wait() //ignore error
	return strings.TrimSpace(out.String()) == "running", nil
}

//EnsureBuildkitDaemon makes sure that the buildkit docker container named "sanic-buildkitd" is running
func EnsureBuildkitDaemon() error {
	running, err := daemonRunning()
	if err != nil {
		return fmt.Errorf("could not connect to docker, is it installed and running? %s", err.Error())
	}
	if running {
		return nil
	}
	cmd := exec.Command("docker",
		"run", "-d",
		"--name", "sanic-buildkitd",
		"--privileged",
		"moby/buildkit:latest", //TODO version pin / configure buildkit version?
		"--addr", "tcp://0.0.0.0:2149")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not start the builder docker image locally: %s", err.Error())
	}
	return nil
}

//GetBuildkitAddress returns a buildkit-compatible tcp://(ip):(port) string with which to connect to the buildkit daemon
func GetBuildkitAddress() (string, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", "sanic-buildkitd")
	out := &bytes.Buffer{}
	cmd.Stdout = out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("could not find buildkit daemon's IP Address, are you using custom networking on your docker daemon? %s", err.Error())
	}
	ip := strings.TrimSpace(out.String())
	return fmt.Sprintf("tcp://%s:2149", ip), nil
}