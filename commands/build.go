package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/git"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/urfave/cli"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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

func buildOptions(serviceDir string) *client.SolveOpt {
	solveOpt := &client.SolveOpt{
		LocalDirs: map[string]string{
			"context":    serviceDir,
			"dockerfile": serviceDir,
		},
	}

	return solveOpt
}

func getRegistry() (registry string, insecure bool, err error) {
	provisioner, err := provisioners.GetProvisioner()

	if err != nil {
		return
	}

	registry, err = provisioner.Registry()
	if err != nil {
		return
	}
	insecure = false
	if strings.HasPrefix(registry, "http://") {
		insecure = true
		registry = registry[len("http://"):]
	} else if strings.HasPrefix(registry, "https://") {
		registry = registry[len("https://"):]
	} else {
		err = fmt.Errorf("Registry must start with 'http://' or 'https://', got '%s'", registry)
	}
	return
}

func exportEntries(serviceName, buildTag string, push bool, writer io.WriteCloser) ([]client.ExportEntry, error) {
	registry, insecure, err := getRegistry()
	if err != nil {
		return nil, err
	}
	fullImageName := fmt.Sprintf("%s/%s:%s", registry, serviceName, buildTag)
	insecureString := "false"
	if insecure {
		insecureString = "true"
	}

	if push {
		return []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":              fullImageName,
					"push":              "true",
					"registry.insecure": insecureString,
				},
			},
		}, nil
	}
	return []client.ExportEntry{
		{
			Type: "docker",
			Attrs: map[string]string{
				"name": fullImageName,
			},
			Output: writer,
		},
	}, nil
}

func buildService(
	ctx context.Context,
	buildInterface build.Interface,
	buildLogger build.Logger,
	serviceDir string,
	registry string,
	buildTag string,
	cliContext *cli.Context,
) error {
	serviceName := filepath.Base(serviceDir)
	if registry == "" {
		buildInterface.StartJob(serviceName, fmt.Sprintf("%s:%s", serviceName, buildTag))
	} else {
		buildInterface.StartJob(serviceName, fmt.Sprintf("%s/%s:%s", registry, serviceName, buildTag))
	}
	statusChannel := make(chan *client.SolveStatus)

	push := cliContext.Bool("push")

	buildOpts := buildOptions(serviceDir)
	var resultR *io.PipeReader
	var resultW *io.PipeWriter
	var err error
	if !push {
		resultR, resultW = io.Pipe()
	}
	buildOpts.Exports, err = exportEntries(serviceName, buildTag, push, resultW)
	if err != nil {
		buildInterface.FailJob(serviceName, err)
		buildLogger.Log(serviceName, time.Now(), "Could not configure pushing / saving for image! ", err.Error())
		return err
	}

	err = util.RunContextuallyInParallel(
		ctx,
		func(ctx context.Context) error {
			buildkitClient, err := client.New(ctx, build.BuildkitDaemonAddr, client.WithFailFast())
			if err != nil {
				buildInterface.FailJob(serviceName, err)
				buildLogger.Log(serviceName, time.Now(), "Could not connect to build daemon! ", err.Error())
				return err
			}
			buildLogger.Log(serviceName, time.Now(), "Starting build of ", serviceDir)
			solveStatus, err := buildkitClient.Build(ctx, *buildOpts, "", dockerfile.Build, statusChannel)
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
			//Load the built service into the docker engine
			if !push {
				cmd := exec.Command("docker", "load")
				cmd.Stdin = resultR
				if err = cmd.Start(); err != nil {
					return err
				}
				err = util.WaitCmdContextually(ctx, cmd)
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
					logErr := buildLogger.ProcessStatus(serviceName, status)
					if logErr != nil {
						fmt.Fprintln(os.Stderr, logErr.Error())
					}
				}
			}
		},
	)

	if err == nil {
		buildInterface.SucceedJob(serviceName)
		buildLogger.Log(serviceName, time.Now(), "Build succeeded!")
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

	services, err := util.FindServices()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = build.EnsureBuildkitDaemon()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	buildTag, err := git.GetCurrentTreeHash(s.GetSanicRoot(), services...)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	buildInterface := createBuildInterface(cliContext.Bool("plaintext"))
	defer func() {
		r := recover()
		buildInterface.Close()
		if r != nil {
			panic(r)
		}
	}()

	buildLogger := build.NewFlatfileLogger(filepath.Join(s.GetSanicRoot(), "logs"))
	buildLogger.AddLogLineListener(buildInterface.ProcessLog)
	defer buildLogger.Close()

	jobs := make([]func(context.Context) error, 0, len(services))

	registry, _, err := getRegistry()
	if err != nil {
		registry = "'"
	}

	for _, serviceDir := range services {
		finalServiceDir := serviceDir
		jobs = append(jobs, func(ctx context.Context) error {
			return buildService(
				ctx,
				buildInterface,
				buildLogger,
				finalServiceDir,
				registry,
				buildTag,
				cliContext,
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
		cli.BoolFlag{
			Name:  "push",
			Usage: "pushes to the configured registry for the current environment instead of loading locally",
		},
	},
}
