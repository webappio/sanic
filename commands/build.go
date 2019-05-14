package commands

import "github.com/urfave/cli"

var BuildCommand = cli.Command {
	Name:	"build",
	Usage:	"build some (or all, by default) services",
	Action:  func(c *cli.Context) error {
		return nil
	},
}