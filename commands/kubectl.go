package commands

import (
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
		return cli.NewExitError(err.Error(), 1)
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return cli.NewExitError("the kubernetes configuration doesn't exist yet, use sanic deploy first", 1)
	}
	kubeExecutableLocation, err := exec.LookPath("kubectl")
	if err != nil {
		return cli.NewExitError(fmt.Sprintf(
			"the kubectl executable was not found on your PATH, it needs to be installed manually. Error:%s\n",
			err.Error()), 1)
	}
	env := append(os.Environ(), "KUBECONFIG="+kubeConfigLocation)
	err = syscall.Exec(kubeExecutableLocation, append([]string{kubeExecutableLocation}, cliContext.Args()...), env)
	if err != nil {
		return cli.NewExitError(err, 1)
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
