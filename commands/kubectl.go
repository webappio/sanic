package commands

import (
	"errors"
	"fmt"
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"syscall"
)

func kubectlCommandAction(cliContext *cli.Context) error {
	provisioner, err := provisioners.GetProvisioner()
	if err != nil {
		return err
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return errors.New("the kubernetes configuration doesn't exist yet, use sanic deploy first")
	}
	kubeExecutableLocation, err := exec.LookPath("kubectl")
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not find kubectl, is it installed? %s", err.Error()), 1)
	}

	env, err := getKubectlEnvironment()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	err = syscall.Exec(kubeExecutableLocation, append([]string{kubeExecutableLocation}, cliContext.Args()...), env)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

var kubectlCommand = cli.Command{
	Name:            "kubectl",
	Usage:           "a wrapper around the base kubectl command, configured to use the current cluster",
	Action:          kubectlCommandAction,
	SkipArgReorder:  true,
	SkipFlagParsing: true,
}
