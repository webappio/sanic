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

func buildOptions(serviceDir, buildTag string) (*client.SolveOpt, error) {
	provisioner, err := provisioners.GetProvisioner()

	if err != nil {
		return nil, err
	}

	solveOpt := &client.SolveOpt{
		LocalDirs: map[string]string{
			"context":    serviceDir,
			"dockerfile": serviceDir,
		},
	}

	if provisioner.RegistryPushDefault() {
		registry, err := provisioner.Registry()
		if err != nil {
			return nil, err
		}
		solveOpt.Exports = []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name":              fmt.Sprintf("%s/%s:%s", registry, filepath.Base(serviceDir), buildTag),
					"push":              "true",
					"registry.insecure": "true", //TODO probably shouldn't be by default
				},
			},
		}
	}

	return solveOpt, nil
}

func buildService(
	ctx context.Context,
	buildInterface build.Interface,
	buildLogger build.Logger,
	serviceDir string,
	buildTag string,
) error {

	serviceID := fmt.Sprintf("%s:%s", filepath.Base(serviceDir), buildTag)
	buildInterface.StartJob(serviceID)
	statusChannel := make(chan *client.SolveStatus)
	err := util.RunContextuallyInParallel(
		ctx,
		func(ctx context.Context) error {
			buildkitClient, err := client.New(ctx, build.BuildkitDaemonAddr, client.WithFailFast())
			if err != nil {
				buildInterface.FailJob(serviceID, err)
				buildLogger.Log(serviceID, time.Now(), "Could not connect to build daemon! ", err.Error())
				return err
			}
			buildLogger.Log(serviceID, time.Now(), "Starting build of ", serviceDir)
			buildOpts, err := buildOptions(serviceDir, buildTag)
			if err != nil {
				buildInterface.FailJob(serviceID, err)
				buildLogger.Log(serviceID, time.Now(), "Could not resolve build options (e.g., where to push to)!", err.Error())
				return err
			}
			solveStatus, err := buildkitClient.Build(ctx, *buildOpts, "", dockerfile.Build, statusChannel)
			if solveStatus != nil {
				//TODO if this is null should print a warning that we failed to push
				//e.g., when we haven't deployed yet
				for k, v := range solveStatus.ExporterResponse {
					buildLogger.Log(serviceID, time.Now(), fmt.Sprintf("exporter: %s=%s", k, v))
				}
			}
			if err != nil {
				buildLogger.Log(serviceID, time.Now(), "FAILED: ", err.Error())
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
					logErr := buildLogger.ProcessStatus(serviceID, status)
					if logErr != nil {
						fmt.Fprintln(os.Stderr, logErr.Error())
					}
				}
			}
		},
	)

	if err == nil {
		buildInterface.SucceedJob(serviceID)
		buildLogger.Log(serviceID, time.Now(), "Build succeeded!")
	} else if err != context.Canceled {
		buildInterface.FailJob(serviceID, err)
		buildLogger.Log(serviceID, time.Now(), "Build failed! ", err.Error())
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

	buildTag, err := git.GetCurrentTreeHash(s.GetSanicRoot(), services...)
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
				buildTag[:12],
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
