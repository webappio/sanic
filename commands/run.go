package commands

import (
	"github.com/distributed-containers-inc/sanic/config"
	"github.com/distributed-containers-inc/sanic/shell"
	"github.com/urfave/cli"
	"os"
)

func runCommandAction(c *cli.Context) error {
	if c.NArg() == 0 {
		return newUsageError(c)
	}

	sanicEnv := getSanicEnv()
	configPath := getSanicConfigPath()
	if sanicEnv == "" || configPath == "" {
		return cli.NewExitError("you must be in an environment to use this command. see sanic env", 1)
	}

	cfg, err := config.Read()
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	env, err := config.CurrentEnvironment(cfg)
	if err != nil {
		return wrapErrorWithExitCode(err, 1)
	}

	commandName := c.Args().First()
	for _, command := range env.Commands {
		if command.Name == commandName {
			err, code := shell.Exec(sanicEnv, configPath, command.Command)
			if code == 0 {
				return nil
			} else {
				return wrapErrorWithExitCode(err, code)
			}
		}
	}
	return cli.NewExitError("Command "+commandName+" was not found in environment "+os.Getenv("SANIC_ENV")+".", 1)

}

var RunCommand = cli.Command{
	Name:   "run",
	Usage:  "run a configured script in the configuration",
	Action: runCommandAction,
}
