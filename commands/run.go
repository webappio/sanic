package commands

import (
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/urfave/cli"
)

func runCommandAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return newUsageError(c)
	}

	s, err := shell.Current()
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	cfg, err := config.Read()
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	env, err := cfg.CurrentEnvironment(s)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	commandName := c.Args().First()
	for _, command := range env.Commands {
		if command.Name == commandName {
			code, err := s.ShellExec(command.Command)
			if code == 0 {
				return nil
			}
			return wrapErrorWithExitCode(err, code)
		}
	}
	return cli.NewExitError("Command "+commandName+" was not found in environment "+s.GetSanicEnvironment(), 1)

}

var runCommand = cli.Command{
	Name:   "run",
	Usage:  "run a configured script in the configuration",
	Action: runCommandAction,
}
