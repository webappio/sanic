package build

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/dockerbridge"
	"os/exec"
)

//BuildkitDaemonContainerName is the name of the docker container which contains the buildkit daemon
const BuildkitDaemonContainerName = "sanic-buildkitd"

//EnsureBuildkitDaemon makes sure that the buildkit docker container named "sanic-buildkitd" is running
func EnsureBuildkitDaemon() error {
	running, err := containers.CheckRunning(BuildkitDaemonContainerName)
	if err != nil {
		return err
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
	ip, err := containers.GetIPAddress(BuildkitDaemonContainerName)
	return fmt.Sprintf("tcp://%s:2149", ip), nil
}
