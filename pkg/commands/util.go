package commands

import (
	"errors"
	"fmt"
	"github.com/webappio/sanic/pkg/config"
	"github.com/webappio/sanic/pkg/provisioners"
	"github.com/webappio/sanic/pkg/provisioners/provisioner"
	"github.com/webappio/sanic/pkg/shell"
	"github.com/urfave/cli"
	"strings"
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

func getNamespaceFromArgs(args []string) (string, error) {
	for i, arg := range args {
		if arg == "--namespace" || arg == "-n" {
			if (i + 1) >= len(args) {
				return "", errors.New("no argument provided with namespace flag")
			}
			return args[i+1], nil
		}

		if strings.Contains(arg, "--namespace=") || strings.Contains(arg, "-n=") {
			split := strings.SplitAfterN(arg, "=", 2)
			if len(split) != 2 {
				return "", errors.New("no argument provided with namespace flag")
			}
			return split[1], nil
		}
	}

	return "", nil
}

func getNamespaceFromEnv() (string, error) {
	cfg, err := config.Read()
	if err != nil {
		return "", err
	}

	s, err := shell.Current()
	if err != nil {
		return "", err
	}

	env, err := cfg.CurrentEnvironment(s);
	if err != nil {
		return "", err
	}

	return env.Namespace, nil
}
