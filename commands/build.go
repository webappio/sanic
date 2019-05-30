package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
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

func loadDockerTar(ctx context.Context, r io.Reader) error {
	cmd := exec.Command("docker", "load") //TODO hack
	cmd.Stdin = r
	//cmd.Stdout = os.Stdout intentionally ignored
	cmd.Stderr = os.Stderr
	cmd.Start()

	processDone := make(chan error)
	go func() {
		processDone <- cmd.Wait()
		close(processDone)
	}()

	select {
	case err := <-processDone:
		return err
	case <-ctx.Done():
		cmd.Process.Kill()
		return ctx.Err()
	}
}

func buildService(
	ctx context.Context,
	serviceDir string,
	buildkitAddress string,
	logErrorsChannel chan error,
	buildLogger build.Logger) error {

	serviceName := filepath.Base(serviceDir)
	c, err := client.New(ctx, buildkitAddress, client.WithFailFast())
	if err != nil {
		return err
	}
	pipeR, pipeW := io.Pipe()

	statusChannel := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)
	buildDone := make(chan interface{})
	eg.Go(func() error {
		buildLogger.Log(serviceName, time.Now(), "Starting build of ", serviceDir)
		_, err = c.Build(
			ctx,
			client.SolveOpt{
				Exports: []client.ExportEntry{
					{
						Type: "docker",
						Attrs: map[string]string{
							"name": serviceName + ":latest",
						},
						Output: pipeW,
					},
				},
				LocalDirs: map[string]string{
					"context":    serviceDir,
					"dockerfile": serviceDir,
				},
			},
			"",
			dockerfile.Build,
			statusChannel)
		pipeR.CloseWithError(err)
		if err != nil {
			buildLogger.Log(serviceName, time.Now(), "FAILED: ", err.Error())
		}
		return err
	})
	eg.Go(func() error {
		if err := loadDockerTar(ctx, pipeR); err != nil {
			return err
		}
		buildDone <- true
		return pipeR.Close()
	})
	eg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case status, ok := <-statusChannel:
				if !ok {
					return nil
				}
				logErr := buildLogger.ProcessStatus(serviceName, status)
				if logErr != nil {
					logErrorsChannel <- logErr
				}
			}
		}
	})

	select {
	case <-ctx.Done(): //cancelled (e.g., ctrl+c or error returned from goroutine)
		return ctx.Err()
	case <-buildDone: //successfully built + loaded
		return nil
	}
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

	buildInterface := createBuildInterface(cliContext.Bool("plain-interface"))
	defer buildInterface.Close()

	buildLogger := build.NewFlatfileLogger(filepath.Join(s.GetSanicRoot(), "logs"))
	buildLogger.AddLogLineListener(buildInterface.ProcessLog)
	defer buildLogger.Close()

	jobErrorsChannel := make(chan error)
	logErrorsChannel := make(chan error, 1024)
	buildingJobs := 0

	for _, serviceDir := range services {
		serviceName := filepath.Base(serviceDir)
		ctx, cancelJob := context.WithCancel(context.Background())
		buildInterface.AddCancelListener(cancelJob)

		finalServiceDir := serviceDir

		buildInterface.StartJob(serviceName)
		buildLogger.Log(serviceName, time.Now(), "Queued for building.")
		go func() {
			jobError := buildService(
				ctx,
				finalServiceDir,
				cliContext.String("buildkit-addr"),
				logErrorsChannel,
				buildLogger)
			if jobError == nil {
				buildLogger.Log(serviceName, time.Now(), "Build succeeded!")
				buildInterface.SucceedJob(serviceName)
			} else {
				buildInterface.FailJob(serviceName, jobError)
				buildLogger.Log(serviceName, time.Now(), "Build failed! ", jobError.Error())
			}
			jobErrorsChannel <- jobError
		}()
		buildingJobs++
	}

	var jobErrors []error
	for i := 0; i < buildingJobs; i++ {
		jobError := <-jobErrorsChannel
		if jobError == context.Canceled {
			fmt.Println() //clear the ^C
			return cli.NewExitError("", 1)
		}
		if jobError != nil {
			jobErrors = append(jobErrors, jobError)
		}
	}

	close(jobErrorsChannel)
	close(logErrorsChannel)
	for logErr := range logErrorsChannel {
		fmt.Fprintf(os.Stderr, "Error while attempting to log: %s\n", logErr)
	}
	if len(jobErrors) != 0 {
		return cli.NewExitError(cli.NewMultiError(jobErrors...).Error(), 1)
	}

	return nil
}

var buildCommand = cli.Command{
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
