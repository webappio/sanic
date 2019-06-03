package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/distributed-containers-inc/sanic/provisioners/localdev"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"time"
)

func findServices(path string) ([]string, error) {
	var ret []string

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "Dockerfile" {
			ret = append(ret, filepath.Dir(path))
		}

		return nil
	})

	return ret, err
}

func createBuildInterface(forceNoninteractive bool) build.Interface {
	if !forceNoninteractive {
		interactiveInterface, err := build.NewInteractiveInterface()
		if err == nil {
			return interactiveInterface
		}
		fmt.Fprintf(os.Stderr, "Failed to launch interactive interface: %s\n", err.Error())
	}
	return build.NewPlaintextInterface()
}

func buildOptions(serviceDir string) client.SolveOpt {
	return client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":              fmt.Sprintf("172.17.0.4:%d/%s:latest", localdev.RegistryNodePort, filepath.Base(serviceDir)), //TODO BEFORE COMMIT
					"push":              "true",
					"registry.insecure": "true",
				},
				//Output: pipeW,
			},
		},
		LocalDirs: map[string]string{
			"context":    serviceDir,
			"dockerfile": serviceDir,
		},
	}
}

func buildService(
	ctx context.Context,
	buildInterface build.Interface,
	buildLogger build.Logger,
	serviceDir string,
) error {

	serviceName := filepath.Base(serviceDir)
	buildInterface.StartJob(serviceName)
	statusChannel := make(chan *client.SolveStatus)
	err := util.RunContextuallyInParallel(
		ctx,
		func(ctx context.Context) error {
			buildkitClient, err := client.New(ctx, build.BuildkitDaemonAddr, client.WithFailFast())
			if err != nil {
				buildInterface.FailJob(serviceName, err)
				buildLogger.Log(serviceName, time.Now(), "Could not connect to build daemon! ", err.Error())
				return err
			}
			buildLogger.Log(serviceName, time.Now(), "Starting build of ", serviceDir)
			solveStatus, err := buildkitClient.Build(ctx, buildOptions(serviceDir), "", dockerfile.Build, statusChannel)
			if solveStatus != nil {
				//TODO if this is null should print a warning that we failed to push
				//e.g., when we haven't deployed yet
				for k, v := range solveStatus.ExporterResponse {
					buildLogger.Log(serviceName, time.Now(), fmt.Sprintf("exporter: %s=%s", k, v))
				}
			}
			if err != nil {
				buildLogger.Log(serviceName, time.Now(), "FAILED: ", err.Error())
			}
			return err
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
					logErr := buildLogger.ProcessStatus(serviceName, status)
					if logErr != nil {
						fmt.Fprintln(os.Stderr, logErr.Error())
					}
				}
			}
		},
	)

	if err == nil {
		buildLogger.Log(serviceName, time.Now(), "Build succeeded!")
		buildInterface.SucceedJob(serviceName)
	} else if err != context.Canceled {
		buildInterface.FailJob(serviceName, err)
		buildLogger.Log(serviceName, time.Now(), "Build failed! ", err.Error())
	}
	return err
}

//adapted from
//https://web.archive.org/web/20190516153923/https://raw.githubusercontent.com/moby/buildkit/master/examples/build-using-dockerfile/main.go
func buildCommandAction(cliContext *cli.Context) error {
	s, err := shell.Current()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	services, err := findServices(s.GetSanicRoot())
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = build.EnsureBuildkitDaemon()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	buildInterface := createBuildInterface(cliContext.Bool("plaintext"))
	defer buildInterface.Close()

	buildLogger := build.NewFlatfileLogger(filepath.Join(s.GetSanicRoot(), "logs"))
	buildLogger.AddLogLineListener(buildInterface.ProcessLog)
	defer buildLogger.Close()

	jobs := make([]func(context.Context) error, 0, len(services))

	for _, serviceDir := range services {
		finalServiceDir := serviceDir
		jobs = append(jobs, func(ctx context.Context) error {
			return buildService(
				ctx,
				buildInterface,
				buildLogger,
				finalServiceDir,
			)
		})
	}

	userCancelledBuild := false
	ctx, cancelJob := context.WithCancel(context.Background())
	buildInterface.AddCancelListener(cancelJob)
	buildInterface.AddCancelListener(func() { userCancelledBuild = true })
	err = util.RunContextuallyInParallel(ctx, jobs...)

	if userCancelledBuild {
		fmt.Println() //clear the ^C
		return cli.NewExitError("", 1)
	}

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

var buildCommand = cli.Command{
	Name:   "build",
	Usage:  "build some (or all, by default) services",
	Action: buildCommandAction,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "plaintext",
			Usage:  "use a plaintext interface",
			EnvVar: "PLAINTEXT_INTERFACE",
		},
	},
}
