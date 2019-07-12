package build

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func (builder *Builder) buildkitSolveOpts(serviceDir, fullImageName string, writer io.WriteCloser) *client.SolveOpt {
	solveOpt := &client.SolveOpt{
		LocalDirs: map[string]string{
			"context":    serviceDir,
			"dockerfile": serviceDir,
		},
		Session: []session.Attachable{authprovider.NewDockerAuthProvider()},
	}

	insecureString := "false"
	if builder.RegistryInsecure {
		insecureString = "true"
	}
	if builder.DoPush {
		solveOpt.Exports = []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":              fullImageName,
					"push":              "true",
					"registry.insecure": insecureString,
				},
			},
		}
	} else {
		solveOpt.Exports = []client.ExportEntry{
			{
				Type: "docker",
				Attrs: map[string]string{
					"name": fullImageName,
				},
				Output: writer,
			},
		}
	}

	return solveOpt
}

//BuildService builds a specific sevice directory with a specific context
func (builder *Builder) BuildService(ctx context.Context, serviceDir string) error {
	serviceName := filepath.Base(serviceDir)
	fullImageName := fmt.Sprintf("%s:%s", serviceName, builder.BuildTag)
	if builder.Registry != "" {
		fullImageName = fmt.Sprintf("%s/%s:%s", builder.Registry, serviceName, builder.BuildTag)
	}
	builder.Interface.StartJob(serviceName, fullImageName)
	statusChannel := make(chan *client.SolveStatus)

	var resultR *io.PipeReader
	var resultW *io.PipeWriter
	if !builder.DoPush {
		resultR, resultW = io.Pipe()
	}
	buildOpts := builder.buildkitSolveOpts(serviceDir, fullImageName, resultW)

	err := util.RunContextuallyInParallel(
		ctx,
		func(ctx context.Context) error {
			buildkitClient, err := client.New(ctx, BuildkitDaemonAddr, client.WithFailFast())
			if err != nil {
				builder.Interface.FailJob(serviceName, err)
				builder.Logger.Log(serviceName, time.Now(), "Could not connect to build daemon! ", err.Error())
				return err
			}
			builder.Logger.Log(serviceName, time.Now(), "Starting build of ", serviceDir)
			solveStatus, err := buildkitClient.Build(ctx, *buildOpts, "", dockerfile.Build, statusChannel)
			if solveStatus != nil {
				//TODO if this is null should print a warning that we failed to push
				//e.g., when we haven't deployed yet
				for k, v := range solveStatus.ExporterResponse {
					builder.Logger.Log(serviceName, time.Now(), fmt.Sprintf("exporter: %s=%s", k, v))
				}
			}
			if err != nil {
				builder.Logger.Log(serviceName, time.Now(), "FAILED: ", err.Error())
			}
			return err
		},
		func(ctx context.Context) error {
			//Load the built service into the docker engine
			if !builder.DoPush {
				cmd := exec.Command("docker", "load")
				cmd.Stdin = resultR
				if err := cmd.Start(); err != nil {
					return err
				}
				err := util.WaitCmdContextually(ctx, cmd)
				resultR.CloseWithError(err)
				return err
			}
			return nil
		},
		func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return context.Canceled
				case status, ok := <-statusChannel:
					if !ok {
						return nil
					}
					for _, status := range status.Statuses {
						if status.ID == "pushing layers" {
							builder.Interface.SetPushing(serviceName)
						}
					}
					logErr := builder.Logger.ProcessStatus(serviceName, status)
					if logErr != nil {
						fmt.Fprintln(os.Stderr, logErr.Error())
					}
				}
			}
		},
	)

	if err == nil {
		builder.Interface.SucceedJob(serviceName)
		builder.Logger.Log(serviceName, time.Now(), "Build succeeded!")
	} else if err != context.Canceled {
		builder.Interface.FailJob(serviceName, err)
		builder.Logger.Log(serviceName, time.Now(), "Build failed! ", err.Error())
	}
	return err
}
