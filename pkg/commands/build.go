package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/pkg/bridge/git"
	"github.com/distributed-containers-inc/sanic/pkg/build"
	"github.com/distributed-containers-inc/sanic/pkg/config"
	"github.com/distributed-containers-inc/sanic/pkg/shell"
	"github.com/distributed-containers-inc/sanic/pkg/util"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"sync"
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

//adapted from
//https://web.archive.org/web/20190516153923/https://raw.githubusercontent.com/moby/buildkit/master/examples/build-using-dockerfile/main.go
func buildCommandAction(cliContext *cli.Context) error {
	registry := ""
	registryInsecure := false
	if addr := cliContext.String("registry"); addr != "" {
		registry = addr
	} else if cliContext.Bool("push") {
		provisioner, err := getProvisioner()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("you must be in an environment with a provisioner to use --push while building: %s", err.Error()), 1)
		}

		registry, registryInsecure, err = provisioner.Registry()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("you specified --push, but a registry was not found: %s. Try \"sanic deploy\" first.", err.Error()), 1)
		}

		if registryInsecure {
			err := provisioner.CheckRegistryInsecureOK()
			if err != nil {
				return cli.NewExitError(fmt.Sprintf("we can't push to the registry: %v", err), 1)
			}
		}
	}

	var buildRoot string
	s, err := shell.Current()
	if err != nil {
		fmt.Fprintln(os.Stderr, "[WARNING] sanic is building dockerfiles recursively in your current directory. It's recommended to use a sanic environment for consistency.")
		buildRoot, err = os.Getwd()
		if err != nil {
			return cli.NewExitError(fmt.Sprintf("error while getting current directory: %s", err.Error()), 1)
		}
	} else {
		buildRoot = s.GetSanicRoot()
	}

	var ignorePaths []string
	if cfg, err := config.Read(); err == nil {
		ignorePaths = cfg.Build.IgnoreDirs
	}
	services, err := util.FindServices(buildRoot, ignorePaths)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if len(services) == 0 {
		return cli.NewExitError(fmt.Sprintf("%s (or some of its subdirectories) should contain a Dockerfile"), 1)
	}


	buildTag := cliContext.String("tag")
	if buildTag == "" {
		var serviceDirs []string
		for _, service := range services {
			serviceDirs = append(serviceDirs, service.Dir)
		}
		buildTag, err = git.GetCurrentTreeHash(buildRoot, serviceDirs...)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
	}

	buildInterface := createBuildInterface(cliContext.Bool("plaintext"))
	defer func() {
		r := recover()
		buildInterface.Close()
		if r != nil {
			panic(r)
		}
	}()

	buildLogger := build.NewFlatfileLogger(filepath.Join(buildRoot, "logs"), cliContext.Bool("verbose"))
	buildLogger.AddLogLineListener(buildInterface.ProcessLog)
	defer buildLogger.Close()

	builder := build.Builder{
		Registry:         registry,
		RegistryInsecure: registryInsecure,
		BuildTag:         buildTag,
		Logger:           buildLogger,
		Interface:        buildInterface,
		DoPush:           cliContext.Bool("push"),
	}

	buildFailed := false

	var wg sync.WaitGroup
	for _, service := range services {
		finalService := service
		go func() {
			ctx, cancelJob := context.WithCancel(context.Background())
			buildInterface.AddCancelListener(cancelJob)
			err := builder.BuildService(
				ctx,
				finalService,
			)
			if err != nil {
				buildFailed = true
				buildLogger.Log(finalService.Name, time.Now(), "Error: ", err.Error())
			}
			wg.Done()
		}()
		wg.Add(1)
	}

	userCancelledBuild := false
	buildInterface.AddCancelListener(func() { userCancelledBuild = true })

	wg.Wait()

	if userCancelledBuild {
		fmt.Println() //clear the ^C
		return cli.NewExitError("", 1)
	}

	if buildFailed {
		return cli.NewExitError("", 1)
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
		cli.StringFlag{
			Name:  "tag,t",
			Usage: "sets the tag of all built images to the specified one",
		},
		cli.StringFlag{
			Name:  "registry",
			Usage: "sets the registry of all built images to the specified one (i.e., for use with --push)",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "enables verbose logging, mostly for sanic development",
		},
	},
}
