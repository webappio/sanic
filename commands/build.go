package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/bridge/git"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/distributed-containers-inc/sanic/util"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
)

func getRegistry() (registryAddr string, registryInsecure bool, err error) {
	provisioner, err := getProvisioner()

	if err != nil {
		return
	}

	return provisioner.Registry()
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

//adapted from
//https://web.archive.org/web/20190516153923/https://raw.githubusercontent.com/moby/buildkit/master/examples/build-using-dockerfile/main.go
func buildCommandAction(cliContext *cli.Context) error {
	registry := ""
	registryInsecure := false
	if cliContext.Bool("push") {
		var err error
		registry, registryInsecure, err = getRegistry()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("could not get registry to push to: %s", err.Error()), 1)
		}
		provisioner, err := getProvisioner()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("you must be in an environment with a provisioner to use --push while building: %s", err.Error()), 1)
		}
		fmt.Println("Because you specified --push, sanic is ensuring that there is a valid cluster running...")
		err = provisioner.EnsureCluster()
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}

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

	buildLogger := build.NewFlatfileLogger(filepath.Join(s.GetSanicRoot(), "logs"), cliContext.Bool("verbose"))
	buildLogger.AddLogLineListener(buildInterface.ProcessLog)
	defer buildLogger.Close()

	jobs := make([]func(context.Context) error, 0, len(services))

	builder := build.Builder{
		Registry: registry,
		RegistryInsecure: registryInsecure,
		BuildTag: buildTag,
		Logger: buildLogger,
		Interface: buildInterface,
		DoPush: cliContext.Bool("push"),
	}

	for _, serviceDir := range services {
		finalServiceDir := serviceDir
		jobs = append(jobs, func(ctx context.Context) error {
			return builder.BuildService(
				ctx,
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
		cli.BoolFlag{
			Name:  "push",
			Usage: "pushes to the configured registry for the current environment instead of loading locally",
		},
		cli.BoolFlag{
			Name: "verbose",
			Usage: "enables verbose logging, mostly for sanic development",
		},
	},
}
