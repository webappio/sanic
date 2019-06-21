package commands

import (
	"errors"
	"fmt"
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/distributed-containers-inc/sanic/shell"
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
	provisioner, err := getProvisioner()
	if err != nil {
		return nil, err
	}
	kubeConfigLocation := provisioner.KubeConfigLocation()
	if _, err := os.Stat(kubeConfigLocation); os.IsNotExist(err) {
		return nil, errors.New("the kubernetes configuration doesn't exist yet, use sanic deploy first if in localdev")
	}
	return append(os.Environ(), "KUBECONFIG="+kubeConfigLocation), nil
}

func getProvisioner() (provisioners.Provisioner, error) {
	s, err := shell.Current()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Read()
	if err != nil {
		return nil, err
	}

	env, err := cfg.CurrentEnvironment(s)
	if err != nil {
		return nil, err
	}

	if env.ClusterProvisioner == "" {
		return nil, errors.New("the environment " + s.GetSanicEnvironment() +
			" does not have a 'clusterProvisioner' key defined in it. Try clusterProvisioner: localdev to start.")
	}
	return provisioners.GetProvisioner(env.ClusterProvisioner, env.ClusterProvisionerArgs), nil
}
