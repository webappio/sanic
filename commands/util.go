package commands

import (
	"errors"
	"fmt"
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/urfave/cli"
	"os"
)

func newUsageError(ctx *cli.Context) error {
	argsUsage := ctx.Command.ArgsUsage
	if argsUsage == "" {
		argsUsage = "[arguments ...]"
	}
	return cli.NewExitError(fmt.Sprintf(
		"Incorrect usage.\nCorrect usage: %s %s",
		ctx.Command.HelpName, argsUsage),
		1)
}

func getKubectlEnvironment() ([]string, error) {
	provisioner, err := provisioners.GetProvisioner()
	if err != nil {
		return nil, err
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return nil, errors.New("the kubernetes configuration doesn't exist yet, use sanic deploy first")
	}
	return append(os.Environ(), "KUBECONFIG="+kubeConfigLocation), nil
}