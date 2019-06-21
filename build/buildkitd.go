package build

import (
	"bytes"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/docker"
	"os/exec"
	"time"
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
	containers.ForceRemove(BuildkitDaemonContainerName) //ignore error intentionally
	stderr := &bytes.Buffer{}
	cmd := exec.Command("docker",
		"run", "-d",
		"--name", "sanic-buildkitd",
		"--privileged",
		"--network", "host",
		"-v", "sanic-buildkitd:/var/lib/buildkit",
		"moby/buildkit:latest", //TODO version pin / configure buildkit version?
		"--addr", BuildkitDaemonAddr)
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("could not start the builder docker image locally: %s\n%s", err.Error(), stderr.String())
	}
	time.Sleep(200 * time.Millisecond) //TODO HACK should poll for it to start instead of flat sleep
	return nil
}
