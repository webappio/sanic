package commands

import (
	"errors"
	"fmt"
	"github.com/distributed-containers-inc/sanic/pkg/config"
	"github.com/distributed-containers-inc/sanic/pkg/provisioners"
	"github.com/distributed-containers-inc/sanic/pkg/provisioners/provisioner"
	"github.com/distributed-containers-inc/sanic/pkg/shell"
	"github.com/urfave/cli"
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

func getProvisioner() (provisioner.Provisioner, error) {
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
