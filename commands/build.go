package commands

import (
	"context"
	"fmt"
	"github.com/distributed-containers-inc/sanic/build"
	"github.com/moby/buildkit/client"
	dockerfile "github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/urfave/cli"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	serviceDir string,
	buildkitAddress string,
	ctx context.Context,
	logErrorsChannel chan error,
	buildInterface build.Interface,
	buildLogger build.Logger) error {

	serviceName := filepath.Base(serviceDir)
	c, err := client.New(ctx, buildkitAddress, client.WithFailFast())
	if err != nil {
		return err
	}
	pipeR, pipeW := io.Pipe()

	buildInterface.StartJob(serviceName)

	statusChannel := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)
	buildDone := make(chan interface{})
	eg.Go(func() error {
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
			buildInterface.FailJob(serviceName, err)
		} else {
			buildInterface.SucceedJob(serviceName)
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
				buildInterface.ProcessStatus(serviceName, status)
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
	projectRoot := getProjectRootPath()
	if projectRoot == "" {
		return cli.NewExitError("you must be in an environment to build, see 'sanic env'", 1)
	}
	services, err := findServices(projectRoot)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	buildInterface := createBuildInterface(cliContext.Bool("plain-interface"))
	defer buildInterface.Close()

	buildLogger := build.NewFlatfileLogger(filepath.Join(projectRoot, "logs"))
	defer buildLogger.Close()

	jobErrorsChannel := make(chan error)
	logErrorsChannel := make(chan error, 1024)
	buildingJobs := 0

	for _, serviceDir := range services {
		ctx, cancelJob := context.WithCancel(context.Background())
		buildInterface.AddCancelListener(cancelJob)

		finalServiceDir := serviceDir

		go func() {
			jobErrorsChannel <- buildService(
				finalServiceDir,
				cliContext.String("buildkit-addr"),
				ctx,
				logErrorsChannel,
				buildInterface,
				buildLogger)
		}()
		buildingJobs += 1
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
