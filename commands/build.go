package commands

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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
		} else {
			fmt.Fprintf(os.Stderr, "Failed to launch interactive interface: %s\n", err.Error())
		}
	}
	return build.NewPlaintextInterface()
}

//adapted from
//https://web.archive.org/web/20190516153923/https://raw.githubusercontent.com/moby/buildkit/master/examples/build-using-dockerfile/main.go
func buildCommandAction(cliContext *cli.Context) error {
	projectRoot := getProjectRootPath()
	if projectRoot == "" {
		return cli.NewExitError("you must be in an environment to build, see 'sanic env'", 1)
	}
	services, err := findServices(projectRoot)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	buildInterface := createBuildInterface(cliContext.Bool("plain-interface"))
	buildLogger := build.FlatfileLogger{
		LogDirectory: filepath.Join(projectRoot, "logs"),
	}
	defer buildLogger.Close()
	defer buildInterface.Close()

	var buildJobsGroup sync.WaitGroup
	jobErrorsChannel := make(chan error)
	logErrorsChannel := make(chan error, 1024)

	for _, serviceDir := range services {
		serviceName := filepath.Base(serviceDir)

		ctx := appcontext.Context()

		c, err := client.New(ctx, cliContext.String("buildkit-addr"), client.WithFailFast())
		if err != nil {
			return err
		}
		pipeR, pipeW := io.Pipe()
		solveOpt, err := solveOpt(serviceDir, pipeW)
		if err != nil {
			return err
		}

		buildInterface.StartJob(serviceName)

		statusChannel := make(chan *client.SolveStatus)
		eg, ctx := errgroup.WithContext(ctx)
		eg.Go(func() error {
			_, err = c.Build(ctx, *solveOpt, "", dockerfile.Build, statusChannel)
			pipeR.CloseWithError(err)
			if err != nil {
				buildInterface.FailJob(serviceName, err)
			} else {
				buildInterface.SucceedJob(serviceName)
			}
			return err
		})
		eg.Go(func() error {
			if err := loadDockerTar(pipeR); err != nil {
				return err
			}
			return pipeR.Close()
		})
		eg.Go(func() error {
			for status := range statusChannel {
				logErr := buildLogger.ProcessStatus(serviceName, status)
				if logErr != nil {
					logErrorsChannel <- logErr
				}
				buildInterface.ProcessStatus(serviceName, status)
			}
			return nil
		})

		buildJobsGroup.Add(1)
		go func() {
			defer buildJobsGroup.Done()
			if err := eg.Wait(); err != nil {
				buildInterface.FailJob(serviceName, err)
				jobErrorsChannel <- err
			}
		}()
	}

	buildJobsGroup.Wait()
	close(jobErrorsChannel)
	close(logErrorsChannel)
	for logErr := range logErrorsChannel {
		fmt.Fprintf(os.Stderr, "Error while attempting to log: %s\n", logErr)
	}
	var jobErrors []error
	for job := range jobErrorsChannel {
		if job != nil {
			jobErrors = append(jobErrors, job)
		}
	}
	if len(jobErrors) != 0 {
		return cli.NewExitError(cli.NewMultiError(jobErrors...).Error(), 1)
	}

	return nil
}

func solveOpt(serviceDir string, w io.WriteCloser) (*client.SolveOpt, error) {
	return &client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				Type: "docker", // TODO: use containerd image store when it is integrated to Docker
				Attrs: map[string]string{
					"name": filepath.Base(serviceDir) + ":latest",
				},
				Output: w,
			},
		},
		LocalDirs: map[string]string{
			"context":    serviceDir,
			"dockerfile": serviceDir,
		},
	}, nil
}

func loadDockerTar(r io.Reader) error {
	cmd := exec.Command("docker", "load") //TODO hack
	cmd.Stdin = r
	//cmd.Stdout = os.Stdout intentionally ignored
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var BuildCommand = cli.Command{
	Name:   "build",
	Usage:  "build some (or all, by default) services",
	Action: buildCommandAction,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:   "buildkit-addr",
			Usage:  "buildkit daemon address",
			EnvVar: "BUILDKIT_HOST",
			Value:  "tcp://0.0.0.0:2149", //see hack/start_buildkitd.sh
		},
		cli.BoolFlag{
			Name:   "plain-interface",
			Usage:  "use a plaintext interface",
			EnvVar: "PLAINTEXT_INTERFACE",
		},
	},
}
