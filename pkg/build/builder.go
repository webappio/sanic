package build

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/webappio/sanic/pkg/util"
	"os/exec"
	"time"
)

//Builder uses buildkit to build a list of service directories
type Builder struct {
	Registry         string
	RegistryInsecure bool
	BuildTag         string
	Logger           Logger
	Interface        Interface
	DoPush           bool
}

func (builder *Builder) runCommandAndOutput(cmd *exec.Cmd, ctx context.Context, serviceName string) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "could not pipe stdout from docker build command")
	}

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	go func() {
		buildOut := bufio.NewScanner(stdout)
		for buildOut.Scan() {
			err = builder.Logger.Log(serviceName, time.Now(), buildOut.Text())
			if err != nil {
				return
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}

	err = util.WaitCmdContextually(ctx, cmd)
	return errors.Wrapf(err, "error: %v", stderr)
}

//BuildService builds a specific sevice directory with a specific context
func (builder *Builder) BuildService(ctx context.Context, service util.BuildableService) error {
	fullImageName := fmt.Sprintf("%s:%s", service.Name, builder.BuildTag)
	if builder.Registry != "" {
		fullImageName = fmt.Sprintf("%s/%s:%s", builder.Registry, service.Name, builder.BuildTag)
	}

	builder.Interface.StartJob(service.Name, fullImageName)

	cmd := exec.Command("docker", "build",
		"--build-arg", "SANIC_ENV",
		"--build-arg", "CI",
		".",
		"--file", service.Dockerfile,
		"--tag", fullImageName)
	cmd.Dir = service.Dir


	err := builder.runCommandAndOutput(cmd, ctx, service.Name)
	if err != nil {
		builder.Interface.FailJob(service.Name, err)
		builder.Logger.Log(service.Name, time.Now(), "Build failed! ", err.Error())
		return errors.Wrap(err, "could not build "+service.Name)
	}

	if builder.DoPush {
		cmd = exec.Command("docker", "push", fullImageName)
		builder.Logger.Log(service.Name, time.Now(), "pushing image to registry...")
		err = builder.runCommandAndOutput(cmd, ctx, service.Name)
		if err != nil {
			builder.Interface.FailJob(service.Name, err)
			return errors.Wrap(err, "could not push " + service.Name)
		}
	}

	builder.Interface.SucceedJob(service.Name)
	builder.Logger.Log(service.Name, time.Now(), "Build succeeded!")
	return err
}
