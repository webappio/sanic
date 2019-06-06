package containers

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

//ForceRemove is a wrapper around docker rm -f (image ...)
func ForceRemove(containerIdentifier ...string) error {
	cmd := exec.Command("docker", append([]string{"rm", "--force"}, containerIdentifier...)...)
	return cmd.Run()
}
