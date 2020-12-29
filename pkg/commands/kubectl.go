package commands

import (
	"github.com/layer-devops/sanic/pkg/config"
	"github.com/layer-devops/sanic/pkg/shell"
	"github.com/urfave/cli"
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

	args := []string{}
	if env, err := cfg.CurrentEnvironment(s); err == nil {
		if env.Namespace != "" {
			args = append(args, "--namespace="+env.Namespace)
		}
	}

	args = append(args, cliContext.Args()...)
	cmd, err := provisioner.KubectlCommand(args...)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
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
