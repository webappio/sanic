package commands

import (
	"fmt"
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"syscall"
)

func kubectlCommandAction(cliContext *cli.Context) error {
	cfg, err := config.Read()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	s, err := shell.Current()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	provisioner, err := getProvisioner()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return cli.NewExitError("the kubernetes configuration doesn't exist yet, use sanic deploy first", 1)
	}
	kubeExecutableLocation, err := exec.LookPath("kubectl")
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("could not find kubectl, is it installed? %s", err.Error()), 1)
	}

	env, err := getKubectlEnvironment()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	args := []string{kubeExecutableLocation}
	if env, err := cfg.CurrentEnvironment(s); err == nil {
		if env.Namespace != "" {
			args = append(args, "--namespace="+env.Namespace)
		}
	}
	args = append(args, cliContext.Args()...)

	err = syscall.Exec(kubeExecutableLocation, args, env)
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
