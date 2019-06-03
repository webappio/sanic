package build

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/dockerbridge"
	"os/exec"
)

//BuildkitDaemonContainerName is the name of the docker container which contains the buildkit daemon
const BuildkitDaemonContainerName = "sanic-buildkitd"

//BuildkitDaemonAddr is the address on the host at which the buildkit server will be listening
const BuildkitDaemonAddr = "tcp://127.0.0.1:31652"

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
		"--network", "host",
		"moby/buildkit:latest", //TODO version pin / configure buildkit version?
		"--addr", BuildkitDaemonAddr)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not start the builder docker image locally: %s", err.Error())
	}
	return nil
}
