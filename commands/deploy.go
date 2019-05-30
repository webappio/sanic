package commands

import (
	"github.com/distributed-containers-inc/sanic/provisioners"
	"github.com/urfave/cli"
)

func deployCommandAction(cliContext *cli.Context) error {
	err := provisioners.EnsureCluster()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

var deployCommand = cli.Command{
	Name:   "deploy",
	Usage:  "deploy some (or all, by default) services",
	Action: deployCommandAction,
}
