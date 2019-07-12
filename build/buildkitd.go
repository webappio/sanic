package build

import (
	"bytes"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/docker"
	"github.com/moby/buildkit/client/llb"
	"os/exec"
	"time"
)

//BuildkitDaemonContainerName is the name of the docker container which contains the buildkit daemon
const BuildkitDaemonContainerName = "sanic-buildkitd"

//BuildkitDaemonAddr is the address on the host at which the buildkit server will be listening
const BuildkitDaemonAddr = "tcp://127.0.0.1:31652"

func waitBuildkitRunning() error {
	var dummyImageData bytes.Buffer
	dt, err := llb.Image("docker.io/library/alpine:latest@sha256:1072e499f3f655a032e88542330cf75b02e7bdf673278f701d7ba61629ee3ebe").Marshal(llb.LinuxAmd64)
	if err != nil {
		return err
	}
	err = llb.WriteTo(dt, &dummyImageData)
	fmt.Printf("Data is: %d\n", len(dummyImageData.String()))
	if err != nil {
		return err
	}
	timer := time.NewTicker(time.Millisecond * 100)
	defer timer.Stop()

	done := false
	go func() {
		time.Sleep(time.Second * 5)
		done = true
	}()

	for range timer.C {
		cmd := exec.Command(
			"docker",
			"exec",
			"-i",
			BuildkitDaemonContainerName,
			"buildctl",
			"--addr", BuildkitDaemonAddr,
			"build",
		)
		cmd.Stdin = &dummyImageData
		err = cmd.Run()
		if err == nil {
			return nil
		}
		if done {
			break
		}
	}
	return fmt.Errorf("buildkit daemon never came up")
}

//EnsureBuildkitDaemon makes sure that the buildkit docker container named "sanic-buildkitd" is running
func EnsureBuildkitDaemon() error {
	running, err := containers.CheckRunning(BuildkitDaemonContainerName)
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	fmt.Println("The build daemon is not running yet. Starting it...")
	containers.ForceRemove(BuildkitDaemonContainerName) //ignore error intentionally
	stderr := &bytes.Buffer{}
	cmd := exec.Command("docker",
		"run", "-d",
		"--name", BuildkitDaemonContainerName,
		"--restart", "always",
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
	return waitBuildkitRunning()
}
